import {Component, EventEmitter, Input, OnInit, Output, ViewChild} from '@angular/core';
import {SemanticModalComponent} from 'ng-semantic/ng-semantic';
import {Workflow, WorkflowNode, WorkflowNodeTrigger, WorkflowTriggerCondition} from '../../../model/workflow.model';
import {Project} from '../../../model/project.model';
import {WorkflowStore} from '../../../service/workflow/workflow.store';

@Component({
    selector: 'app-workflow-trigger',
    templateUrl: './workflow.trigger.html',
    styleUrls: ['./workflow.trigger.scss']
})
export class WorkflowTriggerComponent {

    @ViewChild('triggerModal')
    modal: SemanticModalComponent;

    @Output() triggerChange = new EventEmitter<WorkflowNodeTrigger>();
    @Input() triggerSrcNode: WorkflowNode;
    @Input() workflow: Workflow;
    @Input() project: Project;
    @Input() trigger: WorkflowNodeTrigger;

    operators: Array<string>;
    conditionNames: Array<string>;



    constructor(private _workflowStore: WorkflowStore) {
    }

    show(data?: {}): void {
        this.modal.show(data);
        this._workflowStore.getTriggerCondition(this.project.key, this.workflow.name, this.triggerSrcNode.id).first().subscribe( wtc => {
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
