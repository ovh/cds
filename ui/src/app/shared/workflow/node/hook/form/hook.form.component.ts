
import {Component, EventEmitter, Input, OnInit, Output} from '@angular/core';
import {cloneDeep} from 'lodash';
import {zip as observableZip} from 'rxjs';
import {finalize, first} from 'rxjs/operators';
import {ProjectPlatform} from '../../../../../model/platform.model';
import {IdName, Project} from '../../../../../model/project.model';
import {WorkflowHookModel} from '../../../../../model/workflow.hook.model';
import {
    WNodeHook,
    Workflow, WorkflowNode, WorkflowNodeHook, WorkflowNodeOutgoingHook
} from '../../../../../model/workflow.model';
import {HookService} from '../../../../../service/hook/hook.service';
import {WorkflowStore} from '../../../../../service/workflow/workflow.store';
import {HookEvent} from '../hook.event';

@Component({
    selector: 'app-workflow-node-hook-form',
    templateUrl: './hook.form.html',
    styleUrls: ['./hook.form.scss']
})
export class WorkflowNodeHookFormComponent implements OnInit {

    _hook: WNodeHook = new WNodeHook();
    _outgoingHook: WorkflowNodeOutgoingHook = new WorkflowNodeOutgoingHook();
    canDelete = false;

    @Input() project: Project;
    @Input() workflow: Workflow;
    @Input() node: WorkflowNode;
    @Input() isOutgoing: boolean;
    @Input() loading: boolean;
    @Input() readonly: boolean;
    @Input('hook')
    set hook(data: WNodeHook) {
        if (data) {
            this.canDelete = true;
            this._hook = cloneDeep<WNodeHook>(data);
            if (this.hooksModel) {
                this.selectedHookModel = this.hooksModel.find(hm => hm.id === this._hook.model.id);
            }
            this.displayConfig = Object.keys(this._hook.config).length !== 0;
        }
    }
    get hook() {
        return this._hook;
    }
    @Input('outgoingHook')
    set outgoingHook(data: WorkflowNodeOutgoingHook) {
        if (data) {
            this.canDelete = true;
            this._outgoingHook = cloneDeep<WorkflowNodeOutgoingHook>(data);
            if (this.outgoingHookModels) {
                this.selectedOutgoingHookModel = this.outgoingHookModels.find(hm => hm.id === this._outgoingHook.model.id);
            }
            this.displayConfig = Object.keys(this._outgoingHook.config).length !== 0;
            this._hook = new WNodeHook();
            this._hook.id = this.outgoingHook.id;
            this._hook.config = cloneDeep(this.outgoingHook.config);
            this._hook.model = cloneDeep(this.outgoingHook.model);
            this.displayConfig = Object.keys(this._hook.config).length !== 0;
        }
    }
    get outgoingHook() {
        return this._outgoingHook;
    }

    @Output() hookEvent = new EventEmitter<HookEvent>();

    hooksModel: Array<WorkflowHookModel>;
    outgoingHookModels: Array<WorkflowHookModel>;

    selectedHookModel: WorkflowHookModel;
    selectedOutgoingHookModel: WorkflowHookModel;

    availableWorkflows: Array<IdName>;
    availableHooks: Array<WorkflowNodeHook>;

    operators: {};
    conditionNames: Array<string>;
    loadingModels = true;
    loadingHooks = true;
    displayConfig = false;
    invalidJSON = false;
    updateMode = false;
    codeMirrorConfig: any;
    selectedPlatform: ProjectPlatform;
    availablePlatforms: Array<ProjectPlatform>;

    constructor(private _hookService: HookService, private _workflowStore: WorkflowStore) { }

    updateHook(): void {
        this.hook.model = this.selectedHookModel;
        this.hook.config = cloneDeep(this.selectedHookModel.default_config);
        this.displayConfig = Object.keys(this.hook.config).length !== 0;
    }

    updatePlatform(): void {
        Object.keys(this.hook.config).forEach( k => {
            if (k === 'platform') {
                this.hook.config[k].value = this.selectedPlatform.name;
            } else {
                if (this.selectedPlatform.config[k]) {
                    this.hook.config[k] = cloneDeep(this.selectedPlatform.config[k])
                }
            }
        });
    }

    updateWorkflow(): void {
        this.loadingHooks = true;
        this._workflowStore.getWorkflows(this.hook.config['target_project'].value, this.hook.config['target_workflow'].value)
            .pipe(
                finalize(() => this.loadingHooks = false)
            ).subscribe(
                data => {
                    let key = this.project.key + '-' + this.hook.config['target_workflow'].value;
                    let wf = data.get(key);
                    if (wf) {
                        this.availableHooks = Workflow.getAllHooks(wf).filter(h => wf.hook_models[h.hook_model_id].name === 'Workflow');
                    }
                }
            );
    }

    show(): void {
        this.loadingModels = true;
        observableZip(
            this._hookService.getHookModel(this.project, this.workflow, this.node),
            this._workflowStore.getTriggerCondition(this.project.key, this.workflow.name, this.node.id),
            (hookModels, triggerConditions) => {
                this.hooksModel = hookModels;

                if (this._hook && this._hook.model) {
                    this.selectedHookModel = this.hooksModel.find(hm => hm.id === this._hook.model.id);
                }
                if (this.selectedHookModel != null && this.hook.id) {
                  this.updateMode = true;
                }

                if (this._outgoingHook && this._outgoingHook.model) {
                    this.selectedOutgoingHookModel = this.outgoingHookModels.find(hm => hm.id === this._outgoingHook.model.id);
                }
                if (this.selectedOutgoingHookModel != null && this.outgoingHook.id) {
                    this.updateMode = true;
                }
                this.operators = triggerConditions.operators;
                this.conditionNames = triggerConditions.names;
            }
        ).pipe(
            first(),
            finalize(() => this.loadingModels = false)
        )
        .subscribe();
    }

    addHook(): void {
        let h = new HookEvent('add', this.hook);
        if (this.isOutgoing) {
            h.name = this.outgoingHook.name;
        }
        this.hookEvent.emit(h);
    }

    deleteHook(): void {
        this.hookEvent.emit(new HookEvent('delete', this.hook));
    }

    changeCodeMirror(code: string) {
        if (typeof code === 'string') {
            this.invalidJSON = false;
            try {
                JSON.parse(code);
            } catch (e) {
                this.invalidJSON = true;
            }
        }
    }

    ngOnInit(): void {
        this.availablePlatforms = this.project.platforms.filter(pf =>  pf.model.hook);
        if (this.hook && this.hook.config && this.hook.config['platform']) {
            this.selectedPlatform = this.project.platforms.find(pf => pf.name === this.hook.config['platform'].value);
        }
        this.codeMirrorConfig = {
            matchBrackets: true,
            autoCloseBrackets: true,
            mode: 'application/json',
            lineWrapping: true,
            autoRefresh: true,
            readOnly: this.readonly,
        };
        this.show();
    }
}
