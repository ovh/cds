import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, OnInit } from '@angular/core';
import { ActivatedRoute, Params, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { Application } from 'app/model/application.model';
import { Environment } from 'app/model/environment.model';
import { AllKeys } from 'app/model/keys.model';
import { Pipeline } from 'app/model/pipeline.model';
import { Project } from 'app/model/project.model';
import { Workflow } from 'app/model/workflow.model';
import { KeyService } from 'app/service/keys/keys.service';
import { PipelineCoreService } from 'app/service/pipeline/pipeline.core.service';
import { AsCodeSaveModalComponent } from 'app/shared/ascode/save-modal/ascode.save-modal.component';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { ParameterEvent } from 'app/shared/parameter/parameter.event.model';
import { ToastService } from 'app/shared/toast/ToastService';
import {
    AddPipelineParameter,
    CancelPipelineEdition,
    DeletePipelineParameter,
    FetchPipeline,
    UpdatePipelineParameter
} from 'app/store/pipelines.action';
import { PipelinesState, PipelinesStateModel } from 'app/store/pipelines.state';
import { ProjectState } from 'app/store/project.state';
import cloneDeep from 'lodash-es/cloneDeep';
import { Subscription } from 'rxjs';
import { filter, finalize, first } from 'rxjs/operators';
import { Tab } from 'app/shared/tabs/tabs.component';
import { NzModalService } from 'ng-zorro-antd/modal';
import { RouterService } from 'app/service/services.module';

@Component({
    selector: 'app-pipeline-show',
    templateUrl: './pipeline.show.html',
    styleUrls: ['./pipeline.show.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class PipelineShowComponent implements OnInit, OnDestroy {

    public permFormLoading = false;
    public paramFormLoading = false;

    project: Project;
    pipeline: Pipeline;
    editMode: boolean;
    pipelineSubscriber: Subscription;
    projectSubscription: Subscription;
    asCodeEditorSubscription: Subscription;
    appAsCode: Application;

    applications: Array<Application> = new Array<Application>();
    workflows: Array<Workflow> = new Array<Workflow>();
    environments: Array<Environment> = new Array<Environment>();
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

    keys: AllKeys;
    asCodeEditorOpen: boolean;
    nzTagColor: string = '';

    // tabs
    tabs: Array<Tab>;
    selectedTab: Tab;

    readOnly: boolean;

    constructor(
        private _store: Store,
        private _routeActivated: ActivatedRoute,
        private _router: Router,
        private _toast: ToastService,
        public _translate: TranslateService,
        private _keyService: KeyService,
        private _pipCoreService: PipelineCoreService,
        private _cd: ChangeDetectorRef,
        private _modalService: NzModalService,
        private _routerService: RouterService
    ) {
        this.project = this._routeActivated.snapshot.data['project'];
        this.application = this._routeActivated.snapshot.data['application'];
        this.workflowName = this._routeActivated.snapshot.queryParams['workflow'];

        this.buildNumber = this.getQueryParam('buildNumber');
        this.version = this.getQueryParam('version');
        this.envName = this.getQueryParam('envName');
        this.branch = this.getQueryParam('branch');
        this.remote = this.getQueryParam('remote');

        this.projectSubscription = this._store.select(ProjectState.projectSnapshot)
            .subscribe((p: Project) => {
                this.project = p;
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

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

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
        this.initTabs();
        this.projectKey = this._routeActivated.snapshot.params['key'];
        this.pipName = this._routeActivated.snapshot.params['pipName'];

        this._routeActivated.params.subscribe(_ => {
            const params = this._routerService.getRouteSnapshotParams({}, this._router.routerState.snapshot.root);
            
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
                let current_tab = this.tabs.find((t) => t.key === tab);
                if (current_tab) {
                    this.selectTab(current_tab);
                }
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

        this.pipelineSubscriber = this._store.select(PipelinesState.current)
            .pipe(
                filter((pip) => pip != null),
            )
            .subscribe((pip: PipelinesStateModel) => {
                if (!pip || !pip.pipeline || pip.pipeline.name !== this.pipName || pip.currentProjectKey !== this.projectKey) {
                    return;
                }
                this.editMode = pip.editMode;
                this.readOnly = (pip.pipeline.workflow_ascode_holder && !!pip.pipeline.workflow_ascode_holder.from_template) ||
                    !this.project.permissions.writable;
                if (pip.editMode) {
                    this.pipeline = cloneDeep(pip.editPipeline);
                    if (this.pipeline.workflow_ascode_holder) {
                        let rootAppId = this.pipeline.workflow_ascode_holder.workflow_data.node.context.application_id;
                        this.appAsCode = this.pipeline.workflow_ascode_holder.applications[rootAppId];
                    }
                } else {
                    this.pipeline = cloneDeep(pip.pipeline);
                }

                if (this.pipeline.from_repository && (!this.pipeline.ascode_events || this.pipeline.ascode_events.length === 0)) {
                    this.nzTagColor = 'green';
                } else if (this.pipeline.from_repository && this.pipeline?.ascode_events?.length > 0) {
                    this.nzTagColor = 'orange';
                } else {
                    this.nzTagColor = '';
                }

                if (this.pipeline.usage) {
                    this.applications = this.pipeline.usage.applications || [];
                    this.workflows = this.pipeline.usage.workflows || [];
                    this.environments = this.pipeline.usage.environments || [];
                }

                this.usageCount = this.applications.length + this.environments.length + this.workflows.length;
                this.initTabs();
                this._cd.markForCheck();
            }, () => {
                this._router.navigate(['/project', this.projectKey], { queryParams: { tab: 'pipelines' } });
            });
    }

    initTabs() {
        let usageText = 'Usage';
        if (this.pipeline) {
            usageText = 'Usage (' + this.usageCount + ')';
        }
        this.tabs = [<Tab>{
            title: 'Pipeline',
            key: 'pipeline',
            default: true,
            icon: 'apartment',
            iconTheme: 'outline'
        }, <Tab>{
            title: 'Parameters',
            key: 'parameters',
            icon: 'font-colors',
            iconTheme: 'outline'
        }, <Tab>{
            title: usageText,
            icon: 'global',
            iconTheme: 'outline',
            key: 'usage'
        }]
        if (!this.pipeline?.from_repository) {
            this.tabs.push(<Tab>{
                title: 'Audits',
                icon: 'history',
                iconTheme: 'outline',
                key: 'audits'
            })
        }
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

    parameterEvent(event: ParameterEvent): void {
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
                })).subscribe(() => {
                    if (this.editMode) {
                        this._toast.info('', this._translate.instant('pipeline_ascode_updated'));
                    } else {
                        this._toast.success('', this._translate.instant('parameter_added'));
                    }
                });
                break;
            case 'update':
                this._store.dispatch(new UpdatePipelineParameter({
                    projectKey: this.project.key,
                    pipelineName: this.pipeline.name,
                    parameterName: event.parameter.previousName || event.parameter.name,
                    parameter: event.parameter
                })).subscribe(() => {
                    if (this.editMode) {
                        this._toast.info('', this._translate.instant('pipeline_ascode_updated'));
                    } else {
                        this._toast.success('', this._translate.instant('parameter_updated'));
                    }
                });
                break;
            case 'delete':
                this._store.dispatch(new DeletePipelineParameter({
                    projectKey: this.project.key,
                    pipelineName: this.pipeline.name,
                    parameter: event.parameter
                })).subscribe(() => {
                    if (this.editMode) {
                        this._toast.info('', this._translate.instant('pipeline_ascode_updated'));
                    } else {
                        this._toast.success('', this._translate.instant('parameter_deleted'));
                    }
                });
                break;
        }
    }

    cancelPipeline(): void {
        if (this.editMode) {
            this._store.dispatch(new CancelPipelineEdition());
        }
    }

    saveEditMode(): void {
        if (this.editMode && this.pipeline.from_repository) {
            // show modal to save as code
            this._modalService.create({
                nzWidth: '900px',
                nzTitle: 'Save pipeline as code',
                nzContent: AsCodeSaveModalComponent,
                nzData: {
                    dataToSave: this.pipeline,
                    dataType: 'pipeline',
                    project: this.project,
                    workflow: this.pipeline.workflow_ascode_holder,
                    name: this.pipeline.name,
                }
            });
        }
    }
}
