package put

import (
	"github.com/TIBCOSoftware/flogo-lib/core/activity"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	trigger "github.com/yxuco/flogo-enterprise-app/fabric/trigger/transaction"
)

// Create a new logger
var log = shim.NewLogger("activity-fabric-put")

const (
	ivKey        = "key"
	ivValueType  = "valueType"
	ivValue      = "value"
	ivData       = "data"
	ivIsPrivate  = "isPrivate"
	ivCollection = "collection"
	ovCode       = "code"
	ovMessage    = "message"
	objectType   = "object"
)

// PutActivity is a stub for executing Hyperledger Fabric put operations
type PutActivity struct {
	metadata *activity.Metadata
}

// NewActivity creates a new PutActivity
func NewActivity(metadata *activity.Metadata) activity.Activity {
	return &PutActivity{metadata: metadata}
}

// Metadata implements activity.Activity.Metadata
func (a *PutActivity) Metadata() *activity.Metadata {
	return a.metadata
}

// Eval implements activity.Activity.Eval
func (a *PutActivity) Eval(ctx activity.Context) (done bool, err error) {
	// check input args
	key := ctx.GetInput(ivKey).(string)
	log.Debugf("input key: %s", key)
	vtype := ctx.GetInput(ivValueType).(string)
	log.Debugf("value type: %s", vtype)
	value := ctx.GetInput(ivValue)
	if vtype == objectType {
		value = ctx.GetInput(ivData)
	}
	log.Debugf("input value: %+v", value)
	collection := ""
	isPrivate := ctx.GetInput(ivIsPrivate).(bool)
	if isPrivate {
		collection = ctx.GetInput(ivCollection).(string)
		log.Debugf("private collection: %s", collection)
	}

	// get chaincode stub
	stub, err := GetData("$flow."+trigger.FabricStub, ctx)
	if err != nil {
		log.Errorf("failed to get stub: %+v", err)
	} else {
		log.Infof("fetched stub of type %T: %+v", stub, stub)
	}

	// set output
	ctx.SetOutput(ovCode, 200)
	ctx.SetOutput(ovMessage, "done")
	return true, nil
}

// GetData resolves and returns data from the flow's context, unmarshals JSON string to map[string]interface{}.
// The name to Resolve is a valid output attribute of a flogo activity, e.g., `activity[app_16].value` or `$flow.content`,
// which is shown in normal flogo mapper as, e.g., "{{$flow.content}}"
func GetData(toResolve string, context activity.Context) (value interface{}, err error) {
	actionCtx := context.ActivityHost()
	log.Debugf("fabricop context data: %+v", actionCtx.WorkingData())
	actValue, err := actionCtx.GetResolver().Resolve(toResolve, actionCtx.WorkingData())
	return actValue, err
}
