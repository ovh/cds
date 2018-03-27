import {Component, ViewChild} from '@angular/core';
import {Project} from '../../../model/project.model';
import {ActivatedRoute, Router} from '@angular/router';
import {Subscription} from 'rxjs/Subscription';
import {Workflow, WorkflowNode, WorkflowNodeJoin} from '../../../model/workflow.model';
import {WorkflowStore} from '../../../service/workflow/workflow.store';
import {AutoUnsubscribe} from '../../../shared/decorator/autoUnsubscribe';
import {TranslateService} from '@ngx-translate/core';
import {ToastService} from '../../../shared/toast/ToastService';
import {cloneDeep} from 'lodash';
import {WorkflowJoinTriggerSrcComponent} from '../../../shared/workflow/join/trigger/src/trigger.src.component';
import {WorkflowGraphComponent} from '../graph/workflow.graph.component';
import {ActiveModal} from 'ng2-semantic-ui/dist';
import {WorkflowNodeRunParamComponent} from '../../../shared/workflow/node/run/node.run.param.component';
import {WorkflowCoreService} from '../../../service/workflow/workflow.core.service';
import {PermissionValue} from '../../../model/permission.model';
import {PermissionEvent} from '../../../shared/permission/permission.event.model';
import {WarningModalComponent} from '../../../shared/modal/warning/warning.component';
import {finalize, first} from 'rxjs/operators';

@Component({
    selector: 'app-workflow',
    templateUrl: './workflow.html',
    styleUrls: ['./workflow.scss']
})
@AutoUnsubscribe()
export class WorkflowShowComponent {

    project: Project;
    detailedWorkflow: Workflow;
    previewWorkflow: Workflow;
    workflowSubscription: Subscription;
    workflowPreviewSubscription: Subscription;
    direction: string;

    @ViewChild('workflowGraph')
    workflowGraph: WorkflowGraphComponent;
    @ViewChild('workflowJoinTriggerSrc')
    workflowJoinTriggerSrc: WorkflowJoinTriggerSrcComponent;
    @ViewChild('workflowNodeRunParam')
    runWithParamComponent: WorkflowNodeRunParamComponent;
    @ViewChild('permWarning')
    permWarningModal: WarningModalComponent;

    selectedNode: WorkflowNode;
    selectedJoin: WorkflowNodeJoin;

    selectedTab = 'workflows';

    permissionEnum = PermissionValue;
    permFormLoading = false;

    loading = false;
    // For usage
    usageCount = 0;

    constructor(private activatedRoute: ActivatedRoute, private _workflowStore: WorkflowStore, private _router: Router,
                private _translate: TranslateService, private _toast: ToastService,
                private _workflowCoreService: WorkflowCoreService) {
        // Update data if route change
        this.activatedRoute.data.subscribe(datas => {
            this.project = datas['project'];
        });

        this.activatedRoute.queryParams.subscribe(params => {
            if (params['tab']) {
                this.selectedTab = params['tab'];
            }
        });

        this.activatedRoute.params.subscribe(params => {
            let workflowName = params['workflowName'];
            let projkey = params['key'];

            this._workflowCoreService.toggleAsCodeEditor({open: false, save: false});
            this._workflowCoreService.setWorkflowPreview(null);
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

        this._workflowCoreService.setCurrentWorkflowRun(null);

        this.workflowPreviewSubscription = this._workflowCoreService.getWorkflowPreview()
            .subscribe((wfPreview) => {
                this.previewWorkflow = wfPreview;
                if (wfPreview != null) {
                    this._workflowCoreService.toggleAsCodeEditor({open: false, save: false});
                }
            });
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

    public openDeleteJoinSrcModal(data: { source, target }) {
        let pID = Number(data.source.replace('node-', ''));
        let cID = Number(data.target.replace('join-', ''));

        this.selectedNode = Workflow.getNodeByID(pID, this.detailedWorkflow);
        this.selectedJoin = this.detailedWorkflow.joins.find(j => j.id === cID);

        if (this.workflowJoinTriggerSrc) {
            this.workflowJoinTriggerSrc.show();
        }
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

    public addSourceToJoin(data: { source: WorkflowNode, target: WorkflowNodeJoin }): void {
        let clonedWorkflow: Workflow = cloneDeep(this.detailedWorkflow);
        let currentJoin = clonedWorkflow.joins.find(j => j.id === data.target.id);
        this.selectedNode = data.source;
        if (currentJoin.source_node_id.find(id => id === this.selectedNode.id)) {
            return;
        }
        currentJoin.source_node_ref.push(this.selectedNode.ref);
        this.updateWorkflow(clonedWorkflow);
    }

    deleteJoinSrc(action: string): void {
        let clonedWorkflow: Workflow = cloneDeep(this.detailedWorkflow);

        switch (action) {
            case 'delete_join':
                clonedWorkflow.joins = clonedWorkflow.joins.filter(j => j.id !== this.selectedJoin.id);
                Workflow.removeOldRef(clonedWorkflow);
                break;
            default:
                let currentJoin = clonedWorkflow.joins.find(j => j.id === this.selectedJoin.id);
                currentJoin.source_node_ref = currentJoin.source_node_ref.filter(ref => ref !== this.selectedNode.ref);
        }

        this.updateWorkflow(clonedWorkflow, this.workflowJoinTriggerSrc.modal);
    }

    updateWorkflow(w: Workflow, modal?: ActiveModal<boolean, boolean, void>): void {
        this.loading = true;
        this._workflowStore.updateWorkflow(this.project.key, w).pipe(
            finalize(() => this.loading = false),
            first()
          )
          .subscribe(() => {
            this._toast.success('', this._translate.instant('workflow_updated'));
            if (modal) {
                modal.approve(true);
            }
            if (this.workflowGraph) {
                this._workflowCoreService.linkJoinEvent(null);
            }
            this._router.navigate(['/project', this.project.key, 'workflow', w.name]);
        });
    }

    runWithParameter(): void {
        if (this.runWithParamComponent) {
            this.runWithParamComponent.show();
        }
    }
}
