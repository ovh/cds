import {Component, EventEmitter, Input, OnInit, Output} from '@angular/core';
import {PermissionValue} from '../../../../../model/permission.model';
import {PipelineStatus} from '../../../../../model/pipeline.model';
import {Workflow, WorkflowNodeCondition, WorkflowNodeConditions} from '../../../../../model/workflow.model';
import {Table} from '../../../../table/table';

@Component({
    selector: 'app-workflow-node-condition-list',
    templateUrl: './condition.list.html',
    styleUrls: ['./condition.list.scss']
})
export class WorkflowNodeConditionListComponent extends Table implements OnInit {

    @Input() conditions: WorkflowNodeConditions;
    @Output() conditionChange = new EventEmitter<WorkflowNodeCondition[]>();
    @Input() workflow: Workflow;
    @Input() operators: {};

    codeMirrorConfig: {};
    permission = PermissionValue;
    statuses = [PipelineStatus.SUCCESS, PipelineStatus.FAIL, PipelineStatus.SKIPPED];
    mode: 'advanced'|'basic' = 'basic';
    data: any[] = [];

    constructor() {
        super();
        this.codeMirrorConfig = {
            matchBrackets: true,
            autoCloseBrackets: true,
            mode: 'lua',
            lineWrapping: true,
            lineNumbers: true,
            autoRefresh: true,
            readOnly: true,
        };
    }

    ngOnInit() {
        if (this.conditions.lua_script) {
            this.mode = 'advanced';
        }

        this.data = this.getData();
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
