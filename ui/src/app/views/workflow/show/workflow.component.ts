import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnInit, ViewChild } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { GroupPermission } from 'app/model/group.model';
import { Project } from 'app/model/project.model';
import { Workflow } from 'app/model/workflow.model';
import { WorkflowCoreService } from 'app/service/workflow/workflow.core.service';
import { WorkflowStore } from 'app/service/workflow/workflow.store';
import { AsCodeSaveModalComponent } from 'app/shared/ascode/save-modal/ascode.save-modal.component';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { WarningModalComponent } from 'app/shared/modal/warning/warning.component';
import { PermissionEvent } from 'app/shared/permission/permission.event.model';
import { ToastService } from 'app/shared/toast/ToastService';
import { WorkflowNodeRunParamComponent } from 'app/shared/workflow/node/run/node.run.param.component';
import * as actionsWorkflow from 'app/store/workflow.action';
import { CancelWorkflowEditMode } from 'app/store/workflow.action';
import { WorkflowState, WorkflowStateModel } from 'app/store/workflow.state';
import { WorkflowGraphComponent } from 'app/views/workflow/graph/workflow.graph.component';
import cloneDeep from 'lodash-es/cloneDeep';
import { Subscription } from 'rxjs';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-workflow',
    templateUrl: './workflow.html',
    styleUrls: ['./workflow.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowShowComponent implements OnInit {

    project: Project;
    detailedWorkflow: Workflow;
    previewWorkflow: Workflow;
    workflowSubscription: Subscription;
    workflowPreviewSubscription: Subscription;
    groups: Array<GroupPermission>;
    dataSubs: Subscription;
    paramsSubs: Subscription;
    qpsSubs: Subscription;
    direction: string;
    editMode: boolean;
    editModeWorkflowChanged: boolean;
    isReadOnly: boolean;

    @ViewChild('workflowGraph')
    workflowGraph: WorkflowGraphComponent;
    @ViewChild('workflowStartParam')
    runWithParamComponent: WorkflowNodeRunParamComponent;
    @ViewChild('permWarning')
    permWarningModal: WarningModalComponent;
    @ViewChild('updateAsCode')
    updateAsCodeModal: AsCodeSaveModalComponent;

    selectedHookRef: string;

    selectedTab = 'workflows';

    permFormLoading = false;

    loading = false;
    // For usage
    usageCount = 0;

    constructor(
        private _store: Store,
        private activatedRoute: ActivatedRoute,
        private _workflowStore: WorkflowStore,
        private _router: Router,
        private _translate: TranslateService,
        private _toast: ToastService,
        private _workflowCoreService: WorkflowCoreService,
        private _cd: ChangeDetectorRef
    ) { }

    ngOnInit(): void {
        // Update data if route change
        this.dataSubs = this.activatedRoute.data.subscribe(datas => {
            this.project = datas['project'];
        });

        this.workflowSubscription = this._store.select(WorkflowState.getCurrent()).subscribe((s: WorkflowStateModel) => {
            this._cd.markForCheck();
            this.editMode = s.editMode;
            if (s.editMode) {
                this.detailedWorkflow = s.editWorkflow;
                this.editModeWorkflowChanged = s.editModeWorkflowChanged;
            } else {
                this.detailedWorkflow = s.workflow;
            }
            this.isReadOnly = !!this.detailedWorkflow.from_template;
            if (this.detailedWorkflow) {
                let from_repository = this.detailedWorkflow.from_repository;
                this.previewWorkflow = this.detailedWorkflow.preview;
                this.groups = cloneDeep(this.detailedWorkflow.groups);
                if (this.detailedWorkflow.preview) {
                    // check to avoid "can't define property "x": "obj" is not extensible"
                    if (this.previewWorkflow.hasOwnProperty('from_repository')) {
                        this.previewWorkflow.from_repository = from_repository;
                    }
                }
                // If a node is selected, update it
                this.direction = this._workflowStore.getDirection(s.projectKey, this.detailedWorkflow.name);
                this._workflowStore.updateRecentWorkflow(s.projectKey, this.detailedWorkflow);

                if (!this.detailedWorkflow || !this.detailedWorkflow.usage) {
                    return;
                }

                this.usageCount = Object.keys(this.detailedWorkflow.usage).reduce((total, key) => {
                    return total + this.detailedWorkflow.usage[key].length;
                }, 0);
            }
        });


        this.paramsSubs = this.activatedRoute.params.subscribe(() => {
            this._workflowCoreService.toggleAsCodeEditor({ open: false, save: false });
            this._workflowCoreService.setWorkflowPreview(null);
        });

        this.qpsSubs = this.activatedRoute.queryParams.subscribe(params => {
            if (params['tab']) {
                this.selectedTab = params['tab'];
            }
            if (params['hook_ref']) {
                this.selectedHookRef = params['hook_ref'];
            } else {
                delete this.selectedHookRef;
            }
            this._cd.markForCheck();
        });

        this.workflowPreviewSubscription = this._workflowCoreService.getWorkflowPreview()
            .subscribe((wfPreview) => {
                if (wfPreview != null) {
                    this._workflowCoreService.toggleAsCodeEditor({ open: false, save: false });
                }
            });
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
                    this._store.dispatch(new actionsWorkflow.AddGroupInWorkflow({
                        projectKey: this.project.key,
                        workflowName: this.detailedWorkflow.name,
                        group: event.gp
                    })).pipe(finalize(() => this.permFormLoading = false))
                        .subscribe(() => this._toast.success('', this._translate.instant('permission_added')));

                    break;
                case 'update':
                    this._store.dispatch(new actionsWorkflow.UpdateGroupInWorkflow({
                        projectKey: this.project.key,
                        workflowName: this.detailedWorkflow.name,
                        group: event.gp
                    })).subscribe(() => this._toast.success('', this._translate.instant('permission_updated')));
                    break;
                case 'delete':
                    this._store.dispatch(new actionsWorkflow.DeleteGroupInWorkflow({
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
        this._store.dispatch(new actionsWorkflow.RollbackWorkflow({
            projectKey: this.project.key,
            workflowName: this.detailedWorkflow.name,
            auditId
        })).pipe(finalize(() => this.loading = false))
            .subscribe(() => {
                this._toast.success('', this._translate.instant('workflow_updated'));
                this._router.navigate(['/project', this.project.key, 'workflow', this.detailedWorkflow.name]);
            });
    }

    rollbackWorkflow(): void {
        this._store.dispatch(new CancelWorkflowEditMode());
    }

    saveWorkflow(): void {
        if (this.updateAsCodeModal) {
            this.updateAsCodeModal.show(this.detailedWorkflow, 'workflow');
        }
    }
}
