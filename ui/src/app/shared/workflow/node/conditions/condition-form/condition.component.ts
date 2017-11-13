import {Component, EventEmitter, Input, Output} from '@angular/core';
import {WorkflowNodeCondition} from '../../../../../model/workflow.model';

declare var _: any;

@Component({
    selector: 'app-workflow-node-condition-form',
    templateUrl: './condition.form.html',
    styleUrls: ['./condition.form.scss']
})
export class WorkflowNodeConditionFormComponent {

    @Input() operators: {};
    @Input() names: Array<string>;

    @Output() addEvent = new EventEmitter<WorkflowNodeCondition>();

    condition = new WorkflowNodeCondition();

    constructor() { }


    send(): void {
        this.addEvent.emit(_.cloneDeep(this.condition));
    }
}
