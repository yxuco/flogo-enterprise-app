package common

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"

	"github.com/TIBCOSoftware/flogo-lib/logger"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/errors/retry"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
)

const (
	configType = "yaml"
)

// cached Fabric client connections
var clientMap = map[string]*FabricClient{}

// FabricClient holds fabric client pointers for chaincode invocations.
type FabricClient struct {
	name   string
	sdk    *fabsdk.FabricSDK
	client *channel.Client
}

// SetFlogoLogLevel sets log level of a Flogo app logger according to env 'FLOGO_LOG_LEVEL'
func SetFlogoLogLevel(log logger.Logger) {
	loglevel := "DEBUG"
	if l, ok := os.LookupEnv("FLOGO_LOG_LEVEL"); ok {
		loglevel = l
	}
	if level, err := logger.GetLevelForName(loglevel); err != nil {
		log.SetLogLevel(level)
	} else {
		log.SetLogLevel(logger.DebugLevel)
	}
}

// ConnectorSpec contains configuration parameters of a Fabric connector
type ConnectorSpec struct {
	Name           string
	NetworkConfig  []byte
	EntityMatchers []byte
	OrgName        string
	UserName       string
	ChannelID      string
}

// NewFabricClient returns a new or cached fabric client
func NewFabricClient(config ConnectorSpec) (*FabricClient, error) {
	if fbClient, ok := clientMap[config.Name]; ok && fbClient != nil {
		return fbClient, nil
	}
	sdk, err := fabsdk.New(networkConfigProvider(config.NetworkConfig, config.EntityMatchers))
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create new SDK")
	}

	client, err := channel.New(sdk.ChannelContext(config.ChannelID, fabsdk.WithUser(config.UserName), fabsdk.WithOrg(config.OrgName)))
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create new client of channel %s", config.ChannelID)
	}
	fbClient := &FabricClient{
		name:   config.Name,
		sdk:    sdk,
		client: client,
	}
	clientMap[config.Name] = fbClient

	return fbClient, nil
}

func networkConfigProvider(networkConfig []byte, entityMatcherOverride []byte) core.ConfigProvider {
	configProvider := config.FromRaw(networkConfig, configType)

	if entityMatcherOverride != nil {
		return func() ([]core.ConfigBackend, error) {
			matcherProvider := config.FromRaw(entityMatcherOverride, configType)
			matcherBackends, err := matcherProvider()
			if err != nil {
				fmt.Printf("failed to parse entity matchers: %+v\n", err)
				// return the original config provider defined by configPath
				return configProvider()
			}

			currentBackends, err := configProvider()
			if err != nil {
				fmt.Printf("failed to parse network config: %+v\n", err)
				return nil, err
			}

			// return the combined config with matcher precedency
			return append(matcherBackends, currentBackends...), nil
		}
	}
	return configProvider
}

// Close closes Fabric client connection
func (c *FabricClient) Close() {
	c.sdk.Close()
}

// QueryChaincode sends query request to Fabric network
func (c *FabricClient) QueryChaincode(ccID, fcn string, args [][]byte, transient map[string][]byte) ([]byte, error) {
	response, err := c.client.Query(channel.Request{ChaincodeID: ccID, Fcn: fcn, Args: args, TransientMap: transient},
		channel.WithRetry(retry.DefaultChannelOpts),
	)
	if err != nil {
		return nil, err
	}
	return response.Payload, nil
}

// ExecuteChaincode sends invocation request to Fabric network
func (c *FabricClient) ExecuteChaincode(ccID, fcn string, args [][]byte, transient map[string][]byte) ([]byte, error) {
	response, err := c.client.Execute(channel.Request{ChaincodeID: ccID, Fcn: fcn, Args: args, TransientMap: transient},
		channel.WithRetry(retry.DefaultChannelOpts),
	)
	if err != nil {
		return nil, err
	}
	return response.Payload, nil
}

// ReadFile returns content of a specified file
func ReadFile(filePath string) ([]byte, error) {
	f, err := os.Open(Subst(filePath))
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to open file: %s", filePath)
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to read file stat: %s", filePath)
	}
	s := fi.Size()
	cBytes := make([]byte, s)
	n, err := f.Read(cBytes)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to read file: %s", filePath)
	}
	if n == 0 {
		fmt.Printf("file %s is empty\n", filePath)
	}
	return cBytes, err
}

// Subst replaces instances of '${VARNAME}' (eg ${GOPATH}) with the variable.
// Variables names that are not set by the SDK are replaced with the environment variable.
func Subst(path string) string {
	const (
		sepPrefix = "${"
		sepSuffix = "}"
	)

	splits := strings.Split(path, sepPrefix)

	var buffer bytes.Buffer

	// first split precedes the first sepPrefix so should always be written
	buffer.WriteString(splits[0]) // nolint: gas

	for _, s := range splits[1:] {
		subst, rest := substVar(s, sepPrefix, sepSuffix)
		buffer.WriteString(subst) // nolint: gas
		buffer.WriteString(rest)  // nolint: gas
	}

	return buffer.String()
}

// substVar searches for an instance of a variables name and replaces them with their value.
// The first return value is substituted portion of the string or noMatch if no replacement occurred.
// The second return value is the unconsumed portion of s.
func substVar(s string, noMatch string, sep string) (string, string) {
	endPos := strings.Index(s, sep)
	if endPos == -1 {
		return noMatch, s
	}

	v, ok := os.LookupEnv(s[:endPos])
	if !ok {
		return noMatch, s
	}

	return v, s[endPos+1:]
}
