import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnInit, ViewChild } from '@angular/core';
import { ActivatedRoute, Params, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { Application } from 'app/model/application.model';
import { Environment } from 'app/model/environment.model';
import { AllKeys } from 'app/model/keys.model';
import { Pipeline } from 'app/model/pipeline.model';
import { Project } from 'app/model/project.model';
import { AuthentifiedUser } from 'app/model/user.model';
import { Workflow } from 'app/model/workflow.model';
import { KeyService } from 'app/service/keys/keys.service';
import { PipelineCoreService } from 'app/service/pipeline/pipeline.core.service';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { WarningModalComponent } from 'app/shared/modal/warning/warning.component';
import { ParameterEvent } from 'app/shared/parameter/parameter.event.model';
import { ToastService } from 'app/shared/toast/ToastService';
import { AuthenticationState } from 'app/store/authentication.state';
import {
    AddPipelineParameter,
    DeletePipelineParameter,
    FetchPipeline,
    UpdatePipelineParameter
} from 'app/store/pipelines.action';
import { PipelinesState } from 'app/store/pipelines.state';
import { ProjectState, ProjectStateModel } from 'app/store/project.state';
import cloneDeep from 'lodash-es/cloneDeep';
import { Subscription } from 'rxjs';
import { filter, finalize, first } from 'rxjs/operators';

@Component({
    selector: 'app-pipeline-show',
    templateUrl: './pipeline.show.html',
    styleUrls: ['./pipeline.show.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class PipelineShowComponent implements OnInit {

    public permFormLoading = false;
    public paramFormLoading = false;

    project: Project;
    pipeline: Pipeline;
    pipelineSubscriber: Subscription;
    projectSubscription: Subscription;
    asCodeEditorSubscription: Subscription;

    applications: Array<Application> = new Array<Application>();
    workflows: Array<Workflow> = new Array<Workflow>();
    environments: Array<Environment> = new Array<Environment>();
    currentUser: AuthentifiedUser;
    usageCount = 0;

    // optional application data
    workflowName: string;
    application: Application;
    version: string;
    buildNumber: string;
    envName: string;
    branch: string;
    remote: string;
    projectKey: string;
    pipName: string;

    queryParams: Params;
    @ViewChild('paramWarning', { static: false })
    parameterModalWarning: WarningModalComponent;

    keys: AllKeys;
    asCodeEditorOpen: boolean;

    // Selected tab
    selectedTab = 'pipeline';

    constructor(
        private _store: Store,
        private _routeActivated: ActivatedRoute,
        private _router: Router,
        private _toast: ToastService,
        public _translate: TranslateService,
        private _keyService: KeyService,
        private _pipCoreService: PipelineCoreService,
        private _cd: ChangeDetectorRef
    ) {
        this.currentUser = this._store.selectSnapshot(AuthenticationState.user);
        this.project = this._routeActivated.snapshot.data['project'];
        this.application = this._routeActivated.snapshot.data['application'];
        this.workflowName = this._routeActivated.snapshot.queryParams['workflow'];

        this.buildNumber = this.getQueryParam('buildNumber');
        this.version = this.getQueryParam('version');
        this.envName = this.getQueryParam('envName');
        this.branch = this.getQueryParam('branch');
        this.remote = this.getQueryParam('remote');

        this.projectSubscription = this._store.select(ProjectState)
            .subscribe((projectState: ProjectStateModel) => {
                this.project = projectState.project;
                this._cd.markForCheck();
            });


        this.asCodeEditorSubscription = this._pipCoreService.getAsCodeEditor()
            .subscribe((state) => {
                if (state != null) {
                    this.asCodeEditorOpen = state.open;
                }
                if (state != null && !state.save && !state.open && this.pipeline) {
                    let pipName = this.pipeline.name;
                    this.refreshDatas(this.project.key, pipName);
                }
                this._cd.markForCheck();
            });
    }

    refreshDatas(key: string, pipName: string) {
        this._store.dispatch(new FetchPipeline({
            projectKey: key,
            pipelineName: pipName
        }));
    }

    getQueryParam(name: string): string {
        if (this._routeActivated.snapshot.queryParams[name]) {
            return this._routeActivated.snapshot.queryParams[name];
        }
    }

    ngOnInit() {
        this.projectKey = this._routeActivated.snapshot.params['key'];
        this.pipName = this._routeActivated.snapshot.params['pipName'];

        this._routeActivated.params.subscribe(params => {
            if (!this.pipeline || this.projectKey !== params['key'] || this.pipName !== params['pipName']) {
                this.projectKey = params['key'];
                this.pipName = params['pipName'];
                this.refreshListener();
            }

            this.projectKey = params['key'];
            this.pipName = params['pipName'];
            this.refreshDatas(this.projectKey, this.pipName);
            this._cd.markForCheck();
        });

        this._routeActivated.queryParams.subscribe(params => {
            this.queryParams = params;
            let tab = params['tab'];
            if (tab) {
                this.selectedTab = tab;
            }
            this._cd.markForCheck();
        });

        this._keyService.getAllKeys(this.project.key).pipe(
            first(),
            finalize(() => this._cd.markForCheck()))
            .subscribe(k => {
                this.keys = k;

            });
    }

    refreshListener() {
        if (this.pipelineSubscriber) {
            this.pipelineSubscriber.unsubscribe();
        }

        this.pipelineSubscriber = this._store.select(PipelinesState.selectPipeline(this.projectKey, this.pipName))
            .pipe(
                filter((pip) => pip != null),
            )
            .subscribe((pip) => {
                this.pipeline = cloneDeep(pip);
                if (pip.usage) {
                    this.applications = pip.usage.applications || [];
                    this.workflows = pip.usage.workflows || [];
                    this.environments = pip.usage.environments || [];
                }

                this.usageCount = this.applications.length + this.environments.length + this.workflows.length;
                this._cd.markForCheck();
            }, () => {
                this._router.navigate(['/project', this.projectKey], { queryParams: { tab: 'pipelines' } });
            });
    }

    showTab(tab: string): void {
        this._router.navigateByUrl('/project/' + this.project.key + '/pipeline/' + this.pipeline.name + '?tab=' + tab);
    }

    parameterEvent(event: ParameterEvent, skip?: boolean): void {
        if (!skip && this.pipeline.externalChange) {
            this.parameterModalWarning.show(event);
        } else {
            if (event.parameter) {
                event.parameter.value = String(event.parameter.value);
            }
            switch (event.type) {
                case 'add':
                    this.paramFormLoading = true;
                    this._store.dispatch(new AddPipelineParameter({
                        projectKey: this.project.key,
                        pipelineName: this.pipeline.name,
                        parameter: event.parameter
                    })).pipe(finalize(() => {
                        this.paramFormLoading = false;
                        this._cd.markForCheck();
                    }))
                        .subscribe(() => this._toast.success('', this._translate.instant('parameter_added')));
                    break;
                case 'update':
                    this._store.dispatch(new UpdatePipelineParameter({
                        projectKey: this.project.key,
                        pipelineName: this.pipeline.name,
                        parameterName: event.parameter.previousName || event.parameter.name,
                        parameter: event.parameter
                    })).subscribe(() => this._toast.success('', this._translate.instant('parameter_updated')));
                    break;
                case 'delete':
                    this._store.dispatch(new DeletePipelineParameter({
                        projectKey: this.project.key,
                        pipelineName: this.pipeline.name,
                        parameter: event.parameter
                    })).subscribe(() => this._toast.success('', this._translate.instant('parameter_deleted')));
                    break;
            }
        }
    }
}
