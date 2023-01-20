import { ChangeDetectionStrategy, ChangeDetectorRef, Component, EventEmitter, Input, OnChanges, Output, SimpleChanges } from "@angular/core";
import { JSONFormSchema } from "../json-form.component";
import { FormItem } from "../form-item/json-form-field.component";

@Component({
    selector: 'app-json-form-field-conditional',
    templateUrl: './json-form-field-conditional.html',
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class JSONFormFieldConditionalComponent implements OnChanges {
    @Input() disabled: boolean;
    @Input() field: FormItem;
    @Input() jsonFormSchema: JSONFormSchema;
    @Input() model: any;
    @Output() modelChange = new EventEmitter();

    currentType: string;

    constructor(
        private _cd: ChangeDetectorRef
    ) { }

    ngOnChanges(changes: SimpleChanges): void {
        if (this.field && this.model) {
            this.initCondition();
        }
        this._cd.markForCheck();
    }

    emitChange(): void {
        if (!this.model[this.field.name]) {
            delete this.model[this.field.name];
        }
        this.modelChange.emit(this.model);
    }

    initCondition(): void {
        if (this.field.condition) {
            for (let i = 0; i < this.field.condition.length; i++) {
                let c = this.field.condition[i];
                if (this.model[c.refProperty] === c.conditionValue) {
                    if (this.currentType && this.currentType !== c.type) {
                        this.model[this.field.name] = Object.assign({}, this.jsonFormSchema.types[c.type].model);
                    }
                    this.currentType = c.type;
                }
            }
        }
        if (!this.model[this.field.name]) {
            let newModel = Object.assign({}, this.jsonFormSchema.types[this.currentType].model);
            this.jsonFormSchema.types[this.currentType].required.forEach(r => {
                newModel[r] = '';
            })
            this.model[this.field.name] = newModel;
        }
        this.emitChange();
    }
}
