import { ChangeDetectionStrategy, ChangeDetectorRef, Component, EventEmitter, Input, OnInit, Output } from '@angular/core';
import { Project } from 'app/model/project.model';
// eslint-disable-next-line max-len
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
    @Input() set notification(data: WorkflowNotification) {
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

    @Input() editMode: boolean;
    @Input() readOnly: boolean;

    types: Array<string>;
    notifOnSuccess: Array<string>;
    notifOnFailure: Array<string>;
    selectedUsers: string;
    commentEnabled = true;
    statusEnabled = true;
    alwaysSend = true;
    loadingNotifTemplate = false;
    triggerConditions: WorkflowTriggerConditionCache;

    nodes: Array<WNode>;
    _workflow: Workflow;
    @Input() set workflow(data: Workflow) {
        if (data) {
            this._workflow = data;
            this.nodes = Workflow.getAllNodes(data);
            if (this.nodes) {
                this.nodes = this.nodes.filter(n => n.type === WNodeType.PIPELINE);
            }
            this.initNotif();
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
        if (this.nodes && this._notification && !this._notification.id) {
            this._notification.source_node_ref = this.nodes.map(n => n.name);
        }

        if (this._notification && this._notification.type === 'vcs') {
            this.statusEnabled = !this._notification.settings.template.disable_status;
            this.commentEnabled = !this._notification.settings.template.disable_comment;
            this.alwaysSend = this._notification.settings.on_success === 'always';
        }
    }

    formatNode(): void {
        this.setNotificationTemplate();
    }

    deleteNotification(): void {
        this.deleteNotificationEvent.emit(cloneDeep(this._notification));
    }

    createNotification(): void {
        this.loading = true;
        this._notification.node_id = [];
        this._notification.source_node_ref.forEach(source => {
            let n = Workflow.getAllNodes(this.workflow).find(p => p.name === source);
            this._notification.node_id.push(n.id);
        });

        if (this.selectedUsers != null) {
            this._notification.settings.recipients = this.selectedUsers.split(',');
        }
        if (this._notification.type === 'vcs') {
            this._notification.settings.template.disable_comment = !this.commentEnabled;
            this._notification.settings.template.disable_status = !this.statusEnabled;
            if (this.alwaysSend) {
                this._notification.settings.on_success = 'always';
            } else {
                this._notification.settings.on_success = null;
            }
        }
        this.updatedNotification.emit(cloneDeep(this._notification));
    }

    setNotificationTemplate() {
        this.loadingNotifTemplate = true;
        this._notificationService.getNotificationTypes().pipe(first(), finalize(() => {
            this.loadingNotifTemplate = false;
            this._cd.markForCheck();
        })).subscribe(data => {
            if (data && data[this._notification.type]) {
                this._notification.settings = data[this._notification.type];
            }
        });
    }
}
