import {Component, EventEmitter, Input, Output} from '@angular/core';
import {WorkflowTriggerCondition} from '../../../../model/workflow.model';

declare var _: any;

@Component({
    selector: 'app-workflow-trigger-condition-form',
    templateUrl: './trigger.condition.form.html',
    styleUrls: ['./trigger.condition.form.scss']
})
export class WorkflowTriggerConditionFormComponent {

    @Input() operators: Array<string>;
    @Input() names: Array<string>;

    @Output() addEvent = new EventEmitter<WorkflowTriggerCondition>();

    condition = new WorkflowTriggerCondition();

    constructor() { }


    send(): void {
        this.addEvent.emit(_.cloneDeep(this.condition));
    }
}
