import {Component, EventEmitter, Input, Output} from '@angular/core';
import {Table} from '../../../../table/table';
import {Workflow, WorkflowNodeCondition, WorkflowNodeConditions} from '../../../../../model/workflow.model';
import {PermissionValue} from '../../../../../model/permission.model';
import {PipelineStatus} from '../../../../../model/pipeline.model';

@Component({
    selector: 'app-workflow-node-condition-list',
    templateUrl: './condition.list.html',
    styleUrls: ['./condition.list.scss']
})
export class WorkflowNodeConditionListComponent extends Table {

    @Input() conditions: WorkflowNodeConditions;
    @Output() conditionChange = new EventEmitter<WorkflowNodeCondition[]>();
    @Input() workflow: Workflow;
    @Input() operators: {};

    permission = PermissionValue;
    statuses = [PipelineStatus.SUCCESS, PipelineStatus.FAIL, PipelineStatus.SKIPPED];

    constructor() {
        super();
    }

    getData(): any[] {
        if (!this.conditions) {
            this.conditions = new WorkflowNodeConditions();
        }
        if (!this.conditions.plain) {
            this.conditions.plain = new Array<WorkflowNodeCondition>();
        }
        return this.conditions.plain;
    }

    removeCondition(cond: WorkflowNodeCondition): void {
        let newConditions = new WorkflowNodeConditions();
        newConditions.plain = this.conditions.plain.filter(c => c.variable !== cond.variable)
        this.conditionChange.emit(newConditions.plain);
    }

    isStatusVariable(cond: WorkflowNodeCondition): boolean {
        return cond && cond.variable && cond.variable.indexOf('.status') !== -1;
    }
}
