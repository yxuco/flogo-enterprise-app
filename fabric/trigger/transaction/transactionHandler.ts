
import {Injectable, Inject, Injector} from "@angular/core";
import {Http} from "@angular/http";
import {Observable} from "rxjs/Observable";
import {
    WiContrib,
    WiServiceHandlerContribution,
    IActionResult,
    ActionResult,
    WiContribModelService,
    ICreateFlowActionContext,
    CreateFlowActionResult
} from "wi-studio/app/contrib/wi-contrib";
import { ITriggerContribution, IFieldDefinition, MODE } from "wi-studio/common/models/contrib";
import { IValidationResult, ValidationResult } from "wi-studio/common/models/validation";
import * as lodash from "lodash";

@WiContrib({})
@Injectable()
export class transactionHandler extends WiServiceHandlerContribution {

    constructor(private injector: Injector, private http: Http, private contribModelService: WiContribModelService) {
        super(injector, http, contribModelService);
    }

    value = (fieldName: string, context: ITriggerContribution): Observable<any> | any => {
        return null;
    }

    // verify user entries are valid JSON string
    validate = (fieldName: string, context: ITriggerContribution): Observable<IValidationResult> | IValidationResult => {
        if (fieldName === "parameters") {
            if (context.getMode() === MODE.WIZARD || context.getMode() === MODE.SERVERLESS_FLOW) {
                let parameters: IFieldDefinition = context.getField("parameters");
                let valRes;
                if (parameters.value) {
                    try {
                        valRes = JSON.parse(parameters.value);
                        valRes = JSON.stringify(valRes);
                    } catch (e) {
                        return ValidationResult.newValidationResult().setError("SCHEMA_ERROR", "Unexpected string in JSON");
                    }
                }
            }
        } else if (fieldName === "returns") {
            if (context.getMode() === MODE.WIZARD || context.getMode() === MODE.SERVERLESS_FLOW) {
                let returns: IFieldDefinition = context.getField("returns");
                let valRes;
                if (returns.value) {
                    try {
                        valRes = JSON.parse(returns.value);
                        valRes = JSON.stringify(valRes);
                    } catch (e) {
                        return ValidationResult.newValidationResult().setError("SCHEMA_ERROR", "Unexpected string in JSON");
                    }
                }
            }
        }
        return null;
    }

    // used to configure trigger with data from "Add a trigger" wizard
    action = (actionId: string, context: ICreateFlowActionContext): Observable<IActionResult> | IActionResult => {
        let modelService = this.getModelService();
        let result = CreateFlowActionResult.newActionResult();
        if (context.handler && context.handler.settings && context.handler.settings.length > 0) {
            let txnName = <IFieldDefinition>context.getField("name");
            let parameters = <IFieldDefinition>context.getField("parameters");
            let returns = <IFieldDefinition>context.getField("returns");
            if (txnName && txnName.value) {
                let trigger = modelService.createTriggerElement("fabric/fabric-transaction");
                if (trigger && trigger.handler && trigger.handler.settings && trigger.handler.settings.length > 0) {
                    for (let j = 0; j < trigger.handler.settings.length; j++) {
                        if (trigger.handler.settings[j].name === "name") {
                            trigger.handler.settings[j].value = txnName.value;
                            break;
                        }
                    }
                }
                if (trigger && trigger.outputs && trigger.outputs.length > 0) {
                    for (let j = 0; j < trigger.outputs.length; j++) {
                        if (trigger.outputs[j].name === "parameters") {
                            trigger.outputs[j].value = {
                                "value": parameters.value,
                                "metadata": ""
                            };
                            break;
                        }
                    }
                }
                if (trigger && trigger.reply && trigger.reply.length > 0) {
                    for (let j = 0; j < trigger.reply.length; j++) {
                        if (trigger.reply[j].name === "returns") {
                            trigger.reply[j].value = {
                                "value": returns.value,
                                "metadata": ""
                            };
                            break;
                        }
                    }
                }
                let flowModel = modelService.createFlow(txnName.value, context.getFlowDescription());
                result = result.addTriggerFlowMapping(lodash.cloneDeep(trigger), lodash.cloneDeep(flowModel));
            }
        }
        let actionResult = ActionResult.newActionResult().setSuccess(true).setResult(result);
        return actionResult;
    }
}
