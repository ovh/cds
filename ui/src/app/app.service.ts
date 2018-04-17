import {Injectable} from '@angular/core';
import {ProjectStore} from './service/project/project.store';
import {ActivatedRoute} from '@angular/router';
import {ApplicationStore} from './service/application/application.store';
import {NotificationService} from './service/notification/notification.service';
import {AuthentificationStore} from './service/auth/authentification.store';
import {TranslateService} from '@ngx-translate/core';
import {LoadOpts} from './model/project.model';
import {PipelineStore} from './service/pipeline/pipeline.store';
import {WorkflowStore} from './service/workflow/workflow.store';
import {RouterService} from './service/router/router.service';
import {first} from 'rxjs/operators';
import {Event} from './model/event.model';
import {EventStore} from './service/event/event.store';
import {WorkflowEventStore} from './service/workflow/workflow.event.store';
import {WorkflowNodeRun, WorkflowRun} from './model/workflow.run.model';

@Injectable()
export class AppService {

    constructor(private _projStore: ProjectStore, private _routeActivated: ActivatedRoute,
                private _appStore: ApplicationStore, private _notif: NotificationService, private _authStore: AuthentificationStore,
                private _translate: TranslateService, private _pipStore: PipelineStore, private _workflowEventStore: WorkflowEventStore,
                private _wfStore: WorkflowStore, private _routerService: RouterService, private _eventStore: EventStore) {
    }

    manageEvent(event: Event): void {
        if (event.type_event.indexOf('sdk.EventProject') === 0 || event.type_event.indexOf('sdk.EventEnvironment') === 0 ||
            event.type_event === 'sdk.EventApplicationAdd' || event.type_event === 'sdk.EventApplicationUpdate' ||
            event.type_event === 'sdk.EventApplicationDelete' ||
            event.type_event === 'sdk.EventPipelineAdd' || event.type_event === 'sdk.EventPipelineUpdate' ||
            event.type_event === 'sdk.EventPipelineDelete' ||
            event.type_event === 'sdk.EventEnvironmentAdd' || event.type_event === 'sdk.EventEnvironmentUpdate' ||
            event.type_event === 'sdk.EventEnvironmentDelete' ||
            event.type_event === 'sdk.EventWorkflowAdd' || event.type_event === 'sdk.EventWorkflowUpdate' ||
            event.type_event === 'sdk.EventWorkflowDelete') {
            this.updateProjectCache(event);
        }
        if (event.type_event.indexOf('sdk.EventApplication') === 0) {
            this.updateApplicationCache(event);
        } else if (event.type_event.indexOf('sdk.EventPpeline') === 0) {
            this.updatePipelineCache(event);
        } else if (event.type_event.indexOf('sdk.EventWorkflow') === 0) {
            this.updateWorkflowCache(event);
        } else if (event.type_event.indexOf('sdk.EventRunWorkflow') === 0) {
            this.updateWorkflowRunCache(event);
        }
    }

    updateProjectCache(event: Event): void {
        this._projStore.getProjects('').pipe(first()).subscribe(projects => {
            // Project not in cache
            let projectInCache = projects.get(event.project_key);
            if (!projectInCache) {
                return;
            }

            // Get current route
            let params = this._routerService.getRouteParams({}, this._routeActivated);

            // If working on project or sub resources
            if (params['key'] && params['key'] === projectInCache.key) {
                // if modification from another user, display a notification
                if (event.username !== this._authStore.getUser().username) {
                    this._projStore.externalModification(projectInCache.key);
                    this._notif.create(this._translate.instant('warning_project', {username: event.username}));
                    return;
                }
            } else {
                // If no working on current project, remove from cache
                this._projStore.removeFromStore(projectInCache.key);
                return;
            }

            if (event.type_event === 'sdk.EventProjectDelete') {
                this._projStore.removeFromStore(projectInCache.key);
                return
            }

            let opts = [];
            if (event.type_event.indexOf('sdk.EventProjectVariable') === 0) {
                opts.push(new LoadOpts('withVariables', 'variables'));
            } else if (event.type_event.indexOf('sdk.EventProjectPermission') === 0) {
                opts.push(new LoadOpts('withGroups', 'groups'));
            } else if (event.type_event.indexOf('sdk.EventProjectKey') === 0) {
                opts.push(new LoadOpts('withKeys', 'keys'));
            } else if (event.type_event.indexOf('sdk.EventProjectPlatform') === 0) {
                opts.push(new LoadOpts('withPlatforms', 'platforms'));
            } else if (event.type_event.indexOf('sdk.EventApplication') === 0) {
                opts.push(new LoadOpts('withApplicationNames', 'application_names'));
            } else if (event.type_event.indexOf('sdk.EventPipeline') === 0) {
                opts.push(new LoadOpts('withPipelineNames', 'pipeline_names'));
            } else if (event.type_event.indexOf('sdk.EventEnvironment') === 0) {
                opts.push(new LoadOpts('withEnvironments', 'environments'));
            } else if (event.type_event.indexOf('sdk.EventWorkflow') === 0) {
                opts.push(new LoadOpts('withWorkflowNames', 'workflow_names'));
            }
            this._projStore.resync(projectInCache.key, opts).pipe(first()).subscribe(() => {});
        });
    }

    updateApplicationCache(event: Event): void {
        let appKey = event.project_key + '-' + event.application_name;
        if (event.type_event === 'EventApplicationDelete') {
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

            // Get current route
            let params = this._routerService.getRouteParams({}, this._routeActivated);

            // If working on the application
            if (params['key'] && params['key'] === event.project_key && params['appName'] === event.application_name) {
                // modification by another user
                if (event.username !== this._authStore.getUser().username) {
                    this._appStore.externalModification(appKey);
                    this._notif.create(this._translate.instant('warning_application', {username: event.username}));
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
        let pipKey = event.project_key + '-' + event.pipeline_name;
        if (event.type_event === 'EventPipelineDelete') {
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

            let params = this._routerService.getRouteParams({}, this._routeActivated);

            // update pipeline
            if (params['key'] && params['key'] === event.project_key && params['pipName'] === event.pipeline_name) {
                if (event.username !== this._authStore.getUser().username) {
                    this._pipStore.externalModification(pipKey);
                    this._notif.create(this._translate.instant('warning_pipeline', {username: event.username}));
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
        let wfKey = event.project_key + '-' + event.workflow_name;
        if (event.type_event === 'EventWorkflowDelete') {
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

            let params = this._routerService.getRouteParams({}, this._routeActivated);

            // update workflow
            if (params['key'] && params['key'] === event.project_key && params['workflowName'] === event.workflow_name) {
                if (event.username !== this._authStore.getUser().username) {
                    this._wfStore.externalModification(wfKey);
                    this._notif.create(this._translate.instant('warning_workflow', {username: event.username}));
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
        switch (event.type_event) {
            case 'sdk.EventRunWorkflow':
                let wr = WorkflowRun.fromEventRunWorkflow(event);
                this._workflowEventStore.addWorkflowRun(wr);
                break;
            case 'sdk.EventRunWorkflowNode':
                let wnr = WorkflowNodeRun.fromEventRunWorkflowNode(event);
                this._workflowEventStore.broadcastNodeRunEvents(wnr);
                break;
        }
        this._eventStore._eventFilter.getValue();
    }
}
