import { Component, OnInit, ViewChild } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import {
    AddApplicationVariable,
    DeleteApplicationVariable,
    FetchApplication,
    UpdateApplicationVariable
} from 'app/store/project/applications/applications.action';
import { ApplicationsState } from 'app/store/project/applications/applications.state';
import { cloneDeep } from 'lodash';
import { Subscription } from 'rxjs';
import { filter, finalize } from 'rxjs/operators';
import { Application } from '../../../model/application.model';
import { Environment } from '../../../model/environment.model';
import { PermissionValue } from '../../../model/permission.model';
import { Pipeline } from '../../../model/pipeline.model';
import { Project } from '../../../model/project.model';
import { User } from '../../../model/user.model';
import { Workflow } from '../../../model/workflow.model';
import { ApplicationStore } from '../../../service/application/application.store';
import { AuthentificationStore } from '../../../service/auth/authentification.store';
import { ProjectStore } from '../../../service/project/project.store';
import { AutoUnsubscribe } from '../../../shared/decorator/autoUnsubscribe';
import { WarningModalComponent } from '../../../shared/modal/warning/warning.component';
import { ToastService } from '../../../shared/toast/ToastService';
import { VariableEvent } from '../../../shared/variable/variable.event.model';
import { CDSWebWorker } from '../../../shared/worker/web.worker';

@Component({
    selector: 'app-application-show',
    templateUrl: './application.html',
    styleUrls: ['./application.scss']
})
@AutoUnsubscribe()
export class ApplicationShowComponent implements OnInit {

    // Flag to show the page or not
    public readyApp = false;
    public varFormLoading = false;
    public permFormLoading = false;
    public notifFormLoading = false;

    // Project & Application data
    project: Project;
    application: Application;

    // Subscription
    applicationSubscription: Subscription;
    projectSubscription: Subscription;
    workerSubscription: Subscription;
    _routeParamsSub: Subscription;
    _routeDataSub: Subscription;
    _queryParamsSub: Subscription;
    worker: CDSWebWorker;

    // Selected tab
    selectedTab = 'home';

    @ViewChild('varWarning')
    private varWarningModal: WarningModalComponent;

    // queryparam for breadcrum
    workflowName: string;
    workflowNum: string;
    workflowNodeRun: string;
    workflowPipeline: string;

    pipelines: Array<Pipeline> = new Array<Pipeline>();
    workflows: Array<Workflow> = new Array<Workflow>();
    environments: Array<Environment> = new Array<Environment>();
    currentUser: User;
    usageCount = 0;
    perm = PermissionValue;

    constructor(
        private _applicationStore: ApplicationStore,
        private _route: ActivatedRoute,
        private _router: Router,
        private _authStore: AuthentificationStore,
        private _toast: ToastService,
        public _translate: TranslateService,
        private _projectStore: ProjectStore,
        private store: Store
    ) {
        this.currentUser = this._authStore.getUser();
        // Update data if route change
        this._routeDataSub = this._route.data.subscribe(datas => {
            this.project = datas['project'];
        });

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
                this.store.dispatch(new FetchApplication({ projectKey: key, applicationName: appName }))
                    .subscribe(
                        null,
                        () => this._router.navigate(['/project', key], { queryParams: { tab: 'applications' } })
                    );

                if (this.application && this.application.name !== appName) {
                    this.application = null;
                }
                if (!this.application) {
                    this.applicationSubscription = this.store.select(ApplicationsState.selectApplication(key, appName))
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

                        }, () => {
                            this._router.navigate(['/project', key], { queryParams: { tab: 'applications' } });
                        })
                }
            }
        });
    }

    ngOnInit() {
        this._queryParamsSub = this._route.queryParams.subscribe(params => {
            let tab = params['tab'];
            if (tab) {
                this.selectedTab = tab;
            }
        });

        this.projectSubscription = this._projectStore.getProjects(this.project.key)
            .subscribe((proj) => {
                if (!this.project || !proj || !proj.get(this.project.key)) {
                    return;
                }
                this.project = proj.get(this.project.key);
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
                    this.store.dispatch(new AddApplicationVariable({
                        projectKey: this.project.key,
                        applicationName: this.application.name,
                        variable: event.variable
                    })).pipe(finalize(() => {
                        event.variable.updating = false;
                        this.varFormLoading = false;
                    })).subscribe(() => this._toast.success('', this._translate.instant('variable_added')));
                    break;
                case 'update':
                    this.store.dispatch(new UpdateApplicationVariable({
                        projectKey: this.project.key,
                        applicationName: this.application.name,
                        variableName: event.variable.name,
                        variable: event.variable
                    })).pipe(finalize(() => event.variable.updating = false))
                        .subscribe(() => this._toast.success('', this._translate.instant('variable_updated')));
                    break;
                case 'delete':
                    this.store.dispatch(new DeleteApplicationVariable({
                        projectKey: this.project.key,
                        applicationName: this.application.name,
                        variable: event.variable
                    })).subscribe(() => this._toast.success('', this._translate.instant('variable_deleted')));
                    break;
            }
        }
    }
}
