import {
    AfterViewInit,
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    OnDestroy,
    OnInit,
    TemplateRef,
    ViewChild
} from '@angular/core';
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
import { PermissionEvent } from 'app/shared/permission/permission.event.model';
import { ToastService } from 'app/shared/toast/ToastService';
import { WorkflowNodeRunParamComponent } from 'app/shared/workflow/node/run/node.run.param.component';
import * as actionsWorkflow from 'app/store/workflow.action';
import { CancelWorkflowEditMode, SelectWorkflowNode } from 'app/store/workflow.action';
import { WorkflowState, WorkflowStateModel } from 'app/store/workflow.state';
import cloneDeep from 'lodash-es/cloneDeep';
import { Subscription } from 'rxjs';
import { finalize, first } from 'rxjs/operators';
import { Tab } from 'app/shared/tabs/tabs.component';
import { NzModalService } from 'ng-zorro-antd/modal';

@Component({
    selector: 'app-workflow',
    templateUrl: './workflow.html',
    styleUrls: ['./workflow.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowShowComponent implements OnInit, OnDestroy, AfterViewInit {

    project: Project;
    detailedWorkflow: Workflow;
    previewWorkflow: Workflow;
    workflowSubscription: Subscription;
    workflowPreviewSubscription: Subscription;
    groups: Array<GroupPermission>;
    groupsOutsideOrganization: Array<GroupPermission>;
    dataSubs: Subscription;
    paramsSubs: Subscription;
    qpsSubs: Subscription;
    direction: string;
    editMode: boolean;
    editModeWorkflowChanged: boolean;
    isReadOnly: boolean;

    @ViewChild('warnPermission') warnPermission: TemplateRef<any>;

    selectedHookRef: string;

    tabs: Array<Tab>;
    selectedTab: Tab;

    permFormLoading = false;

    loading = false;
    // For usage
    usageCount = 0;

    constructor(
        private _store: Store,
        private activatedRoute: ActivatedRoute,
        private _workflowStore: WorkflowStore,
        private _router: Router,
        public _translate: TranslateService,
        private _toast: ToastService,
        private _workflowCoreService: WorkflowCoreService,
        private _cd: ChangeDetectorRef,
        private _modalService: NzModalService
    ) {
    }

    ngAfterViewInit(): void {
        if (this.detailedWorkflow) {
            this.initTabs();
        }
    }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit(): void {
        // Update data if route change
        this.dataSubs = this.activatedRoute.data.subscribe(datas => {
            this.project = datas['project'];
        });

        this.workflowSubscription = this._store.select(WorkflowState.current).subscribe((s: WorkflowStateModel) => {
            this._cd.markForCheck();
            this.editMode = s.editMode;
            if (s.editMode) {
                this.detailedWorkflow = s.editWorkflow;
                this.editModeWorkflowChanged = s.editModeWorkflowChanged;
            } else {
                this.detailedWorkflow = s.workflow;
            }
            if (this.detailedWorkflow) {
                this.isReadOnly = !!this.detailedWorkflow.from_template && !!this.detailedWorkflow.from_repository;
                let from_repository = this.detailedWorkflow.from_repository;
                this.previewWorkflow = this.detailedWorkflow.preview;
                this.groups = cloneDeep(this.detailedWorkflow.groups);
                if (!!this.detailedWorkflow.organization) {
                    this.groupsOutsideOrganization = this.groups.filter(gp =>
                        gp.group.organization && gp.group.organization !== this.detailedWorkflow.organization);
                }
                if (this.detailedWorkflow.preview) {
                    // check to avoid "can't define property "x": "obj" is not extensible"
                    if (this.previewWorkflow.hasOwnProperty('from_repository')) {
                        this.previewWorkflow.from_repository = from_repository;
                    }
                }
                // If a node is selected, update it
                this.direction = this._workflowStore.getDirection(s.projectKey, this.detailedWorkflow.name);
                
                if (!this.detailedWorkflow || !this.detailedWorkflow.usage) {
                    return;
                }

                this.usageCount = Object.keys(this.detailedWorkflow.usage).reduce((total, key) =>
                    total + this.detailedWorkflow.usage[key].length, 0);

                this.initTabs();
            }
        });


        this.paramsSubs = this.activatedRoute.params.subscribe(() => {
            this._workflowCoreService.toggleAsCodeEditor({ open: false, save: false });
            this._workflowCoreService.setWorkflowPreview(null);
        });

        this.qpsSubs = this.activatedRoute.queryParams.subscribe(params => {
            if (params['tab']) {
                let current_tab = this.tabs.find((t) => t.key === params['tab']);
                if (current_tab) {
                    this.selectTab(current_tab);
                }
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

    initTabs() {
        let graphTab = <Tab>{
            title: 'Workflows',
            key: 'workflows',
            icon: 'share-alt',
            iconTheme: 'outline',
            default: true
        }
        if (this.previewWorkflow) {
            graphTab.warningText = this._translate.instant('workflow_preview_mode');
        }
        let notificationTab = <Tab>{
            title: 'Notifications',
            key: 'notifications',
            icon: 'bell',
            iconTheme: 'outline',
        };
        let permissionTab = <Tab>{
            title: 'Permissions',
            key: 'permissions',
            icon: 'user-switch',
            iconTheme: 'outline',
        }
        if (this.groupsOutsideOrganization && this.groupsOutsideOrganization.length > 0) {
            permissionTab.warningTemplate = this.warnPermission
        }

        this.tabs = new Array<Tab>();
        this.tabs.push(graphTab, notificationTab, permissionTab)

        if (!this.detailedWorkflow.from_repository) {
            this.tabs.push(<Tab>{
                title: 'Audit',
                icon: 'history',
                iconTheme: 'outline',
                key: 'audits',
            });
        }
        this.tabs.push(<Tab>{
            title: 'Usage',
            icon: 'global',
            iconTheme: 'outline',
            key: 'usage'
        });
        if (this.detailedWorkflow.permissions.writable) {
            this.tabs.push(<Tab>{
                title: 'Advanced',
                icon: 'setting',
                iconTheme: 'fill',
                key: 'advanced'
            });
        }
    }

    savePreview() {
        this._workflowCoreService.toggleAsCodeEditor({ open: false, save: true });
    }

    changeDirection() {
        this.direction = this.direction === 'LR' ? 'TB' : 'LR';
    }

    selectTab(tab: Tab): void {
        this.selectedTab = tab;
    }

    showAsCodeEditor() {
        this._workflowCoreService.toggleAsCodeEditor({ open: true, save: false });
    }

    groupManagement(event: PermissionEvent): void {
        switch (event.type) {
            case 'add':
                this.permFormLoading = true;
                this._store.dispatch(new actionsWorkflow.AddGroupInWorkflow({
                    projectKey: this.project.key,
                    workflowName: this.detailedWorkflow.name,
                    group: event.gp
                })).pipe(finalize(() => {
                    this.permFormLoading = false;
                    this._cd.markForCheck();
                })).subscribe(() => this._toast.success('', this._translate.instant('permission_added')));

                break;
            case 'update':
                this._store.dispatch(new actionsWorkflow.UpdateGroupInWorkflow({
                    projectKey: this.project.key,
                    workflowName: this.detailedWorkflow.name,
                    group: event.gp
                })).pipe(finalize(() => {
                    this.permFormLoading = false;
                    this.groups = cloneDeep(this.detailedWorkflow.groups);
                    this._cd.markForCheck();
                })).subscribe(() => this._toast.success('', this._translate.instant('permission_updated')));
                break;
            case 'delete':
                this._store.dispatch(new actionsWorkflow.DeleteGroupInWorkflow({
                    projectKey: this.project.key,
                    workflowName: this.detailedWorkflow.name,
                    group: event.gp
                })).pipe(finalize(() => {
                    this.permFormLoading = false;
                    this.groups = cloneDeep(this.detailedWorkflow.groups);
                    this._cd.markForCheck();
                })).subscribe(() => this._toast.success('', this._translate.instant('permission_deleted')));
                break;
        }
    }

    runWithParameter(): void {
        if (this.detailedWorkflow?.workflow_data?.node) {
            this._store.dispatch(new SelectWorkflowNode({
                node: this.detailedWorkflow.workflow_data.node
            })).pipe(first()).subscribe(() => {
                this._modalService.create({
                    nzWidth: '900px',
                    nzTitle: 'Run worklow',
                    nzContent: WorkflowNodeRunParamComponent,
                })
            });
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
        this._modalService.create({
            nzWidth: '900px',
            nzTitle: 'Save workflow as code',
            nzContent: AsCodeSaveModalComponent,
            nzData: {
                dataToSave: this.detailedWorkflow,
                dataType: 'workflow',
                project: this.project,
                workflow: this.detailedWorkflow,
                name: this.detailedWorkflow.name,
            }
        });
    }
}
