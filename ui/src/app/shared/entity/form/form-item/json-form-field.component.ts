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
    currentModel: any;

    constructor(
        private _cd: ChangeDetectorRef
    ) { }

    ngOnChanges(changes: SimpleChanges): void {
        if (!this.jsonFormSchema || !this.field || !this.model) {
            return;
        }
        this.currentModel = { ...this.model };
        if (!this.currentModel[this.field.name]) {
            this.currentModel[this.field.name] = null;
        }
        this.required = (<string[]>this.jsonFormSchema.types[this.parentType].required).indexOf(this.field.name) !== -1;
        this._cd.markForCheck();
    }

    onValueChanged(value: any): void {
        this.currentModel[this.field.name] = value;
        this._cd.markForCheck();
        this.modelChange.emit(this.currentModel);
    }
}
