import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    EventEmitter,
    Input,
    OnChanges,
    OnInit,
    Output,
    SimpleChanges
} from "@angular/core";
import { FlatSchema } from "../../../model/schema.model";
import { FormItem } from "./form-item/json-form-field.component";
import { DumpOptions, dump, load, LoadOptions } from 'js-yaml'

export class JSONFormSchema {
    types: { [key: string]: JSONFormSchemaTypeItem };
}

export class JSONFormSchemaTypeItem {
    fields: FormItem[];
    required: string[];
}

@Component({
    selector: 'app-json-form',
    templateUrl: './json-form.html',
    styleUrls: ['./json-form.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class JSONFormComponent implements OnInit, OnChanges {
    @Input() schema: FlatSchema;
    @Input() parentType: string;
    @Input() disabled: boolean;
    @Input() data: any;
    @Output() dataChange = new EventEmitter();

    jsonFormSchema: JSONFormSchema;
    model: any;

    constructor(
        private _cd: ChangeDetectorRef
    ) { }

    ngOnInit() {
        const schemaDefs = this.schema.schema['$defs'];
        let allTypes = {};
        this.schema.flatTypes.forEach((v, k) => {
            let items = (v ?? [])
                .filter(value => !value.disabled)
                .map(value => {
                    let item = <FormItem>{
                        name: value.name,
                        type: value.type[0],
                        enum: value.enum,
                        formOrder: value.formOrder,
                        condition: value.condition,
                        description: value.description,
                    };
                    if (item.type === 'object' && value.type.length === 2) {
                        item.objectType = value.type[1];
                    }
                    return item;
                })
                .sort((i, j) => i.formOrder - j.formOrder);
            allTypes[k] = <JSONFormSchemaTypeItem>{
                fields: items,
                required: k === this.parentType ? schemaDefs[k].required : schemaDefs[k]['$defs'][k].required
            };
        });
        this.jsonFormSchema = { types: allTypes };
        this._cd.markForCheck();
    }

    ngOnChanges(changes: SimpleChanges): void {
        try {
            this.model = load(<string>this.data, <LoadOptions>{ onWarning: (e) => { } });
        } catch (e) {
            // TODO: mark form as invalid
        }
        this._cd.markForCheck();
    }

    mergeModelAndData(value: any): void {
        this.model = value;
        this._cd.markForCheck();
        const cleanModel = this.cleanModel(this.parentType, this.model);
        this.dataChange.emit(dump(cleanModel, <DumpOptions>{ lineWidth: 120 }));
    }

    // For given data remove useless fields and empty values
    cleanModel(objectType: string, data: any): any {
        if (!objectType || !data) {
            return null;
        }
        const schema = this.jsonFormSchema.types[objectType];
        let cleanData = {};
        schema.fields.forEach(f => {
            const required = schema.required.indexOf(f.name) !== -1;
            if (f.type === 'object') {
                const subObjectType = (f.condition && f.condition.length > 0) ?
                    f.condition.find(c => data[c.refProperty] && data[c.refProperty] === c.conditionValue)?.type
                    : objectType;
                const cleanSubData = this.cleanModel(subObjectType, data[f.name]);
                if (cleanSubData || required) {
                    cleanData[f.name] = cleanSubData ?? {};
                }
            } else if (data[f.name] || required) {
                cleanData[f.name] = data[f.name] ?? '';
            }
        });
        return cleanData;
    }
}
