import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnInit, ViewChild } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import * as applicationsActions from 'app/store/applications.action';
import { ApplicationsState } from 'app/store/applications.state';
import { AuthenticationState } from 'app/store/authentication.state';
import { ProjectState, ProjectStateModel } from 'app/store/project.state';
import cloneDeep from 'lodash-es/cloneDeep';
import { Subscription } from 'rxjs';
import { filter, finalize } from 'rxjs/operators';
import { Application } from '../../../model/application.model';
import { Environment } from '../../../model/environment.model';
import { Pipeline } from '../../../model/pipeline.model';
import { Project } from '../../../model/project.model';
import { AuthentifiedUser } from '../../../model/user.model';
import { Workflow } from '../../../model/workflow.model';
import { ApplicationStore } from '../../../service/application/application.store';
import { AutoUnsubscribe } from '../../../shared/decorator/autoUnsubscribe';
import { WarningModalComponent } from '../../../shared/modal/warning/warning.component';
import { ToastService } from '../../../shared/toast/ToastService';
import { VariableEvent } from '../../../shared/variable/variable.event.model';
import { CDSWebWorker } from '../../../shared/worker/web.worker';

@Component({
    selector: 'app-application-show',
    templateUrl: './application.html',
    styleUrls: ['./application.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ApplicationShowComponent implements OnInit {

    // Flag to show the page or not
    public readyApp = false;
    public varFormLoading = false;
    public permFormLoading = false;

    // Project & Application data
    project: Project;
    application: Application;

    // Subscription
    applicationSubscription: Subscription;
    projectSubscription: Subscription;
    _routeParamsSub: Subscription;
    _routeDataSub: Subscription;
    _queryParamsSub: Subscription;
    worker: CDSWebWorker;

    // Selected tab
    selectedTab = 'home';

    @ViewChild('varWarning', { static: false })
    private varWarningModal: WarningModalComponent;

    // queryparam for breadcrum
    workflowName: string;
    workflowNum: string;
    workflowNodeRun: string;
    workflowPipeline: string;

    pipelines: Array<Pipeline> = new Array<Pipeline>();
    workflows: Array<Workflow> = new Array<Workflow>();
    environments: Array<Environment> = new Array<Environment>();
    currentUser: AuthentifiedUser;
    usageCount = 0;

    constructor(
        private _applicationStore: ApplicationStore,
        private _route: ActivatedRoute,
        private _router: Router,
        private _toast: ToastService,
        public _translate: TranslateService,
        private _store: Store,
        private _cd: ChangeDetectorRef
    ) {
        this.currentUser = this._store.selectSnapshot(AuthenticationState.user);
        // Update data if route change
        this._routeDataSub = this._route.data.subscribe(datas => {
            this.project = datas['project'];
        });

        this.projectSubscription = this._store.select(ProjectState)
            .subscribe((projectState: ProjectStateModel) => this.project = projectState.project);

        if (this._route.snapshot && this._route.queryParams) {
            this.workflowName = this._route.snapshot.queryParams['workflow'];
            this.workflowNum = this._route.snapshot.queryParams['run'];
            this.workflowNodeRun = this._route.snapshot.queryParams['node'];
        }
        this.workflowPipeline = this._route.snapshot.queryParams['wpipeline'];
        this._routeParamsSub = this._route.params.subscribe(params => {
            let key = params['key'];
            let appName = params['appName'];
            if (key && appName) {
                this._store.dispatch(new applicationsActions.FetchApplication({ projectKey: key, applicationName: appName }))
                    .subscribe(
                        null,
                        () => this._router.navigate(['/project', key], { queryParams: { tab: 'applications' } })
                    );

                if (this.application && this.application.name !== appName) {
                    this.application = null;
                }
                if (!this.application) {
                    if (this.applicationSubscription) {
                        this.applicationSubscription.unsubscribe();
                    }

                    this.applicationSubscription = this._store.select(ApplicationsState.selectApplication(key, appName))
                        .pipe(filter((app) => app != null))
                        .subscribe((app: Application) => {
                            this.readyApp = true;
                            // TODO: to delete when all CDS application will be in store. In fact we make a copy to break the read only rule
                            this.application = cloneDeep(app);
                            if (app.usage) {
                                this.workflows = app.usage.workflows || [];
                                this.environments = app.usage.environments || [];
                                this.pipelines = app.usage.pipelines || [];
                                this.usageCount = this.pipelines.length + this.environments.length + this.workflows.length;
                            }

                            // Update recent application viewed
                            this._applicationStore.updateRecentApplication(key, this.application);
                            this._cd.markForCheck();

                        }, () => {
                            this._router.navigate(['/project', key], { queryParams: { tab: 'applications' } });
                        })
                }
                this._cd.markForCheck();
            }
        });
    }

    ngOnInit() {
        this._queryParamsSub = this._route.queryParams.subscribe(params => {
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
                    })).subscribe(() => this._toast.success('', this._translate.instant('variable_added')));
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
                        .subscribe(() => this._toast.success('', this._translate.instant('variable_updated')));
                    break;
                case 'delete':
                    this._store.dispatch(new applicationsActions.DeleteApplicationVariable({
                        projectKey: this.project.key,
                        applicationName: this.application.name,
                        variable: event.variable
                    })).pipe(finalize(() => this._cd.markForCheck()))
                        .subscribe(() => this._toast.success('', this._translate.instant('variable_deleted')));
                    break;
            }
        }
    }
}
