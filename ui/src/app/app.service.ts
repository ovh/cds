import { Injectable } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { RetentionDryRunEvent } from 'app/model/purge.model';
import { WorkflowNodeRun, WorkflowRun } from 'app/model/workflow.run.model';
import { AsCodeEvent } from 'app/store/ascode.action';
import { UpdateMaintenance } from 'app/store/cds.action';
import { concatMap, first } from 'rxjs/operators';
import { Event, EventType } from './model/event.model';
import { LoadOpts } from './model/project.model';
import { RouterService } from './service/services.module';
import { WorkflowRunService } from './service/workflow/run/workflow.run.service';
import { ToastService } from './shared/toast/ToastService';
import {
    ClearCacheApplication,
    ExternalChangeApplication,
    ResyncApplication
} from './store/applications.action';
import { ApplicationsState } from './store/applications.state';
import { AuthenticationState } from './store/authentication.state';
import { AddEvent } from './store/event.action';
import { DeleteFromCachePipeline, ExternalChangePipeline, ResyncPipeline } from './store/pipelines.action';
import { PipelinesState } from './store/pipelines.state';
import * as projectActions from './store/project.action';
import { ProjectState } from './store/project.state';
import {
    ComputeRetentionDryRunEvent,
    ExternalChangeWorkflow,
    GetWorkflow,
    GetWorkflowNodeRun,
    GetWorkflowRun,
    RemoveWorkflowRunFromList,
    UpdateWorkflowRunList
} from './store/workflow.action';
import { WorkflowState } from './store/workflow.state';
import { AnalysisEvent, AnalysisService } from "./service/analysis/analysis.service";
import { lastValueFrom } from 'rxjs';

@Injectable()
export class AppService {

    // Information about current route
    routeParams: {};


    constructor(
        private _routerService: RouterService,
        private _routeActivated: ActivatedRoute,
        private _router: Router,
        private _translate: TranslateService,
        private _toast: ToastService,
        private _workflowRunService: WorkflowRunService,
        private _store: Store,
        private _analysisService: AnalysisService
    ) {
        this.routeParams = this._routerService.getRouteParams({}, this._routeActivated);
    }

    updateRoute(params: {}) {
        this.routeParams = params;
    }

    async manageEvent(event: Event): Promise<void> {
        if (!event || !event.type_event) {
            return;
        }
        await this._store.dispatch(new AddEvent(event)).toPromise();

        if (event.type_event.indexOf(EventType.MAINTENANCE) === 0) {
            await this._store.dispatch(new UpdateMaintenance(event.payload['enable'])).toPromise();
            return;
        }
        if (event.type_event.indexOf(EventType.ASCODE) === 0) {
            if (event.username === this._store.selectSnapshot(AuthenticationState.summary).user.username) {
                await this._store.dispatch(new AsCodeEvent(event.payload['as_code_event'])).toPromise();
            }
            return;
        }
        if (event.type_event.indexOf(EventType.WORKFLOW_RETENTION_DRYRUN) === 0) {
            if (this.routeParams['key'] && this.routeParams['key'] === event.project_key
                && this.routeParams['workflowName'] === event.workflow_name) {
                let retentionEvent = <RetentionDryRunEvent>event.payload;
                if (retentionEvent.status === 'ERROR') {
                    this._toast.error('', retentionEvent.error);
                }
                await this._store.dispatch(new ComputeRetentionDryRunEvent({
                    projectKey: event.project_key,
                    workflowName: event.workflow_name,
                    event: retentionEvent
                })).toPromise();
            }
            return;
        }
        switch (event.type_event) {
            case EventType.PROJECT_REPOSITORY_ANALYSE:
                let aEvent = new AnalysisEvent(event?.payload['vcs_id'], event?.payload['repository_id'], event?.payload['analysis_id'], event?.payload['status']);
                this._analysisService.sendEvent(aEvent);
                return;
        }

        if (event.type_event.indexOf(EventType.PROJECT_PREFIX) === 0 || event.type_event.indexOf(EventType.ENVIRONMENT_PREFIX) === 0 ||
            event.type_event === EventType.APPLICATION_ADD || event.type_event === EventType.APPLICATION_UPDATE ||
            event.type_event === EventType.APPLICATION_DELETE ||
            event.type_event === EventType.PIPELINE_ADD || event.type_event === EventType.PIPELINE_UPDATE ||
            event.type_event === EventType.PIPELINE_DELETE || event.type_event.indexOf(EventType.PIPELINE_PARAMETER_PREFIX) === 0 ||
            event.type_event === EventType.PIPELINE_ADD || event.type_event === EventType.PIPELINE_UPDATE ||
            event.type_event === EventType.PIPELINE_DELETE ||
            event.type_event === EventType.WORKFLOW_ADD || event.type_event === EventType.WORKFLOW_UPDATE ||
            event.type_event === EventType.WORKFLOW_DELETE) {
            await this.updateProjectCache(event);
        }
        if (event.type_event.indexOf(EventType.APPLICATION_PREFIX) === 0) {
            await this.updateApplicationCache(event);
        } else if (event.type_event.indexOf(EventType.PIPELINE_PREFIX) === 0) {
            await this.updatePipelineCache(event);
        } else if (event.type_event.indexOf(EventType.WORKFLOW_PREFIX) === 0) {
            await this.updateWorkflowCache(event);
        } else if (event.type_event.indexOf(EventType.RUN_WORKFLOW_PREFIX) === 0) {
            await this.updateWorkflowRunCache(event);
        }
    }

    async updateProjectCache(event: Event): Promise<void> {
        if (!event || !event.type_event) {
            return;
        }
        let p = this._store.selectSnapshot(ProjectState.projectSnapshot);
        if (p && p.key === event.project_key) {
            let projectInCache = p;
            // If working on project or sub resources
            if (this.routeParams['key'] && this.routeParams['key'] === projectInCache.key) {
                // if modification from another user, display a notification
                if (event.username !== this._store.selectSnapshot(AuthenticationState.summary).user.username) {
                    await this._store.dispatch(new projectActions.ExternalChangeProject({ projectKey: projectInCache.key })).toPromise();
                    this._toast.info('', this._translate.instant('warning_project', { username: event.username }));
                    return;
                }
            } else {
                // If no working on current project, remove from cache
                await lastValueFrom(this._store.dispatch(new projectActions.DeleteProjectFromCache()));
                return;
            }

            if (event.type_event === EventType.PROJECT_DELETE) {
                await lastValueFrom(this._store.dispatch(new projectActions.DeleteProjectFromCache()));
                return;
            }

            let opts = [];
            if (event.type_event.indexOf(EventType.PROJECT_VARIABLE_PREFIX) === 0) {
                opts.push(new LoadOpts('withVariables', 'variables'));
            } else if (event.type_event.indexOf(EventType.PROJECT_PERMISSION_PREFIX) === 0) {
                opts.push(new LoadOpts('withGroups', 'groups'));
            } else if (event.type_event.indexOf(EventType.PROJECT_KEY_PREFIX) === 0) {
                opts.push(new LoadOpts('withKeys', 'keys'));
            } else if (event.type_event.indexOf(EventType.APPLICATION_PREFIX) === 0) {
                opts.push(new LoadOpts('withApplicationNames', 'application_names'));
            } else if (event.type_event.indexOf(EventType.PIPELINE_PREFIX) === 0) {
                opts.push(new LoadOpts('withPipelineNames', 'pipeline_names'));
            } else if (event.type_event.indexOf(EventType.ENVIRONMENT_PREFIX) === 0) {
                opts.push(new LoadOpts('withEnvironmentNames', 'environment_names'));
            } else if (event.type_event.indexOf(EventType.WORKFLOW_PREFIX) === 0) {
                opts.push(new LoadOpts('withWorkflowNames', 'workflow_names'));
                opts.push(new LoadOpts('withLabels', 'labels'));
            }

            if (event.type_event.indexOf('Variable') === -1 && event.type_event.indexOf('Parameter') === -1
                && event.type_event.indexOf(EventType.ENVIRONMENT_PREFIX) === -1) {
                await this._store.dispatch(new projectActions.FetchProject({ projectKey: projectInCache.key, opts })).toPromise();
            }
        }
    }

    async updateApplicationCache(event: Event): Promise<void> {
        if (!event || !event.type_event) {
            return;
        }
        const payload = { projectKey: event.project_key, applicationName: event.application_name };

        let appState = this._store.selectSnapshot(ApplicationsState.current);
        if (!appState.application ||
            !(appState.application.name === event.application_name &&
                appState.currentProjectKey === event.project_key)) {
            return;
        }

        if (event.type_event === EventType.APPLICATION_DELETE) {
            // If user is on an application that has been deleted by an other user
            if (this.routeParams['key'] && this.routeParams['key'] === event.project_key &&
                this.routeParams['appName'] && this.routeParams['appName'] === event.application_name &&
                event.username !== this._store.selectSnapshot(AuthenticationState.summary).user.username) {
                this._toast.info('', this._translate.instant('application_deleted_by',
                    { appName: this.routeParams['appName'], username: event.username }));
                this._router.navigate(['/project'], this.routeParams['key']);
            }
            await this._store.dispatch(new ClearCacheApplication()).toPromise();
            await this._store.dispatch(new projectActions.DeleteApplicationInProject({
                applicationName: event.application_name
            })).toPromise();
            return;
        }

        // If working on the application
        if (this.routeParams['key'] && this.routeParams['key'] === event.project_key
            && this.routeParams['appName'] === event.application_name) {
            // modification by another user
            if (event.username !== this._store.selectSnapshot(AuthenticationState.summary).user.username) {
                await this._store.dispatch(new ExternalChangeApplication(payload)).toPromise();
                this._toast.info('', this._translate.instant('warning_application', { username: event.username }));
                return;
            }
        } else {
            await this._store.dispatch(new ClearCacheApplication()).toPromise();
            return;
        }

        if (event.type_event.indexOf('Variable') === -1) {
            await this._store.dispatch(new ResyncApplication(payload)).toPromise();
        }
    }

    async updatePipelineCache(event: Event): Promise<void> {
        if (!event || !event.type_event) {
            return;
        }

        let pips = this._store.selectSnapshot(PipelinesState.current);
        if (!pips || !pips.pipeline || pips.pipeline.name !== event.pipeline_name || pips.currentProjectKey !== event.project_key) {
            return;
        }

        if (event.type_event === EventType.PIPELINE_DELETE) {
            await this._store.dispatch(new DeleteFromCachePipeline({
                projectKey: event.project_key,
                pipelineName: event.pipeline_name
            })).toPromise();
            await this._store.dispatch(new projectActions.DeletePipelineInProject({ pipelineName: event.pipeline_name })).toPromise();
            return;
        }

        // update pipeline
        if (this.routeParams['key'] && this.routeParams['key'] === event.project_key
            && this.routeParams['pipName'] === event.pipeline_name) {
            if (event.username !== this._store.selectSnapshot(AuthenticationState.summary).user.username) {
                await this._store.dispatch(new ExternalChangePipeline({
                    projectKey: event.project_key,
                    pipelineName: event.pipeline_name
                })).toPromise();
                this._toast.info('', this._translate.instant('warning_pipeline', { username: event.username }));
                return;
            }
        } else {
            await this._store.dispatch(new DeleteFromCachePipeline({
                projectKey: event.project_key,
                pipelineName: event.pipeline_name
            })).toPromise();
            return;
        }

        await this._store.dispatch(new ResyncPipeline({
            projectKey: event.project_key,
            pipelineName: event.pipeline_name
        })).toPromise();
    }

    async updateWorkflowCache(event: Event): Promise<void> {
        if (!event || !event.type_event) {
            return;
        }
        let wf = this._store.selectSnapshot(WorkflowState.current);
        if (wf != null && wf.workflow && (wf.projectKey !== event.project_key || wf.workflow.name !== event.workflow_name)) {
            if (event.type_event === EventType.WORKFLOW_DELETE) {
                await this._store.dispatch(new projectActions.DeleteWorkflowInProject({ workflowName: event.workflow_name })).toPromise();
                return;
            }

            // update workflow
            if (this.routeParams['key'] && this.routeParams['key'] === event.project_key
                && this.routeParams['workflowName'] === event.workflow_name) {
                if (event.username !== this._store.selectSnapshot(AuthenticationState.summary).user.username) {
                    await this._store.dispatch(new ExternalChangeWorkflow({
                        projectKey: event.project_key,
                        workflowName: event.workflow_name
                    })).toPromise();
                    this._toast.info('', this._translate.instant('warning_workflow', { username: event.username }));
                    return;
                }
            } else {
                return;
            }

            await this._store.dispatch(new GetWorkflow({
                projectKey: event.project_key,
                workflowName: event.workflow_name
            })).toPromise();
        }
    }

    async updateWorkflowRunCache(event: Event): Promise<void> {
        if (!event || !event.type_event) {
            return;
        }
        if (this.routeParams['key'] !== event.project_key || this.routeParams['workflowName'] !== event.workflow_name) {
            return;
        }
        switch (event.type_event) {
            case EventType.RUN_WORKFLOW_PREFIX:
                if (event.payload['to_delete']) {
                    await this._store.dispatch(new RemoveWorkflowRunFromList({
                        projectKey: event.project_key,
                        workflowName: event.workflow_name,
                        num: event.workflow_run_num
                    })).toPromise();

                    if (this.routeParams['number'] === event.workflow_run_num.toString()) {
                        this._toast.info('', 'This run has just been deleted');
                        this._router.navigate(['/project', this.routeParams['key'], 'workflow', event.workflow_name]);
                    }
                    return;
                }
                if (this.routeParams['number'] === event.workflow_run_num.toString()) {
                    // if same run number , then update
                    await this._store.dispatch(new GetWorkflowRun({
                        projectKey: event.project_key,
                        workflowName: event.workflow_name,
                        num: event.workflow_run_num
                    })).toPromise();
                } else {
                    await this._workflowRunService
                        .getWorkflowRun(event.project_key, event.workflow_name, event.workflow_run_num)
                        .pipe(
                            first(),
                            concatMap(wrkRun => this._store.dispatch(new UpdateWorkflowRunList({ workflowRun: wrkRun })))
                        ).toPromise();
                }
                break;
            case EventType.RUN_WORKFLOW_NODE:
                // Refresh node run if user is listening on it
                const wnr = this._store.selectSnapshot<WorkflowNodeRun>((state) => state.workflow.workflowNodeRun);
                let wnrEvent = <WorkflowNodeRun>event.payload;
                if (wnr && wnr.id === wnrEvent.id) {
                    await this._store.dispatch(new GetWorkflowNodeRun({
                        projectKey: event.project_key,
                        workflowName: event.workflow_name,
                        num: event.workflow_run_num,
                        nodeRunID: wnr.id
                    })).toPromise();
                }

                // Refresh workflow run if user is listening on it
                const wr = this._store.selectSnapshot<WorkflowRun>((state) => state.workflow.workflowRun);
                if (wr && wr.num === event.workflow_run_num) {
                    await this._store.dispatch(new GetWorkflowRun({
                        projectKey: event.project_key,
                        workflowName: event.workflow_name,
                        num: event.workflow_run_num
                    })).toPromise();
                }
                break;
        }
    }
}
