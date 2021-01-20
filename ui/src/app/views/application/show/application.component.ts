import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, OnInit, ViewChild } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { Application } from 'app/model/application.model';
import { Environment } from 'app/model/environment.model';
import { Pipeline } from 'app/model/pipeline.model';
import { Project } from 'app/model/project.model';
import { AuthentifiedUser } from 'app/model/user.model';
import { Workflow } from 'app/model/workflow.model';
import { ApplicationStore } from 'app/service/application/application.store';
import { AsCodeSaveModalComponent } from 'app/shared/ascode/save-modal/ascode.save-modal.component';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { WarningModalComponent } from 'app/shared/modal/warning/warning.component';
import { ToastService } from 'app/shared/toast/ToastService';
import { VariableEvent } from 'app/shared/variable/variable.event.model';
import * as applicationsActions from 'app/store/applications.action';
import { CancelApplicationEdition, ClearCacheApplication } from 'app/store/applications.action';
import { ApplicationsState, ApplicationStateModel } from 'app/store/applications.state';
import { AuthenticationState } from 'app/store/authentication.state';
import { ProjectState, ProjectStateModel } from 'app/store/project.state';
import cloneDeep from 'lodash-es/cloneDeep';
import { Subscription } from 'rxjs';
import { filter, finalize } from 'rxjs/operators';

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

    // Selected tab
    selectedTab = 'home';

    @ViewChild('varWarning')
    private varWarningModal: WarningModalComponent;
    @ViewChild('updateEditMode')
    asCodeSaveModal: AsCodeSaveModalComponent;

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
        private _applicationStore: ApplicationStore,
        private _routeActivated: ActivatedRoute,
        private _router: Router,
        private _toast: ToastService,
        public _translate: TranslateService,
        private _store: Store,
        private _cd: ChangeDetectorRef
    ) {
        this.project = this._routeActivated.snapshot.data['project'];

        this.workflowName = this._routeActivated.snapshot.queryParams['workflow'];
        this.workflowNum = this._routeActivated.snapshot.queryParams['run'];
        this.workflowNodeRun = this._routeActivated.snapshot.queryParams['node'];
        this.workflowPipeline = this._routeActivated.snapshot.queryParams['wpipeline'];

        this.projectSubscription = this._store.select(ProjectState)
            .subscribe(
                (projectState: ProjectStateModel) => {
                    this.project = projectState.project;
                    this._cd.markForCheck();
                }
            );

        this._routeParamsSub = this._routeActivated.params.subscribe(params => {
            let projectKey = params['key'];
            this.urlAppName = params['appName'];
            if (this.application && this.application.name !== this.urlAppName) {
                this.application = null;
            }
            if (projectKey && this.urlAppName) {
                this._store.dispatch(new applicationsActions.FetchApplication({ projectKey, applicationName: this.urlAppName }))
                    .subscribe(
                        () => {},
                        () => this._router.navigate(['/project', projectKey], { queryParams: { tab: 'applications' } }),
                        null
                    );
            }
        });

        this.storeSub = this._store.select(ApplicationsState.currentState())
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

                // Update recent application viewed
                this._applicationStore.updateRecentApplication(s.currentProjectKey, this.application);
                this._cd.markForCheck();
        }, () => {
                this._router.navigate(['/project', this.project.key], { queryParams: { tab: 'applications' } });
            });
    }

    ngOnInit() {
        this._queryParamsSub = this._routeActivated.queryParams.subscribe(params => {
            let tab = params['tab'];
            if (tab) {
                this.selectedTab = tab;
            }
            this._cd.markForCheck();
        });
    }

    showTab(tab: string): void {
        this._router.navigateByUrl('/project/' + this.project.key + '/application/' + this.application.name + '?tab=' + tab);
    }

    /**
     * Event on variable
     *
     * @param event
     */
    variableEvent(event: VariableEvent, skip?: boolean) {
        if (!skip && this.application.externalChange) {
            this.varWarningModal.show(event);
        } else {
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
    }

    cancelApplication(): void {
        if (this.editMode) {
            this._store.dispatch(new CancelApplicationEdition());
        }
    }

    saveEditMode(): void {
        if (this.editMode && this.application.from_repository && this.asCodeSaveModal) {
            // show modal to save as code
            this.asCodeSaveModal.show(this.application, 'application');
        }
    }

    ngOnDestroy(): void {
        this._store.dispatch(new ClearCacheApplication());
    }
}
