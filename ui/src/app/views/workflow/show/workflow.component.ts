import { Component, OnInit, ViewChild } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import * as actionsWorkflow from 'app/store/workflows.action';
import { WorkflowsState } from 'app/store/workflows.state';
import cloneDeep from 'lodash-es/cloneDeep';
import { Subscription } from 'rxjs';
import { filter, finalize } from 'rxjs/operators';
import { PermissionValue } from '../../../model/permission.model';
import { Project } from '../../../model/project.model';
import { WNode, Workflow } from '../../../model/workflow.model';
import { WorkflowCoreService } from '../../../service/workflow/workflow.core.service';
import { WorkflowEventStore } from '../../../service/workflow/workflow.event.store';
import { WorkflowStore } from '../../../service/workflow/workflow.store';
import { AutoUnsubscribe } from '../../../shared/decorator/autoUnsubscribe';
import { WarningModalComponent } from '../../../shared/modal/warning/warning.component';
import { PermissionEvent } from '../../../shared/permission/permission.event.model';
import { ToastService } from '../../../shared/toast/ToastService';
import { WorkflowNodeRunParamComponent } from '../../../shared/workflow/node/run/node.run.param.component';
import { WorkflowGraphComponent } from '../graph/workflow.graph.component';

@Component({
    selector: 'app-workflow',
    templateUrl: './workflow.html',
    styleUrls: ['./workflow.scss']
})
@AutoUnsubscribe()
export class WorkflowShowComponent implements OnInit {

    project: Project;
    detailedWorkflow: Workflow;
    previewWorkflow: Workflow;
    workflowSubscription: Subscription;
    workflowPreviewSubscription: Subscription;
    dataSubs: Subscription;
    paramsSubs: Subscription;
    qpsSubs: Subscription;
    direction: string;

    @ViewChild('workflowGraph')
    workflowGraph: WorkflowGraphComponent;
    @ViewChild('workflowStartParam')
    runWithParamComponent: WorkflowNodeRunParamComponent;
    @ViewChild('permWarning')
    permWarningModal: WarningModalComponent;

    selectedNode: WNode;
    selectedNodeID: number;
    selectedNodeRef: string;
    selectedHookRef: string;

    selectedTab = 'workflows';

    permissionEnum = PermissionValue;
    permFormLoading = false;

    loading = false;
    // For usage
    usageCount = 0;

    constructor(
        private store: Store,
        private activatedRoute: ActivatedRoute,
        private _workflowStore: WorkflowStore,
        private _router: Router,
        private _translate: TranslateService,
        private _toast: ToastService,
        private _workflowCoreService: WorkflowCoreService,
        private _workflowEventStore: WorkflowEventStore
    ) {
    }

    ngOnInit(): void {
        // Update data if route change
        this.dataSubs = this.activatedRoute.data.subscribe(datas => {
            this.project = datas['project'];
        });


        this.paramsSubs = this.activatedRoute.params.subscribe(params => {
            let workflowName = params['workflowName'];
            let projkey = params['key'];

            this._workflowCoreService.toggleAsCodeEditor({ open: false, save: false });
            this._workflowCoreService.setWorkflowPreview(null);

            if (!this.activatedRoute.snapshot.queryParams['node_id'] && !this.activatedRoute.snapshot.queryParams['node_ref']) {
                this._workflowEventStore.unselectAll();
            }
            if (projkey && workflowName) {
                if (this.workflowSubscription) {
                    this.workflowSubscription.unsubscribe();
                }
                this.store.dispatch(new actionsWorkflow.FetchWorkflow({
                    projectKey: projkey,
                    workflowName
                })).subscribe(null, () => this._router.navigate(['/project', projkey]));

                this.workflowSubscription = this.store.select(WorkflowsState.selectWorkflow(projkey, workflowName))
                    .pipe(filter((wf) => wf != null && !wf.externalChange))
                    .subscribe((wf) => {
                        // TODO: delete cloneDeep
                        this.detailedWorkflow = cloneDeep(wf);
                        this.previewWorkflow = wf.preview;
                        // If a node is selected, update it
                        this.direction = this._workflowStore.getDirection(projkey, this.detailedWorkflow.name);
                        this._workflowStore.updateRecentWorkflow(projkey, wf);

                        if (!this.detailedWorkflow || !this.detailedWorkflow.usage) {
                            return;
                        }

                        this.usageCount = Object.keys(this.detailedWorkflow.usage).reduce((total, key) => {
                            return total + this.detailedWorkflow.usage[key].length;
                        }, 0);
                    }, () => this._router.navigate(['/project', projkey]));
            }
        });

        this.qpsSubs = this.activatedRoute.queryParams.subscribe(params => {
            if (params['tab']) {
                this.selectedTab = params['tab'];
            }
            if (params['node_id']) {
                this.selectedNodeID = params['node_id'];
            } else {
                delete this.selectedNodeID;
            }
            if (params['node_ref']) {
                this.selectedNodeRef = params['node_ref'];
            } else {
                delete this.selectedNodeRef;
            }
            if (params['hook_ref']) {
                this.selectedHookRef = params['hook_ref'];
            } else {
                delete this.selectedHookRef;
            }
            this.selectNode();
        });

        this.workflowPreviewSubscription = this._workflowCoreService.getWorkflowPreview()
            .subscribe((wfPreview) => {
                if (wfPreview != null) {
                    this._workflowCoreService.toggleAsCodeEditor({ open: false, save: false });
                }
            });
    }

    selectNode() {
        if (!this.detailedWorkflow) {
            return;
        }
        if (this.selectedNodeID) {
            let n = Workflow.getNodeByID(this.selectedNodeID, this.detailedWorkflow);
            if (n) {
                this.selectedNode = n;
                this._workflowEventStore.setSelectedNode(n, true);
                return;
            }
        }
        if (this.selectedNodeRef) {
            let n = Workflow.getNodeByRef(this.selectedNodeRef, this.detailedWorkflow);
            if (n) {
                this.selectedNode = n;
                this._workflowEventStore.setSelectedNode(n, true);
                return;
            }
        }
        if (this.selectedHookRef) {
            let h = Workflow.getHookByRef(this.selectedHookRef, this.detailedWorkflow)
            if (h) {
                this._workflowEventStore.setSelectedHook(h);
                return;
            }
        }
        this._workflowEventStore.setSelectedNode(null, true);
    }

    savePreview() {
        this._workflowCoreService.toggleAsCodeEditor({ open: false, save: true });
    }

    changeDirection() {
        this.direction = this.direction === 'LR' ? 'TB' : 'LR';
    }

    showTab(tab: string): void {
        this._router.navigateByUrl('/project/' + this.project.key + '/workflow/' + this.detailedWorkflow.name + '?tab=' + tab);
    }

    showAsCodeEditor() {
        this._workflowCoreService.toggleAsCodeEditor({ open: true, save: false });
    }

    groupManagement(event: PermissionEvent, skip?: boolean): void {
        if (!skip && this.detailedWorkflow.externalChange) {
            this.permWarningModal.show(event);
        } else {
            switch (event.type) {
                case 'add':
                    this.permFormLoading = true;
                    this.store.dispatch(new actionsWorkflow.AddGroupInWorkflow({
                        projectKey: this.project.key,
                        workflowName: this.detailedWorkflow.name,
                        group: event.gp
                    })).pipe(finalize(() => this.permFormLoading = false))
                        .subscribe(() => this._toast.success('', this._translate.instant('permission_added')));

                    break;
                case 'update':
                    this.store.dispatch(new actionsWorkflow.UpdateGroupInWorkflow({
                        projectKey: this.project.key,
                        workflowName: this.detailedWorkflow.name,
                        group: event.gp
                    })).subscribe(() => this._toast.success('', this._translate.instant('permission_updated')));
                    break;
                case 'delete':
                    this.store.dispatch(new actionsWorkflow.DeleteGroupInWorkflow({
                        projectKey: this.project.key,
                        workflowName: this.detailedWorkflow.name,
                        group: event.gp
                    })).subscribe(() => this._toast.success('', this._translate.instant('permission_deleted')));
                    break;
            }
        }
    }

    runWithParameter(): void {
        if (this.runWithParamComponent) {
            this.runWithParamComponent.show();
        }
    }

    rollback(auditId: number): void {
        this.loading = true;
        this.store.dispatch(new actionsWorkflow.RollbackWorkflow({
            projectKey: this.project.key,
            workflowName: this.detailedWorkflow.name,
            auditId
        })).pipe(finalize(() => this.loading = false))
            .subscribe(() => {
                this._toast.success('', this._translate.instant('workflow_updated'));
                this._router.navigate(['/project', this.project.key, 'workflow', this.detailedWorkflow.name]);
            });
    }
}
