import {Component, EventEmitter, Input, Output, ViewChild} from '@angular/core';
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

@Component({
    selector: 'app-workflow-node-hook-form',
    templateUrl: './hook.form.html',
    styleUrls: ['./hook.form.scss']
})
export class WorkflowNodeHookFormComponent {

    _hook: WorkflowNodeHook = new WorkflowNodeHook();
    canDelete = false;

    @Input() project: Project;
    @Input() workflow: Workflow;
    @Input() node: WorkflowNode;
    @Input() loading: boolean;
    @Input('hook')
    set hook(data: WorkflowNodeHook) {
        if (data) {
            this.canDelete = true;
            this._hook = cloneDeep(data);
            if (this.hooksModel) {
                this.selectedHookModel = this.hooksModel.find(hm => hm.id === this._hook.model.id);
            }
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

    // Ng semantic modal
    @ViewChild('nodeHookFormModal')
    public nodeHookFormModal: ModalTemplate<boolean, boolean, void>;
    modal: ActiveModal<boolean, boolean, void>;
    modalConfig: TemplateModalConfig<boolean, boolean, void>;

    constructor(private _hookService: HookService, private _modalService: SuiModalService, private _workflowStore: WorkflowStore) {
    }

    updateHook(): void {
        this.hook.model = this.selectedHookModel;
        this.hook.config = cloneDeep(this.selectedHookModel.default_config);
    }

    show(): void {
        this._hookService.getHookModel().first().subscribe(hms => {
            this.hooksModel = hms;
            if (this._hook && this._hook.model) {
                this.selectedHookModel = this.hooksModel.find(hm => hm.id === this._hook.model.id);
            }
        });
        this._workflowStore.getTriggerCondition(this.project.key, this.workflow.name, this.node.id).first().subscribe( wtc => {
            this.operators = wtc.operators;
            this.conditionNames = wtc.names;
        });
        this.modalConfig = new TemplateModalConfig<boolean, boolean, void>(this.nodeHookFormModal);
        this.modal = this._modalService.open(this.modalConfig);
    }

    addHook(): void {
        this.hookEvent.emit(new HookEvent('add', this.hook));
    }

    deleteHook(): void {
        this.hookEvent.emit(new HookEvent('delete', this.hook));
    }
}
