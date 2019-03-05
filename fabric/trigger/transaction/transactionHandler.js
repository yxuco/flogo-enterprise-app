"use strict";
var __extends = (this && this.__extends) || (function () {
    var extendStatics = Object.setPrototypeOf ||
        ({ __proto__: [] } instanceof Array && function (d, b) { d.__proto__ = b; }) ||
        function (d, b) { for (var p in b) if (b.hasOwnProperty(p)) d[p] = b[p]; };
    return function (d, b) {
        extendStatics(d, b);
        function __() { this.constructor = d; }
        d.prototype = b === null ? Object.create(b) : (__.prototype = b.prototype, new __());
    };
})();
var __decorate = (this && this.__decorate) || function (decorators, target, key, desc) {
    var c = arguments.length, r = c < 3 ? target : desc === null ? desc = Object.getOwnPropertyDescriptor(target, key) : desc, d;
    if (typeof Reflect === "object" && typeof Reflect.decorate === "function") r = Reflect.decorate(decorators, target, key, desc);
    else for (var i = decorators.length - 1; i >= 0; i--) if (d = decorators[i]) r = (c < 3 ? d(r) : c > 3 ? d(target, key, r) : d(target, key)) || r;
    return c > 3 && r && Object.defineProperty(target, key, r), r;
};
var __metadata = (this && this.__metadata) || function (k, v) {
    if (typeof Reflect === "object" && typeof Reflect.metadata === "function") return Reflect.metadata(k, v);
};
Object.defineProperty(exports, "__esModule", { value: true });
var core_1 = require("@angular/core");
var http_1 = require("@angular/http");
var wi_contrib_1 = require("wi-studio/app/contrib/wi-contrib");
var contrib_1 = require("wi-studio/common/models/contrib");
var validation_1 = require("wi-studio/common/models/validation");
var lodash = require("lodash");
var transactionHandler = (function (_super) {
    __extends(transactionHandler, _super);
    function transactionHandler(injector, http, contribModelService) {
        var _this = _super.call(this, injector, http, contribModelService) || this;
        _this.injector = injector;
        _this.http = http;
        _this.contribModelService = contribModelService;
        _this.value = function (fieldName, context) {
            return null;
        };
        _this.validate = function (fieldName, context) {
            if (fieldName === "parameters") {
                if (context.getMode() === contrib_1.MODE.WIZARD || context.getMode() === contrib_1.MODE.SERVERLESS_FLOW) {
                    var parameters = context.getField("parameters");
                    var valRes = void 0;
                    if (parameters.value) {
                        try {
                            valRes = JSON.parse(parameters.value);
                            valRes = JSON.stringify(valRes);
                        }
                        catch (e) {
                            return validation_1.ValidationResult.newValidationResult().setError("SCHEMA_ERROR", "Unexpected string in JSON");
                        }
                    }
                }
            }
            else if (fieldName === "returns") {
                if (context.getMode() === contrib_1.MODE.WIZARD || context.getMode() === contrib_1.MODE.SERVERLESS_FLOW) {
                    var returns = context.getField("returns");
                    var valRes = void 0;
                    if (returns.value) {
                        try {
                            valRes = JSON.parse(returns.value);
                            valRes = JSON.stringify(valRes);
                        }
                        catch (e) {
                            return validation_1.ValidationResult.newValidationResult().setError("SCHEMA_ERROR", "Unexpected string in JSON");
                        }
                    }
                }
            }
            return null;
        };
        _this.action = function (actionId, context) {
            var modelService = _this.getModelService();
            var result = wi_contrib_1.CreateFlowActionResult.newActionResult();
            if (context.handler && context.handler.settings && context.handler.settings.length > 0) {
                var txnName = context.getField("name");
                var parameters = context.getField("parameters");
                var returns = context.getField("returns");
                if (txnName && txnName.value) {
                    var trigger = modelService.createTriggerElement("fabric/fabric-transaction");
                    if (trigger && trigger.handler && trigger.handler.settings && trigger.handler.settings.length > 0) {
                        for (var j = 0; j < trigger.handler.settings.length; j++) {
                            if (trigger.handler.settings[j].name === "name") {
                                trigger.handler.settings[j].value = txnName.value;
                                break;
                            }
                        }
                    }
                    if (trigger && trigger.outputs && trigger.outputs.length > 0) {
                        for (var j = 0; j < trigger.outputs.length; j++) {
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
                        for (var j = 0; j < trigger.reply.length; j++) {
                            if (trigger.reply[j].name === "returns") {
                                trigger.reply[j].value = {
                                    "value": returns.value,
                                    "metadata": ""
                                };
                                break;
                            }
                        }
                    }
                    var flowModel = modelService.createFlow(txnName.value, context.getFlowDescription());
                    result = result.addTriggerFlowMapping(lodash.cloneDeep(trigger), lodash.cloneDeep(flowModel));
                }
            }
            var actionResult = wi_contrib_1.ActionResult.newActionResult().setSuccess(true).setResult(result);
            return actionResult;
        };
        return _this;
    }
    return transactionHandler;
}(wi_contrib_1.WiServiceHandlerContribution));
transactionHandler = __decorate([
    wi_contrib_1.WiContrib({}),
    core_1.Injectable(),
    __metadata("design:paramtypes", [core_1.Injector, http_1.Http, wi_contrib_1.WiContribModelService])
], transactionHandler);
exports.transactionHandler = transactionHandler;
//# sourceMappingURL=transactionHandler.js.map