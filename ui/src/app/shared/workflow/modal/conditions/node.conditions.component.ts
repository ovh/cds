import { Component, EventEmitter, Input, Output, ViewChild } from '@angular/core';
import { WorkflowService } from 'app/service/workflow/workflow.service';
import { cloneDeep } from 'lodash';
import { ModalTemplate, SuiModalService, TemplateModalConfig } from 'ng2-semantic-ui';
import { ActiveModal } from 'ng2-semantic-ui/dist';
import { finalize, first } from 'rxjs/operators';
import { PermissionValue } from '../../../../model/permission.model';
import { Project } from '../../../../model/project.model';
import { WNode, Workflow, WorkflowNodeConditions } from '../../../../model/workflow.model';
import { VariableService } from '../../../../service/variable/variable.service';

@Component({
    selector: 'app-workflow-node-conditions-old',
    templateUrl: './node.conditions.html',
    styleUrls: ['./node.conditions.scss']
})
export class WorkflowNodeConditionsComponent {

    @Output() conditionsEvent = new EventEmitter<Workflow>();

    _node: WNode;
    @Input('node')
    set node(data: WNode) {
        if (data) {
            if (!data.context.conditions) {
                data.context.conditions = new WorkflowNodeConditions();
            }
            this._node = cloneDeep(data);
            if (data.context.conditions.lua_script) {
                this.mode = 'advanced';
            } else {
                this.mode = 'basic';
            }
        }
    }
    get node() {
        return this._node;
    }

    @Input() workflow: Workflow;
    @Input() project: Project;
    @Input() loading: boolean;
    permission = PermissionValue;

    operators: {};
    conditionNames: Array<string>;
    suggest: Array<string> = [];
    loadingConditions = false;
    mode: 'advanced' | 'basic' = 'basic';

    @ViewChild('nodeConditionsModal')
    public nodeConditionModal: ModalTemplate<boolean, boolean, void>;
    modal: ActiveModal<boolean, boolean, void>;
    modalConfig: TemplateModalConfig<boolean, boolean, void>;

    constructor(
        private _workflowService: WorkflowService,
        private _suiService: SuiModalService,
        private _variableService: VariableService
    ) { }

    conditionsChange(conditions: WorkflowNodeConditions): void {
        this.node.context.conditions = conditions;
    }

    show(): void {
        this.loadingConditions = true;
        this.suggest = [];
        this._variableService.getContextVariable(this.project.key, this.node.context.pipeline_id)
            .subscribe((suggest) => this.suggest = suggest);

        this._workflowService.getTriggerCondition(this.project.key, this.workflow.name, this.node.id)
            .pipe(
                first(),
                finalize(() => this.loadingConditions = false)
            )
            .subscribe(wtc => {
                this.operators = wtc.operators;
                this.conditionNames = wtc.names;
            });
        if (this.nodeConditionModal) {
            this.modalConfig = new TemplateModalConfig<boolean, boolean, void>(this.nodeConditionModal);
            this.modalConfig.mustScroll = true;
            this.modal = this._suiService.open(this.modalConfig);
        }
    }

    saveConditions(): void {
        let clonedWorkflow = cloneDeep(this.workflow);
        let n = Workflow.getNodeByID(this.node.id, clonedWorkflow);
        n.context.conditions = this.node.context.conditions;
        this.conditionsEvent.emit(clonedWorkflow);
    }
}
