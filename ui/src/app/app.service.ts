import { Injectable } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { WorkflowNodeRun, WorkflowRun } from 'app/model/workflow.run.model';
import { AsCodeEvent } from 'app/store/ascode.action';
import { UpdateMaintenance } from 'app/store/cds.action';
import cloneDeep from 'lodash-es/cloneDeep';
import { filter, first } from 'rxjs/operators';
import { Broadcast } from './model/broadcast.model';
import { Event, EventType } from './model/event.model';
import { LoadOpts } from './model/project.model';
import { TimelineFilter } from './model/timeline.model';
import { BroadcastStore } from './service/broadcast/broadcast.store';
import { NavbarService } from './service/navbar/navbar.service';
import { RouterService, TimelineStore } from './service/services.module';
import { WorkflowRunService } from './service/workflow/run/workflow.run.service';
import { ToastService } from './shared/toast/ToastService';
import {
    ClearCacheApplication,
    ExternalChangeApplication,
    ResyncApplication
} from './store/applications.action';
import { ApplicationsState, ApplicationStateModel } from './store/applications.state';
import { AuthenticationState } from './store/authentication.state';
import { DeleteFromCachePipeline, ExternalChangePipeline, ResyncPipeline } from './store/pipelines.action';
import { PipelinesState, PipelinesStateModel } from './store/pipelines.state';
import * as projectActions from './store/project.action';
import { ProjectState, ProjectStateModel } from './store/project.state';
import { ExternalChangeWorkflow, GetWorkflow, GetWorkflowNodeRun, GetWorkflowRun, UpdateWorkflowRunList } from './store/workflow.action';
import { WorkflowState } from './store/workflow.state';

@Injectable()
export class AppService {

    // Information about current route
    routeParams: {};

    filter: TimelineFilter;

    constructor(
        private _routerService: RouterService,
        private _routeActivated: ActivatedRoute,
        private _router: Router,
        private _translate: TranslateService,
        private _broadcastStore: BroadcastStore,
        private _timelineStore: TimelineStore,
        private _toast: ToastService,
        private _workflowRunService: WorkflowRunService,
        private _store: Store,
        private _navbarService: NavbarService,
    ) {
        this.routeParams = this._routerService.getRouteParams({}, this._routeActivated);
    }

    initFilter(filterTimeline: TimelineFilter) {
        this.filter = cloneDeep(filterTimeline);
    }

    updateRoute(params: {}) {
        this.routeParams = params;
    }

    manageEvent(event: Event): void {
        if (!event || !event.type_event) {
            return
        }
        if (event.type_event.indexOf(EventType.MAINTENANCE) === 0) {
            this._store.dispatch(new UpdateMaintenance(event.payload['enable']));
            return;
        }
        if (event.type_event.indexOf(EventType.ASCODE) === 0) {
            if (event.username === this._store.selectSnapshot(AuthenticationState.user).username) {
                this._store.dispatch(new AsCodeEvent(event.payload['as_code_event']));
            }
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
            this.updateProjectCache(event);
            this._navbarService.refreshData();
        }
        if (event.type_event.indexOf(EventType.APPLICATION_PREFIX) === 0) {
            this.updateApplicationCache(event);
        } else if (event.type_event.indexOf(EventType.PIPELINE_PREFIX) === 0) {
            this.updatePipelineCache(event);
        } else if (event.type_event.indexOf(EventType.WORKFLOW_PREFIX) === 0) {
            this.updateWorkflowCache(event);
        } else if (event.type_event.indexOf(EventType.RUN_WORKFLOW_PREFIX) === 0) {
            this.updateWorkflowRunCache(event);
        } else if (event.type_event.indexOf(EventType.BROADCAST_PREFIX) === 0) {
            this.updateBroadcastCache(event);
        }
        this.manageEventForTimeline(event);
    }

    manageEventForTimeline(event: Event) {
        if (!event || !event.type_event) {
            return
        }
        if (event.type_event === EventType.RUN_WORKFLOW_PREFIX) {
            let mustAdd = true;
            // Check if we have to mute it
            if (this.filter && this.filter.projects) {
                let workflowList = this.filter.projects.find(p => p.key === event.project_key);
                if (workflowList) {
                    let w = workflowList.workflow_names.find(wname => wname === event.workflow_name);
                    if (w) {
                        mustAdd = false;
                    }
                }
            }

            if (mustAdd) {
                let e = cloneDeep(event);
                this._timelineStore.add(e);
            }

        }
    }

    updateProjectCache(event: Event): void {
        if (!event || !event.type_event) {
            return
        }
        this._store.selectOnce(ProjectState)
            .pipe(
                filter((projState: ProjectStateModel) => {
                    return projState && projState.project && projState.project.key === event.project_key;
                })
            )
            .subscribe((projectState: ProjectStateModel) => {
                let projectInCache = projectState.project;
                // If working on project or sub resources
                if (this.routeParams['key'] && this.routeParams['key'] === projectInCache.key) {
                    // if modification from another user, display a notification
                    if (event.username !== this._store.selectSnapshot(AuthenticationState.user).username) {
                        this._store.dispatch(new projectActions.ExternalChangeProject({ projectKey: projectInCache.key }));
                        this._toast.info('', this._translate.instant('warning_project', { username: event.username }));
                        return;
                    }
                } else {
                    // If no working on current project, remove from cache
                    this._store.dispatch(new projectActions.DeleteProjectFromCache({ projectKey: projectInCache.key }));
                    return;
                }

                if (event.type_event === EventType.PROJECT_DELETE) {
                    this._store.dispatch(new projectActions.DeleteProjectFromCache({ projectKey: projectInCache.key }));
                    return;
                }

                let opts = [];
                if (event.type_event.indexOf(EventType.PROJECT_VARIABLE_PREFIX) === 0) {
                    opts.push(new LoadOpts('withVariables', 'variables'));
                } else if (event.type_event.indexOf(EventType.PROJECT_PERMISSION_PREFIX) === 0) {
                    opts.push(new LoadOpts('withGroups', 'groups'));
                } else if (event.type_event.indexOf(EventType.PROJECT_KEY_PREFIX) === 0) {
                    opts.push(new LoadOpts('withKeys', 'keys'));
                } else if (event.type_event.indexOf(EventType.PROJECT_INTEGRATION_PREFIX) === 0) {
                    opts.push(new LoadOpts('withIntegrations', 'integrations'));
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

                if (event.type_event.indexOf('Variable') === -1 && event.type_event.indexOf('Parameter') === -1) {
                    this._store.dispatch(new projectActions.ResyncProject({ projectKey: projectInCache.key, opts }));
                }
            });
    }

    updateApplicationCache(event: Event): void {
        if (!event || !event.type_event) {
            return
        }
        const payload = { projectKey: event.project_key, applicationName: event.application_name };

        this._store.selectOnce(ApplicationsState).subscribe((appState: ApplicationStateModel) => {
            if (!appState.application ||
                !(appState.application.name === event.application_name &&
                    appState.currentProjectKey === event.project_key)) {
                return;
            }

            if (event.type_event === EventType.APPLICATION_DELETE) {
                // If user is on an application that has been deleted by an other user
                if (this.routeParams['key'] && this.routeParams['key'] === event.project_key &&
                    this.routeParams['appName'] && this.routeParams['appName'] === event.application_name &&
                    event.username !== this._store.selectSnapshot(AuthenticationState.user).username) {
                    this._toast.info('', this._translate.instant('application_deleted_by',
                        { appName: this.routeParams['appName'], username: event.username }));
                    this._router.navigate(['/project'], this.routeParams['key']);
                }
                this._store.dispatch(new ClearCacheApplication());
                this._store.dispatch(new projectActions.DeleteApplicationInProject({ applicationName: event.application_name }));
                return;
            }

            // If working on the application
            if (this.routeParams['key'] && this.routeParams['key'] === event.project_key
                && this.routeParams['appName'] === event.application_name) {
                // modification by another user
                if (event.username !== this._store.selectSnapshot(AuthenticationState.user).username) {
                    this._store.dispatch(new ExternalChangeApplication(payload));
                    this._toast.info('', this._translate.instant('warning_application', { username: event.username }));
                    return;
                }
            } else {
                this._store.dispatch(new ClearCacheApplication());
                return;
            }

            if (event.type_event.indexOf('Variable') === -1) {
                this._store.dispatch(new ResyncApplication(payload));
            }
        });
    }

    updatePipelineCache(event: Event): void {
        if (!event || !event.type_event) {
            return
        }

        this._store.selectOnce(PipelinesState).subscribe((pips: PipelinesStateModel) => {
            if (!pips || !pips.pipeline || pips.pipeline.name !== event.pipeline_name || pips.currentProjectKey !== event.project_key) {
                return;
            }

            if (event.type_event === EventType.PIPELINE_DELETE) {
                this._store.dispatch(new DeleteFromCachePipeline({
                    projectKey: event.project_key,
                    pipelineName: event.pipeline_name
                }));
                this._store.dispatch(new projectActions.DeletePipelineInProject({ pipelineName: event.pipeline_name }));
                return;
            }

            // update pipeline
            if (this.routeParams['key'] && this.routeParams['key'] === event.project_key
                && this.routeParams['pipName'] === event.pipeline_name) {
                if (event.username !== this._store.selectSnapshot(AuthenticationState.user).username) {
                    this._store.dispatch(new ExternalChangePipeline({
                        projectKey: event.project_key,
                        pipelineName: event.pipeline_name
                    }));
                    this._toast.info('', this._translate.instant('warning_pipeline', { username: event.username }));
                    return;
                }
            } else {
                this._store.dispatch(new DeleteFromCachePipeline({
                    projectKey: event.project_key,
                    pipelineName: event.pipeline_name
                }));
                return;
            }

            this._store.dispatch(new ResyncPipeline({
                projectKey: event.project_key,
                pipelineName: event.pipeline_name
            }))
        });
    }

    updateWorkflowCache(event: Event): void {
        if (!event || !event.type_event) {
            return
        }
        this._store.selectOnce(WorkflowState)
            .pipe(
                filter((wf) => wf != null && wf.workflow
                    && (wf.projectKey !== event.project_key || wf.workflow.name !== event.workflow_name)))
            .subscribe(() => {
                if (event.type_event === EventType.WORKFLOW_DELETE) {
                    this._store.dispatch(new projectActions.DeleteWorkflowInProject({ workflowName: event.workflow_name }));
                    return;
                }

                // update workflow
                if (this.routeParams['key'] && this.routeParams['key'] === event.project_key
                    && this.routeParams['workflowName'] === event.workflow_name) {
                    if (event.username !== this._store.selectSnapshot(AuthenticationState.user).username) {
                        this._store.dispatch(new ExternalChangeWorkflow({
                            projectKey: event.project_key,
                            workflowName: event.workflow_name
                        }));
                        this._toast.info('', this._translate.instant('warning_workflow', { username: event.username }));
                        return;
                    }
                } else {
                    return;
                }

                this._store.dispatch(new GetWorkflow({
                    projectKey: event.project_key,
                    workflowName: event.workflow_name
                }));
            });
    }

    updateWorkflowRunCache(event: Event): void {
        if (!event || !event.type_event) {
            return
        }
        if (this.routeParams['key'] !== event.project_key || this.routeParams['workflowName'] !== event.workflow_name) {
            return;
        }
        switch (event.type_event) {
            case EventType.RUN_WORKFLOW_PREFIX:
                if (this.routeParams['number'] === event.workflow_run_num.toString()) {
                    // if same run number , then update store
                    this._store.dispatch(
                        new GetWorkflowRun({
                            projectKey: event.project_key,
                            workflowName: event.workflow_name,
                            num: event.workflow_run_num
                        }));
                } else {
                    this._workflowRunService
                        .getWorkflowRun(event.project_key, event.workflow_name, event.workflow_run_num)
                        .pipe(first())
                        .subscribe(wrkRun => this._store.dispatch(new UpdateWorkflowRunList({ workflowRun: wrkRun })));
                }
                break;
            case EventType.RUN_WORKFLOW_NODE:
                // Refresh node run if user is listening on it
                const wnr = this._store.selectSnapshot<WorkflowNodeRun>((state) => {
                    return state.workflow.workflowNodeRun;
                });
                let wnrEvent = <WorkflowNodeRun>event.payload;
                if (wnr && wnr.id === wnrEvent.id) {
                    this._store.dispatch(
                        new GetWorkflowNodeRun({
                            projectKey: event.project_key,
                            workflowName: event.workflow_name,
                            num: event.workflow_run_num,
                            nodeRunID: wnr.id
                        }));
                }

                // Refresh workflow run if user is listening on it
                const wr = this._store.selectSnapshot<WorkflowRun>((state) => state.workflow.workflowRun);
                if (wr && wr.num === event.workflow_run_num) {
                    this._store.dispatch(new GetWorkflowRun(
                        {
                            projectKey: event.project_key,
                            workflowName: event.workflow_name,
                            num: event.workflow_run_num
                        }));
                }
                break;
        }
    }

    updateBroadcastCache(event: Event): void {
        if (!event || !event.type_event) {
            return
        }
        switch (event.type_event) {
            case EventType.BROADCAST_ADD:
                let bEvent: Broadcast = <Broadcast>event.payload;
                if (bEvent) {
                    this._broadcastStore.addBroadcastInCache(bEvent);
                }
                break;
            case EventType.BROADCAST_UPDATE:
                let bUpEvent: Broadcast = <Broadcast>event.payload['NewBroadcast'];
                if (bUpEvent) {
                    this._broadcastStore.addBroadcastInCache(bUpEvent);
                }
                break;
            case EventType.BROADCAST_DELETE:
                this._broadcastStore.removeBroadcastFromCache(event.payload['BroadcastID']);
                break;
        }
    }
}
