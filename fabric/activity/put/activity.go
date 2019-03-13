package put

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
	ivValueType     = "valueType"
	ivValue         = "value"
	ivData          = "data"
	ivIsPrivate     = "isPrivate"
	ivCollection    = "collection"
	ivCompositeKeys = "compositeKeys"
	ovCode          = "code"
	ovMessage       = "message"
	ovResult        = "result"
	objectType      = "object"
)

// Create a new logger
var log = shim.NewLogger("activity-fabric-put")

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

// FabricPutActivity is a stub for executing Hyperledger Fabric put operations
type FabricPutActivity struct {
	metadata *activity.Metadata
}

// NewActivity creates a new PutActivity
func NewActivity(metadata *activity.Metadata) activity.Activity {
	return &FabricPutActivity{metadata: metadata}
}

// Metadata implements activity.Activity.Metadata
func (a *FabricPutActivity) Metadata() *activity.Metadata {
	return a.metadata
}

// Eval implements activity.Activity.Eval
func (a *FabricPutActivity) Eval(ctx activity.Context) (done bool, err error) {
	// check input args
	key, ok := ctx.GetInput(ivKey).(string)
	if !ok || key == "" {
		log.Error("state key is not specified")
		ctx.SetOutput(ovCode, 400)
		ctx.SetOutput(ovMessage, "state key is not specified")
		return false, errors.New("state key is not specified")
	}
	log.Debugf("state key: %s", key)
	vtype := ctx.GetInput(ivValueType).(string)
	log.Debugf("value type: %s", vtype)
	value := ctx.GetInput(ivValue)
	if vtype == objectType {
		if obj, ok := ctx.GetInput(ivData).(*data.ComplexObject); ok {
			value = obj.Value
		} else {
			log.Errorf("input data is not a complex object")
			ctx.SetOutput(ovCode, 500)
			ctx.SetOutput(ovMessage, "input data is not a complex object")
			return false, errors.New("input data is not a complex object")
		}
	}
	log.Debugf("input value type %T: %+v", value, value)
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		log.Errorf("failed to marshal value '%+v', error: %+v", value, err)
		ctx.SetOutput(ovCode, 400)
		ctx.SetOutput(ovMessage, fmt.Sprintf("failed to marshal value: %+v", err))
		return false, errors.Wrapf(err, "failed to marshal value: %+v", value)
	}

	// check composite keys
	if keys, ok := ctx.GetInput(ivCompositeKeys).(*data.ComplexObject); ok {
		log.Debugf("Got composite keys: %+v\n", keys.Value)
	} else {
		log.Debugf("No composite key is defined")
	}

	// get chaincode stub
	stub, err := GetData("$flow."+trigger.FabricStub, ctx)
	if err != nil || stub == nil {
		log.Errorf("failed to get stub: %+v", err)
		ctx.SetOutput(ovCode, 500)
		ctx.SetOutput(ovMessage, fmt.Sprintf("failed to get stub: %+v", err))
		return false, errors.Errorf("failed to get stub: %+v", err)
	}

	ccshim, ok := stub.(shim.ChaincodeStubInterface)
	if !ok {
		log.Errorf("stub type %T is not a ChaincodeStubInterface", stub)
		ctx.SetOutput(ovCode, 500)
		ctx.SetOutput(ovMessage, fmt.Sprintf("stub type %T is not a ChaincodeStubInterface", stub))
		return false, errors.Errorf("stub type %T is not a ChaincodeStubInterface", stub)
	}

	isPrivate, ok := ctx.GetInput(ivIsPrivate).(bool)
	if ok && isPrivate {
		// store data on a private collection
		collection, ok := ctx.GetInput(ivCollection).(string)
		if !ok || collection == "" {
			log.Error("private collection is not specified")
			ctx.SetOutput(ovCode, 400)
			ctx.SetOutput(ovMessage, "private collection is not specified")
			return false, errors.New("private collection is not specified")
		}
		if err := ccshim.PutPrivateData(collection, key, jsonBytes); err != nil {
			log.Errorf("failed to store data in private collection %s: %+v", collection, err)
			ctx.SetOutput(ovCode, 500)
			ctx.SetOutput(ovMessage, fmt.Sprintf("failed to store data in private collection %s: %+v", collection, err))
			return false, errors.Wrapf(err, "failed to store data in private collection %s", collection)
		}
		log.Debugf("stored in private collection %s, data: %s", collection, string(jsonBytes))
		ctx.SetOutput(ovCode, 200)
		ctx.SetOutput(ovMessage, fmt.Sprintf("stored in private collection %s, data: %s", collection, string(jsonBytes)))
		if result, ok := ctx.GetOutput(ovResult).(*data.ComplexObject); ok && result != nil {
			log.Debugf("set activity output result: %+v", value)
			result.Value = value
			ctx.SetOutput(ovResult, result)
		}
		return true, nil
	}

	// store data on the ledger
	if err := ccshim.PutState(key, jsonBytes); err != nil {
		log.Errorf("failed to store data on ledger: %+v", err)
		ctx.SetOutput(ovCode, 500)
		ctx.SetOutput(ovMessage, fmt.Sprintf("failed to store data on ledger: %+v", err))
		return false, errors.Errorf("failed to store data on ledger: %+v", err)
	}
	log.Debugf("stored data on ledger: %s", string(jsonBytes))
	ctx.SetOutput(ovCode, 200)
	ctx.SetOutput(ovMessage, fmt.Sprintf("stored data on ledger: %s", string(jsonBytes)))
	if result, ok := ctx.GetOutput(ovResult).(*data.ComplexObject); ok && result != nil {
		log.Debugf("set activity output result: %+v", value)
		result.Value = value
		ctx.SetOutput(ovResult, result)
	}
	return true, nil
}

// GetData resolves and returns data from the flow's context, unmarshals JSON string to map[string]interface{}.
// The name to Resolve is a valid output attribute of a flogo activity, e.g., `activity[app_16].value` or `$flow.content`,
// which is shown in normal flogo mapper as, e.g., "$flow.content"
func GetData(toResolve string, context activity.Context) (value interface{}, err error) {
	actionCtx := context.ActivityHost()
	log.Debugf("Fabric context data: %+v", actionCtx.WorkingData())
	actValue, err := actionCtx.GetResolver().Resolve(toResolve, actionCtx.WorkingData())
	return actValue, err
}
