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
    types: {};
}

export class JSONFormSchemaTypeItem {
    fields: FormItem[];
    model: {};
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

    constructor(private _cd: ChangeDetectorRef) { }

    ngOnInit() {
        let allTypes = {};
        this.schema.flatTypes.forEach((v, k) => {
            let items = Array<FormItem>();
            let currentModel = {};
            if (v) {
                v.forEach(value => {
                    if (value.disabled) {
                        return;
                    }
                    let item = new FormItem();
                    item.name = value.name;
                    item.type = value.type[0];
                    if (item.type === 'object' && value.type.length === 2) {
                        item.objectType = value.type[1];
                    }
                    item.enum = value.enum;
                    item.formOrder = value.formOrder;
                    item.condition = value.condition;
                    item.description = value.description;
                    items.push(item);
                    if (item.type) {
                        currentModel[item.name] = null;
                    } else {
                        currentModel[item.name] = {};
                    }
                });
            }
            items.sort((i, j) => i.formOrder - j.formOrder);
            let schemaDefs = this.schema.schema['$defs'];
            let required = [];
            if (k === this.parentType) {
                required = schemaDefs[k].required;
            } else {
                required = schemaDefs[k]['$defs'][k].required;
            }
            allTypes[k] = <JSONFormSchemaTypeItem>{ fields: items, model: currentModel, required: required };
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

    mergeModelAndData(): void {
        this.omitEmpty(this.model);
        this.data = dump(this.model, <DumpOptions>{ lineWidth: 120 });
        this.dataChange.emit(this.data);
    }

    omitEmpty(root: any): boolean {
        if (!root) {
            return true;
        }
        let keys = Object.keys(root)
        if (!keys || keys.length === 0) {
            return true;
        }
        if (keys) {
            keys.forEach(k => {
                let newElt = root[k];
                let t = typeof newElt;
                if (t === 'object') {
                    let mustDelete = this.omitEmpty(newElt);
                    if (mustDelete) {
                        delete root[k];
                    }
                }
            });
            keys = Object.keys(root)
            if (!keys || keys.length === 0) {
                return true;
            }
        }
        return false;
    }
}
