import {Component, Input} from '@angular/core';
import {Table} from '../../../table/table';
import {WorkflowTriggerCondition} from '../../../../model/workflow.model';
import {Project} from '../../../../model/project.model';

@Component({
    selector: 'app-workflow-trigger-condition-list',
    templateUrl: './trigger.condition.list.html',
    styleUrls: ['./trigger.condition.list.scss']
})
export class WorkflowTriggerConditionListComponent extends Table {

    @Input() conditions: Array<WorkflowTriggerCondition>;
    @Input() project: Project;
    @Input() operators: Array<string>;

    constructor() {
        super();
    }

    getData(): any[] {
        return this.conditions;
    }

    removeCondition(cond: WorkflowTriggerCondition): void {
        this.conditions = this.conditions.filter(c => c.variable !== cond.variable);
    }
}
