import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    OnDestroy,
    OnInit,
} from '@angular/core';
import { ActivatedRoute, NavigationStart, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Select, Store } from '@ngxs/store';
import { Project } from 'app/model/project.model';
import { Workflow } from 'app/model/workflow.model';
import { FeatureNames, FeatureService } from 'app/service/feature/feature.service';
import { WorkflowCoreService } from 'app/service/workflow/workflow.core.service';
import { AsCodeSaveModalComponent } from 'app/shared/ascode/save-modal/ascode.save-modal.component';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { ToastService } from 'app/shared/toast/ToastService';
import { AddFeatureResult, FeaturePayload } from 'app/store/feature.action';
import { ProjectState, ProjectStateModel } from 'app/store/project.state';
import {
    CleanWorkflowRun,
    CleanWorkflowState,
    GetWorkflow,
    SelectHook,
    UpdateFavoriteWorkflow
} from 'app/store/workflow.action';
import { WorkflowState } from 'app/store/workflow.state';
import { Observable, Subscription } from 'rxjs';
import { finalize } from 'rxjs/operators';
import { NzModalService } from 'ng-zorro-antd/modal';
import {
    WorkflowTemplateApplyModalComponent
} from 'app/shared/workflow-template/apply-modal/workflow-template.apply-modal.component';


@Component({
    selector: 'app-workflow',
    templateUrl: './workflow.html',
    styleUrls: ['./workflow.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowComponent implements OnInit, OnDestroy {

    project: Project;

    @Select(WorkflowState.getWorkflow()) workflow$: Observable<Workflow>;
    workflow: Workflow;
    workflowSubscription: Subscription;

    runNumber: number;

    projectSubscription: Subscription;
    qpRouteSubscription: Subscription;
    paramsRouteSubscription: Subscription;
    eventsRouteSubscription: Subscription;

    loading = true;
    loadingFav = false;

    asCodeEditorSubscription: Subscription;
    asCodeEditorOpen = false;

    selectedNodeID: number;
    selectedNodeRef: string;
    selectecHookRef: string;

    workflowV3Enabled: boolean;
    asCodeTagColor: string = '';
    templateTagColor: string = '';
    previewV3TagColor: string = '';

    constructor(
        private _activatedRoute: ActivatedRoute,
        private _router: Router,
        private _workflowCore: WorkflowCoreService,
        private _toast: ToastService,
        private _translate: TranslateService,
        private _store: Store,
        private _cd: ChangeDetectorRef,
        private _featureService: FeatureService,
        private _modalService: NzModalService
    ) { }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit(): void {
        this.projectSubscription = this._store.select(ProjectState).subscribe((projectState: ProjectStateModel) => {
            this.project = projectState.project;
            if (this.project && this.workflow && this.project.key !== this.workflow.project_key) {
                delete this.workflow;
            }
            this._cd.detectChanges();
            if (!this.project) {
                return;
            }

            let data = { project_key: this.project.key };
            this._featureService.isEnabled(FeatureNames.WorkflowRetentionPolicy, data).subscribe(f => {
                this._store.dispatch(new AddFeatureResult(<FeaturePayload>{
                    key: f.name,
                    result: {
                        paramString: JSON.stringify(data),
                        enabled: f.enabled,
                        exists: f.exists
                    }
                }));
            });
            this._featureService.isEnabled(FeatureNames.WorkflowRetentionMaxRuns, data).subscribe(f => {
                this._store.dispatch(new AddFeatureResult(<FeaturePayload>{
                    key: f.name,
                    result: {
                        paramString: JSON.stringify(data),
                        enabled: f.enabled,
                        exists: f.exists
                    }
                }));
            });
            this._featureService.isEnabled(FeatureNames.WorkflowV3, data).subscribe(f => {
                this.workflowV3Enabled = f.enabled;
            });
        });

        this.asCodeEditorSubscription = this._workflowCore.getAsCodeEditor()
            .subscribe((state) => {
                if (state != null) {
                    this.asCodeEditorOpen = state.open;
                    this._cd.markForCheck();
                }
            });

        this.qpRouteSubscription = this._activatedRoute.queryParams.subscribe(qps => {
            if (qps['node_id']) {
                this.selectedNodeID = Number(qps['node_id']);
                delete this.selectecHookRef;
            }
            if (qps['node_ref']) {
                this.selectedNodeRef = qps['node_ref'];
                delete this.selectecHookRef;
            }
            if (qps['hook_ref']) {
                this.selectecHookRef = qps['hook_ref'];
                delete this.selectedNodeRef;
                delete this.selectedNodeID;
            }
            this._cd.markForCheck();
        });

        this._store.dispatch(new CleanWorkflowState());
        this.workflowSubscription = this.workflow$.subscribe(w => {
            if (!w) {
                return;
            }
            this.workflow = w;
            if (this.selectecHookRef) {
                let h = Workflow.getHookByRef(this.selectecHookRef, this.workflow);
                if (h) {
                    this._store.dispatch(new SelectHook({ hook: h, node: this.workflow.workflow_data.node }));
                }
            }
            if (this.workflow && this.workflow.from_repository && (!this.workflow.as_code_events || this.workflow.as_code_events.length === 0)) {
                this.asCodeTagColor = 'green';
            } else if (this.workflow && this.workflow.as_code_events && this.workflow.as_code_events.length > 0) {
                this.asCodeTagColor = 'orange';
            } else if (this.workflow && !this.workflow.from_repository && (!this.workflow.as_code_events || this.workflow.as_code_events.length === 0)) {
                this.asCodeTagColor = '';
            }
            this._cd.markForCheck();
        });

        // Workflow subscription
        this.paramsRouteSubscription = this._activatedRoute.params.subscribe(params => {
            let projectKey = params['key'];
            let workflowName = params['workflowName'];

            if (projectKey && workflowName) {
                this.loading = true;
                this._store.dispatch(new GetWorkflow({ projectKey, workflowName }))
                    .pipe(finalize(() => this.loading = false))
                    .subscribe(null, () => this._router.navigate(['/project', projectKey]));
            }
        });

        // unselect all when returning on workflow main page
        this.eventsRouteSubscription = this._router.events.subscribe(e => {
            this.runNumber = this._activatedRoute.children[0].snapshot.params['number'];
            this._cd.markForCheck();

            if (e instanceof NavigationStart && this.workflow) {
                if (e.url.indexOf('/project/' + this.project.key + '/workflow/') === 0 && e.url.indexOf('/run/') === -1) {
                    this._store.dispatch(new CleanWorkflowRun({}));
                }
            }
        });
    }

    updateFav() {
        if (this.loading || !this.workflow) {
            return;
        }
        this.loadingFav = true;
        this._store.dispatch(new UpdateFavoriteWorkflow({
            projectKey: this.project.key,
            workflowName: this.workflow.name
        })).pipe(finalize(() => {
            this.loadingFav = false;
            this._cd.markForCheck()
        }))
            .subscribe(() => this._toast.success('', this._translate.instant('common_favorites_updated')));
    }

    showTemplateFrom(): void {
        this._modalService.create({
            nzTitle: 'Update workflow from template',
            nzWidth: '1100px',
            nzContent: WorkflowTemplateApplyModalComponent,
            nzComponentParams: {
                projectIn: this.project,
                workflowIn: this.workflow
            },
            nzFooter: null
        });
    }

    initTemplateFromWorkflow(): void {
        this._router.navigate(['settings', 'workflow-template', 'add'], {
            queryParams: {
                from: this.project.key + '/' + this.workflow.name,
            }
        });
    }

    openSaveAsCodeModal(): void {
        if (!this.project.vcs_servers) {
            this._toast.error('', this._translate.instant('project_vcs_no'));
            return;
        }
        if (!this.workflow.workflow_data || !this.workflow.workflow_data.node ||
            !this.workflow.workflow_data.node.context ||
            !this.workflow.workflow_data.node.context.application_id
        ) {
            this._toast.error('', this._translate.instant('common_no_application'));
            return;
        }
        let app = this.workflow.applications[this.workflow.workflow_data.node.context.application_id];
        if (!app || !app.repository_fullname) {
            this._toast.error('', this._translate.instant('application_repo_no'));
            return;
        }

        this._modalService.create({
            nzTitle: 'Migrate as code',
            nzWidth: '900px',
            nzContent: AsCodeSaveModalComponent,
            nzComponentParams: {
                dataToSave: null,
                dataType: 'workflow',
                project: this.project,
                workflow: this.workflow,
                name: this.workflow.name,
            }
        });
    }
}
