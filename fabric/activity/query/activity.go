package query

import (
	"encoding/json"
	"fmt"

	"github.com/TIBCOSoftware/flogo-lib/core/activity"
	"github.com/TIBCOSoftware/flogo-lib/core/data"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
	"github.com/pkg/errors"
	"github.com/yxuco/flogo-enterprise-app/fabric/common"
)

const (
	ivQuery       = "query"
	ivQueryParams = "queryParams"
	ivPageSize    = "pageSize"
	ivBookmark    = "start"
	ivIsPrivate   = "isPrivate"
	ivCollection  = "collection"
	ovCode        = "code"
	ovMessage     = "message"
	ovBookmark    = "bookmark"
	ovCount       = "count"
	ovResult      = "result"
)

// Create a new logger
var log = shim.NewLogger("activity-fabric-query")

func init() {
	common.SetChaincodeLogLevel(log)
}

// FabricQueryActivity is a stub for executing Hyperledger Fabric query operations
type FabricQueryActivity struct {
	metadata *activity.Metadata
}

// NewActivity creates a new FabricQueryActivity
func NewActivity(metadata *activity.Metadata) activity.Activity {
	return &FabricQueryActivity{metadata: metadata}
}

// Metadata implements activity.Activity.Metadata
func (a *FabricQueryActivity) Metadata() *activity.Metadata {
	return a.metadata
}

// Eval implements activity.Activity.Eval
func (a *FabricQueryActivity) Eval(ctx activity.Context) (done bool, err error) {
	// check input args
	query, ok := ctx.GetInput(ivQuery).(string)
	if !ok || query == "" {
		log.Error("query statement is not specified\n")
		ctx.SetOutput(ovCode, 400)
		ctx.SetOutput(ovMessage, "query statement is not specified")
		return false, errors.New("query statement is not specified")
	}
	log.Debugf("query statement: %s\n", query)
	queryParams, err := getQueryParams(ctx)
	if err != nil {
		ctx.SetOutput(ovCode, 400)
		ctx.SetOutput(ovMessage, err.Error())
		return false, err
	}
	log.Debugf("query parameters: %+v\n", queryParams)

	queryStatement, err := prepareQueryStatement(query, queryParams)
	if err != nil {
		ctx.SetOutput(ovCode, 400)
		ctx.SetOutput(ovMessage, err.Error())
		return false, err
	}

	// get chaincode stub
	stub, err := common.GetChaincodeStub(ctx)
	if err != nil || stub == nil {
		ctx.SetOutput(ovCode, 500)
		ctx.SetOutput(ovMessage, err.Error())
		return false, err
	}

	if isPrivate, ok := ctx.GetInput(ivIsPrivate).(bool); ok && isPrivate {
		// query private data
		return queryPrivateData(ctx, stub, queryStatement)
	}

	// query state data
	return queryData(ctx, stub, queryStatement)
}

func getQueryParams(ctx activity.Context) (map[string]interface{}, error) {
	queryParams, ok := ctx.GetInput(ivQueryParams).(*data.ComplexObject)
	if !ok || queryParams == nil || queryParams.Value == nil {
		log.Error("query parameters are not specified\n")
		return nil, errors.New("query parameters are not specified")
	}
	params, ok := queryParams.Value.(map[string]interface{})
	if !ok {
		log.Errorf("query parameter type %T is not JSON object\n", queryParams.Value)
		return nil, errors.Errorf("query parameter type %T is not JSON object\n", queryParams.Value)
	}
	return params, nil
}

func prepareQueryStatement(query string, queryParams map[string]interface{}) (string, error) {
	// TODO: construct query statement by replace parameters in query
	return query, nil
}

func queryPrivateData(ctx activity.Context, ccshim shim.ChaincodeStubInterface, query string) (bool, error) {
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

	// query data from a private collection
	collection, ok := ctx.GetInput(ivCollection).(string)
	if !ok || collection == "" {
		log.Error("private collection is not specified\n")
		ctx.SetOutput(ovCode, 400)
		ctx.SetOutput(ovMessage, "private collection is not specified")
		return false, errors.New("private collection is not specified")
	}

	// query private data
	if pageSize > 0 {
		log.Infof("private data query does not support pagination, so ignore specified page size %d and bookmark %s\n", pageSize, bookmark)
	}
	resultIterator, err := ccshim.GetPrivateDataQueryResult(collection, query)
	if err != nil {
		log.Errorf("failed to execute query %s on private collection %s: %+v\n", query, collection, err)
		ctx.SetOutput(ovCode, 500)
		ctx.SetOutput(ovMessage, fmt.Sprintf("failed to execute query %s on private collection %s: %+v", query, collection, err))
		return false, errors.Wrapf(err, "failed to execute query %s on private collection %s", query, collection)
	}
	defer resultIterator.Close()

	jsonBytes, err := common.ConstructQueryResponse(resultIterator, false, nil)
	if err != nil {
		log.Errorf("failed to collect result from iterator: %+v\n", err)
		ctx.SetOutput(ovCode, 500)
		ctx.SetOutput(ovMessage, fmt.Sprintf("failed to collect result from iterator: %+v", err))
		return false, errors.Wrapf(err, "failed to collect result from iterator")
	}

	if jsonBytes == nil {
		log.Infof("no data returned for query %s on private collection %s\n", query, collection)
		ctx.SetOutput(ovCode, 300)
		ctx.SetOutput(ovMessage, fmt.Sprintf("no data returned for query %s on private collection %s", query, collection))
		ctx.SetOutput(ovCount, 0)
		return true, nil
	}
	log.Debugf("query result from private collection %s: %s\n", collection, string(jsonBytes))

	var value interface{}
	if err := json.Unmarshal(jsonBytes, &value); err != nil {
		log.Errorf("failed to parse JSON data: %+v\n", err)
		ctx.SetOutput(ovCode, 500)
		ctx.SetOutput(ovMessage, fmt.Sprintf("failed to parse JSON data: %+v", err))
		return false, errors.Wrapf(err, "failed to parse JSON data %s", string(jsonBytes))
	}

	ctx.SetOutput(ovCode, 200)
	ctx.SetOutput(ovMessage, fmt.Sprintf("result of query %s from private collection %s: %s", query, collection, string(jsonBytes)))
	if result, ok := ctx.GetOutput(ovResult).(*data.ComplexObject); ok && result != nil {
		log.Debugf("set activity output result: %+v\n", value)
		result.Value = value
		ctx.SetOutput(ovResult, result)
		ctx.SetOutput(ovBookmark, "")
		if vArray, ok := value.([]interface{}); ok {
			ctx.SetOutput(ovCount, len(vArray))
		} else {
			ctx.SetOutput(ovCount, 0)
		}
	}
	return true, nil
}

func queryData(ctx activity.Context, ccshim shim.ChaincodeStubInterface, query string) (bool, error) {
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

	// query state data
	var resultIterator shim.StateQueryIteratorInterface
	var resultMetadata *pb.QueryResponseMetadata
	var err error
	if pageSize > 0 {
		if resultIterator, resultMetadata, err = ccshim.GetQueryResultWithPagination(query, pageSize, bookmark); err != nil {
			log.Errorf("failed to execute query %s with page size %d: %+v\n", query, pageSize, err)
			ctx.SetOutput(ovCode, 500)
			ctx.SetOutput(ovMessage, fmt.Sprintf("failed to execute query %s with page size %d: %+v", query, pageSize, err))
			return false, errors.Wrapf(err, "failed to execute query %s with page size %d", query, pageSize)
		}
	} else {
		if resultIterator, err = ccshim.GetQueryResult(query); err != nil {
			log.Errorf("failed to execute query %s: %+v\n", query, err)
			ctx.SetOutput(ovCode, 500)
			ctx.SetOutput(ovMessage, fmt.Sprintf("failed to execute query %s: %+v", query, err))
			return false, errors.Wrapf(err, "failed to execute query %s", query)
		}
	}
	defer resultIterator.Close()

	jsonBytes, err := common.ConstructQueryResponse(resultIterator, false, nil)
	if err != nil {
		log.Errorf("failed to collect result from iterator: %+v\n", err)
		ctx.SetOutput(ovCode, 500)
		ctx.SetOutput(ovMessage, fmt.Sprintf("failed to collect result from iterator: %+v", err))
		return false, errors.Wrapf(err, "failed to collect result from iterator")
	}

	if jsonBytes == nil {
		log.Infof("no data returned for query %s\n", query)
		ctx.SetOutput(ovCode, 300)
		ctx.SetOutput(ovMessage, fmt.Sprintf("no data returned for query %s", query))
		ctx.SetOutput(ovCount, 0)
		return true, nil
	}
	log.Debugf("query returned data: %s\n", string(jsonBytes))

	var value interface{}
	if err := json.Unmarshal(jsonBytes, &value); err != nil {
		log.Errorf("failed to parse JSON data: %+v\n", err)
		ctx.SetOutput(ovCode, 500)
		ctx.SetOutput(ovMessage, fmt.Sprintf("failed to parse JSON data: %+v", err))
		return false, errors.Wrapf(err, "failed to parse JSON data %s", string(jsonBytes))
	}

	ctx.SetOutput(ovCode, 200)
	ctx.SetOutput(ovMessage, fmt.Sprintf("data returned for query %s: %s", query, string(jsonBytes)))
	if result, ok := ctx.GetOutput(ovResult).(*data.ComplexObject); ok && result != nil {
		log.Debugf("set activity output result: %+v\n", value)
		result.Value = value
		ctx.SetOutput(ovResult, result)
		if resultMetadata != nil {
			ctx.SetOutput(ovCount, resultMetadata.FetchedRecordsCount)
			ctx.SetOutput(ovBookmark, resultMetadata.Bookmark)
		} else {
			ctx.SetOutput(ovBookmark, "")
			if vArray, ok := value.([]interface{}); ok {
				ctx.SetOutput(ovCount, len(vArray))
			} else {
				ctx.SetOutput(ovCount, 0)
			}
		}
	}
	return true, nil
}
