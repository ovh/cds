import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { ProjectIntegration } from 'app/model/integration.model';
import { Project } from 'app/model/project.model';
import { Workflow, WorkflowNotification, WorkflowProjectIntegration } from 'app/model/workflow.model';
import { NotificationService } from 'app/service/notification/notification.service';
import { ToastService } from 'app/shared/toast/ToastService';
// eslint-disable-next-line max-len
import { AddNotificationWorkflow, DeleteIntegrationWorkflow, DeleteNotificationWorkflow,
    UpdateIntegrationsWorkflow, UpdateNotificationWorkflow } from 'app/store/workflow.action';
import cloneDeep from 'lodash-es/cloneDeep';
import { finalize, first } from 'rxjs/operators';

@Component({
    selector: 'app-workflow-notification-list',
    templateUrl: './workflow.notification.list.html',
    styleUrls: ['./workflow.notification.list.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class WorkflowNotificationListComponent {

    tab: 'newNotification' | 'newEvent';
    newNotification: WorkflowNotification;
    loading = false;
    loadingNotifTemplate = false;
    selectedNotification: number;
    mapNodesNotif: Map<number, Array<string>>;
    _workflow: Workflow;

    selectedIntegration: ProjectIntegration;

    @Input() editMode: boolean;
    @Input() readOnly: boolean;

    @Input()
    set workflow(data: Workflow) {
        if (data) {
            this._workflow = data;
            this.refreshNotif();
        }
    }

    get workflow() {
        return this._workflow;
    }

    eventIntegrations: ProjectIntegration[];
    _project: Project;
    @Input()
    set project(proj: Project) {
        this._project = proj;
        if (proj && proj.integrations) {
            this.eventIntegrations = proj.integrations.filter((integ) => integ.model.event && !integ.model.public);
        }
    }
    get project(): Project {
        return this._project;
    }

    constructor(
        private store: Store,
        private _notificationService: NotificationService,
        private _translate: TranslateService,
        private _toast: ToastService,
        private _cd: ChangeDetectorRef
    ) { }

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
            this._cd.markForCheck();
        })).subscribe(() => {
            if (this.editMode) {
                this._toast.info('', this._translate.instant('workflow_ascode_updated'));
            } else {
                this._toast.success('', this._translate.instant('workflow_updated'));
            }
        });
    }

    setNotificationTemplate() {
        this.loadingNotifTemplate = true;
        this._notificationService.getNotificationTypes().pipe(first(), finalize(() => {
            this.loadingNotifTemplate = false;
            this._cd.markForCheck();
        })).subscribe(data => {
            if (data && data[this.newNotification.type]) {
                this.newNotification.settings = data[this.newNotification.type];
            }
        });
    }

    openNewNotifArea() {
        this.tab = 'newNotification';
        this.selectedNotification = null;
        this.newNotification = new WorkflowNotification();
        this.setNotificationTemplate();
    }

    openEditionArea(i: number) {
        this.tab = null;
        this.selectedNotification = i;
        this.newNotification = null;
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
            this._cd.markForCheck();
        })).subscribe(() => {
            if (this.editMode) {
                this._toast.info('', this._translate.instant('workflow_ascode_updated'));
            } else {
                this._toast.success('', this._translate.instant('workflow_updated'));
            }
        });
    }

    deleteNotification(n: WorkflowNotification): void {
        this.loading = true;
        this.store.dispatch(new DeleteNotificationWorkflow({
            projectKey: this.project.key,
            workflowName: this.workflow.name,
            notification: n
        })).pipe(finalize(() => {
            this.loading = false;
            delete this.selectedNotification;
            delete this.newNotification;
            this._cd.markForCheck();
        })).subscribe(() => {
            if (this.editMode) {
                this._toast.info('', this._translate.instant('workflow_ascode_updated'));
            } else {
                this._toast.success('', this._translate.instant('workflow_updated'));
            }
        });
    }

    refreshNotif(): void {
        let mapNodes = Workflow.getMapNodes(this.workflow);
        this.mapNodesNotif = new Map<number, Array<string>>();
        if (this.workflow.notifications) {
            this.workflow.notifications.forEach(n => {
                let listNodes = new Array<string>();
                if (n.node_id) {
                    n.node_id.forEach(id => {
                        let node = mapNodes.get(id);
                        if (node) {
                            listNodes.push(node.name);
                        }
                    });
                    this.mapNodesNotif.set(n.id, listNodes);
                }
            });
        }
    }

    openNewEventArea() {
        this.tab = 'newEvent';
        this.selectedNotification = null;
    }

    addEvent(integration: ProjectIntegration) {
        this.loading = true;
        let workflowIntegrations = new Array<WorkflowProjectIntegration>();
        let wi = new WorkflowProjectIntegration();
        wi.project_integration = integration;
        wi.project_integration_id = integration.id;
        workflowIntegrations.push(wi);
        if (this.workflow.integrations) {
            workflowIntegrations = [wi].concat(this.workflow.integrations);
        }
        this.store.dispatch(new UpdateIntegrationsWorkflow({
            projectKey: this.project.key,
            workflowName: this.workflow.name,
            integrations: workflowIntegrations,
        })).pipe(finalize(() => {
            this.loading = false;
            this._cd.markForCheck();
        })).subscribe();
    }

    deleteEvent(projectIntegrationID: number) {
        this.loading = true;
        this.store.dispatch(new DeleteIntegrationWorkflow({
            projectKey: this.project.key,
            workflowName: this.workflow.name,
            projectIntegrationID,
        })).pipe(finalize(() => {
            this.loading = false;
            this._cd.markForCheck();
        })).subscribe();
    }

    filterEvent(wpi: WorkflowProjectIntegration): boolean {
        return wpi.project_integration.model.event;
    }
}
