package get

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/TIBCOSoftware/flogo-lib/core/activity"
	"github.com/TIBCOSoftware/flogo-lib/core/data"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/pkg/errors"
	trigger "github.com/yxuco/flogo-enterprise-app/fabric/trigger/transaction"
)

const (
	ivKey           = "key"
	ivIsPrivate     = "isPrivate"
	ivCollection    = "collection"
	ivCompositeKeys = "compositeKeys"
	ovCode          = "code"
	ovMessage       = "message"
	ovKey           = "key"
	ovResult        = "result"
)

// Create a new logger
var log = shim.NewLogger("activity-fabric-get")

func init() {
	loglevel := "DEBUG"
	if l, ok := os.LookupEnv("CORE_CHAINCODE_LOGGING_LEVEL"); ok {
		loglevel = l
	}
	if level, err := shim.LogLevel(loglevel); err != nil {
		log.SetLevel(level)
	} else {
		log.SetLevel(shim.LogDebug)
	}
}

// FabricGetActivity is a stub for executing Hyperledger Fabric put operations
type FabricGetActivity struct {
	metadata *activity.Metadata
}

// NewActivity creates a new PutActivity
func NewActivity(metadata *activity.Metadata) activity.Activity {
	return &FabricGetActivity{metadata: metadata}
}

// Metadata implements activity.Activity.Metadata
func (a *FabricGetActivity) Metadata() *activity.Metadata {
	return a.metadata
}

// Eval implements activity.Activity.Eval
func (a *FabricGetActivity) Eval(ctx activity.Context) (done bool, err error) {
	// check input args
	key, ok := ctx.GetInput(ivKey).(string)
	if !ok || key == "" {
		log.Error("state key is not specified\n")
		ctx.SetOutput(ovCode, 400)
		ctx.SetOutput(ovMessage, "state key is not specified")
		return false, errors.New("state key is not specified")
	}
	log.Debugf("state key: %s\n", key)

	// get chaincode stub
	stub, err := resolveFlowData("$flow."+trigger.FabricStub, ctx)
	if err != nil || stub == nil {
		log.Errorf("failed to get stub: %+v\n", err)
		ctx.SetOutput(ovCode, 500)
		ctx.SetOutput(ovMessage, fmt.Sprintf("failed to get stub: %+v", err))
		return false, errors.Errorf("failed to get stub: %+v", err)
	}

	ccshim, ok := stub.(shim.ChaincodeStubInterface)
	if !ok {
		log.Errorf("stub type %T is not a ChaincodeStubInterface\n", stub)
		ctx.SetOutput(ovCode, 500)
		ctx.SetOutput(ovMessage, fmt.Sprintf("stub type %T is not a ChaincodeStubInterface", stub))
		return false, errors.Errorf("stub type %T is not a ChaincodeStubInterface", stub)
	}

	if isPrivate, ok := ctx.GetInput(ivIsPrivate).(bool); ok && isPrivate {
		// retrieve data from a private collection
		return retrievePrivateData(ctx, ccshim, key)
	}

	// retrieve data for the key
	return retrieveData(ctx, ccshim, key)
}

func retrievePrivateData(ctx activity.Context, ccshim shim.ChaincodeStubInterface, key string) (bool, error) {
	// retrieve data from a private collection
	collection, ok := ctx.GetInput(ivCollection).(string)
	if !ok || collection == "" {
		log.Error("private collection is not specified\n")
		ctx.SetOutput(ovCode, 400)
		ctx.SetOutput(ovMessage, "private collection is not specified")
		return false, errors.New("private collection is not specified")
	}
	jsonBytes, err := ccshim.GetPrivateData(collection, key)
	if err != nil {
		log.Errorf("failed to retrieve data from private collection %s: %+v\n", collection, err)
		ctx.SetOutput(ovCode, 500)
		ctx.SetOutput(ovMessage, fmt.Sprintf("failed to retrieve data from private collection %s: %+v", collection, err))
		return false, errors.Wrapf(err, "failed to retrieve data from private collection %s", collection)
	}
	if jsonBytes == nil {
		log.Infof("no data found for key %s on private collection %s\n", key, collection)
		ctx.SetOutput(ovCode, 300)
		ctx.SetOutput(ovMessage, fmt.Sprintf("no data found for key %s on private collection %s\n", key, collection))
		ctx.SetOutput(ovKey, key)
		return true, nil
	}
	log.Debugf("retrieved from private collection %s, data: %s\n", collection, string(jsonBytes))

	var value interface{}
	if err := json.Unmarshal(jsonBytes, &value); err != nil {
		log.Errorf("failed to parse JSON data: %+v\n", err)
		ctx.SetOutput(ovCode, 500)
		ctx.SetOutput(ovMessage, fmt.Sprintf("failed to parse JSON data: %+v", err))
		return false, errors.Wrapf(err, "failed to parse JSON data %s", string(jsonBytes))
	}

	ctx.SetOutput(ovCode, 200)
	ctx.SetOutput(ovMessage, fmt.Sprintf("retrieved data from private collection %s, data: %s", collection, string(jsonBytes)))
	if result, ok := ctx.GetOutput(ovResult).(*data.ComplexObject); ok && result != nil {
		log.Debugf("set activity output result: %+v\n", value)
		result.Value = value
		ctx.SetOutput(ovResult, result)
		ctx.SetOutput(ovKey, key)
	}
	return true, nil
}

func retrieveData(ctx activity.Context, ccshim shim.ChaincodeStubInterface, key string) (bool, error) {
	// retrieve data for the key
	jsonBytes, err := ccshim.GetState(key)
	if err != nil {
		log.Errorf("failed to retrieve data for key %s: %+v\n", key, err)
		ctx.SetOutput(ovCode, 500)
		ctx.SetOutput(ovMessage, fmt.Sprintf("failed to retrieve data for key %s: %+v", key, err))
		return false, errors.Wrapf(err, "failed to retrieve data for key %s", key)
	}
	if jsonBytes == nil {
		log.Infof("no data found for key %s\n", key)
		ctx.SetOutput(ovCode, 300)
		ctx.SetOutput(ovMessage, fmt.Sprintf("no data found for key %s\n", key))
		ctx.SetOutput(ovKey, key)
		return true, nil
	}
	log.Debugf("retrieved data from ledger: %s\n", string(jsonBytes))

	var value interface{}
	if err := json.Unmarshal(jsonBytes, &value); err != nil {
		log.Errorf("failed to parse JSON data: %+v\n", err)
		ctx.SetOutput(ovCode, 500)
		ctx.SetOutput(ovMessage, fmt.Sprintf("failed to parse JSON data: %+v", err))
		return false, errors.Wrapf(err, "failed to parse JSON data %s", string(jsonBytes))
	}

	ctx.SetOutput(ovCode, 200)
	ctx.SetOutput(ovMessage, fmt.Sprintf("retrieved data for key %s: %s", key, string(jsonBytes)))
	if result, ok := ctx.GetOutput(ovResult).(*data.ComplexObject); ok && result != nil {
		log.Debugf("set activity output result: %+v\n", value)
		result.Value = value
		ctx.SetOutput(ovResult, result)
		ctx.SetOutput(ovKey, key)
	}
	return true, nil
}

// resolveFlowData resolves and returns data from the flow's context, unmarshals JSON string to map[string]interface{}.
// The name to Resolve is a valid output attribute of a flogo activity, e.g., `activity[app_16].value` or `$flow.content`,
// which is shown in normal flogo mapper as, e.g., "$flow.content"
func resolveFlowData(toResolve string, context activity.Context) (value interface{}, err error) {
	actionCtx := context.ActivityHost()
	log.Debugf("Fabric context data: %+v", actionCtx.WorkingData())
	actValue, err := actionCtx.GetResolver().Resolve(toResolve, actionCtx.WorkingData())
	return actValue, err
}
