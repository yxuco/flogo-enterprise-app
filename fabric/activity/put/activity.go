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
		log.Error("state key is not specified\n")
		ctx.SetOutput(ovCode, 400)
		ctx.SetOutput(ovMessage, "state key is not specified")
		return false, errors.New("state key is not specified")
	}
	log.Debugf("state key: %s\n", key)
	vtype := ctx.GetInput(ivValueType).(string)
	log.Debugf("value type: %s\n", vtype)
	value := ctx.GetInput(ivValue)
	if vtype == objectType {
		if obj, ok := ctx.GetInput(ivData).(*data.ComplexObject); ok {
			value = obj.Value
		} else {
			log.Errorf("input data is not a complex object\n")
			ctx.SetOutput(ovCode, 500)
			ctx.SetOutput(ovMessage, "input data is not a complex object")
			return false, errors.New("input data is not a complex object")
		}
	}
	log.Debugf("input value type %T: %+v\n", value, value)

	// check composite keys
	if keys, ok := ctx.GetInput(ivCompositeKeys).(string); ok {
		log.Debugf("Got composite keys: %s\n", keys)
		keyMap := make(map[string][]string)
		if err := json.Unmarshal([]byte(keys), &keyMap); err != nil {
			log.Warningf("failed to unmarshal composite keys: %+v\n", err)
		} else {
			log.Debugf("Parsed composite keys: %+v\n", keyMap)
		}
	} else {
		log.Debugf("No composite key is defined\n")
	}

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
		// store data on a private collection
		return storePrivateData(ctx, ccshim, key, value)
	}

	// store data on the ledger
	return storeData(ctx, ccshim, key, value)
}

func storePrivateData(ctx activity.Context, ccshim shim.ChaincodeStubInterface, key string, value interface{}) (bool, error) {
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		log.Errorf("failed to marshal value '%+v', error: %+v\n", value, err)
		ctx.SetOutput(ovCode, 400)
		ctx.SetOutput(ovMessage, fmt.Sprintf("failed to marshal value: %+v", err))
		return false, errors.Wrapf(err, "failed to marshal value: %+v", value)
	}
	// store data on a private collection
	collection, ok := ctx.GetInput(ivCollection).(string)
	if !ok || collection == "" {
		log.Error("private collection is not specified\n")
		ctx.SetOutput(ovCode, 400)
		ctx.SetOutput(ovMessage, "private collection is not specified")
		return false, errors.New("private collection is not specified")
	}
	if err := ccshim.PutPrivateData(collection, key, jsonBytes); err != nil {
		log.Errorf("failed to store data in private collection %s: %+v\n", collection, err)
		ctx.SetOutput(ovCode, 500)
		ctx.SetOutput(ovMessage, fmt.Sprintf("failed to store data in private collection %s: %+v", collection, err))
		return false, errors.Wrapf(err, "failed to store data in private collection %s", collection)
	}
	log.Debugf("stored in private collection %s, data: %s\n", collection, string(jsonBytes))

	// store composite keys if required
	if keys, err := getCompositeKeys(ctx, ccshim, value); err == nil && keys != nil && len(keys) > 0 {
		for _, k := range keys {
			cv := []byte{0x00}
			if err := ccshim.PutPrivateData(collection, k, cv); err != nil {
				log.Errorf("failed to store composite key %s on collection %s: %+v\n", k, collection, err)
			} else {
				log.Debugf("stored composite key %s on collection %s\n", k, collection)
			}
		}
	}

	ctx.SetOutput(ovCode, 200)
	ctx.SetOutput(ovMessage, fmt.Sprintf("stored in private collection %s, data: %s", collection, string(jsonBytes)))
	if result, ok := ctx.GetOutput(ovResult).(*data.ComplexObject); ok && result != nil {
		log.Debugf("set activity output result: %+v\n", value)
		result.Value = value
		ctx.SetOutput(ovResult, result)
	}
	return true, nil
}

func storeData(ctx activity.Context, ccshim shim.ChaincodeStubInterface, key string, value interface{}) (bool, error) {
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		log.Errorf("failed to marshal value '%+v', error: %+v\n", value, err)
		ctx.SetOutput(ovCode, 400)
		ctx.SetOutput(ovMessage, fmt.Sprintf("failed to marshal value: %+v", err))
		return false, errors.Wrapf(err, "failed to marshal value: %+v", value)
	}
	// store data on the ledger
	if err := ccshim.PutState(key, jsonBytes); err != nil {
		log.Errorf("failed to store data on ledger: %+v\n", err)
		ctx.SetOutput(ovCode, 500)
		ctx.SetOutput(ovMessage, fmt.Sprintf("failed to store data on ledger: %+v", err))
		return false, errors.Errorf("failed to store data on ledger: %+v", err)
	}
	log.Debugf("stored data on ledger: %s\n", string(jsonBytes))

	// store composite keys if required
	if keys, err := getCompositeKeys(ctx, ccshim, value); err == nil && keys != nil && len(keys) > 0 {
		for _, k := range keys {
			cv := []byte{0x00}
			if err := ccshim.PutState(k, cv); err != nil {
				log.Errorf("failed to store composite key %s: %+v\n", k, err)
			} else {
				log.Debugf("stored composite key %s\n", k)
			}
		}
	}

	ctx.SetOutput(ovCode, 200)
	ctx.SetOutput(ovMessage, fmt.Sprintf("stored data on ledger: %s", string(jsonBytes)))
	if result, ok := ctx.GetOutput(ovResult).(*data.ComplexObject); ok && result != nil {
		log.Debugf("set activity output result: %+v\n", value)
		result.Value = value
		ctx.SetOutput(ovResult, result)
	}
	return true, nil
}

// collect composite keys as specified in activity input 'ivCompositeKeys'
func getCompositeKeys(ctx activity.Context, ccshim shim.ChaincodeStubInterface, value interface{}) ([]string, error) {
	// verify that value is a map
	obj, ok := value.(map[string]interface{})
	if !ok {
		log.Debugf("No composite keys because state value is not a map\n")
		return nil, nil
	}

	// check composite keys
	if keyDefs, ok := ctx.GetInput(ivCompositeKeys).(string); ok {
		log.Debugf("Got composite key definitions: %s\n", keyDefs)
		keyMap := make(map[string][]string)
		if err := json.Unmarshal([]byte(keyDefs), &keyMap); err != nil {
			log.Warningf("failed to unmarshal composite key definitions: %+v\n", err)
			return nil, err
		}
		log.Debugf("Parsed composite keys: %+v\n", keyMap)
		var keys []string
		for k, v := range keyMap {
			if ck := compositeKey(ccshim, k, v, obj); ck != "" {
				keys = append(keys, ck)
			}
		}
		return keys, nil
	}
	log.Debugf("No composite key is defined")
	return nil, nil
}

// construct composite key if all specified attributes exist in the value object
func compositeKey(ccshim shim.ChaincodeStubInterface, name string, attributes []string, value map[string]interface{}) string {
	if name == "" || attributes == nil || len(attributes) == 0 {
		log.Debugf("invalid composite key definition: name %s attributes %+v\n", name, attributes)
		return ""
	}
	var keyValues []string
	for _, k := range attributes {
		if v, ok := value[k]; ok {
			keyValues = append(keyValues, fmt.Sprintf("%v", v))
		} else {
			log.Debugf("composite key attribute %s is not found in state value\n", k)
			return ""
		}
	}
	key, err := ccshim.CreateCompositeKey(name, keyValues)
	if err != nil {
		log.Errorf("failed to create composite key %s with values %+v\n", name, keyValues)
		return ""
	}
	return key
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
