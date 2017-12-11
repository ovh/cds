import {Injectable} from '@angular/core';
import {LastModification} from './model/lastupdate.model';
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

@Injectable()
export class AppService {

    constructor(private _projStore: ProjectStore, private _routeActivated: ActivatedRoute,
                private _appStore: ApplicationStore, private _notif: NotificationService, private _authStore: AuthentificationStore,
                private _translate: TranslateService, private _pipStore: PipelineStore,
                private _wfStore: WorkflowStore, private _routerService: RouterService) {
    }

    updateCache(lastUpdate: LastModification) {
        let opts = [];
        if (!lastUpdate) {
            return;
        }

        switch (lastUpdate.type) {
            case 'project':
                opts = [
                    new LoadOpts('withWorkflows', 'workflows')
                ];
                this.updateProjectCache(lastUpdate, opts);
                break;
            case 'application':
                this.updateApplicationCache(lastUpdate);
                break;
            case 'pipeline':
                this.updatePipelineCache(lastUpdate);
                break;
            case 'workflow':
                this.updateWorkflowCache(lastUpdate);
                break;
            case 'project.environment':
                opts = [
                    new LoadOpts('withEnvironments', 'environments')
                ];
                this.updateProjectCache(lastUpdate, opts);
                break;
            case 'project.variable':
                opts = [
                    new LoadOpts('withVariables', 'variables')
                ];
                this.updateProjectCache(lastUpdate, opts);
                break;
            case 'project.application':
                opts = [
                    new LoadOpts('withApplicationNames', 'application_names')
                ];
                this.updateProjectCache(lastUpdate, opts);
                break;
            case 'project.pipeline':
                opts = [
                    new LoadOpts('withPipelineNames', 'pipeline_names')
                ];
                this.updateProjectCache(lastUpdate, opts);
                break;
            case 'project.workflow':
                opts = [
                    new LoadOpts('withWorkflowNames', 'workflow_names')
                ];
                this.updateProjectCache(lastUpdate, opts);
                break;
        }
    }

    updateProjectCache(lastUpdate: LastModification, opts: LoadOpts[]): void {
        // Get all projects
        this._projStore.getProjects('', opts).pipe(first()).subscribe(projects => {
            // Project not in cache
            if (!projects.get(lastUpdate.key)) {
                return;
            }

            // Project
            if ((new Date(projects.get(lastUpdate.key).last_modified)).getTime() < (lastUpdate.last_modified + 1) * 1000) {
                // Get current route params)
                let params = this._routerService.getRouteParams({}, this._routeActivated);

                // If working on project on sub resources
                if (params['key'] && params['key'] === lastUpdate.key) {
                    if (lastUpdate.username !== this._authStore.getUser().username) {
                        this._projStore.externalModification(lastUpdate.key);
                        this._notif.create(this._translate.instant('project_modification', {username: lastUpdate.username}));
                    }

                    // If working on sub resources - resync project
                    if (params['pipName'] || params['appName'] || lastUpdate.username === this._authStore.getUser().username) {
                        this._projStore.resync(lastUpdate.key, opts).pipe(first()).subscribe(() => {});
                    }
                } else {
                    // remove from cache
                    this._projStore.removeFromStore(lastUpdate.key);
                }
            }
        });
    }

    updateApplicationCache(lastUpdate: LastModification): void {
        this._appStore.getApplications(lastUpdate.key, null).pipe(first()).subscribe(apps => {
            if (!apps) {
                return;
            }

            let appKey = lastUpdate.key + '-' + lastUpdate.name;
            if (!apps.get(appKey)) {
                return;
            }

            if ((new Date(apps.get(appKey).last_modified)).getTime() < (lastUpdate.last_modified + 1) * 1000) {

                let params = this._routerService.getRouteParams({}, this._routeActivated);

                if (params['key'] && params['key'] === lastUpdate.key && params['appName'] === lastUpdate.name) {

                    if (lastUpdate.username !== this._authStore.getUser().username) {
                        this._appStore.externalModification(appKey);
                        this._notif.create(this._translate.instant('application_modification', {username: lastUpdate.username}));
                    }

                    if (params['pipName'] || lastUpdate.username === this._authStore.getUser().username) {
                        this._appStore.resync(lastUpdate.key, lastUpdate.name);
                    }
                } else {
                    this._appStore.removeFromStore(appKey);
                }
            }
        });

    }

    updatePipelineCache(lastUpdate: LastModification): void {
        this._pipStore.getPipelines(lastUpdate.name).pipe(first()).subscribe(pips => {
            if (!pips) {
                return;
            }

            let pipKey = lastUpdate.key + '-' + lastUpdate.name;
            if (!pips.get(pipKey)) {
                return;
            }

            if (pips.get(pipKey).last_modified < lastUpdate.last_modified) {
                let params = this._routerService.getRouteParams({}, this._routeActivated);

                // delete linked applications from cache
                this._pipStore.getPipelineResolver(lastUpdate.key, lastUpdate.name)
                    .subscribe((pip) => {
                        if (pip && pip.usage && Array.isArray(pip.usage.applications)) {
                            pip.usage.applications.forEach((app) => this._appStore.removeFromStore(lastUpdate.key + '-' + app.name));
                        }
                    });

                // update pipeline
                if (params['key'] && params['key'] === lastUpdate.key && params['pipName'] === lastUpdate.name) {
                    if (lastUpdate.username !== this._authStore.getUser().username) {
                        this._pipStore.externalModification(pipKey);
                        this._notif.create(this._translate.instant('pipeline_modification', {username: lastUpdate.username}));
                    }

                    if (params['buildNumber'] || lastUpdate.username === this._authStore.getUser().username) {
                        this._pipStore.resync(lastUpdate.key, lastUpdate.name);
                    }
                } else {
                    this._pipStore.removeFromStore(pipKey);
                }
            }
        });
    }

    updateWorkflowCache(lastUpdate: LastModification): void {
        this._wfStore.getWorkflows(lastUpdate.name).pipe(first()).subscribe(wfs => {
            if (!wfs) {
                return;
            }

            let wfKey = lastUpdate.key + '-' + lastUpdate.name;
            if (!wfs.get(wfKey)) {
                return;
            }

            if (new Date(wfs.get(wfKey).last_modified).getTime() < lastUpdate.last_modified * 1000) {
                let params = this._routerService.getRouteParams({}, this._routeActivated);

                // delete linked applications from cache
                this._wfStore.getWorkflowResolver(lastUpdate.key, lastUpdate.name)
                    .subscribe((pip) => {
                        if (pip && pip.usage && Array.isArray(pip.usage.applications)) {
                            pip.usage.applications.forEach((app) => this._appStore.removeFromStore(lastUpdate.key + '-' + app.name));
                        }
                    });

                // update pipeline
                if (params['key'] && params['key'] === lastUpdate.key && params['pipName'] === lastUpdate.name) {
                    if (lastUpdate.username !== this._authStore.getUser().username) {
                        this._wfStore.externalModification(wfKey);
                        this._notif.create(this._translate.instant('pipeline_modification', {username: lastUpdate.username}));
                    }

                    if (params['buildNumber'] || lastUpdate.username === this._authStore.getUser().username) {
                        this._wfStore.resync(lastUpdate.key, lastUpdate.name);
                    }
                } else {
                    this._wfStore.removeFromStore(wfKey);
                }
            }
        });
    }
}
