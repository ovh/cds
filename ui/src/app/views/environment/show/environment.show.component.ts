import { Component, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { Environment } from 'app/model/environment.model';
import { PermissionValue } from 'app/model/permission.model';
import { Pipeline } from 'app/model/pipeline.model';
import { Project } from 'app/model/project.model';
import { User } from 'app/model/user.model';
import { Workflow } from 'app/model/workflow.model';
import { AuthentificationStore } from 'app/service/auth/authentification.store';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { ToastService } from 'app/shared/toast/ToastService';
import { VariableEvent } from 'app/shared/variable/variable.event.model';
import { CDSWebWorker } from 'app/shared/worker/web.worker';
import * as projectActions from 'app/store/project.action';
import { ProjectState, ProjectStateModel } from 'app/store/project.state';
import { cloneDeep } from 'lodash-es';
import { Subscription } from 'rxjs';
import { filter, finalize } from 'rxjs/operators';

@Component({
    selector: 'app-environment-show',
    templateUrl: './environment.show.html',
    styleUrls: ['./environment.show.scss']
})
@AutoUnsubscribe()
export class EnvironmentShowComponent implements OnInit {

    // Flag to show the page or not
    public readyEnv = false;
    public varFormLoading = false;
    public permFormLoading = false;
    public notifFormLoading = false;

    // Project & Application data
    project: Project;
    environment: Environment;

    // Subscription
    environmentSubscription: Subscription;
    projectSubscription: Subscription;
    workerSubscription: Subscription;
    _routeParamsSub: Subscription;
    _routeDataSub: Subscription;
    _queryParamsSub: Subscription;
    worker: CDSWebWorker;

    // Selected tab
    selectedTab = 'variables';

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
        private _route: ActivatedRoute,
        private _router: Router,
        private _authStore: AuthentificationStore,
        private _toast: ToastService,
        public _translate: TranslateService,
        private store: Store
    ) {
        this.currentUser = this._authStore.getUser();
        // Update data if route change
        this._routeDataSub = this._route.data.subscribe(datas => {
            this.project = datas['project'];
        });

        this.projectSubscription = this.store.select(ProjectState)
            .subscribe((projectState: ProjectStateModel) => this.project = projectState.project);

        if (this._route.snapshot && this._route.queryParams) {
            this.workflowName = this._route.snapshot.queryParams['workflow'];
            this.workflowNum = this._route.snapshot.queryParams['run'];
            this.workflowNodeRun = this._route.snapshot.queryParams['node'];
        }
        this.workflowPipeline = this._route.snapshot.queryParams['wpipeline'];

        this._routeParamsSub = this._route.params.subscribe(params => {
            let key = params['key'];
            let envName = params['envName'];
            if (key && envName) {
                this.store.dispatch(new projectActions.FetchEnvironmentInProject({ projectKey: key, envName }))
                    .subscribe(
                        null,
                        () => this._router.navigate(['/project', key], { queryParams: { tab: 'environments' } })
                    );

                if (this.environment && this.environment.name !== envName) {
                    this.environment = null;
                }
                if (!this.environment) {
                    if (this.environmentSubscription) {
                        this.environmentSubscription.unsubscribe();
                    }

                    this.environmentSubscription = this.store.select(ProjectState.selectEnvironment(envName))
                        .pipe(filter((env) => env != null))
                        .subscribe((env: Environment) => {
                            this.readyEnv = true;
                            this.environment = cloneDeep(env);
                            if (env.usage) {
                                this.workflows = env.usage.workflows || [];
                                this.environments = env.usage.environments || [];
                                this.pipelines = env.usage.pipelines || [];
                                this.usageCount = this.pipelines.length + this.environments.length + this.workflows.length;
                            }
                        }, () => {
                            this._router.navigate(['/project', key], { queryParams: { tab: 'environments' } });
                        });
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
    }

    showTab(tab: string): void {
        this._router.navigateByUrl('/project/' + this.project.key + '/environment/' + this.environment.name + '?tab=' + tab);
    }

    /**
     * Event on variable
     * @param event
     */
    variableEvent(event: VariableEvent): void {
        event.variable.value = String(event.variable.value);
        switch (event.type) {
            case 'add':
                this.varFormLoading = true;
                this.store.dispatch(new projectActions.AddEnvironmentVariableInProject({
                    projectKey: this.project.key,
                    environmentName: this.environment.name,
                    variable: event.variable
                })).pipe(finalize(() => this.varFormLoading = false))
                    .subscribe(() => this._toast.success('', this._translate.instant('variable_added')));
                break;
            case 'update':
                this.store.dispatch(new projectActions.UpdateEnvironmentVariableInProject({
                    projectKey: this.project.key,
                    environmentName: this.environment.name,
                    variableName: event.variable.name,
                    changes: event.variable
                })).pipe(finalize(() => event.variable.updating = false))
                    .subscribe(() => this._toast.success('', this._translate.instant('variable_updated')));
                break;
            case 'delete':
                this.store.dispatch(new projectActions.DeleteEnvironmentVariableInProject({
                    projectKey: this.project.key,
                    environmentName: this.environment.name,
                    variable: event.variable
                })).pipe(finalize(() => event.variable.updating = false))
                    .subscribe(() => this._toast.success('', this._translate.instant('variable_deleted')));
                break;
        }
    }
}
