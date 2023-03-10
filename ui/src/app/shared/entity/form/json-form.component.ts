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
    @Input() data: string;
    @Output() dataChange = new EventEmitter();

    jsonFormSchema: JSONFormSchema;
    model: any;

    constructor(
        private _cd: ChangeDetectorRef
    ) { }

    ngOnInit() {
        const schemaDefs = this.schema.schema['$defs'];
        let allTypes = {};
        console.log(this.schema);
        this.schema.flatTypes.forEach((v, k) => {
            if (k === 'V2Action') {
                console.log(v);
            }
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
                        pattern: value.pattern,
                    };
                    if ( (item.type === 'object' || item.type === 'array') && value.type.length === 2) {
                        item.objectType = value.type[1];
                    }
                    if (item.type === 'map') {
                        item.keyMapType = value.type[1];
                        item.objectType = value.type[2];
                    }
                    return item;
                })
                .sort((i, j) => i.formOrder - j.formOrder);
            let required = [];
            if (schemaDefs[k]) {
                // If sub jsonschema
               if (schemaDefs[k]['$defs']) {
                   required = schemaDefs[k]['$defs'][k].required
               } else {
                   required = schemaDefs[k].required
               }
            }
            allTypes[k] = <JSONFormSchemaTypeItem>{
                fields: items,
                required: required
            };
        });
        console.log(allTypes);
        this.jsonFormSchema = { types: allTypes };
        this._cd.markForCheck();
    }

    ngOnChanges(changes: SimpleChanges): void {
        try {
            this.model = load(this.data && this.data !== '' ? this.data : '{}', <LoadOptions>{ onWarning: (e) => { } });
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
