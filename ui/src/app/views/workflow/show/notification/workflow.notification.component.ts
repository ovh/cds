import { Component, Input } from '@angular/core';
import {Workflow, WorkflowNotification} from '../../../../model/workflow.model';
import {cloneDeep} from 'lodash';
import {Project} from '../../../../model/project.model';

@Component({
    selector: 'app-workflow-notifications',
    templateUrl: './workflow.notifications.html',
    styleUrls: ['./workflow.notifications.scss']
})
export class WorkflowNotificationComponent {

    _workflow: Workflow;
    @Input('workflow')
    set workflow (data: Workflow) {
        this._workflow = cloneDeep(data);
    }
    get workflow() {
        return this._workflow;
    }
    @Input() project: Project;

    constructor() { }
}
