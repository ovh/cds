import { Component, Input } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { NotificationService } from 'app/service/notification/notification.service';
import { AddNotificationWorkflow, DeleteNotificationWorkflow, UpdateNotificationWorkflow } from 'app/store/workflows.action';
import { cloneDeep } from 'lodash';
import { finalize, first } from 'rxjs/operators';
import { Project } from '../../../../../model/project.model';
import { Workflow, WorkflowNotification } from '../../../../../model/workflow.model';
import { ToastService } from '../../../../../shared/toast/ToastService';

@Component({
    selector: 'app-workflow-notification-list',
    templateUrl: './workflow.notification.list.html',
    styleUrls: ['./workflow.notification.list.scss']
})
export class WorkflowNotificationListComponent {

    newNotification: WorkflowNotification;
    loading = false;
    loadingNotifTemplate = false;
    selectedNotification: number;
    mapNodesNotif: Map<number, Array<string>>;
    _workflow: Workflow;
    @Input('workflow')
    set workflow(data: Workflow) {
        if (data) {
            this._workflow = data;
            this.refreshNotif();
        }
    }

    get workflow() {
        return this._workflow;
    }

    @Input() project: Project;

    constructor(
        private store: Store,
        private _notificationService: NotificationService,
        private _translate: TranslateService,
        private _toast: ToastService
    ) {
    }

    createNotification(n: WorkflowNotification): void {
        this.loading = true;
        if (n.settings && n.settings.recipients) {
            n.settings.recipients = n.settings.recipients.map(r => r.trim());
        }

        this.store.dispatch(new AddNotificationWorkflow({
            projectKey: this.project.key,
            workflowName: this.workflow.name,
            notification: n
        })).pipe(finalize(() => {
            this.loading = false;
            delete this.selectedNotification;
            delete this.newNotification;
        })).subscribe(() => this._toast.success('', this._translate.instant('workflow_updated')));
    }

    copy(index: number) {
        this.newNotification = cloneDeep(this.workflow.notifications[index]);
        delete this.newNotification.id;
    }

    setNotificationTemplate() {
        this.loadingNotifTemplate = true;
        this._notificationService.getNotificationTypes().pipe(first(), finalize(() => {
            this.loadingNotifTemplate = false;
        })).subscribe(data => {
            if (data && data[this.newNotification.type]) {
                this.newNotification.settings = data[this.newNotification.type];
            }
        });
    }

    openNewNotifArea() {
        this.newNotification = new WorkflowNotification();
        this.setNotificationTemplate();
        delete this.selectedNotification;
    }

    openEditionArea(i: number) {
        this.selectedNotification = i;
        delete this.newNotification;
    }

    updateNotification(n: WorkflowNotification): void {
        this.loading = true;
        if (n.settings && n.settings.recipients) {
            n.settings.recipients = n.settings.recipients.map(r => r.trim());
        }
        this.store.dispatch(new UpdateNotificationWorkflow({
            projectKey: this.project.key,
            workflowName: this.workflow.name,
            notification: n
        })).pipe(finalize(() => {
            this.loading = false;
            delete this.selectedNotification;
            delete this.newNotification;
        })).subscribe(() => this._toast.success('', this._translate.instant('workflow_updated')));
    }

    deleteNotification(n: WorkflowNotification): void {
        this.loading = true
        this.store.dispatch(new DeleteNotificationWorkflow({
            projectKey: this.project.key,
            workflowName: this.workflow.name,
            notification: n
        })).pipe(finalize(() => {
            this.loading = false;
            delete this.selectedNotification;
            delete this.newNotification;
        })).subscribe(() => this._toast.success('', this._translate.instant('workflow_updated')));
    }

    refreshNotif(): void {
        let mapNodes = Workflow.getMapNodes(this.workflow);
        this.mapNodesNotif = new Map<number, Array<string>>();
        if (this.workflow.notifications) {
            this.workflow.notifications.forEach(n => {
                let listNodes = new Array<string>();
                n.source_node_id.forEach(id => {
                    let node = mapNodes.get(id);
                    if (node) {
                        listNodes.push(node.name);
                    }
                });
                this.mapNodesNotif.set(n.id, listNodes);
            });
        }
    }


}
