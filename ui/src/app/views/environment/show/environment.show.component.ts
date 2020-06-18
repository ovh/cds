import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { Environment } from 'app/model/environment.model';
import { Pipeline } from 'app/model/pipeline.model';
import { Project } from 'app/model/project.model';
import { AuthentifiedUser } from 'app/model/user.model';
import { Workflow } from 'app/model/workflow.model';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { ToastService } from 'app/shared/toast/ToastService';
import { VariableEvent } from 'app/shared/variable/variable.event.model';
import { CDSWebWorker } from 'app/shared/worker/web.worker';
import { AuthenticationState } from 'app/store/authentication.state';
import { CleanEnvironmentState } from 'app/store/environment.action';
import * as envActions from 'app/store/environment.action';
import { EnvironmentState, EnvironmentStateModel } from 'app/store/environment.state';
import { ProjectState, ProjectStateModel } from 'app/store/project.state';
import { cloneDeep } from 'lodash-es';
import { Subscription } from 'rxjs';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-environment-show',
    templateUrl: './environment.show.html',
    styleUrls: ['./environment.show.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class EnvironmentShowComponent implements OnInit, OnDestroy {

    // Flag to show the page or not
    public readyEnv = false;
    public varFormLoading = false;
    public permFormLoading = false;

    // Project & Application data
    project: Project;
    environment: Environment;

    // Subscription
    environmentSubscription: Subscription;
    projectSubscription: Subscription;
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

    workflows: Array<Workflow> = new Array<Workflow>();
    currentUser: AuthentifiedUser;
    usageCount = 0;

    constructor(
        private _route: ActivatedRoute,
        private _router: Router,
        private _toast: ToastService,
        public _translate: TranslateService,
        private _store: Store,
        private _cd: ChangeDetectorRef
    ) {
        this.currentUser = this._store.selectSnapshot(AuthenticationState.user);
        this.project = this._route.snapshot.data['project'];
        this.projectSubscription = this._store.select(ProjectState)// Update data if route change
            .subscribe((projectState: ProjectStateModel) => this.project = projectState.project);
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
            let envName = params['envName'];
            if (key && envName) {
                this._store.dispatch(new envActions.FetchEnvironment({ projectKey: key, envName }))
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

                    this.environmentSubscription = this._store.select(EnvironmentState.currentState())
                        .subscribe((s: EnvironmentStateModel) => {
                            if (!s.environment) {
                                return;
                            }
                            this.readyEnv = true;
                            this.environment = cloneDeep(s.environment);
                            if (this.environment.usage) {
                                this.workflows = this.environment.usage.workflows || [];
                                this.usageCount = this.workflows.length;
                            }
                            this._cd.markForCheck();
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

    ngOnDestroy() {
        this._store.dispatch(new CleanEnvironmentState())
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
                this._store.dispatch(new envActions.AddEnvironmentVariable({
                    projectKey: this.project.key,
                    environmentName: this.environment.name,
                    variable: event.variable
                })).pipe(finalize(() => {
                    this.varFormLoading = false;
                    this._cd.markForCheck();
                }))
                    .subscribe(() => this._toast.success('', this._translate.instant('variable_added')));
                break;
            case 'update':
                this._store.dispatch(new envActions.UpdateEnvironmentVariable({
                    projectKey: this.project.key,
                    environmentName: this.environment.name,
                    variableName: event.variable.name,
                    changes: event.variable
                })).pipe(finalize(() => {
                    event.variable.updating = false;
                    this._cd.markForCheck();
                }))
                    .subscribe(() => this._toast.success('', this._translate.instant('variable_updated')));
                break;
            case 'delete':
                this._store.dispatch(new envActions.DeleteEnvironmentVariable({
                    projectKey: this.project.key,
                    environmentName: this.environment.name,
                    variable: event.variable
                })).pipe(finalize(() => {
                    event.variable.updating = false;
                    this._cd.markForCheck();
                }))
                    .subscribe(() => this._toast.success('', this._translate.instant('variable_deleted')));
                break;
        }
    }
}
