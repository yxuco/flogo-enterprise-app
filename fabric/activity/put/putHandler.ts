
import {Injectable, Injector, Inject} from "@angular/core";
import {Http} from "@angular/http";
import {Observable} from "rxjs/Observable";
import {
    WiContrib,
    WiServiceHandlerContribution,
    IContributionTypes,
    ActionResult,
    IActionResult,
    WiContribModelService,
    IFieldDefinition,
    IActivityContribution    
} from "wi-studio/app/contrib/wi-contrib";
import { IValidationResult, ValidationResult } from "wi-studio/common/models/validation";

@WiContrib({})
@Injectable()
export class putHandler extends WiServiceHandlerContribution {
    constructor( @Inject(Injector) injector) {
        super(injector);
    }

    value = (fieldName: string, context: IActivityContribution): Observable<any> | any => {
        return null;
    }

    validate = (fieldName: string, context: IActivityContribution): Observable<IValidationResult> | IValidationResult => {
        if (fieldName === "value") {
            let vresult: IValidationResult = ValidationResult.newValidationResult();
            let valueTypeFieldDef: IFieldDefinition = context.getField("valueType");
            let valueFieldDef: IFieldDefinition = context.getField("value");
            if (valueTypeFieldDef.value && valueTypeFieldDef.value === "object") {
                vresult.setVisible(false);
            } else {
                vresult.setVisible(true);
            }
            return vresult;
        } else if (fieldName === "collection") {
            let vresult: IValidationResult = ValidationResult.newValidationResult();
            let isPrivateFieldDef: IFieldDefinition = context.getField("isPrivate");
            let collectionFieldDef: IFieldDefinition = context.getField("collection");
            if (isPrivateFieldDef.value && isPrivateFieldDef.value === true) {
                if (collectionFieldDef.display && collectionFieldDef.display.visible == false) {
                    vresult.setVisible(true);
                }
            } else {
                vresult.setVisible(false);
            }
            return vresult;
        } else if (fieldName === "data") {
            let vresult: IValidationResult = ValidationResult.newValidationResult();
            let valueTypeFieldDef: IFieldDefinition = context.getField("valueType");
            let dataFieldDef: IFieldDefinition = context.getField("data");
            if (valueTypeFieldDef.value && valueTypeFieldDef.value === "object") {
                if (dataFieldDef.display && dataFieldDef.display.visible == false) {
                    vresult.setVisible(true);
                }
                if (dataFieldDef.value === null || dataFieldDef.value === "") {
                    vresult.setError("FABTIC-PUT-1010", "Data definition must not be empty");
                } else {
                    let valRes;
                    try {
                        valRes = JSON.parse(dataFieldDef.value.value);
                        valRes = JSON.stringify(valRes);
                    } catch (e) {
                        vresult.setError("FABTIC-PUT-1020", "Invalid JSON: " + e.toString());
                    }
                }
            } else {
                vresult.setVisible(false)
            }
            return vresult;
        }
        return null;
    }

    action = (actionId: string, context: IContributionTypes): Observable<IActionResult> | IActionResult => {
        return Observable.create(observer => {
            let aresult: IActionResult = ActionResult.newActionResult();
            observer.next(aresult);
        });
    }
}
