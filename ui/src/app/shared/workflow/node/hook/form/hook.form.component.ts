
import {zip as observableZip} from 'rxjs';
import {Component, EventEmitter, Input, Output, ViewChild, OnInit} from '@angular/core';
import {
    Workflow, WorkflowNode, WorkflowNodeHook
} from '../../../../../model/workflow.model';
import {HookService} from '../../../../../service/hook/hook.service';
import {WorkflowHookModel} from '../../../../../model/workflow.hook.model';
import {ModalTemplate, SuiModalService, TemplateModalConfig} from 'ng2-semantic-ui';
import {ActiveModal} from 'ng2-semantic-ui/dist';
import {cloneDeep} from 'lodash';
import {Project} from '../../../../../model/project.model';
import {WorkflowStore} from '../../../../../service/workflow/workflow.store';
import {HookEvent} from '../hook.event';
import {first, finalize} from 'rxjs/operators';
import {ProjectPlatform} from '../../../../../model/platform.model';

@Component({
    selector: 'app-workflow-node-hook-form',
    templateUrl: './hook.form.html',
    styleUrls: ['./hook.form.scss']
})
export class WorkflowNodeHookFormComponent implements OnInit {

    _hook: WorkflowNodeHook = new WorkflowNodeHook();
    canDelete = false;

    @Input() project: Project;
    @Input() workflow: Workflow;
    @Input() node: WorkflowNode;
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

    @Output() hookEvent = new EventEmitter<HookEvent>();

    hooksModel: Array<WorkflowHookModel>;
    selectedHookModel: WorkflowHookModel;
    operators: {};
    conditionNames: Array<string>;
    loadingModels = true;
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

    constructor(private _hookService: HookService, private _modalService: SuiModalService, private _workflowStore: WorkflowStore) {
        this.codeMirrorConfig = {
            matchBrackets: true,
            autoCloseBrackets: true,
            mode: 'application/json',
            lineWrapping: true,
            autoRefresh: true
        };
    }

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

    show(): void {
        this.loadingModels = true;
        observableZip(
            this._hookService.getHookModel(this.project, this.workflow, this.node),
            this._workflowStore.getTriggerCondition(this.project.key, this.workflow.name, this.node.id),
            (hms, wtc) => {
                this.hooksModel = hms;
                if (this._hook && this._hook.model) {
                    this.selectedHookModel = this.hooksModel.find(hm => hm.id === this._hook.model.id);
                }
                if (this.selectedHookModel != null && this.hook.id) {
                  this.updateMode = true;
                }
                this.operators = wtc.operators;
                this.conditionNames = wtc.names;
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
    }
}
