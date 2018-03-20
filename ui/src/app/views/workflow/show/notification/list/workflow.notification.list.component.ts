import {Component, Input} from '@angular/core';
import {Workflow, WorkflowNotification} from '../../../../../model/workflow.model';
import {WorkflowStore} from '../../../../../service/workflow/workflow.store';
import {TranslateService} from '@ngx-translate/core';
import {ToastService} from '../../../../../shared/toast/ToastService';
import {finalize} from 'rxjs/operators';
import {Project} from '../../../../../model/project.model';
import {cloneDeep} from 'lodash';

@Component({
    selector: 'app-workflow-notification-list',
    templateUrl: './workflow.notification.list.html',
    styleUrls: ['./workflow.notification.list.scss']
})
export class WorkflowNotificationListComponent {

    newNotification: WorkflowNotification;
    loading = false;
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

    constructor(private _workflowStore: WorkflowStore, private _translate: TranslateService, private _toast: ToastService) {
    }

    createNotification(n: WorkflowNotification): void {
        this.loading = true;
        let workflowToUpdate = cloneDeep(this.workflow);
        if (!workflowToUpdate.notifications) {
            workflowToUpdate.notifications = new Array<WorkflowNotification>();
        }
        if (n.settings && n.settings.recipients) {
            n.settings.recipients = n.settings.recipients.map(r => r.trim());
        }
        workflowToUpdate.notifications.push(n);

        this.updateWorkflow(workflowToUpdate);
    }

    copy(index: number) {
        this.newNotification = cloneDeep(this.workflow.notifications[index]);
        delete this.newNotification.id;
    }


    openNewNotifArea() {
        this.newNotification = new WorkflowNotification();
        delete this.selectedNotification;
    }

    openEditionArea(i: number) {
        this.selectedNotification = i;
        delete this.newNotification;
    }

    updateNotification(n: WorkflowNotification): void {
        let workflowToUpdate = cloneDeep(this.workflow);
        if (n.settings && n.settings.recipients) {
            n.settings.recipients = n.settings.recipients.map(r => r.trim());
        }
        workflowToUpdate.notifications = workflowToUpdate.notifications.map(notif => {
            if (notif.id !== n.id) {
                return notif;
            } else {
                return n;
            }
        });
        this.updateWorkflow(workflowToUpdate);
    }

    deleteNotification(n: WorkflowNotification): void {
        let workflowToUpdate = cloneDeep(this.workflow);
        workflowToUpdate.notifications = workflowToUpdate.notifications.filter(no => {
            return n.id !== no.id;
        });
        this.updateWorkflow(workflowToUpdate);
    }

    updateWorkflow(workflowToUpdate: Workflow): void {
        this.loading = true;
        this._workflowStore.updateWorkflow(this.project.key, workflowToUpdate).pipe(finalize(() => {
            this.loading = false;
            delete this.selectedNotification;
            delete this.newNotification;
        })).subscribe(() => {
            this._toast.success('', this._translate.instant('workflow_updated'));
        });
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
