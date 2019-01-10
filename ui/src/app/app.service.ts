import { Injectable } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { cloneDeep } from 'lodash';
import { first } from 'rxjs/operators';
import { Subscription } from 'rxjs/Subscription';
import { Broadcast, BroadcastEvent } from './model/broadcast.model';
import { Event, EventType } from './model/event.model';
import { PipelineStatus } from './model/pipeline.model';
import { LoadOpts } from './model/project.model';
import { TimelineFilter } from './model/timeline.model';
import { WorkflowNodeRun, WorkflowRun } from './model/workflow.run.model';
import { ApplicationStore } from './service/application/application.store';
import { AuthentificationStore } from './service/auth/authentification.store';
import { BroadcastStore } from './service/broadcast/broadcast.store';
import { PipelineStore } from './service/pipeline/pipeline.store';
import { ProjectStore } from './service/project/project.store';
import { ActionStore, RouterService, TimelineStore } from './service/services.module';
import { WorkflowRunService } from './service/workflow/run/workflow.run.service';
import { WorkflowEventStore } from './service/workflow/workflow.event.store';
import { WorkflowStore } from './service/workflow/workflow.store';
import { ToastService } from './shared/toast/ToastService';

@Injectable()
export class AppService {

    // Information about current route
    routeParams: {};

    filterSub: Subscription;
    filter: TimelineFilter;

    constructor(private _projStore: ProjectStore, private _routerService: RouterService, private _routeActivated: ActivatedRoute,
                private _appStore: ApplicationStore, private _authStore: AuthentificationStore, private _actionStore: ActionStore,
                private _translate: TranslateService, private _pipStore: PipelineStore, private _workflowEventStore: WorkflowEventStore,
                private _wfStore: WorkflowStore, private _broadcastStore: BroadcastStore, private _timelineStore: TimelineStore,
                private _toast: ToastService, private _workflowRunService: WorkflowRunService) {
        this.routeParams = this._routerService.getRouteParams({}, this._routeActivated);
    }

    initFilter(filter: TimelineFilter) {
        this.filter = cloneDeep(filter);
    }

    updateRoute(params: {}) {
        this.routeParams = params;
    }

    manageEvent(event: Event): void {
        if (!event || !event.type_event) {
            return
        }
        if (event.type_event.indexOf(EventType.ACTION_PREFIX) === 0) {
            this._actionStore.resync();
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
        this._projStore.getProjects('').pipe(first()).subscribe(projects => {
            // Project not in cache
            let projectInCache = projects.get(event.project_key);
            if (!projectInCache) {
                return;
            }

            // If working on project or sub resources
            if (this.routeParams['key'] && this.routeParams['key'] === projectInCache.key) {
                // if modification from another user, display a notification
                if (event.username !== this._authStore.getUser().username) {
                    this._projStore.externalModification(projectInCache.key);
                    this._toast.info('', this._translate.instant('warning_project', {username: event.username}));
                    return;
                }
            } else {
                // If no working on current project, remove from cache
                this._projStore.removeFromStore(projectInCache.key);
                return;
            }

            if (event.type_event === EventType.PROJECT_DELETE) {
                this._projStore.removeFromStore(projectInCache.key);
                return
            }

            let opts = [];
            if (event.type_event.indexOf(EventType.PROJECT_VARIABLE_PREFIX) === 0) {
                opts.push(new LoadOpts('withVariables', 'variables'));
            } else if (event.type_event.indexOf(EventType.PROJECT_PERMISSION_PREFIX) === 0) {
                opts.push(new LoadOpts('withGroups', 'groups'));
            } else if (event.type_event.indexOf(EventType.PROJECT_KEY_PREFIX) === 0) {
                opts.push(new LoadOpts('withKeys', 'keys'));
            } else if (event.type_event.indexOf(EventType.PROJECT_PLATFORM_PREFIX) === 0) {
                opts.push(new LoadOpts('withPlatforms', 'platforms'));
            } else if (event.type_event.indexOf(EventType.APPLICATION_PREFIX) === 0) {
                opts.push(new LoadOpts('withApplicationNames', 'application_names'));
            } else if (event.type_event.indexOf(EventType.PIPELINE_PREFIX) === 0) {
                opts.push(new LoadOpts('withPipelineNames', 'pipeline_names'));
            } else if (event.type_event.indexOf(EventType.ENVIRONMENT_PREFIX) === 0) {
                opts.push(new LoadOpts('withEnvironments', 'environments'));
            } else if (event.type_event.indexOf(EventType.WORKFLOW_PREFIX) === 0) {
                opts.push(new LoadOpts('withWorkflowNames', 'workflow_names'));
                opts.push(new LoadOpts('withLabels', 'labels'));
            }
            this._projStore.resync(projectInCache.key, opts).pipe(first()).subscribe();
        });
    }

    updateApplicationCache(event: Event): void {
        if (!event || !event.type_event) {
            return
        }
        let appKey = event.project_key + '-' + event.application_name;
        if (event.type_event === EventType.APPLICATION_DELETE) {
            this._appStore.removeFromStore(appKey);
            return;
        }

        this._appStore.getApplications(event.project_key, null).pipe(first()).subscribe(apps => {
            if (!apps) {
                return;
            }

            if (!apps.get(appKey)) {
                return;
            }

            // If working on the application
            if (this.routeParams['key'] && this.routeParams['key'] === event.project_key
                && this.routeParams['appName'] === event.application_name) {
                // modification by another user
                if (event.username !== this._authStore.getUser().username) {
                    this._appStore.externalModification(appKey);
                    this._toast.info('', this._translate.instant('warning_application', {username: event.username}));
                    return;
                }
            } else {
                this._appStore.removeFromStore(appKey);
                return;
            }

            this._appStore.resync(event.project_key, event.application_name);

        });

    }

    updatePipelineCache(event: Event): void {
        if (!event || !event.type_event) {
            return
        }
        let pipKey = event.project_key + '-' + event.pipeline_name;
        if (event.type_event === EventType.PIPELINE_DELETE) {
            this._appStore.removeFromStore(pipKey);
            return;
        }

        this._pipStore.getPipelines(event.project_key).pipe(first()).subscribe(pips => {
            if (!pips) {
                return;
            }

            if (!pips.get(pipKey)) {
                return;
            }

            // update pipeline
            if (this.routeParams['key'] && this.routeParams['key'] === event.project_key
                && this.routeParams['pipName'] === event.pipeline_name) {
                if (event.username !== this._authStore.getUser().username) {
                    this._pipStore.externalModification(pipKey);
                    this._toast.info('', this._translate.instant('warning_pipeline', {username: event.username}));
                    return;
                }
            } else {
                this._pipStore.removeFromStore(pipKey);
                return;
            }

            this._pipStore.resync(event.project_key, event.pipeline_name);
        });
    }

    updateWorkflowCache(event: Event): void {
        if (!event || !event.type_event) {
            return
        }
        let wfKey = event.project_key + '-' + event.workflow_name;
        if (event.type_event === EventType.WORKFLOW_DELETE) {
            this._appStore.removeFromStore(wfKey);
            return;
        }
        this._wfStore.getWorkflows(event.project_key).pipe(first()).subscribe(wfs => {
            if (!wfs) {
                return;
            }

            if (!wfs.get(wfKey)) {
                return;
            }

            // update workflow
            if (this.routeParams['key'] && this.routeParams['key'] === event.project_key
                && this.routeParams['workflowName'] === event.workflow_name) {
                if (event.username !== this._authStore.getUser().username) {
                    this._wfStore.externalModification(wfKey);
                    this._toast.info('', this._translate.instant('warning_workflow', {username: event.username}));
                    return
                }
            } else {
                this._wfStore.removeFromStore(wfKey);
                return;
            }

            this._wfStore.resync(event.project_key, event.workflow_name);
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
                let wr = WorkflowRun.fromEventRunWorkflow(event);
                this._workflowEventStore.broadcastWorkflowRun(event.project_key, event.workflow_name, wr);
                break;
            case EventType.RUN_WORKFLOW_NODE:
                if (this.routeParams['number'] === event.workflow_run_num.toString()) {
                    let wnr = WorkflowNodeRun.fromEventRunWorkflowNode(event);
                    if (PipelineStatus.isDone(wnr.status)) {
                        // Usefull to load tests and artifacts
                        this._workflowRunService.getWorkflowNodeRun(
                            event.project_key,
                            event.workflow_name,
                            wnr.num,
                            wnr.id
                        ).subscribe((wfNodeRun) => this._workflowEventStore.broadcastNodeRunEvents(<WorkflowNodeRun>wfNodeRun));
                    } else {
                        this._workflowEventStore.broadcastNodeRunEvents(wnr);
                    }
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
                let bEvent: BroadcastEvent = <BroadcastEvent>event.payload['Broadcast'];
                if (bEvent) {
                    this._broadcastStore.addBroadcastInCache(Broadcast.fromEvent(bEvent));
                }
                break;
            case EventType.BROADCAST_UPDATE:
                let bUpEvent: BroadcastEvent = <BroadcastEvent>event.payload['NewBroadcast'];
                if (bUpEvent) {
                    this._broadcastStore.addBroadcastInCache(Broadcast.fromEvent(bUpEvent));
                }
                break;
            case EventType.BROADCAST_DELETE:
                this._broadcastStore.removeBroadcastFromCache(event.payload['BroadcastID']);
                break;
        }
    }
}
