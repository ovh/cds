import {ChangeDetectionStrategy, Component, EventEmitter, Input, Output} from "@angular/core";
import {FlatElementTypeCondition} from "../../../../model/schema.model";
import {JSONFormSchema} from "../json-form.component";

export class FormItem {
    name: string;
    type: string;
    objectType?: string;
    enum?: string[];
    formOrder: number;
    condition: FlatElementTypeCondition[];
    description: string;
}
@Component({
    selector: 'app-json-form-field',
    templateUrl: './json-form-field.html',
    styleUrls: ['./json-form-field.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class JSONFormFieldComponent {

    _field: FormItem;
    @Input() set field(data: FormItem) {
        this._field = data;
        this.initModel();
    };

    get field(): FormItem {
        return this._field;
    }

    _jsonFormSchema: JSONFormSchema
    @Input() set jsonFormSchema(data: JSONFormSchema) {
        this._jsonFormSchema = data;
        this.initModel();
    }
    get jsonFormSchema(): JSONFormSchema {
        return this._jsonFormSchema;
    }

    _model: any;
    @Input() set model(model: any) {
        this._model = model;
        this.initModel();
    }
    get model() {
        return this._model;
    }

    @Input() parentType: string;

    @Output() modelChange = new EventEmitter();

    required: boolean;

    emitChange(): void {
        let required = <string[]>this.jsonFormSchema.types[this.parentType].required;
        if (!this.model[this.field.name] && required.indexOf(this.field.name) === -1) {
            delete this.model[this.field.name];
        }
        this.modelChange.emit(this.model);
    }

    updateParentModel(parentField: string, childModel: {}) {
        this.model[parentField] = childModel;
        this.emitChange();
    }

    initModel() {
        if (!this.jsonFormSchema || !this.field || !this.model) {
            return;
        }
        if (this.field.type !== 'object') {
            // check required
            let required = <string[]>this.jsonFormSchema.types[this.parentType].required;
            let index = required.indexOf(this._field.name);
            this.required = index !== -1;
            return;
        }
        if (this.jsonFormSchema && this.field.objectType && !this._model[this.field.name]) {
            this._model[this.field.name] = {}
        }
    }
}
