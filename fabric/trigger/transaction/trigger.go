package transaction

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"time"

	"github.com/TIBCOSoftware/flogo-lib/core/data"
	"github.com/TIBCOSoftware/flogo-lib/core/trigger"

	//"github.com/TIBCOSoftware/flogo-lib/logger"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	jschema "github.com/xeipuuv/gojsonschema"
)

const (
	sTransaction = "name"
	sValidation  = "validation"
	oParameters  = "parameters"
	oTxID        = "txID"
	oTxTime      = "txTime"
	rReturns     = "returns"

	// FabricStub is the name of flow property for passing chaincode stub to activities
	FabricStub = "_chaincode_stub"
)

// Create a new logger
//var log = logger.GetLogger("trigger-fabric-transaction")
var log = shim.NewLogger("trigger-fabric-transaction")

// TriggerMap maps transaction name in trigger handler setting to the trigger,
// so we can lookup trigger by transaction name
var triggerMap = map[string]*Trigger{}

// GetTrigger returns the cached trigger for a specified transaction name;
// return false in the second value if no trigger is cached for the specified name
func GetTrigger(name string) (*Trigger, bool) {
	trig, ok := triggerMap[name]
	return trig, ok
}

// TriggerFactory Fabric Trigger factory
type TriggerFactory struct {
	metadata *trigger.Metadata
}

// NewFactory create a new Trigger factory
func NewFactory(md *trigger.Metadata) trigger.Factory {
	return &TriggerFactory{metadata: md}
}

// New Creates a new trigger instance for a given id
func (t *TriggerFactory) New(config *trigger.Config) trigger.Trigger {
	return &Trigger{metadata: t.metadata, config: config}
}

// ParameterIndex stores transaction parameters and its location in raw JSON schema string
// start and end location is used to sort the parameter list to match the parameter order in schema
type ParameterIndex struct {
	name     string
	jsonType string
	start    int
	end      int
}

// Trigger is a stub for the Trigger implementation
type Trigger struct {
	metadata   *trigger.Metadata
	config     *trigger.Config
	handlers   []*trigger.Handler
	parameters []ParameterIndex
}

// Initialize implements trigger.Init.Initialize
func (t *Trigger) Initialize(ctx trigger.InitContext) error {
	t.handlers = ctx.GetHandlers()
	for _, handler := range t.handlers {
		name := handler.GetStringSetting(sTransaction)
		log.Info("init transaction trigger:", name)
		_, ok := triggerMap[name]
		if ok {
			log.Warningf("transaction name %s used by multiple trigger handlers, only the last handler is effective", name)
			//log.Warnf("transaction name %s used by multiple trigger handlers, only the last handler is effective", name)
		}
		triggerMap[name] = t

		// collect input parameter name and types from metadata
		params, ok := handler.GetOutput()[oParameters].(*data.ComplexObject)
		if ok {
			if index, err := rootParameters(params.Metadata); err == nil {
				t.parameters = index
			} else {
				t.parameters = nil
				log.Errorf("failed to initialize transaction parameters: %+v", err)
			}
		}

		// verify validation setting, value is not used
		handler.GetSetting(sValidation)
		validate := false
		if v, ok := handler.GetSetting(sValidation); ok {
			if bv, err := data.CoerceToBoolean(v); err == nil {
				validate = bv
			}
		}
		log.Info("validate output:", validate)
	}
	return nil
}

// Metadata implements trigger.Trigger.Metadata
func (t *Trigger) Metadata() *trigger.Metadata {
	return t.metadata
}

// Start implements trigger.Trigger.Start
func (t *Trigger) Start() error {
	return nil
}

// Stop implements trigger.Trigger.Start
func (t *Trigger) Stop() error {
	// stop the trigger
	return nil
}

// addIndex adds a new parameter position to the index, ignore or merge index if index region overlaps.
func addIndex(parameters []ParameterIndex, param ParameterIndex) []ParameterIndex {
	for i, v := range parameters {
		if param.start > v.start && param.start < v.end {
			// ignore if new param's start postion falls in region covered by a known parameter
			return parameters
		} else if v.start > param.start && v.start < param.end {
			// replace old parameter region if its start position falls in the region covered by the new parameter
			updated := append(parameters[:i], param)
			if len(parameters) > i+1 {
				// check the remaining knonw parameters
				for _, p := range parameters[i+1:] {
					if !(p.start > param.start && p.start < param.end) {
						updated = append(updated, p)
					}
				}
			}
			return updated
		}
	}
	// append new parameter
	return append(parameters, param)
}

func rootParameters(metadata string) ([]ParameterIndex, error) {
	// extract root object properties from JSON schema
	var rawProperties struct {
		Data json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal([]byte(metadata), &rawProperties); err != nil {
		log.Errorf("failed to extract properties from metadata: %+v", err)
		return nil, err
	}

	// extract parameter names from raw object properties
	var params map[string]json.RawMessage
	if err := json.Unmarshal(rawProperties.Data, &params); err != nil {
		log.Errorf("failed to extract parameters from object schema: %+v", err)
		return nil, err
	}

	// collect parameter locations in the raw object schema
	var paramIndex []ParameterIndex
	for p, v := range params {
		// encode parameter name with quotes
		key, _ := json.Marshal(p)
		// key may exist in raw schema multiple times,
		// so check each occurence to determine its correct location in the raw schema
		items := bytes.Split(rawProperties.Data, key)
		pos := 0
		for _, seg := range items {
			if pos == 0 {
				// first segment should not be the key definition
				pos += len(seg)
				continue
			}
			vpos := bytes.Index(seg, v)
			if vpos >= 0 {
				// the segment contains the key definition, so collect its position in raw schema
				endPos := pos + len(key) + vpos + len(v)
				// extract JSON type of the parameter
				var paramDef struct {
					RawType string `json:"type"`
				}
				if err := json.Unmarshal(v, &paramDef); err != nil {
					log.Errorf("failed to extract JSON type of parameter %s: %+v", p, err)
				}
				paramType := jschema.TYPE_OBJECT
				if paramDef.RawType != "" {
					paramType = paramDef.RawType
				}
				paramIndex = addIndex(paramIndex, ParameterIndex{name: p, jsonType: paramType, start: pos, end: endPos})
			}
			pos += len(key) + len(seg)
		}
	}

	// sort parameter index by start location in raw schema
	sort.Slice(paramIndex, func(i, j int) bool {
		return paramIndex[i].start < paramIndex[j].start
	})
	return paramIndex, nil
}

// Invoke starts the trigger and invokes the action registered in the handler,
// and returns result as JSON string
func (t *Trigger) Invoke(stub shim.ChaincodeStubInterface, fn string, args []string) (string, error) {
	//func (t *Trigger) Invoke(stub interface{}, fn string, args []string) (string, error) {
	log.Debugf("fabric.Trigger invokes fn %s with args %+v", fn, args)
	if t.parameters == nil {
		log.Errorf("parameters not defined for transaction %s", fn)
		return "", fmt.Errorf("parameters not defined for transaction %s", fn)
	}
	for _, handler := range t.handlers {
		if f := handler.GetStringSetting(sTransaction); f != fn {
			log.Warningf("handler transaction %s is different from requested function %s", f, fn)
			//log.Warnf("handler transaction %s is different from requested function %s", f, fn)
			continue
		}

		// construct transaction input data
		transData, err := prepareTriggerData(t.parameters, args)
		if err != nil {
			return "", err
		}
		if log.IsEnabledFor(shim.LogDebug) {
			// debug flow data
			triggerData, _ := json.Marshal(transData)
			log.Debugf("trigger output data: %s", string(triggerData))
		}

		// set trigger data
		params, _ := handler.GetOutput()[oParameters].(*data.ComplexObject)
		params.Value = transData
		triggerData := make(map[string]interface{})
		triggerData[oParameters] = params
		triggerData[FabricStub] = stub
		triggerData[oTxID] = stub.GetTxID()
		if ts, err := stub.GetTxTimestamp(); err == nil {
			triggerData[oTxTime] = time.Unix(ts.Seconds, int64(ts.Nanos)).UTC().Format("2006-01-02T15:04:05.000000-0700")
		}

		// execute flogo flow
		log.Debugf("flogo flow started transaction %s with timestamp %s", triggerData[oTxID], triggerData[oTxTime])
		results, err := handler.Handle(context.Background(), triggerData)
		if err != nil {
			log.Errorf("flogo flow returned error: %+v", err)
			return "", err
		}
		if len(results) != 0 {
			if dataAttr, ok := results[rReturns]; ok {
				// return serialized JSON string
				replyData, err := json.Marshal(dataAttr.Value())
				if err != nil {
					log.Errorf("failed to serialize reply: %+v", err)
					return "", err
				}
				log.Debugf("flogo flow returned data: %s", string(replyData))
				return string(replyData), nil
			}
			log.Warningf("flogo flow result does not contain attribute %s", rReturns)
		}
		log.Info("flogo flow did not return any data")
		return "", nil
	}
	log.Warningf("no flogo handler is activated for transaction %s", fn)
	return "", nil
}

// construct trigger output data for specified parameter index, and values of the parameters
func prepareTriggerData(paramIndex []ParameterIndex, values []string) (interface{}, error) {
	log.Debugf("prepareFlowData with parameters %+v values %+v", paramIndex, values)
	if paramIndex == nil || len(paramIndex) == 0 {
		// unknown parameter schema
		return nil, errors.New("parameter schema is not defined")
	}

	if len(paramIndex) < len(values) {
		// some data values are not defined by parameter index
		return nil, fmt.Errorf("parameter list %d is shorter than data items %d", len(paramIndex), len(values))
	}

	// convert string array to object with name-values as defined by parameter index
	result := make(map[string]interface{})
	for i, v := range values {
		if obj := unmarshalString(v, paramIndex[i].jsonType, paramIndex[i].name); obj != nil {
			result[paramIndex[i].name] = obj
		}
	}
	return result, nil
}

// unmarshalString returns unmarshaled object if input is a valid JSON object or array,
// or returns the input string if it is not a valid JSON format
func unmarshalString(data, jsonType, name string) interface{} {
	s := strings.TrimSpace(data)
	switch jsonType {
	case jschema.TYPE_STRING:
		return s
	case jschema.TYPE_ARRAY:
		var result []interface{}
		if err := json.Unmarshal([]byte(data), &result); err != nil {
			log.Warningf("failed to parse parameter %s as JSON array: data '%s' error %+v", name, data, err)
		}
		return result
	case jschema.TYPE_BOOLEAN:
		b, err := strconv.ParseBool(s)
		if err != nil {
			log.Warningf("failed to convert parameter %s to boolean: data '%s' error %+v", name, data, err)
			return false
		}
		return b
	case jschema.TYPE_INTEGER:
		i, err := strconv.Atoi(s)
		if err != nil {
			log.Warningf("failed to convert parameter %s to integer: data '%s' error %+v", name, data, err)
			return 0
		}
		return i
	case jschema.TYPE_NUMBER:
		if !strings.Contains(s, ".") {
			i, err := strconv.Atoi(s)
			if err != nil {
				log.Warningf("failed to convert parameter %s to integer: data '%s' error %+v", name, data, err)
				return 0
			}
			return i
		}
		n, err := strconv.ParseFloat(s, 64)
		if err != nil {
			log.Warningf("failed to convert parameter %s to float: data '%s' error %+v", name, data, err)
			return 0.0
		}
		return n
	default:
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(data), &result); err != nil {
			log.Warningf("failed to convert parameter %s to object: data '%s' error %+v", name, data, err)
		}
		return result
	}
}
