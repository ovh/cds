import { Component, EventEmitter, Input, Output } from '@angular/core';
import { NotificationService } from 'app/service/notification/notification.service';
import cloneDeep from 'lodash-es/cloneDeep';
import { finalize, first } from 'rxjs/operators';
import { notificationOnFailure, notificationOnSuccess, notificationTypes } from '../../../../../model/notification.model';
import { Project } from '../../../../../model/project.model';
import { WNode, WNodeType, Workflow, WorkflowNotification } from '../../../../../model/workflow.model';

@Component({
    selector: 'app-workflow-notifications-form',
    templateUrl: './workflow.notifications.form.html',
    styleUrls: ['./workflow.notifications.form.scss']
})
export class WorkflowNotificationFormComponent {

    _notification: WorkflowNotification;
    @Input('notification')
    set notification(data: WorkflowNotification) {
        if (data) {
            this._notification = cloneDeep(data);
            if (this._notification.settings.recipients) {
                this.selectedUsers = this._notification.settings.recipients.join(',');
            }

            this.initNotif();
        }
    }

    get notification() {
        return this._notification;
    }

    types: Array<string>;
    notifOnSuccess: Array<string>;
    notifOnFailure: Array<string>;
    selectedUsers: string;
    nodeError = false;
    loadingNotifTemplate = false;

    nodes: Array<WNode>;
    _workflow: Workflow;
    @Input('workflow')
    set workflow(data: Workflow) {
        if (data) {
            this._workflow = data;
            this.nodes = Workflow.getAllNodes(data);
            if (this.nodes) {
                this.nodes = this.nodes.filter(n => n.type === WNodeType.PIPELINE);
            }
            this.initNotif()
        }
    }
    get workflow() {
        return this._workflow;
    }

    @Output() updatedNotification = new EventEmitter<WorkflowNotification>();
    @Output() deleteNotificationEvent = new EventEmitter<WorkflowNotification>();

    @Input() loading: boolean;
    @Input() project: Project;
    @Input() canDelete: boolean;

    constructor(private _notificationService: NotificationService) {
        this.notifOnSuccess = notificationOnSuccess;
        this.notifOnFailure = notificationOnFailure;
        this.types = notificationTypes;
    }

    initNotif(): void {
        if (this.nodes && this.notification && !this.notification.id) {
            this.notification.source_node_ref = this.nodes.map(n => {
                return n.name;
            });
        }

    }

    formatNode(): void {
        this.setNotificationTemplate();
        this.notification.source_node_ref = this.notification.source_node_ref.map(id => id.toString());
    }

    deleteNotification(): void {
        this.deleteNotificationEvent.emit(this.notification);
    }

    createNotification(): void {
        if (!this.notification.source_node_ref || this.notification.source_node_ref.length === 0) {
            this.nodeError = true;
            return;
        }
        this.nodeError = false;

        this.loading = true;

        if (this.selectedUsers) {
            this.notification.settings.recipients = this.selectedUsers.split(',');
        }
        this.updatedNotification.emit(this.notification);
    }

    setNotificationTemplate() {
        this.loadingNotifTemplate = true;
        this._notificationService.getNotificationTypes().pipe(first(), finalize(() => {
            this.loadingNotifTemplate = false;
        })).subscribe(data => {
            if (data && data[this.notification.type]) {
                this.notification.settings = data[this.notification.type];
            }
        });
    }
}
