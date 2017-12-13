import {Component, Input, OnInit} from '@angular/core';
import {
    notificationOnFailure,
    notificationOnSuccess,
    notificationTypes,
    UserNotificationSettings,
    UserNotificationTemplate
} from '../../../../../model/notification.model';
import {TranslateService} from '@ngx-translate/core'
import {Workflow, WorkflowNode, WorkflowNotification} from '../../../../../model/workflow.model';
import {WorkflowStore} from '../../../../../service/workflow/workflow.store';
import {Project} from '../../../../../model/project.model';
import {ToastService} from '../../../../../shared/toast/ToastService';
import {finalize} from 'rxjs/operators'
import {cloneDeep} from 'lodash';

@Component({
    selector: 'app-workflow-notifications-form',
    templateUrl: './workflow.notifications.form.html',
    styleUrls: ['./workflow.notifications.form.scss']
})
export class WorkflowNotificationFormComponent implements OnInit {

    types: Array<string>;
    notifOnSuccess: Array<string>;
    notifOnFailure: Array<string>;
    selectedType: string;
    selectedUsers: string;
    loading = false;
    userNotification: UserNotificationSettings;
    notificationTemplate: UserNotificationTemplate;
    selectedNodes: Array<WorkflowNode>;
    nodeError = false;
    ready = false;

    nodes: Array<WorkflowNode>;
    _workflow: Workflow;
    @Input('workflow')
    set workflow(data: Workflow) {
        if (data) {
            this._workflow = data;
            this.nodes = Workflow.getAllNodes(data);
        }
    }

    get workflow() {
        return this._workflow;
    }

    @Input() project: Project;

    constructor(private _workflowStore: WorkflowStore, private _translate: TranslateService, private _toast: ToastService) {
        this.userNotification = new UserNotificationSettings();
        this.notificationTemplate = new UserNotificationTemplate();
        this.notifOnSuccess = notificationOnSuccess;
        this.notifOnFailure = notificationOnFailure;
        this.selectedType = notificationTypes[0];
        this.types = notificationTypes;
    }

    ngOnInit(): void {
        this.selectedNodes = cloneDeep(this.nodes);
        this.ready = true;
    }

    createNotification(): void {
        if (!this.selectedNodes || this.selectedNodes.length === 0) {
            this.nodeError = true;
            return;
        }
        this.nodeError = false;

        this.loading = true;
        let notification = new WorkflowNotification();

        if (this.selectedUsers) {
            this.userNotification.recipients = this.selectedUsers.split(',');
        }
        this.userNotification.template = this.notificationTemplate;
        notification.type = this.selectedType;
        notification.settings = this.userNotification;

        this.selectedNodes.forEach(sn => {
            notification.source_node_ref.push(sn.id.toString());
        });

        if (!this.workflow.notifications) {
            this.workflow.notifications = new Array<WorkflowNotification>();
        }
        this.workflow.notifications.push(notification);
        this._workflowStore.updateWorkflow(this.project.key, this.workflow).pipe(finalize(() => {
            this.loading = false;
        })).subscribe(() => {
            this._toast.success('', this._translate.instant('workflow_updated'));
        });
    }
}
