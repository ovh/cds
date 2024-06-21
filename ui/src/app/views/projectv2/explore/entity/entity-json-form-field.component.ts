import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    EventEmitter,
    Input,
    OnChanges,
    OnDestroy,
    OnInit,
    Output,
    SimpleChanges,
    ViewChild
} from "@angular/core";
import { AutoUnsubscribe } from "app/shared/decorator/autoUnsubscribe";
import { FormItem, JSONFormSchema, JSONFormSchemaOneOfItem } from "./form.model";
import { EntityType } from "app/model/entity.model";
import { NzCodeEditorComponent } from "ng-zorro-antd/code-editor";
import { FlatElementTypeCondition } from "app/model/schema.model";
import { Subscription } from "rxjs";
import { ProjectService } from "app/service/project/project.service";
import { PluginService } from "app/service/plugin.service";
import { Store } from "@ngxs/store";
import { ActivatedRoute } from "@angular/router";
import { DragulaService } from "ng2-dragula";
import { PreferencesState } from "app/store/preferences.state";
import { ProjectState } from "app/store/project.state";
import { LoadOptions, load } from "js-yaml";

class selectValues {
    label: string;
    description: string;
    branch: string;
    value: string;
}

@Component({
    selector: 'app-entity-json-form-field',
    templateUrl: './entity-json-form-field.html',
    styleUrls: ['./entity-json-form-field.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class EntityJSONFormFieldComponent implements OnInit, OnChanges, OnDestroy {
    @ViewChild('editor') editor: NzCodeEditorComponent;

    @Input() field: FormItem;
    @Input() jsonFormSchema: JSONFormSchema;
    @Input() model: any;
    @Input() parentType: string;
    @Input() disabled: boolean;
    @Input() hideLabel: boolean;
    @Input() entityType: EntityType;
    @Input() indent: number = 0;
    @Output() modelChange = new EventEmitter();

    required: boolean;
    oneOf: Map<string, JSONFormSchemaOneOfItem>;
    oneOfSelected: string[] = new Array<string>();
    oneOfSelectOpts: string[];
    timestamp = Date.now();
    currentModel: any;
    isConditionnal: boolean;
    selectedCondition: FlatElementTypeCondition;
    conditionRefProperties: string[];
    resizingSubscription: Subscription;
    values: selectValues[] = [];

    constructor(
        private _cd: ChangeDetectorRef,
        private _projectService: ProjectService,
        private _pluginService: PluginService,
        private _store: Store,
        private _activatedRouter: ActivatedRoute,
        private _dragulaService: DragulaService
    ) {
        this.trackByIndex = this.trackByIndex.bind(this);
    }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit(): void {
        if (!this._dragulaService.find('array-field')) {
            this._dragulaService.createGroup('array-field', {
                moves(el, source, handle) {
                    const f = (element: Element) => {
                        if (element.classList.contains('move')) {
                            return true;
                        }
                        ;
                        if (element.parentElement) {
                            return f(element.parentElement);
                        }
                        return false;
                    };
                    return f(handle);
                },
                direction: 'vertical'
            });
        }

        this.resizingSubscription = this._store.select(PreferencesState.resizing).subscribe(resizing => {
            if (this.editor && !resizing) {
                this.editor.layout();
            }
            this._cd.markForCheck();
        });
    }

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

        if (this.field?.mode === 'split' && this.field?.enum?.length > 0) {
            this.values = [];
            let tmpMapModel = new Map<string, boolean>();
            this.field.enum.forEach(e => {
                let v = new selectValues();
                let itemSplit = e.split('@')
                if (itemSplit.length == 2) {
                    v.branch = itemSplit[1];
                }
                let pathSplit = itemSplit[0].split('/');
                v.label = pathSplit[pathSplit.length - 1];
                v.description = itemSplit[0].replace('/' + v.label, '');
                v.value = e;
                this.values.push(v);

                if (!tmpMapModel.has(itemSplit[0])) {
                    tmpMapModel.set(itemSplit[0], true);
                    let valueWithoutBranch = new selectValues();
                    valueWithoutBranch.label = v.label;
                    valueWithoutBranch.description = v.description;
                    valueWithoutBranch.value = v.description + '/' + v.label;
                    this.values.push(valueWithoutBranch);
                }
            });
        }

        this._cd.markForCheck();
    }

    trackByIndex(index: number, v) {
        return this.timestamp + '-' + index;
    }

    updateItemStruct(index: number) {
        this.currentModel[this.field.name][index]['oneOfSelected'] = this.oneOfSelected[index];
        this._cd.markForCheck();
        this.modelChange.emit(this.currentModel);
    }

    addArrayItem() {
        if (!this.currentModel[this.field.name]) {
            this.currentModel[this.field.name] = [];
        }
        if (this.field.type === 'string') {
            this.currentModel[this.field.name].push('');
        } else {
            this.currentModel[this.field.name].push({});
        }
        if (this.oneOf) {
            this.oneOfSelected.push(this.oneOfSelectOpts[0]);
        }
        this._cd.markForCheck();
    }

    addMapItem() {
        if (!this.currentModel[this.field.name]) {
            this.currentModel[this.field.name] = {}
        }
        if (this.field.objectType === 'string') {
            this.currentModel[this.field.name] = [];
        }
        this._cd.markForCheck();
    }

    onKeyMapChanged(value: any, previousValue): void {
        this.currentModel[this.field.name][value] = this.currentModel[this.field.name][previousValue];
        delete this.currentModel[this.field.name][previousValue];

        this._cd.markForCheck();
        this.modelChange.emit(this.currentModel);
    }

    onFullArrayChanged(value: any): void {
        this.currentModel[this.field.name] = value;
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

                if (this.field.prefix) {
                    branchSplit[0] = branchSplit[0].slice(this.field.prefix.length);
                }
                let routeParams = this._activatedRouter.snapshot.params;
                let entitySplit = branchSplit[0].split('/');
                let entityName = '';
                let repoName = '';
                let vcsName = '';
                let projKey = '';
                switch (entitySplit.length) {
                    case 1:
                        entityName = entitySplit[0];
                        repoName = routeParams['repoName'];
                        vcsName = routeParams['vcsName'];
                        projKey = this._store.selectSnapshot(ProjectState.projectSnapshot).key;
                        break;
                    case 3:
                        entityName = entitySplit[2];
                        repoName = entitySplit[0] + '/' + entitySplit[1];
                        vcsName = routeParams['vcsName'];
                        projKey = this._store.selectSnapshot(ProjectState.projectSnapshot).key;
                        break;
                    case 4:
                        entityName = entitySplit[3];
                        repoName = entitySplit[1] + '/' + entitySplit[2];
                        vcsName = entitySplit[0];
                        projKey = this._store.selectSnapshot(ProjectState.projectSnapshot).key;
                        break;
                    case 5:
                        entityName = entitySplit[4];
                        repoName = entitySplit[2] + '/' + entitySplit[3];
                        vcsName = entitySplit[1];
                        projKey = entitySplit[0];
                        break;
                    default:
                        console.error("Unable to load ", entitySplit);
                        return;
                }

                if (this.entityType === EntityType.Action && entitySplit.length === 1) {
                    this._pluginService.getPlugin(entityName).subscribe(pl => {
                        if (pl.inputs) {
                            let keys = Object.keys(pl.inputs);
                            if (keys.length > 0) {
                                this.currentModel['with'] = {};
                                keys.forEach(k => {
                                    this.currentModel['with'][k] = pl.inputs[k].default;
                                });
                            }
                            this.modelChange.emit(this.currentModel);
                        }
                    });
                } else {
                    this._projectService.getRepoEntity(projKey, vcsName, repoName, this.entityType, entityName, branch).subscribe(e => {
                        let ent = load(e.data && e.data !== '' ? e.data : '{}', <LoadOptions>{
                            onWarning: (e) => {
                            }
                        });
                        switch (this.entityType) {
                            case EntityType.Action:
                                if (ent.inputs) {
                                    let keys = Object.keys(ent.inputs);
                                    if (keys.length > 0) {
                                        this.currentModel['with'] = {};
                                        keys.forEach(k => {
                                            this.currentModel['with'][k] = ent.inputs[k].default;
                                        });
                                    }
                                }
                                break;
                        }
                        this.modelChange.emit(this.currentModel);
                    });
                }

                break;
        }
        if (this.field.type === 'array' || this.field.type === 'map') {
            this.currentModel[this.field.name][index] = value;
        } else {
            this.currentModel[this.field.name] = value;
        }
        this._cd.markForCheck();
        this.modelChange.emit(this.currentModel);
    }

    onDragArrayItem(field: string, array: any): void {
        this.timestamp = Date.now();
        this.currentModel[field] = array;
        this._cd.markForCheck();
        this.modelChange.emit(this.currentModel);
    }

    onArrayItemDelete(field: string, index: number): void {
        this.currentModel[field] = this.currentModel[field].filter((v: any, i: number) => i !== index);
        this._cd.markForCheck();
        this.modelChange.emit(this.currentModel);
    }

    onMapItemDelete(field: string, key: string): void {
        delete this.currentModel[field][key];
        this._cd.markForCheck();
        this.modelChange.emit(this.currentModel);
    }
}
