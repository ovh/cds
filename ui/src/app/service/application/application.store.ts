import {Injectable} from '@angular/core';
import {Map, List} from 'immutable';
import {BehaviorSubject, Observable} from 'rxjs/Rx';
import {Application} from '../../model/application.model';
import {ApplicationService} from './application.service';
import {RepositoryPoller} from '../../model/polling.model';
import {Project} from '../../model/project.model';
import {Hook} from '../../model/hook.model';
import {Variable} from '../../model/variable.model';
import {GroupPermission} from '../../model/group.model';
import {ProjectStore} from '../project/project.store';
import {Trigger} from '../../model/trigger.model';
import {ApplyTemplateRequest} from '../../model/template.model';


@Injectable()
export class ApplicationStore {

    static RECENT_APPLICATION_KEY = 'CDS-RECENT-APPLICATION';

    // List of all applications.
    private _application: BehaviorSubject<Map<string, Application>> = new BehaviorSubject(Map<string, Application>());

    private _recentApplications: BehaviorSubject<List<Application>> = new BehaviorSubject(List<Application>());


    constructor(private _applicationService: ApplicationService, private _projectStore: ProjectStore) {
        this.loadRecentApplication();

    }

    loadRecentApplication(): void {
        this._recentApplications.next(JSON.parse(localStorage.getItem(ApplicationStore.RECENT_APPLICATION_KEY)));
    }

    /**
     * Get recent application.
     * @returns {Observable<List<Application>>}
     */
    getRecentApplications(): Observable<List<Application>> {
        return new Observable<List<Application>>(fn => this._recentApplications.subscribe(fn));
    }

    /**
     * Use by router to preload application
     * @param key
     * @param appName
     * @returns {Observable<Application>}
     */
    getApplicationResolver(key: string, appName: string): Observable<Application> {
        let store = this._application.getValue();
        let appKey = key + '-' + appName;
        if (store.size === 0 || !store.get(appKey)) {
            return this._applicationService.getApplication(key, appName).map( res => {
                this._application.next(store.set(appKey, res));
                return res;
            });
        } else {
            return Observable.of(store.get(appKey));
        }
    }

    /**
     * Update recent application viewed.
     * @param key Project unique key
     * @param application Application to add
     */
    updateRecentApplication(key: string, application: Application): void {
        application.project_key = key;
        let currentRecentApps: Array<Application> = JSON.parse(localStorage.getItem(ApplicationStore.RECENT_APPLICATION_KEY));
        if (currentRecentApps) {
            let index: number = currentRecentApps.findIndex(app => app.name === application.name && app.project_key === key);
            if (index >= 0) {
                currentRecentApps.splice(index, 1);
            }
        } else {
            currentRecentApps = new Array<Application>();
        }
        currentRecentApps.splice(0, 0, application);
        currentRecentApps = currentRecentApps.splice(0, 15);
        localStorage.setItem(ApplicationStore.RECENT_APPLICATION_KEY, JSON.stringify(currentRecentApps));
        this._recentApplications.next(List(currentRecentApps));
    }

    /**
     * Get an Application
     * @returns {Observable<Application>}
     */
    getApplications(key: string, appName: string): Observable<Map<string, Application>> {
        let store = this._application.getValue();
        let appKey = key + '-' + appName;

        if (!store.get(appKey)) {
            this._applicationService.getApplication(key, appName).subscribe(res => {
                this._application.next(store.set(appKey, res));
            }, err => {
                this._application.error(err);
            });
        }
        return new Observable<Map<string, Application>>(fn => this._application.subscribe(fn));
    }

    /**
     * Clone application
     * @param key Project unique key
     * @param appName Application to clone
     * @param app New application data
     * @returns Observable<Application>
     */
    cloneApplication(key: string, appName: string, app: Application): Observable<Application> {
        return this._applicationService.cloneApplication(key, appName, app).map(appResult => appResult);
    }

    /**
     * Application template to create an application.
     * @param key Project unique key
     * @param request Request
     * @returns {Observable<Project>}
     */
    applyTemplate(key: string, request: ApplyTemplateRequest): Observable<Project> {
        return this._applicationService.applyTemplate(key, request).map(p => {
            this._projectStore.updateApplications(key, p);
            return p;
        });
    }

    /**
     * Update the given application
     * @param key Project unique key
     * @param oldName Old application name
     * @param application Application to update
     * @returns {Observable<Application>}
     */
    renameApplication(key: string, oldName: string, newName: string): Observable<Application> {
        return this._applicationService.renameApplication(key, oldName, newName).map(app => {
            let cache = this._application.getValue();
            let appKey = key + '-' + oldName;
            if (cache.get(appKey)) {
                let pToUpdate = cache.get(appKey);
                pToUpdate.last_modified = app.last_modified;
                pToUpdate.name = app.name;
                this._application.next(cache.set(key + '-' + app.name, pToUpdate).remove(appKey));
            }

            this._projectStore.updateApplicationName(key, oldName, newName);

            return app;
        });
    }

    /**
     * Delete an application
     * @param key Project unique key
     * @param appName Application name to delete
     * @returns {Observable<boolean>}
     */
    deleteApplication(key: string, appName: string): Observable<boolean> {
        return this._applicationService.deleteApplication(key, appName).map(res => {
            let cache = this._application.getValue();
            let appKey = key + '-' + appName;
            this._application.next(cache.delete(appKey));
            return res;
        });
    }

    /**
     * Connect a repository to an application
     * @param key Project unique key
     * @param currentName Application current name
     * @param repoManName Repository manager name
     * @param repoFullname Repository name
     * @returns {Observable<Application>}
     */
    connectRepository(key: string, currentName: string, repoManName: string, repoFullname: string): Observable<Application> {
        return this._applicationService.connectRepository(key, currentName, repoManName, repoFullname)
            .map(app => {
                let cache = this._application.getValue();
                let appKey = key + '-' + currentName;
                let appToUpdate = cache.get(appKey);
                if (appToUpdate) {
                    appToUpdate.last_modified = app.last_modified;
                    appToUpdate.repositories_manager = app.repositories_manager;
                    appToUpdate.repository_fullname = app.repository_fullname;
                    this._application.next(cache.set(appKey, appToUpdate));
                }
                return app;
            });
    }

    /**
     * Remove the attached repository
     * @param application Application
     * @returns {Observable<Application>}
     */
    removeRepository(key: string, currentName: string, repoManName: string): Observable<Application> {
        return this._applicationService.removeRepository(key, currentName, repoManName)
            .map(app => {
                let cache = this._application.getValue();
                let appKey = key + '-' + currentName;
                if (cache.get(appKey)) {
                    let pToUpdate = cache.get(appKey);
                    pToUpdate.last_modified = app.last_modified;
                    delete pToUpdate.repositories_manager;
                    delete pToUpdate.repository_fullname;
                    this._application.next(cache.set(appKey, pToUpdate));
                }
                return app;
            });
    }

    /**
     * Add poller to the application for the given pipeline
     * @param key Project unique key
     * @param appName Application name
     * @param pipName Pipeline name
     * @param poller Poller
     * @returns {Observable<Application>}
     */
    addPoller(key: string, appName: string, pipName: string, poller: RepositoryPoller): Observable<Application> {
        return this._applicationService.addPoller(key, appName, pipName, poller).map(app => {
            return this.refreshApplicationPollerCache(key, appName, app);
        });
    }

    /**
     * Delete the given poller from the application
     * @param key Project unique key
     * @param appName Application name
     * @param poller Poller to delete
     * @returns {Observable<Application>}
     */
    deletePoller(key: string, appName: string, poller: RepositoryPoller): Observable<Application> {
        return this._applicationService.deletePoller(key, appName, poller.pipeline.name).map( app => {
           return this.refreshApplicationPollerCache(key, appName, app);
        });
    }

    /**
     * Refresh application cache
     * @param key Project unique key
     * @param appName Application Name
     * @param application updated poller application
     * @returns {: Application}
     */
    refreshApplicationPollerCache(key: string, appName: string, application: Application): Application {
        let cache = this._application.getValue();
        let appKey = key + '-' + appName;
        let appToUpdate = cache.get(appKey);
        if (appToUpdate) {
            appToUpdate.last_modified = application.last_modified;
            appToUpdate.pollers = application.pollers;
            this._application.next(cache.set(appKey, appToUpdate));
            return appToUpdate;
        }
        return application;
    }

    /**
     * Add a hook for the current application on the given pipeline name
     * @param p Project
     * @param a Application
     * @param hook Hook to add
     * @returns {Observable<Application>}
     */
    addHook(p: Project, a: Application, hook: Hook): Observable<Application> {
        return this._applicationService.addHook(p.key, a.name, a.repositories_manager.name, a.repository_fullname, hook.pipeline.name)
            .map( app => {
                return this.refreshApplicationHookCache(p.key, a.name, app);
        });
    }

    /**
     * Remove a hook from an application
     * @param p Project
     * @param a Application
     * @param h Hook to delete
     * @returns {Observable<Application>}
     */
    removeHook(p: Project, a: Application, h: Hook): Observable<Application> {
        return this._applicationService.deleteHook(p.key, a.name, a.repositories_manager.name, h.id).map(app => {
            return this.refreshApplicationHookCache(p.key, a.name, app);
        });
    }

    /**
     * Refresh application cache
     * @param key Project unique key
     * @param appName Application Name
     * @param application updated hook application
     * @returns {: Application}
     */
    refreshApplicationHookCache(key: string, appName: string, application: Application): Application {
        let cache = this._application.getValue();
        let appKey = key + '-' + appName;
        let appToUpdate = cache.get(appKey);
        if (appToUpdate) {
            appToUpdate.last_modified = application.last_modified;
            appToUpdate.hooks = application.hooks;
            this._application.next(cache.set(appKey, appToUpdate));
            return appToUpdate;
        }
        return application;
    }

    /**
     * Add a variable in the application
     * @param key Project unique key
     * @param appName Application name
     * @param v Variable to add
     * @returns {Observable<Application>}
     */
    addVariable(key: string, appName: string, v: Variable): Observable<Application> {
        return this._applicationService.addVariable(key, appName, v).map(app => {
            return this.refreshApplicationVariableCache(key, appName, app);
        });
    }

    /**
     * Update a variable in the application
     * @param key Project unique key
     * @param appName Application name
     * @param v Variable to update
     * @returns {Observable<Application>}
     */
    updateVariable(key: string, appName: string, v: Variable): Observable<Application> {
        return this._applicationService.updateVariable(key, appName, v).map(app => {
            return this.refreshApplicationVariableCache(key, appName, app);
        });
    }

    /**
     * Delete a variable from the application
     * @param key Project unique key
     * @param appName Application name
     * @param v Variable to delete
     * @returns {Observable<Application>}
     */
    removeVariable(key: string, appName: string, v: Variable): Observable<Application> {
        return this._applicationService.removeVariable(key, appName, v).map(app => {
            return this.refreshApplicationVariableCache(key, appName, app);
        });
    }

    /**
     * Refresh application cache
     * @param key Project unique key
     * @param appName Application Name
     * @param application updated variable application
     * @returns {: Application}
     */
    refreshApplicationVariableCache(key: string, appName: string, application: Application): Application {
        let cache = this._application.getValue();
        let appKey = key + '-' + appName;
        let appToUpdate = cache.get(appKey);
        if (appToUpdate) {
            appToUpdate.last_modified = application.last_modified;
            appToUpdate.variables = application.variables;
            this._application.next(cache.set(appKey, appToUpdate));
            return appToUpdate;
        }
        return application;
    }

    /**
     * Add a permission on the given application
     * @param key Project unique key
     * @param appName Application name
     * @param gp Permission to add
     * @returns {Observable<Application>}
     */
    addPermission(key: string, appName: string, gp: GroupPermission): Observable<Application> {
        return this._applicationService.addPermission(key, appName, gp).map(app => {
            return this.refreshApplicationPermissionCache(key, appName, app);
        });
    }

    /**
     * Update a permission on the given application
     * @param key Project unique key
     * @param appName Application name
     * @param gp Permission to update
     * @returns {Observable<Application>}
     */
    updatePermission(key: string, appName: string, gp: GroupPermission): Observable<Application> {
        return this._applicationService.updatePermission(key, appName, gp).map(app => {
            return this.refreshApplicationPermissionCache(key, appName, app);
        });
    }

    /**
     * Remove a permission from the given application
     * @param key Project unique key
     * @param appName Application name
     * @param gp Permission to remove
     * @returns {Observable<Application>}
     */
    removePermission(key: string, appName: string, gp: GroupPermission): Observable<Application> {
        return this._applicationService.removePermission(key, appName, gp).map(app => {
            return this.refreshApplicationPermissionCache(key, appName, app);
        });
    }

    /**
     * Refresh application cache
     * @param key Project unique key
     * @param appName Application Name
     * @param application updated permissions application
     * @returns {: Application}
     */
    refreshApplicationPermissionCache(key: string, appName: string, application: Application): Application {
        let cache = this._application.getValue();
        let appKey = key + '-' + appName;
        let appToUpdate = cache.get(appKey);
        if (appToUpdate) {
            appToUpdate.last_modified = application.last_modified;
            appToUpdate.groups = application.groups;
            this._application.next(cache.set(appKey, appToUpdate));
            return appToUpdate;
        }
        return application;
    }

    /**
     * Add a trigger on the given application/pipeline
     * @param key Project unique key
     * @param appName Application name
     * @param pipName PIpeline name
     * @param t Trigger to add
     * @returns {Observable<Application>}
     */
    addTrigger(key: string, appName: string, pipName: string, t: Trigger): Observable<Application> {
        return this._applicationService.addTrigger(key, appName, pipName, t).map( app => {
            return this.refreshApplicationWorkflowCache(key, appName, app);
        });
    }

    /**
     * UPdate a trigger on the given application/pipeline
     * @param key Project unique key
     * @param appName Application name
     * @param pipName PIpeline name
     * @param t Trigger to update
     * @returns {Observable<Application>}
     */
    updateTrigger(key: string, appName: string, pipName: string, t: Trigger): Observable<Application> {
        return this._applicationService.updateTrigger(key, appName, pipName, t).map( app => {
            return this.refreshApplicationWorkflowCache(key, appName, app);
        });
    }

    /**
     * Remove the given trigger
     * @param key Project unique key
     * @param appName Application name
     * @param pipName PIpeline name
     * @param t Trigger to remove
     * @returns {Observable<Application>}
     */
    removeTrigger(key: string, appName: string, pipName: string, t: Trigger): Observable<Application> {
        return this._applicationService.removeTrigger(key, appName, pipName, t).map( app => {
            return this.refreshApplicationWorkflowCache(key, appName, app);
        });
    }

    /**
     * Refresh application cache
     * @param key Project unique key
     * @param appName Application Name
     * @param application updated workflow application
     * @returns {: Application}
     */
    refreshApplicationWorkflowCache(key: string, appName: string, application: Application): Application {
        let cache = this._application.getValue();
        let appKey = key + '-' + appName;
        let appToUpdate = cache.get(appKey);
        if (appToUpdate) {
            appToUpdate.last_modified = application.last_modified;
            appToUpdate.workflows = application.workflows;
            this._application.next(cache.set(appKey, appToUpdate));
            return appToUpdate;
        }
        return application;
    }

}
