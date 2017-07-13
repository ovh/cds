import {Injectable} from '@angular/core';
import {ProjectLastUpdates} from './model/lastupdate.model';
import {ProjectStore} from './service/project/project.store';
import {ActivatedRoute, Router} from '@angular/router';
import {ApplicationStore} from './service/application/application.store';
import {NotificationService} from './service/notification/notification.service';
import {AuthentificationStore} from './service/auth/authentification.store';
import {TranslateService} from 'ng2-translate';
import {PipelineStore} from './service/pipeline/pipeline.store';
import {RouterService} from './service/router/router.service';

@Injectable()
export class AppService {

    constructor(private _projStore: ProjectStore, private _routeActivated: ActivatedRoute,
        private _appStore: ApplicationStore, private _notif: NotificationService, private _authStore: AuthentificationStore,
        private _translate: TranslateService, private _pipStore: PipelineStore, private _routerService: RouterService) {
    }

    updateCache(lastUpdates: Array<ProjectLastUpdates>) {
        if (!lastUpdates) {
            return;
        }
        // Get current route params
        let params = this._routerService.getRouteParams({}, this._routeActivated);

        // Get all projects
        this._projStore.getProjects().first().subscribe(projects => {

            // browse last updates
            lastUpdates.forEach(plu => {
                // Project not in cache
                if (!projects.get(plu.name)) {
                    return;
                }

                // Project
                if ((new Date(projects.get(plu.name).last_modified)).getTime() < plu.last_modified * 1000) {
                    // If working on project on sub resources
                    if (params['key'] && params['key'] === plu.name) {
                        if (plu.username !== this._authStore.getUser().username) {
                            this._projStore.externalModification(plu.name);
                            this._notif.create(this._translate.instant('project_modification', { username: plu.username}));
                        }

                        // If working on sub resources - resync project
                        if (params['pipName'] || params['appName'] || plu.username === this._authStore.getUser().username) {
                            this._projStore.resync(plu.name).first().subscribe(() => {});
                        }
                    } else {
                        // remove from cache
                        this._projStore.removeFromStore(plu.name);
                    }
                }

                if (plu.applications && plu.applications.length > 0) {
                    // update application cache
                    this.updateApplicationCache(plu, params);
                }

                if (plu.pipelines && plu.pipelines.length > 0) {
                    this.updatePipelineCache(plu, params);
                }
            });
        });

    }

    updateApplicationCache(plu: ProjectLastUpdates, params: {}): void {
        this._appStore.getApplications(plu.name).first().subscribe(apps => {
            if (!apps) {
                return;
            }

            plu.applications.forEach( a => {
                let appKey = plu.name + '-' + a.name;
                if (!apps.get(appKey)) {
                    return;
                }

                if ((new Date(apps.get(appKey).last_modified)).getTime() < a.last_modified * 1000) {
                    if (params['key'] && params['key'] === plu.name && params['appName'] === a.name ) {

                        if (a.username !== this._authStore.getUser().username) {
                            this._appStore.externalModification(appKey);
                            this._notif.create(this._translate.instant('application_modification', { username: plu.username}));
                        }

                        if (params['pipName'] || a.username === this._authStore.getUser().username) {
                            this._appStore.resync(plu.name, a.name);
                        }
                    } else {
                        this._appStore.removeFromStore(appKey);
                    }
                }
            });
        });

    }

    updatePipelineCache(plu: ProjectLastUpdates, params: {}): void {
        this._pipStore.getPipelines(plu.name).first().subscribe(pips => {
            if (!pips) {
                return;
            }

            plu.pipelines.forEach(p => {
                let pipKey = plu.name + '-' + p.name;
                if (!pips.get(pipKey)) {
                    return;
                }

                if (pips.get(pipKey).last_modified < p.last_modified) {
                    if (params['key'] && params['key'] === plu.name && params['pipName'] === p.name) {
                        this._pipStore.externalModification(pipKey);

                        if (p.username !== this._authStore.getUser().username) {
                            this._notif.create(this._translate.instant('pipeline_modification', {username: plu.username}));
                        }

                        if (params['buildNumber'] || p.username === this._authStore.getUser().username) {
                            this._pipStore.resync(plu.name, p.name);
                        }
                    } else {
                        this._pipStore.removeFromStore(pipKey);
                    }
                }
            });
        });

    }
}
