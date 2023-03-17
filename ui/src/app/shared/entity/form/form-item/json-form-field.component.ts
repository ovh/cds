import { ChangeDetectionStrategy, ChangeDetectorRef, Component, EventEmitter, Input, OnChanges, Output, SimpleChanges } from "@angular/core";
import { FlatElementTypeCondition } from "../../../../model/schema.model";
import {JSONFormSchema, JSONFormSchemaOneOfItem} from "../json-form.component";
import {EntityAction} from "../../../../model/project.model";
import {ProjectService} from "../../../../service/project/project.service";
import {ActivatedRoute} from "@angular/router";
import {ProjectState} from "../../../../store/project.state";
import {Store} from "@ngxs/store";
import { load, LoadOptions } from 'js-yaml'

export class FormItem {
    name: string;
    type: string;
    objectType?: string;
    keyMapType?: string;
    keyMapPattern?: string;
    enum?: string[];
    formOrder: number;
    condition: FlatElementTypeCondition[];
    description: string;
    pattern: string;
    onchange: string;
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
    @Input() hideLabel: boolean;
    @Output() modelChange = new EventEmitter();

    required: boolean;
    oneOf: Map<string, JSONFormSchemaOneOfItem>;
    oneOfSelected: string[] = new Array<string>();
    oneOfSelectOpts: string[];


    currentModel: any;
    isConditionnal: boolean;
    selectedCondition: FlatElementTypeCondition;
    conditionRefProperties: string[];

    constructor(
        private _cd: ChangeDetectorRef,
        private _projectService: ProjectService,
        private _store: Store,
        private _activatedRouter: ActivatedRoute
    ) { }

    ngOnChanges(changes: SimpleChanges): void {
        if (!this.jsonFormSchema || !this.field || !this.model) {
            return;
        }
        this.currentModel = Object.assign({}, this.model);
        if (!this.currentModel[this.field.name]) {
            this.currentModel[this.field.name] = null;
        }

        if (this.jsonFormSchema.types[this.parentType].required) {
            this.required = (<string[]>this.jsonFormSchema.types[this.parentType].required)?.indexOf(this.field.name) !== -1;
        } else {
            this.required = false;
        }

        // Init oneOf data to display select
        if (this.field.objectType && this.jsonFormSchema.types[this.field.objectType]?.oneOf?.size > 0) {
            this.oneOf = this.jsonFormSchema.types[this.field.objectType].oneOf;
            this.oneOfSelectOpts = Array.from(this.oneOf.keys());
            if (this.oneOfSelected.length === 0 && this.currentModel[this.field.name]) {
                this.currentModel[this.field.name].forEach((v, i) => {
                    this.oneOfSelectOpts.forEach(opt => {
                        if (v[opt]) {
                            this.oneOfSelected[i] = opt;
                        }
                    })
                });
            }
        }

        this.isConditionnal = this.field.condition && this.field.condition.length > 0;
        this.selectedCondition = (this.field.condition ?? []).find(c => this.currentModel[c.refProperty] && this.currentModel[c.refProperty] === c.conditionValue);
        this.conditionRefProperties = (this.field.condition ?? []).map(c => c.refProperty).filter((ref, index, arr) => arr.indexOf(ref) === index);
        this._cd.markForCheck();
    }

    trackByIndex(index: number) {
        return index;
    }

    updateItemStruct(index: number) {
        this.currentModel[this.field.name][index]['oneOfSelected'] = this.oneOfSelected[index];
        this._cd.markForCheck();
        this.modelChange.emit(this.currentModel);
    }

    addArrayItem() {
        this.currentModel[this.field.name].push({})
        this.oneOfSelected.push(this.oneOfSelectOpts[0]);
        this._cd.markForCheck();
    }

    addMapItem() {
        this.currentModel[this.field.name][''] = {};
        this._cd.markForCheck();
    }

    onKeyMapChanged(value: any, previousValue): void {
        this.currentModel[this.field.name][value] = this.currentModel[this.field.name][previousValue];
        delete this.currentModel[this.field.name][previousValue];

        this._cd.markForCheck();
        this.modelChange.emit(this.currentModel);
    }

    onValueChanged(value: any, index?: any): void {
        switch (this.field.onchange) {
            case 'loadentity':
                let branch: string;
                let branchSplit = value.split('@');
                if (branchSplit.length === 2) {
                    branch = branchSplit[1];
                }

                let routeParams = this._activatedRouter.snapshot.params;
                let actionName = routeParams['actionName'];
                let repoName = routeParams['repoName'];
                let vcsName = routeParams['vcsName'];
                let project = this._store.selectSnapshot(ProjectState.projectSnapshot);

                this._projectService.getRepoEntity(project.key, vcsName, repoName, EntityAction, actionName, branch).subscribe(e => {
                    let action = load(e.data && e.data !== '' ? e.data : '{}', <LoadOptions>{ onWarning: (e) => { } });
                    console.log(action);
                });

                break;
            default:
                if (this.field.type === 'array' || this.field.type === 'map') {
                    this.currentModel[this.field.name][index] = value;
                } else {
                    this.currentModel[this.field.name] = value;
                }
                this._cd.markForCheck();
                this.modelChange.emit(this.currentModel);
        }
    }
}
