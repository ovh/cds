import {Component, EventEmitter, Input, OnInit, Output, ViewChild} from '@angular/core';
import {Workflow, WorkflowNode, WorkflowNodeCondition, WorkflowNodeConditions} from '../../../../model/workflow.model';
import {Project} from '../../../../model/project.model';
import {WorkflowStore} from '../../../../service/workflow/workflow.store';
import {PermissionValue} from '../../../../model/permission.model';
import {ModalTemplate, SuiModalService, TemplateModalConfig} from 'ng2-semantic-ui';
import {ActiveModal} from 'ng2-semantic-ui/dist';

@Component({
    selector: 'app-workflow-node-conditions',
    templateUrl: './node.conditions.html',
    styleUrls: ['./node.conditions.scss']
})
export class WorkflowNodeConditionsComponent implements OnInit {

    @Output() conditionsEvent = new EventEmitter<WorkflowNode>();

    _node: WorkflowNode;
    @Input('node')
    set node(data: WorkflowNode) {
        if (data) {
            if (!data.context.conditions) {
                data.context.conditions = new WorkflowNodeConditions();
            }
            this._node = data;
        }
    }
    get node() {
        return this._node;
    }

    @Input() workflow: Workflow;
    @Input() project: Project;
    permission = PermissionValue;

    operators: {};
    conditionNames: Array<string>;

    @ViewChild('nodeConditionsModal')
    public nodeConditionModal: ModalTemplate<boolean, boolean, void>;
    modal: ActiveModal<boolean, boolean, void>;
    modalConfig: TemplateModalConfig<boolean, boolean, void>;

    constructor(private _workflowStore: WorkflowStore, private _suiService: SuiModalService) { }

    ngOnInit(): void {
        this._workflowStore.getTriggerCondition(this.project.key, this.workflow.name, this.node.id).first().subscribe(wtc => {
            this.operators = wtc.operators;
            this.conditionNames = wtc.names;
        });
    }

    addCondition(condition: WorkflowNodeCondition): void {
        if (!this.node.context.conditions) {
            this.node.context.conditions = new WorkflowNodeConditions();
        }
        if (!this.node.context.conditions.plain) {
            this.node.context.conditions.plain = new Array<WorkflowNodeCondition>();
        }
        let index = this.node.context.conditions.plain.findIndex(c => c.variable === condition.variable);
        if (index === -1) {
            this.node.context.conditions.plain.push(condition);
        }
    }

    show(): void {
        if (this.nodeConditionModal) {
            this.modalConfig = new TemplateModalConfig<boolean, boolean, void>(this.nodeConditionModal);
            this.modal = this._suiService.open(this.modalConfig);
        }
    }

    saveConditions(): void {
        this.conditionsEvent.emit(this.node);
    }
}
