"use strict";
var __decorate = (this && this.__decorate) || function (decorators, target, key, desc) {
    var c = arguments.length, r = c < 3 ? target : desc === null ? desc = Object.getOwnPropertyDescriptor(target, key) : desc, d;
    if (typeof Reflect === "object" && typeof Reflect.decorate === "function") r = Reflect.decorate(decorators, target, key, desc);
    else for (var i = decorators.length - 1; i >= 0; i--) if (d = decorators[i]) r = (c < 3 ? d(r) : c > 3 ? d(target, key, r) : d(target, key)) || r;
    return c > 3 && r && Object.defineProperty(target, key, r), r;
};
Object.defineProperty(exports, "__esModule", { value: true });
var wi_contrib_1 = require("wi-studio/app/contrib/wi-contrib");
var core_1 = require("@angular/core");
var common_1 = require("@angular/common");
var http_1 = require("@angular/http");
var transactionHandler_1 = require("./transactionHandler");
var transactionModule = (function () {
    function transactionModule() {
    }
    return transactionModule;
}());
transactionModule = __decorate([
    core_1.NgModule({
        imports: [
            common_1.CommonModule,
            http_1.HttpModule
        ],
        exports: [],
        declarations: [],
        entryComponents: [],
        providers: [
            { provide: wi_contrib_1.WiServiceContribution, useClass: transactionHandler_1.transactionHandler }
        ],
        bootstrap: []
    })
], transactionModule);
exports.default = transactionModule;
//# sourceMappingURL=transaction.module.js.map