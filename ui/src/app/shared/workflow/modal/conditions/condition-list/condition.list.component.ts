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

    @Input('conditions')
    set conditions(data: WorkflowNodeConditions) {
      if (!data) {
          data = new WorkflowNodeConditions();
      }
      if (!data.plain) {
          data.plain = new Array<WorkflowNodeCondition>();
      }
      this._conditions = data;
      this.getDataForCurrentPage();
    }
    get conditions(): WorkflowNodeConditions {
      return this._conditions;
    }
    @Output() conditionChange = new EventEmitter<WorkflowNodeCondition[]>();
    @Input() workflow: Workflow;
    @Input() operators: {};
    @Input() mode: 'advanced'|'basic';

    codeMirrorConfig: {};
    permission = PermissionValue;
    statuses = [PipelineStatus.SUCCESS, PipelineStatus.FAIL, PipelineStatus.SKIPPED];
    data: any[] = [];
    _conditions: WorkflowNodeConditions;

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
        this.getDataForCurrentPage();
    }

    getDataForCurrentPage(): any[] {
        this.data = super.getDataForCurrentPage();
        return this.data;
    }

    getData(): any[] {
        if (!this.conditions || !this.conditions.plain) {
            return [];
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
