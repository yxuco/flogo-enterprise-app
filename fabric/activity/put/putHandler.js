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
var __param = (this && this.__param) || function (paramIndex, decorator) {
    return function (target, key) { decorator(target, key, paramIndex); }
};
Object.defineProperty(exports, "__esModule", { value: true });
var core_1 = require("@angular/core");
var Observable_1 = require("rxjs/Observable");
var wi_contrib_1 = require("wi-studio/app/contrib/wi-contrib");
var validation_1 = require("wi-studio/common/models/validation");
var putHandler = (function (_super) {
    __extends(putHandler, _super);
    function putHandler(injector) {
        var _this = _super.call(this, injector) || this;
        _this.value = function (fieldName, context) {
            return null;
        };
        _this.validate = function (fieldName, context) {
            if (fieldName === "value") {
                var vresult = validation_1.ValidationResult.newValidationResult();
                var valueTypeFieldDef = context.getField("valueType");
                var valueFieldDef = context.getField("value");
                if (valueTypeFieldDef.value && valueTypeFieldDef.value === "object") {
                    vresult.setVisible(false);
                }
                else {
                    vresult.setVisible(true);
                }
                return vresult;
            }
            else if (fieldName === "collection") {
                var vresult = validation_1.ValidationResult.newValidationResult();
                var isPrivateFieldDef = context.getField("isPrivate");
                var collectionFieldDef = context.getField("collection");
                if (isPrivateFieldDef.value && isPrivateFieldDef.value === true) {
                    if (collectionFieldDef.display && collectionFieldDef.display.visible == false) {
                        vresult.setVisible(true);
                    }
                }
                else {
                    vresult.setVisible(false);
                }
                return vresult;
            }
            else if (fieldName === "data") {
                var vresult = validation_1.ValidationResult.newValidationResult();
                var valueTypeFieldDef = context.getField("valueType");
                var dataFieldDef = context.getField("data");
                if (valueTypeFieldDef.value && valueTypeFieldDef.value === "object") {
                    if (dataFieldDef.display && dataFieldDef.display.visible == false) {
                        vresult.setVisible(true);
                    }
                    if (dataFieldDef.value === null || dataFieldDef.value === "") {
                        vresult.setError("FABTIC-PUT-1010", "Data definition must not be empty");
                    }
                    else {
                        var valRes = void 0;
                        try {
                            valRes = JSON.parse(dataFieldDef.value.value);
                            valRes = JSON.stringify(valRes);
                        }
                        catch (e) {
                            vresult.setError("FABTIC-PUT-1020", "Invalid JSON: " + e.toString());
                        }
                    }
                }
                else {
                    vresult.setVisible(false);
                }
                return vresult;
            }
            return null;
        };
        _this.action = function (actionId, context) {
            return Observable_1.Observable.create(function (observer) {
                var aresult = wi_contrib_1.ActionResult.newActionResult();
                observer.next(aresult);
            });
        };
        return _this;
    }
    return putHandler;
}(wi_contrib_1.WiServiceHandlerContribution));
putHandler = __decorate([
    wi_contrib_1.WiContrib({}),
    core_1.Injectable(),
    __param(0, core_1.Inject(core_1.Injector)),
    __metadata("design:paramtypes", [Object])
], putHandler);
exports.putHandler = putHandler;
//# sourceMappingURL=putHandler.js.map