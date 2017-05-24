import {Component, EventEmitter, Input, OnInit, Output, ViewChild} from '@angular/core';
import {SemanticModalComponent} from 'ng-semantic/ng-semantic';
import {
    Workflow, WorkflowNode, WorkflowNodeJoin, WorkflowNodeJoinTrigger,
    WorkflowTriggerCondition
} from '../../../../model/workflow.model';
import {Project} from '../../../../model/project.model';
import {WorkflowStore} from '../../../../service/workflow/workflow.store';

@Component({
    selector: 'app-workflow-trigger-join',
    templateUrl: './workflow.trigger.join.html',
    styleUrls: ['./workflow.trigger.join.scss']
})
export class WorkflowTriggerJoinComponent {

    @ViewChild('triggerJoinModal')
    modal: SemanticModalComponent;

    @Output() triggerChange = new EventEmitter<WorkflowNodeJoinTrigger>();
    @Input() join: WorkflowNodeJoin;
    @Input() workflow: Workflow;
    @Input() project: Project;
    @Input() trigger: WorkflowNodeJoinTrigger;

    operators: Array<string>;
    conditionNames: Array<string>;



    constructor(private _workflowStore: WorkflowStore) {
    }

    show(data?: {}): void {
        this.modal.show(data);
        this._workflowStore.getTriggerJoinCondition(this.project.key, this.workflow.name, this.join.id).first().subscribe( wtc => {
            this.operators = wtc.operators;
            this.conditionNames = wtc.names;
        });


    }

    hide(): void {
        this.modal.hide();
    }

    destNodeChange(node: WorkflowNode): void {
        this.trigger.workflow_dest_node = node;
    }

    saveTrigger(): void {
        this.triggerChange.emit(this.trigger);
    }

    addCondition(condition: WorkflowTriggerCondition): void {
        if (!this.trigger.conditions) {
            this.trigger.conditions = new Array<WorkflowTriggerCondition>();
        }
        let index = this.trigger.conditions.findIndex(c => c.variable === condition.variable);
        if (index === -1) {
            this.trigger.conditions.push(condition);
        }
    }
}
