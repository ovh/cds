import { Component, OnInit, ViewChild } from '@angular/core';
import { ActivatedRoute, Params, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { AddPipelineParameter, DeletePipelineParameter, FetchPipeline, UpdatePipelineParameter } from 'app/store/pipelines.action';
import { PipelinesState } from 'app/store/pipelines.state';
import { ProjectState, ProjectStateModel } from 'app/store/project.state';
import cloneDeep from 'lodash-es/cloneDeep';
import { Subscription } from 'rxjs';
import { filter, finalize, first } from 'rxjs/operators';
import { Application } from '../../../model/application.model';
import { Environment } from '../../../model/environment.model';
import { AllKeys } from '../../../model/keys.model';
import { Pipeline } from '../../../model/pipeline.model';
import { Project } from '../../../model/project.model';
import { User } from '../../../model/user.model';
import { Workflow } from '../../../model/workflow.model';
import { AuthentificationStore } from '../../../service/auth/authentification.store';
import { KeyService } from '../../../service/keys/keys.service';
import { PipelineCoreService } from '../../../service/pipeline/pipeline.core.service';
import { AutoUnsubscribe } from '../../../shared/decorator/autoUnsubscribe';
import { WarningModalComponent } from '../../../shared/modal/warning/warning.component';
import { ParameterEvent } from '../../../shared/parameter/parameter.event.model';
import { ToastService } from '../../../shared/toast/ToastService';

@Component({
    selector: 'app-pipeline-show',
    templateUrl: './pipeline.show.html',
    styleUrls: ['./pipeline.show.scss']
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
    currentUser: User;
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
    @ViewChild('paramWarning')
    parameterModalWarning: WarningModalComponent;

    keys: AllKeys;
    asCodeEditorOpen: boolean;

    // Selected tab
    selectedTab = 'pipeline';

    constructor(
        private store: Store,
        private _routeActivated: ActivatedRoute,
        private _router: Router,
        private _toast: ToastService,
        public _translate: TranslateService,
        private _authentificationStore: AuthentificationStore,
        private _keyService: KeyService,
        private _pipCoreService: PipelineCoreService
    ) {
        this.currentUser = this._authentificationStore.getUser();
        this.project = this._routeActivated.snapshot.data['project'];
        this.application = this._routeActivated.snapshot.data['application'];
        this.workflowName = this._routeActivated.snapshot.queryParams['workflow'];

        this.buildNumber = this.getQueryParam('buildNumber');
        this.version = this.getQueryParam('version');
        this.envName = this.getQueryParam('envName');
        this.branch = this.getQueryParam('branch');
        this.remote = this.getQueryParam('remote');

        this.projectSubscription = this.store.select(ProjectState)
            .subscribe((projectState: ProjectStateModel) => this.project = projectState.project);


        this.asCodeEditorSubscription = this._pipCoreService.getAsCodeEditor()
            .subscribe((state) => {
                if (state != null) {
                    this.asCodeEditorOpen = state.open;
                }
                if (state != null && !state.save && !state.open && this.pipeline) {
                    let pipName = this.pipeline.name;
                    this.refreshDatas(this.project.key, pipName);
                }
            });
    }

    refreshDatas(key: string, pipName: string) {
        this.store.dispatch(new FetchPipeline({
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
        });

        this._routeActivated.queryParams.subscribe(params => {
            this.queryParams = params;
            let tab = params['tab'];
            if (tab) {
                this.selectedTab = tab;
            }
        });

        this._keyService.getAllKeys(this.project.key).pipe(first()).subscribe(k => {
            this.keys = k;
        });
    }

    refreshListener() {
        if (this.pipelineSubscriber) {
            this.pipelineSubscriber.unsubscribe();
        }

        this.pipelineSubscriber = this.store.select(PipelinesState.selectPipeline(this.projectKey, this.pipName))
            .pipe(
                filter((pip) => pip != null)
            )
            .subscribe((pip) => {
                this.pipeline = cloneDeep(pip);
                if (pip.usage) {
                    this.applications = pip.usage.applications || [];
                    this.workflows = pip.usage.workflows || [];
                    this.environments = pip.usage.environments || [];
                }

                this.usageCount = this.applications.length + this.environments.length + this.workflows.length;
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
                    this.store.dispatch(new AddPipelineParameter({
                        projectKey: this.project.key,
                        pipelineName: this.pipeline.name,
                        parameter: event.parameter
                    })).pipe(finalize(() => this.paramFormLoading = false))
                        .subscribe(() => this._toast.success('', this._translate.instant('parameter_added')));
                    break;
                case 'update':
                    this.store.dispatch(new UpdatePipelineParameter({
                        projectKey: this.project.key,
                        pipelineName: this.pipeline.name,
                        parameterName: event.parameter.previousName || event.parameter.name,
                        parameter: event.parameter
                    })).subscribe(() => this._toast.success('', this._translate.instant('parameter_updated')));
                    break;
                case 'delete':
                    this.store.dispatch(new DeletePipelineParameter({
                        projectKey: this.project.key,
                        pipelineName: this.pipeline.name,
                        parameter: event.parameter
                    })).subscribe(() => this._toast.success('', this._translate.instant('parameter_deleted')));
                    break;
            }
        }
    }
}
