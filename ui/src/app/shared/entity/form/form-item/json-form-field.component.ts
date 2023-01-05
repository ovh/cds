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
}
@Component({
    selector: 'app-json-form-field',
    templateUrl: './json-form-field.html',
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class JSONFormFieldComponent {
    @Input() field: FormItem;

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

    @Output() modelChange = new EventEmitter();

    emitChange(): void {
        if (!this.model[this.field.name]) {
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
            return;
        }
        if (this.jsonFormSchema && this.field.objectType && !this._model[this.field.name]) {
            this._model[this.field.name] = {}
        }
    }
}
