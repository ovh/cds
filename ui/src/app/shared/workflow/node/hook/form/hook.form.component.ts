
import {Component, EventEmitter, Input, OnInit, Output, ViewChild} from '@angular/core';
import {cloneDeep} from 'lodash';
import {ModalTemplate, SuiModalService, TemplateModalConfig} from 'ng2-semantic-ui';
import {ActiveModal} from 'ng2-semantic-ui/dist';
import {zip as observableZip} from 'rxjs';
import {finalize, first} from 'rxjs/operators';
import {ProjectPlatform} from '../../../../../model/platform.model';
import {IdName, Project} from '../../../../../model/project.model';
import {WorkflowHookModel} from '../../../../../model/workflow.hook.model';
import {
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

    _hook: WorkflowNodeHook = new WorkflowNodeHook();
    _outgoingHook: WorkflowNodeOutgoingHook = new WorkflowNodeOutgoingHook();
    canDelete = false;

    @Input() project: Project;
    @Input() workflow: Workflow;
    @Input() node: WorkflowNode;
    @Input() isOutgoing: boolean;
    @Input() loading: boolean;
    @Input() readonly: boolean;
    @Input('hook')
    set hook(data: WorkflowNodeHook) {
        if (data) {
            this.canDelete = true;
            this._hook = cloneDeep<WorkflowNodeHook>(data);
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
            this._hook = new WorkflowNodeHook();
            this._hook.id = this.outgoingHook.id;
            this._hook.config = cloneDeep(this.outgoingHook.config);
            this._hook.model = cloneDeep(this.outgoingHook.model);
            this.displayConfig = Object.keys(this._hook.config).length !== 0;
            console.log(this._hook);
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

    // Ng semantic modal
    @ViewChild('nodeHookFormModal')
    public nodeHookFormModal: ModalTemplate<boolean, boolean, void>;
    modal: ActiveModal<boolean, boolean, void>;
    modalConfig: TemplateModalConfig<boolean, boolean, void>;

    constructor(private _hookService: HookService, private _modalService: SuiModalService, private _workflowStore: WorkflowStore) { }

    updateHook(): void {
        this.hook.model = this.selectedHookModel;
        this.hook.config = cloneDeep(this.selectedHookModel.default_config);
        this.displayConfig = Object.keys(this.hook.config).length !== 0;
    }

    updateOutgoingHook(): void {
        this.hook.model = this.selectedOutgoingHookModel;
        this.hook.config = cloneDeep(this.selectedOutgoingHookModel.default_config);
        this.displayConfig = Object.keys(this.hook.config).length !== 0;

        // Specific behavior for the 'workflow' hooks
        if (this.hook.model.name === 'Workflow') {
            // Current limitation: trigger only workflow in the same project
            this.hook.config['target_project'].value = this.project.key;
            // Load the workflow for the current project, but exclude the current workflow
            this.availableWorkflows = this.project.workflow_names.filter(idName => idName.name !== this.workflow.name);
        }
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
                finalize(() => this.loadingHooks = true)
            ).subscribe(
                data => {
                    let key = this.project.key + '-' + this.hook.config['target_workflow'].value;
                    let wf = data.get(key);
                    if (wf) {
                        this.availableHooks = Workflow.getAllHooks(wf).filter(h => h.model.name === 'Workflow');
                    }
                }
            );
    }

    show(): void {
        this.loadingModels = true;
        observableZip(
            this._hookService.getHookModel(this.project, this.workflow, this.node),
            this._hookService.getOutgoingHookModel(this.project, this.workflow, this.node),
            this._workflowStore.getTriggerCondition(this.project.key, this.workflow.name, this.node.id),
            (hookModels, outgoingHookModels, triggerConditions) => {
                this.hooksModel = hookModels;
                this.outgoingHookModels = outgoingHookModels;

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

        this.modalConfig = new TemplateModalConfig<boolean, boolean, void>(this.nodeHookFormModal);
        this.modalConfig.mustScroll = true;
        this.modal = this._modalService.open(this.modalConfig);
    }

    addHook(): void {
        this.hookEvent.emit(new HookEvent('add', this.hook));
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
    }
}
