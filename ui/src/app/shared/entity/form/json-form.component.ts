import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    EventEmitter,
    Input,
    OnInit,
    Output
} from "@angular/core";
import {FlatSchema} from "../../../model/schema.model";
import {FormItem} from "./form-item/json-form-field.component";
import {DumpOptions, dump, load, LoadOptions} from 'js-yaml'

export class JSONFormSchema {
    fields: FormItem[];
    types: {};
}

export class JSONFormSchemaTypeItem {
    fields: FormItem[];
    model: {};
}

@Component({
    selector: 'app-json-form',
    template: `
    <ng-container *ngIf="jsonFormSchema && model">
        <ng-container *ngFor="let f of jsonFormSchema.fields">
            <app-json-form-field [field]="f" [jsonFormSchema]="jsonFormSchema" [(model)]="model" (modelChange)="mergeModelAndData()"></app-json-form-field>
        </ng-container>
    </ng-container>`,
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class JSONFormComponent implements OnInit {

    @Input() schema: FlatSchema;

    _data: {};
    @Input() set data(d: any) {
        this._data = d;
        try {
            this.model = load(<string>this._data, <LoadOptions>{onWarning: (e)=> {}});
        } catch (e) {
            // TODO: mark form as invalid
        }
    }
    get data() {
        return this._data;
    }
    @Output() dataChange = new EventEmitter();

    jsonFormSchema: JSONFormSchema;
    model: any;

    constructor(private _cd: ChangeDetectorRef) {}

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
                   items.push(item);
                   if (item.type) {
                       currentModel[item.name] = null;
                   } else {
                       currentModel[item.name] = {};
                   }
               });
           }
           items.sort((i, j) => i.formOrder - j.formOrder);
           allTypes[k] = <JSONFormSchemaTypeItem>{fields: items, model: currentModel};
        });

        let mainRef = this.schema.schema.$ref.replace('#/$defs/', '');
        this.jsonFormSchema = {fields: allTypes[mainRef].fields, types: allTypes};
        this._cd.markForCheck();
    }

    mergeModelAndData(): void {
        this.omitEmpty(this.model);
        this._data = dump(this.model, <DumpOptions>{lineWidth: 120});
        this.dataChange.emit(this._data);
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
