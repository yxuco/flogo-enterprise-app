package fabrequest

import (
	"encoding/json"
	"fmt"

	"github.com/TIBCOSoftware/flogo-lib/core/activity"
	"github.com/TIBCOSoftware/flogo-lib/core/data"
	"github.com/TIBCOSoftware/flogo-lib/logger"
	"github.com/pkg/errors"
	client "github.com/yxuco/flogo-enterprise-app/fabclient/common"
	"github.com/yxuco/flogo-enterprise-app/fabric/common"
)

const (
	ivConnection     = "connectionName"
	ivRequestType    = "requestType"
	ivChaincode      = "chaincodeID"
	ivTransaction    = "transactionName"
	ivParameters     = "parameters"
	ovCode           = "code"
	ovMessage        = "message"
	ovResult         = "result"
	conName          = "name"
	conConfig        = "config"
	conEntityMatcher = "entityMatcher"
	conChannel       = "channelID"
	conOrgName       = "orgName"
	conUserName      = "userName"
	opInvoke         = "invoke"
	opQuery          = "query"
)

// Create a new logger
var log = logger.GetLogger("activity-fabclient-request")

func init() {
	client.SetFlogoLogLevel(log)
}

// FabricRequestActivity is a stub for sending Hyperledger Fabric invoke/query request
type FabricRequestActivity struct {
	metadata *activity.Metadata
}

// NewActivity creates a new FabricRequestActivity
func NewActivity(metadata *activity.Metadata) activity.Activity {
	return &FabricRequestActivity{metadata: metadata}
}

// Metadata implements activity.Activity.Metadata
func (a *FabricRequestActivity) Metadata() *activity.Metadata {
	return a.metadata
}

// Eval implements activity.Activity.Eval
func (a *FabricRequestActivity) Eval(ctx activity.Context) (done bool, err error) {
	// check input args
	ccID, ok := ctx.GetInput(ivChaincode).(string)
	if !ok || ccID == "" {
		log.Error("chaincode ID is not specified\n")
		ctx.SetOutput(ovCode, 400)
		ctx.SetOutput(ovMessage, "chaincode ID is not specified")
		return false, errors.New("chaincode ID is not specified")
	}
	log.Debugf("chaincode ID: %s\n", ccID)
	txName, ok := ctx.GetInput(ivTransaction).(string)
	if !ok || txName == "" {
		log.Error("transaction name is not specified\n")
		ctx.SetOutput(ovCode, 400)
		ctx.SetOutput(ovMessage, "transaction name is not specified")
		return false, errors.New("transaction name is not specified")
	}
	log.Debugf("transaction name: %s\n", txName)
	reqType, ok := ctx.GetInput(ivRequestType).(string)
	if !ok || reqType == "" {
		log.Warn("request type is not specified, assume `query`\n")
		reqType = opQuery
	}
	log.Debugf("request type: %s\n", reqType)

	params, err := getParameters(ctx)
	if err != nil {
		ctx.SetOutput(ovCode, 400)
		ctx.SetOutput(ovMessage, fmt.Sprintf("invalid parameters: %+v", err))
		return false, err
	}
	client, err := getFabricClient(ctx)
	if err != nil {
		ctx.SetOutput(ovCode, 500)
		ctx.SetOutput(ovMessage, fmt.Sprintf("fabric connector failure: %+v", err))
		return false, err
	}

	// invoke fabric transaction
	var response []byte
	if reqType == opInvoke {
		response, err = client.ExecuteChaincode(ccID, txName, params)
	} else {
		response, err = client.QueryChaincode(ccID, txName, params)
	}

	if err != nil {
		log.Errorf("Fabric returned error %+v\n", err)
		ctx.SetOutput(ovCode, 500)
		ctx.SetOutput(ovMessage, fmt.Sprintf("Fabric request returned error: %+v", err))
		return false, errors.Wrapf(err, "Fabric request returned error")
	}

	if response == nil {
		log.Debugf("no data returned from fabric\n")
		ctx.SetOutput(ovCode, 300)
		ctx.SetOutput(ovMessage, "no data returned from fabric")
		return true, nil
	}
	var value interface{}
	if err := json.Unmarshal(response, &value); err != nil {
		log.Errorf("failed to unmarshal fabric response %+v, error: %+v\n", response, err)
		ctx.SetOutput(ovCode, 300)
		ctx.SetOutput(ovMessage, fmt.Sprintf("data returned from fabric is not JSON: %v", response))
		return true, nil
	}
	if result, ok := ctx.GetOutput(ovResult).(*data.ComplexObject); ok && result != nil {
		log.Debugf("set activity output result: %+v\n", value)
		result.Value = value
		ctx.SetOutput(ovResult, result)
	}
	return true, nil
}

func getFabricClient(ctx activity.Context) (*client.FabricClient, error) {
	conn := ctx.GetInput(ivConnection)
	configs, err := client.GetSettings(conn)
	if err != nil {
		return nil, err
	}
	networkConfig, err := client.ExtractFileContent(configs[conConfig])
	if err != nil {
		return nil, errors.Wrapf(err, "invalid network config")
	}
	entityMatcher, err := client.ExtractFileContent(configs[conEntityMatcher])
	if err != nil {
		return nil, errors.Wrapf(err, "invalid entity-matchers-override")
	}
	return client.NewFabricClient(client.ConnectorSpec{
		Name:           configs[conName].(string),
		NetworkConfig:  networkConfig,
		EntityMatchers: entityMatcher,
		OrgName:        configs[conOrgName].(string),
		UserName:       configs[conUserName].(string),
		ChannelID:      configs[conChannel].(string),
	})
}

func getParameters(ctx activity.Context) ([][]byte, error) {
	var result [][]byte
	// extract parameter definitions from metadata
	paramObj, ok := ctx.GetInput(ivParameters).(*data.ComplexObject)
	if !ok {
		log.Debug("parameter is not a complex object\n")
		return result, nil
	}
	paramIndex, err := common.OrderedParameters([]byte(paramObj.Metadata))
	if err != nil {
		log.Errorf("failed to extract parameter definition from metadata: %+v\n", err)
		return result, nil
	}
	if len(paramIndex) == 0 {
		log.Debug("no parameter defined in metadata\n")
		return result, nil
	}

	// extract parameter values in the order of parameter index
	paramValue, ok := paramObj.Value.(map[string]interface{})
	if !ok {
		log.Debugf("parameter value of type %T is not a JSON object\n", paramObj.Value)
		return result, nil
	}
	for _, p := range paramIndex {
		// TODO: assuming string params here to be consistent with implementaton of trigger and chaincode-shim
		// should change all places to use []byte for best portability
		param := ""
		if v, ok := paramValue[p.Name]; ok && v != nil {
			param = fmt.Sprintf("%v", v)
			log.Debugf("add chaincode parameter: %s=%s", p.Name, param)
		}
		result = append(result, []byte(param))
	}
	return result, nil
}
