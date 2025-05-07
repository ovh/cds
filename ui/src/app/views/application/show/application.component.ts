import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { Application } from 'app/model/application.model';
import { Environment } from 'app/model/environment.model';
import { Pipeline } from 'app/model/pipeline.model';
import { Project } from 'app/model/project.model';
import { Workflow } from 'app/model/workflow.model';
import { AsCodeSaveModalComponent } from 'app/shared/ascode/save-modal/ascode.save-modal.component';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { ToastService } from 'app/shared/toast/ToastService';
import { VariableEvent } from 'app/shared/variable/variable.event.model';
import * as applicationsActions from 'app/store/applications.action';
import { CancelApplicationEdition, ClearCacheApplication } from 'app/store/applications.action';
import { ApplicationsState, ApplicationStateModel } from 'app/store/applications.state';
import { ProjectState } from 'app/store/project.state';
import cloneDeep from 'lodash-es/cloneDeep';
import { Subscription } from 'rxjs';
import { filter, finalize } from 'rxjs/operators';
import { Tab } from 'app/shared/tabs/tabs.component';
import { NzModalService } from 'ng-zorro-antd/modal';
import { RouterService } from 'app/service/services.module';

@Component({
    selector: 'app-application-show',
    templateUrl: './application.html',
    styleUrls: ['./application.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ApplicationShowComponent implements OnInit, OnDestroy {

    // Flag to show the page or not
    public readyApp = false;
    public varFormLoading = false;
    public permFormLoading = false;

    // Project & Application data
    urlAppName: string;
    project: Project;
    readOnlyApplication: Application;
    application: Application;
    editMode: boolean;
    readOnly: boolean;

    // Subscription
    projectSubscription: Subscription;
    _routeParamsSub: Subscription;
    _queryParamsSub: Subscription;

    // tabs
    tabs: Array<Tab>;
    selectedTab: Tab;

    // queryparam for breadcrum
    workflowName: string;
    workflowNum: string;
    workflowNodeRun: string;
    workflowPipeline: string;

    pipelines: Array<Pipeline> = new Array<Pipeline>();
    workflows: Array<Workflow> = new Array<Workflow>();
    environments: Array<Environment> = new Array<Environment>();
    usageCount = 0;

    storeSub: Subscription;

    constructor(
        private _routeActivated: ActivatedRoute,
        private _router: Router,
        private _toast: ToastService,
        public _translate: TranslateService,
        private _store: Store,
        private _cd: ChangeDetectorRef,
        private _modalService: NzModalService,
        private _routerService: RouterService
    ) {
        this.project = this._routeActivated.snapshot.data['project'];

        this.workflowName = this._routeActivated.snapshot.queryParams['workflow'];
        this.workflowNum = this._routeActivated.snapshot.queryParams['run'];
        this.workflowNodeRun = this._routeActivated.snapshot.queryParams['node'];
        this.workflowPipeline = this._routeActivated.snapshot.queryParams['wpipeline'];

        this.projectSubscription = this._store.select(ProjectState.projectSnapshot)
            .subscribe((p: Project) => {
                this.project = p;
                this._cd.markForCheck();
            });

        this._routeParamsSub = this._routeActivated.params.subscribe(_router => {
            const params = this._routerService.getRouteSnapshotParams({}, this._router.routerState.snapshot.root);

            let projectKey = params['key'];
            this.urlAppName = params['appName'];
            if (this.application && this.application.name !== this.urlAppName) {
                this.application = null;
            }
            if (projectKey && this.urlAppName) {
                this._store.dispatch(new applicationsActions.FetchApplication({
                    projectKey,
                    applicationName: this.urlAppName
                }))
                    .subscribe(
                        () => {
                        },
                        () => this._router.navigate(['/project', projectKey], { queryParams: { tab: 'applications' } }),
                        null
                    );
            }
        });

        this.storeSub = this._store.select(ApplicationsState.current)
            .pipe(filter((s: ApplicationStateModel) => s.application != null && s.application.name === this.urlAppName))
            .subscribe((s: ApplicationStateModel) => {
                this.readyApp = true;
                this.readOnly = (s.application.workflow_ascode_holder && !!s.application.workflow_ascode_holder.from_template) ||
                    !this.project.permissions.writable;
                this.editMode = s.editMode;
                this.readOnlyApplication = s.application;
                if (this.editMode) {
                    this.application = cloneDeep(s.editApplication);
                } else {
                    this.application = cloneDeep(s.application);
                }

                if (this.application.usage) {
                    this.workflows = this.application.usage.workflows || [];
                    this.environments = this.application.usage.environments || [];
                    this.pipelines = this.application.usage.pipelines || [];
                    this.usageCount = this.pipelines.length + this.environments.length + this.workflows.length;
                }
                this.initTabs();

                this._cd.markForCheck();
            }, () => {
                this._router.navigate(['/project', this.project.key], { queryParams: { tab: 'applications' } });
            });
    }

    ngOnInit() {
        this.initTabs();
        this._queryParamsSub = this._routeActivated.queryParams.subscribe(params => {
            let tab = params['tab'];
            if (tab) {
                let current_tab = this.tabs.find((t) => t.key === tab);
                if (current_tab) {
                    this.selectTab(current_tab);
                }
            }
            this._cd.markForCheck();
        });
    }

    initTabs() {
        let usageText = 'Usage';
        if (this.application) {
            usageText = 'Usage (' + this.usageCount + ')';
        }
        this.tabs = [<Tab>{
            title: 'Overview',
            key: 'home',
            icon: 'home',
            iconTheme: 'outline',
            default: true,
        }, <Tab>{
            title: 'Variables',
            key: 'variables',
            icon: 'font-colors',
            iconTheme: 'outline'
        }, <Tab>{
            title: usageText,
            icon: 'global',
            iconTheme: 'outline',
            key: 'usage'
        }, <Tab>{
            title: 'Keys',
            icon: 'lock',
            iconTheme: 'outline',
            key: 'keys'
        }]
        if (this.project?.permissions?.writable) {
            this.tabs.push(<Tab>{
                title: 'Advanced',
                icon: 'setting',
                iconTheme: 'fill',
                key: 'advanced'
            })
        }
    }

    selectTab(tab: Tab): void {
        this.selectedTab = tab;
    }

    /**
     * Event on variable
     *
     * @param event
     */
    variableEvent(event: VariableEvent) {
        event.variable.value = String(event.variable.value);
        switch (event.type) {
            case 'add':
                this.varFormLoading = true;
                this._store.dispatch(new applicationsActions.AddApplicationVariable({
                    projectKey: this.project.key,
                    applicationName: this.application.name,
                    variable: event.variable
                })).pipe(finalize(() => {
                    event.variable.updating = false;
                    this.varFormLoading = false;
                    this._cd.markForCheck();
                })).subscribe(() => {
                    if (this.editMode) {
                        this._toast.info('', this._translate.instant('application_ascode_updated'));
                    } else {
                        this._toast.success('', this._translate.instant('variable_added'));
                    }

                });
                break;
            case 'update':
                this._store.dispatch(new applicationsActions.UpdateApplicationVariable({
                    projectKey: this.project.key,
                    applicationName: this.application.name,
                    variableName: event.variable.name,
                    variable: event.variable
                })).pipe(finalize(() => {
                    event.variable.updating = false;
                    this._cd.markForCheck();
                }))
                    .subscribe(() => {
                        if (this.editMode) {
                            this._toast.info('', this._translate.instant('application_ascode_updated'));
                        } else {
                            this._toast.success('', this._translate.instant('variable_updated'));
                        }
                    });
                break;
            case 'delete':
                this._store.dispatch(new applicationsActions.DeleteApplicationVariable({
                    projectKey: this.project.key,
                    applicationName: this.application.name,
                    variable: event.variable
                })).pipe(finalize(() => this._cd.markForCheck()))
                    .subscribe(() => {
                        if (this.editMode) {
                            this._toast.info('', this._translate.instant('application_ascode_updated'));
                        } else {
                            this._toast.success('', this._translate.instant('variable_deleted'));
                        }
                    });
                break;

        }
    }

    cancelApplication(): void {
        if (this.editMode) {
            this._store.dispatch(new CancelApplicationEdition());
        }
    }

    saveEditMode(): void {
        if (this.editMode && this.application.from_repository) {
            this._modalService.create({
                nzWidth: '900px',
                nzTitle: 'Save application as code',
                nzContent: AsCodeSaveModalComponent,
                nzData: {
                    dataToSave: this.application,
                    dataType: 'application',
                    project: this.project,
                    workflow: this.application.workflow_ascode_holder,
                    name: this.readOnlyApplication.name,
                }
            });
        }
    }

    ngOnDestroy(): void {
        this._store.dispatch(new ClearCacheApplication());
    }
}
