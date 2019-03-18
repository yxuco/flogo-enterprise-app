package getrange

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/TIBCOSoftware/flogo-lib/core/activity"
	"github.com/TIBCOSoftware/flogo-lib/core/data"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/pkg/errors"
	trigger "github.com/yxuco/flogo-enterprise-app/fabric/trigger/transaction"
)

const (
	ivStartKey   = "startKey"
	ivEndKey     = "endKey"
	ivPageSize   = "pageSize"
	ivBookmark   = "start"
	ivIsPrivate  = "isPrivate"
	ivCollection = "collection"
	ovCode       = "code"
	ovMessage    = "message"
	ovBookmark   = "bookmark"
	ovCount      = "count"
	ovResult     = "result"
)

// Create a new logger
var log = shim.NewLogger("activity-fabric-getrange")

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

// FabricRangeActivity is a stub for executing Hyperledger Fabric getRange operations
type FabricRangeActivity struct {
	metadata *activity.Metadata
}

// NewActivity creates a new FabricRangeActivity
func NewActivity(metadata *activity.Metadata) activity.Activity {
	return &FabricRangeActivity{metadata: metadata}
}

// Metadata implements activity.Activity.Metadata
func (a *FabricRangeActivity) Metadata() *activity.Metadata {
	return a.metadata
}

// Eval implements activity.Activity.Eval
func (a *FabricRangeActivity) Eval(ctx activity.Context) (done bool, err error) {
	// check input args
	startKey, ok := ctx.GetInput(ivStartKey).(string)
	if !ok || startKey == "" {
		log.Error("start key is not specified\n")
		ctx.SetOutput(ovCode, 400)
		ctx.SetOutput(ovMessage, "start key is not specified")
		return false, errors.New("start key is not specified")
	}
	log.Debugf("start key: %s\n", startKey)
	endKey, ok := ctx.GetInput(ivEndKey).(string)
	if !ok || endKey == "" {
		log.Error("end key is not specified\n")
		ctx.SetOutput(ovCode, 400)
		ctx.SetOutput(ovMessage, "end key is not specified")
		return false, errors.New("end key is not specified")
	}
	log.Debugf("end key: %s\n", endKey)

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
		// retrieve data range from a private collection
		return retrievePrivateRange(ctx, ccshim, startKey, endKey)
	}

	// retrieve data range [startKey, endKey)
	return retrieveRange(ctx, ccshim, startKey, endKey)
}

func retrievePrivateRange(ctx activity.Context, ccshim shim.ChaincodeStubInterface, startKey, endKey string) (bool, error) {
	// check pagination
	pageSize := int32(0)
	bookmark := ""
	if psize, ok := ctx.GetInput(ivPageSize).(int); ok {
		pageSize = int32(psize)
		log.Debugf("pageSize=%d\n", pageSize)
	}
	if pageSize > 0 {
		if bookmark, ok := ctx.GetInput(ivBookmark).(string); ok && bookmark != "" {
			log.Debugf("bookmark=%s\n", bookmark)
		}
	}

	// retrieve data from a private collection
	collection, ok := ctx.GetInput(ivCollection).(string)
	if !ok || collection == "" {
		log.Error("private collection is not specified\n")
		ctx.SetOutput(ovCode, 400)
		ctx.SetOutput(ovMessage, "private collection is not specified")
		return false, errors.New("private collection is not specified")
	}

	// retrieve private data range [startKey, endKey)
	if pageSize > 0 {
		log.Infof("private data query does not support pagination, so ignore specified page size %d and bookmark %s\n", pageSize, bookmark)
	}
	resultIterator, err := ccshim.GetPrivateDataByRange(collection, startKey, endKey)
	if err != nil {
		log.Errorf("failed to retrieve data range [%s, %s) from private collection %s: %+v\n", startKey, endKey, collection, err)
		ctx.SetOutput(ovCode, 500)
		ctx.SetOutput(ovMessage, fmt.Sprintf("failed to retrieve data range [%s, %s) from private collection %s: %+v\n", startKey, endKey, collection, err))
		return false, errors.Wrapf(err, "failed to retrieve data range [%s, %s) from private collection %s", startKey, endKey, collection)
	}
	defer resultIterator.Close()

	var jsonBytes []byte
	if buffer, err := constructQueryResponse(resultIterator); err == nil {
		jsonBytes = buffer.Bytes()
	} else {
		log.Errorf("failed to collect result from iteragor: %+v\n", err)
		ctx.SetOutput(ovCode, 500)
		ctx.SetOutput(ovMessage, fmt.Sprintf("failed to collect result from iterator: %+v", err))
		return false, errors.Wrapf(err, "failed to collect result from iterator")
	}

	if jsonBytes == nil {
		log.Infof("no data found in key range [%s, %s) from private collection %s\n", startKey, endKey, collection)
		ctx.SetOutput(ovCode, 300)
		ctx.SetOutput(ovMessage, fmt.Sprintf("no data found in key range [%s, %s] from private collection %s\n", startKey, endKey, collection))
		ctx.SetOutput(ovCount, 0)
		return true, nil
	}
	log.Debugf("retrieved data range from private collection %s: %s\n", collection, string(jsonBytes))

	var value interface{}
	if err := json.Unmarshal(jsonBytes, &value); err != nil {
		log.Errorf("failed to parse JSON data: %+v\n", err)
		ctx.SetOutput(ovCode, 500)
		ctx.SetOutput(ovMessage, fmt.Sprintf("failed to parse JSON data: %+v", err))
		return false, errors.Wrapf(err, "failed to parse JSON data %s", string(jsonBytes))
	}

	ctx.SetOutput(ovCode, 200)
	ctx.SetOutput(ovMessage, fmt.Sprintf("retrieved data in key range [%s, %s) from private collection %s: %s", startKey, endKey, collection, string(jsonBytes)))
	if result, ok := ctx.GetOutput(ovResult).(*data.ComplexObject); ok && result != nil {
		log.Debugf("set activity output result: %+v\n", value)
		result.Value = value
		ctx.SetOutput(ovResult, result)
		ctx.SetOutput(ovBookmark, "")
		if vArray, ok := value.([]map[string]interface{}); ok {
			ctx.SetOutput(ovCount, len(vArray))
		} else {
			ctx.SetOutput(ovCount, 0)
		}
	}
	return true, nil
}

func retrieveRange(ctx activity.Context, ccshim shim.ChaincodeStubInterface, startKey, endKey string) (bool, error) {
	// check pagination
	pageSize := int32(0)
	bookmark := ""
	if psize, ok := ctx.GetInput(ivPageSize).(int); ok {
		pageSize = int32(psize)
		log.Debugf("pageSize=%d\n", pageSize)
	}
	if pageSize > 0 {
		if bookmark, ok := ctx.GetInput(ivBookmark).(string); ok && bookmark != "" {
			log.Debugf("bookmark=%s\n", bookmark)
		}
	}

	// retrieve data range [startKey, endKey)
	var resultIterator shim.StateQueryIteratorInterface
	var resultMetadata *pb.QueryResponseMetadata
	var err error
	if pageSize > 0 {
		if resultIterator, resultMetadata, err = ccshim.GetStateByRangeWithPagination(startKey, endKey, pageSize, bookmark); err != nil {
			log.Errorf("failed to retrieve data range [%s, %s) with page size %d: %+v\n", startKey, endKey, pageSize, err)
			ctx.SetOutput(ovCode, 500)
			ctx.SetOutput(ovMessage, fmt.Sprintf("failed to retrieve data range [%s, %s) with page size %d: %+v\n", startKey, endKey, pageSize, err))
			return false, errors.Wrapf(err, "failed to retrieve data range [%s, %s) with page size %d", startKey, endKey, pageSize)
		}
	} else {
		if resultIterator, err = ccshim.GetStateByRange(startKey, endKey); err != nil {
			log.Errorf("failed to retrieve data range [%s, %s): %+v\n", startKey, endKey, err)
			ctx.SetOutput(ovCode, 500)
			ctx.SetOutput(ovMessage, fmt.Sprintf("failed to retrieve data range [%s, %s): %+v\n", startKey, endKey, err))
			return false, errors.Wrapf(err, "failed to retrieve data range [%s, %s)", startKey, endKey)
		}
	}
	defer resultIterator.Close()

	var jsonBytes []byte
	if buffer, err := constructQueryResponse(resultIterator); err == nil {
		jsonBytes = buffer.Bytes()
	} else {
		log.Errorf("failed to collect result from iteragor: %+v\n", err)
		ctx.SetOutput(ovCode, 500)
		ctx.SetOutput(ovMessage, fmt.Sprintf("failed to collect result from iterator: %+v", err))
		return false, errors.Wrapf(err, "failed to collect result from iterator")
	}

	if jsonBytes == nil {
		log.Infof("no data found in key range [%s, %s]\n", startKey, endKey)
		ctx.SetOutput(ovCode, 300)
		ctx.SetOutput(ovMessage, fmt.Sprintf("no data found in key range [%s, %s)\n", startKey, endKey))
		ctx.SetOutput(ovCount, 0)
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
	ctx.SetOutput(ovMessage, fmt.Sprintf("retrieved data in key range [%s, %s): %s", startKey, endKey, string(jsonBytes)))
	if result, ok := ctx.GetOutput(ovResult).(*data.ComplexObject); ok && result != nil {
		log.Debugf("set activity output result: %+v\n", value)
		result.Value = value
		ctx.SetOutput(ovResult, result)
		if resultMetadata != nil {
			ctx.SetOutput(ovCount, resultMetadata.FetchedRecordsCount)
			ctx.SetOutput(ovBookmark, resultMetadata.Bookmark)
		} else {
			ctx.SetOutput(ovBookmark, "")
			if vArray, ok := value.([]map[string]interface{}); ok {
				ctx.SetOutput(ovCount, len(vArray))
			} else {
				ctx.SetOutput(ovCount, 0)
			}
		}
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

func constructQueryResponse(resultsIterator shim.StateQueryIteratorInterface) (*bytes.Buffer, error) {
	// buffer is a JSON array containing QueryResults
	var buffer bytes.Buffer
	buffer.WriteString("[")

	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		// Add a comma before array members, suppress it for the first array member
		if bArrayMemberAlreadyWritten == true {
			buffer.WriteString(",")
		}
		buffer.WriteString("{\"key\":")
		buffer.WriteString("\"")
		buffer.WriteString(queryResponse.Key)
		buffer.WriteString("\"")

		buffer.WriteString(", \"value\":")
		// Record is a JSON object, so we write as-is
		buffer.WriteString(string(queryResponse.Value))
		buffer.WriteString("}")
		bArrayMemberAlreadyWritten = true
	}
	buffer.WriteString("]")

	return &buffer, nil
}
