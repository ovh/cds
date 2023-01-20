import { ChangeDetectionStrategy, ChangeDetectorRef, Component, EventEmitter, Input, OnChanges, Output, SimpleChanges } from "@angular/core";
import { FlatElementTypeCondition } from "../../../../model/schema.model";
import { JSONFormSchema } from "../json-form.component";

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
export class JSONFormFieldComponent implements OnChanges {
    @Input() field: FormItem;
    @Input() jsonFormSchema: JSONFormSchema;
    @Input() model: any;
    @Input() parentType: string;
    @Input() disabled: boolean;
    @Output() modelChange = new EventEmitter();

    required: boolean;

    constructor(
        private _cd: ChangeDetectorRef
    ) { }

    ngOnChanges(changes: SimpleChanges): void {
        this.initModel();
        this._cd.markForCheck();
    }

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
            let index = required.indexOf(this.field.name);
            this.required = index !== -1;
            return;
        }
        if (this.jsonFormSchema && this.field.objectType && !this.model[this.field.name]) {
            this.model[this.field.name] = {}
        }
    }
}
