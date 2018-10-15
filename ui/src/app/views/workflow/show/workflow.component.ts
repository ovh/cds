import {Component, OnInit, ViewChild} from '@angular/core';
import {ActivatedRoute, Router} from '@angular/router';
import {TranslateService} from '@ngx-translate/core';
import {Subscription} from 'rxjs';
import {finalize, first} from 'rxjs/operators';
import {PermissionValue} from '../../../model/permission.model';
import {Project} from '../../../model/project.model';
import {WNode, Workflow} from '../../../model/workflow.model';
import {WorkflowCoreService} from '../../../service/workflow/workflow.core.service';
import {WorkflowEventStore} from '../../../service/workflow/workflow.event.store';
import {WorkflowStore} from '../../../service/workflow/workflow.store';
import {AutoUnsubscribe} from '../../../shared/decorator/autoUnsubscribe';
import {WarningModalComponent} from '../../../shared/modal/warning/warning.component';
import {PermissionEvent} from '../../../shared/permission/permission.event.model';
import {ToastService} from '../../../shared/toast/ToastService';
import {WorkflowNodeRunParamComponent} from '../../../shared/workflow/node/run/node.run.param.component';
import {WorkflowGraphComponent} from '../graph/workflow.graph.component';

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
    direction: string;

    @ViewChild('workflowGraph')
    workflowGraph: WorkflowGraphComponent;
    @ViewChild('workflowNodeRunParam')
    runWithParamComponent: WorkflowNodeRunParamComponent;
    @ViewChild('permWarning')
    permWarningModal: WarningModalComponent;

    selectedNode: WNode;
    selectedNodeID: number;
    selectedNodeRef: string;

    selectedTab = 'workflows';

    permissionEnum = PermissionValue;
    permFormLoading = false;

    loading = false;
    // For usage
    usageCount = 0;

    constructor(private activatedRoute: ActivatedRoute, private _workflowStore: WorkflowStore, private _router: Router,
                private _translate: TranslateService, private _toast: ToastService,
                private _workflowCoreService: WorkflowCoreService, private _workflowEventStore: WorkflowEventStore) {
    }

    ngOnInit(): void {
        // Update data if route change
        this.activatedRoute.data.subscribe(datas => {
            this.project = datas['project'];
        });


        this.activatedRoute.params.subscribe(params => {
            let workflowName = params['workflowName'];
            let projkey = params['key'];

            this._workflowCoreService.toggleAsCodeEditor({open: false, save: false});
            this._workflowCoreService.setWorkflowPreview(null);

            if (!this.activatedRoute.snapshot.queryParams['node_id'] && !this.activatedRoute.snapshot.queryParams['node_ref']) {
                this._workflowEventStore.unselectAll();
            }
            if (projkey && workflowName) {
                if (this.workflowSubscription) {
                    this.workflowSubscription.unsubscribe();
                }

                this.workflowSubscription = this._workflowStore.getWorkflows(projkey, workflowName).subscribe(ws => {
                    if (ws) {
                        let updatedWorkflow = ws.get(projkey + '-' + workflowName);
                        if (updatedWorkflow && !updatedWorkflow.externalChange) {
                            if (this.detailedWorkflow && this.detailedWorkflow.last_modified === updatedWorkflow.last_modified) {
                                return;
                            }
                            this.detailedWorkflow = updatedWorkflow;

                            // If a node is selected, update it
                            this.direction = this._workflowStore.getDirection(projkey, this.detailedWorkflow.name);
                            this._workflowStore.updateRecentWorkflow(projkey, updatedWorkflow);

                            if (!this.detailedWorkflow || !this.detailedWorkflow.usage) {
                                return;
                            }

                            this.usageCount = Object.keys(this.detailedWorkflow.usage).reduce((total, key) => {
                                return total + this.detailedWorkflow.usage[key].length;
                            }, 0);
                        }
                    }
                }, () => {
                    this._router.navigate(['/project', projkey]);

                });
            }
        });

        this.activatedRoute.queryParams.subscribe(params => {
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
            this.selectNode();
        });

        this.workflowPreviewSubscription = this._workflowCoreService.getWorkflowPreview()
            .subscribe((wfPreview) => {
                this.previewWorkflow = wfPreview;
                if (wfPreview != null) {
                    this._workflowCoreService.toggleAsCodeEditor({open: false, save: false});
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
        this._workflowEventStore.setSelectedNode(null, true);
    }

    savePreview() {
        this._workflowCoreService.toggleAsCodeEditor({open: false, save: true});
    }

    changeDirection() {
        this.direction = this.direction === 'LR' ? 'TB' : 'LR';
    }

    showTab(tab: string): void {
        this._router.navigateByUrl('/project/' + this.project.key + '/workflow/' + this.detailedWorkflow.name + '?tab=' + tab);
    }

    showAsCodeEditor() {
      this._workflowCoreService.toggleAsCodeEditor({open: true, save: false});
    }

    groupManagement(event: PermissionEvent, skip?: boolean): void {
        if (!skip && this.detailedWorkflow.externalChange) {
            this.permWarningModal.show(event);
        } else {
            switch (event.type) {
                case 'add':
                    this.permFormLoading = true;
                    this._workflowStore.addPermission(this.project.key, this.detailedWorkflow, event.gp).pipe(finalize(() => {
                        this.permFormLoading = false;
                    })).subscribe(() => {
                        this._toast.success('', this._translate.instant('permission_added'));

                    });
                    break;
                case 'update':
                    this._workflowStore.updatePermission(this.project.key, this.detailedWorkflow, event.gp).subscribe(() => {
                        this._toast.success('', this._translate.instant('permission_updated'));
                    });
                    break;
                case 'delete':
                    this._workflowStore.deletePermission(this.project.key, this.detailedWorkflow, event.gp).subscribe(() => {
                        this._toast.success('', this._translate.instant('permission_deleted'));
                    });
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
      this._workflowStore.rollbackWorkflow(this.project.key, this.detailedWorkflow.name, auditId)
        .pipe(
          finalize(() => this.loading = false),
          first()
        )
        .subscribe((wf) => {
          this._toast.success('', this._translate.instant('workflow_updated'));
          this._router.navigate(['/project', this.project.key, 'workflow', wf.name]);
        });
    }
}
