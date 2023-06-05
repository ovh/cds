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
    oneOf: Map<string, JSONFormSchemaOneOfItem>;
}

export class JSONFormSchemaOneOfItem {
    keyFormItem: FormItem;
    fields: FormItem[];
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
    @Input() entityType: string;
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
                        pattern: value.pattern,
                        onchange: value.onchange,
                        mode: value.mode,
                        prefix: value.prefix,
                        code: value.code
                    };
                    if ((item.type === 'object' || item.type === 'array') && value.type.length === 2) {
                        item.objectType = value.type[1];
                    }
                    if (item.type === 'map') {
                        item.keyMapType = value.type[1];
                        item.keyMapPattern = value.type[2]
                        item.objectType = value.type[3];
                    }
                    return item;
                })
                .sort((i, j) => i.formOrder - j.formOrder);
            let required = [];
            let oneOf = new Map<string, JSONFormSchemaOneOfItem>();
            if (schemaDefs[k]) {
                // If sub jsonschema
                if (schemaDefs[k]['$defs']) {
                    required = schemaDefs[k]['$defs'][k].required
                } else {
                    required = schemaDefs[k].required
                }
                if (schemaDefs[k].oneOf) {
                    let oneOfListItemName = schemaDefs[k].oneOf.map(o => {
                        return o.required[0];
                    });
                    schemaDefs[k].oneOf.forEach(v => {
                        let oneOfItem = new JSONFormSchemaOneOfItem();
                        let listAllowedItem = items.filter(i => {
                            if (v.not && v.not.required) {
                                if (v.not.required.indexOf(i.name) !== -1) {
                                    return false;
                                }
                            }
                            let indexOf = oneOfListItemName.indexOf(i.name);
                            if (i.name === v.required[0]) {
                                oneOfItem.keyFormItem = i;
                            }
                            return indexOf === -1;
                        });
                        oneOfItem.fields = listAllowedItem;

                        oneOf.set(v.required[0], oneOfItem);
                    });
                }
            }
            allTypes[k] = <JSONFormSchemaTypeItem>{
                fields: items,
                required: required,
                oneOf: oneOf
            };
        });
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
            const required = schema.required && schema.required.indexOf(f.name) !== -1;
            if (f.type === 'object') {
                const subObjectType = (f.condition && f.condition.length > 0) ?
                    f.condition.find(c => data[c.refProperty] && data[c.refProperty] === c.conditionValue)?.type
                    : f.objectType;

                const cleanSubData = this.cleanModel(subObjectType, data[f.name]);
                if (cleanSubData || required) {
                    cleanData[f.name] = cleanSubData ?? {};
                }
            } else if (f.type === 'map') {
                if (data[f.name]) {
                    let keys = Object.keys(data[f.name])
                    keys.forEach(k => {
                        let d = data[f.name][k]
                        if (f.objectType === 'string') {
                            if (!cleanData[f.name]) {
                                cleanData[f.name] = {};
                            }
                            cleanData[f.name][k] = d;
                        } else {
                            const cleanSubData = this.cleanModel(f.objectType, d);
                            if (!cleanData[f.name]) {
                                cleanData[f.name] = {};
                            }
                            cleanData[f.name][k] = cleanSubData;
                        }

                    });
                }
            } else if (f.type === 'array') {
                if (data[f.name]) {
                    data[f.name].forEach((d, i) => {
                        const cleanSubData = this.cleanModel(f.objectType, d);
                        if (cleanSubData) {
                            if (!cleanData[f.name]) {
                                cleanData[f.name] = [];
                            }
                            cleanData[f.name][i] = cleanSubData ?? {};
                        }
                    })
                }
            } else if (data[f.name] || required) {
                cleanData[f.name] = data[f.name] ?? '';
            }

            // One of check
            if (schema.oneOf.size > 0) {
                let keys = Array.from(schema.oneOf.keys());
                let oneOfSelected = data['oneOfSelected'];
                if (oneOfSelected) {
                    keys.forEach(k => {
                        if (!cleanData[k]) {
                            return;
                        }
                        if (k !== oneOfSelected) {
                            delete cleanData[k];
                        } else {
                            let currentKeys = Object.keys(cleanData);
                            let allowedKeys = schema.oneOf.get(k).fields;
                            currentKeys.forEach(subKey => {
                                let ff = allowedKeys.find(i => {
                                    return i.name === subKey || i.name === k
                                });
                                if (!ff && subKey !== k) {
                                    delete cleanData[subKey];
                                }
                            });
                        }
                    });
                }
            }
        });
        return cleanData;
    }
}
