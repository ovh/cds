import {ChangeDetectionStrategy, Component, EventEmitter, Input, Output} from "@angular/core";
import {JSONFormSchema} from "../json-form.component";
import {FormItem} from "../form-item/json-form-field.component";

@Component({
    selector: 'app-json-form-field-conditional',
    templateUrl: './json-form-field-conditional.html',
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class JSONFormFieldConditionalComponent {

    _field: FormItem;
    @Input() set field(data: FormItem) {
        this._field = data;
        if (data && this.model) {
            this.initCondition();
        }
    };
    get field(): FormItem {
        return this._field;
    }

    _jsonFormSchema: JSONFormSchema
    @Input() set jsonFormSchema(data: JSONFormSchema) {
        this._jsonFormSchema = data;
    }
    get jsonFormSchema(): JSONFormSchema {
        return this._jsonFormSchema;
    }

    _model: any;
    @Input() set model(model: any) {
        this._model = model;
        if (this.field && model) {
            this.initCondition();
        }
    }
    get model() {
        return this._model;
    }

    currentType: string;

    @Output() modelChange = new EventEmitter();

    emitChange(): void {
        if (!this.model[this.field.name]) {
            delete this.model[this.field.name];
        }
        this.modelChange.emit(this.model);
    }

    initCondition(): void {
        if (this._field.condition) {
            for (let i=0; i<this._field.condition.length; i++) {
                let c = this._field.condition[i];
                if (this._model[c.refProperty] === c.conditionValue) {
                    if (this.currentType && this.currentType !== c.type) {
                        this._model[this.field.name] = Object.assign({}, this._jsonFormSchema.types[c.type].model);
                    }
                    this.currentType = c.type;
                }
            }
        }
        if (!this._model[this.field.name]) {
            let newModel = Object.assign({}, this._jsonFormSchema.types[this.currentType].model);
            this._jsonFormSchema.types[this.currentType].required.forEach(r => {
                newModel[r] = '';
            })
            this._model[this.field.name] = newModel;
        }
        this.emitChange();
    }
}
