import { Component, Input } from '@angular/core';
import {Workflow} from '../../../../model/workflow.model';
import {cloneDeep} from 'lodash';

@Component({
    selector: 'app-workflow-notifications',
    templateUrl: './workflow.notifications.html',
    styleUrls: ['./workflow.notifications.scss']
})
export class WorkflowNotificationComponent {

    _workflow: Workflow;
    @Input('workflow')
    set (data: Workflow) {
        this._workflow = cloneDeep(data);
    }
    get() {
        return this._workflow;
    }

    constructor() { }
}
