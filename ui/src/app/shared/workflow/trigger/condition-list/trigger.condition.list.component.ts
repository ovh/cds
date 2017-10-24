import {Component, EventEmitter, Input, Output} from '@angular/core';
import {Table} from '../../../table/table';
import {WorkflowTriggerCondition, WorkflowTriggerConditions} from '../../../../model/workflow.model';
import {Project} from '../../../../model/project.model';

@Component({
    selector: 'app-workflow-trigger-condition-list',
    templateUrl: './trigger.condition.list.html',
    styleUrls: ['./trigger.condition.list.scss']
})
export class WorkflowTriggerConditionListComponent extends Table {

    @Input() conditions: WorkflowTriggerConditions;
    @Output() conditionsChange = new EventEmitter<WorkflowTriggerConditions>();
    @Input() project: Project;
    @Input() operators: {};

    constructor() {
        super();
    }

    getData(): any[] {
        if (!this.conditions) {
            this.conditions = new WorkflowTriggerConditions();
        }
        if (!this.conditions.plain) {
            this.conditions.plain = new Array<WorkflowTriggerCondition>();
        }
        return this.conditions.plain;
    }

    removeCondition(cond: WorkflowTriggerCondition): void {
        var newConditions = new WorkflowTriggerConditions();
        newConditions.plain = this.conditions.plain.filter(c => c.variable !== cond.variable)
        this.conditionsChange.emit(newConditions);
    }
}
