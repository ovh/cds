import { ChangeDetectionStrategy, ChangeDetectorRef, Component, EventEmitter, Input, OnInit, Output } from '@angular/core';
import { Project } from 'app/model/project.model';
// tslint:disable-next-line: max-line-length
import { notificationOnFailure, notificationOnSuccess, notificationTypes, WNode, WNodeType, Workflow, WorkflowNotification, WorkflowTriggerConditionCache } from 'app/model/workflow.model';
import { NotificationService } from 'app/service/notification/notification.service';
import cloneDeep from 'lodash-es/cloneDeep';
import { finalize, first } from 'rxjs/operators';

@Component({
    selector: 'app-workflow-notifications-form',
    templateUrl: './workflow.notifications.form.html',
    styleUrls: ['./workflow.notifications.form.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class WorkflowNotificationFormComponent implements OnInit {

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
    commentEnabled = true;
    alwaysSend = true;
    nodeError = false;
    loadingNotifTemplate = false;
    triggerConditions: WorkflowTriggerConditionCache;

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

    constructor(private _notificationService: NotificationService, private _cd: ChangeDetectorRef) {
        this.notifOnSuccess = notificationOnSuccess;
        this.notifOnFailure = notificationOnFailure;
        this.types = notificationTypes;
    }

    ngOnInit() {
        this.loading = true;
        this._notificationService.getConditions(this.project.key, this.workflow.name)
            .pipe(
                first(),
                finalize(() => {
                    this.loading = false;
                    this._cd.markForCheck();
                })
            )
            .subscribe((wtc) => this.triggerConditions = wtc);
    }

    initNotif(): void {
        if (this.nodes && this.notification && !this.notification.id) {
            this.notification.source_node_ref = this.nodes.map(n => {
                return n.name;
            });
        }

        if (this.notification && this.notification.type === 'vcs') {
            this.commentEnabled = !this.notification.settings.template.disable_comment;
            this.alwaysSend = this.notification.settings.on_success === 'always';
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

        if (this.selectedUsers != null) {
            this.notification.settings.recipients = this.selectedUsers.split(',');
        }
        if (this.notification.type === 'vcs') {
            this.notification.settings.template.disable_comment = !this.commentEnabled;
            if (this.alwaysSend) {
                this.notification.settings.on_success = 'always';
            } else {
                this.notification.settings.on_success = null;
            }
        }
        this.updatedNotification.emit(this.notification);
    }

    setNotificationTemplate() {
        this.loadingNotifTemplate = true;
        this._notificationService.getNotificationTypes().pipe(first(), finalize(() => {
            this.loadingNotifTemplate = false;
            this._cd.markForCheck();
        })).subscribe(data => {
            if (data && data[this.notification.type]) {
                this.notification.settings = data[this.notification.type];
            }
        });
    }
}
