import {Injectable} from '@angular/core';
import {Map, List} from 'immutable';
import {Observable} from 'rxjs/Observable';
import {BehaviorSubject} from 'rxjs/BehaviorSubject'
import {Application} from '../../model/application.model';
import {ApplicationService} from './application.service';
import {RepositoryPoller} from '../../model/polling.model';
import {Project} from '../../model/project.model';
import {Hook} from '../../model/hook.model';
import {Variable} from '../../model/variable.model';
import {GroupPermission} from '../../model/group.model';
import {ProjectStore} from '../project/project.store';
import {Trigger} from '../../model/trigger.model';
import {NavbarRecentData} from '../../model/navbar.model';
import {Notification} from '../../model/notification.model';
import {Scheduler} from '../../model/scheduler.model';
import 'rxjs/add/observable/of';


@Injectable()
export class ApplicationStore {

    static RECENT_APPLICATIONS_KEY = 'CDS-RECENT-APPLICATIONS';

    // List of all applications.
    private _application: BehaviorSubject<Map<string, Application>> = new BehaviorSubject(Map<string, Application>());

    private _recentApplications: BehaviorSubject<List<NavbarRecentData>> = new BehaviorSubject(List<NavbarRecentData>());


    constructor(private _applicationService: ApplicationService, private _projectStore: ProjectStore) {
        this.loadRecentApplication();

    }

    loadRecentApplication(): void {
        let arrayApp = JSON.parse(localStorage.getItem(ApplicationStore.RECENT_APPLICATIONS_KEY));
        this._recentApplications.next(List.of(...arrayApp));
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
    getApplicationResolver(key: string, appName: string, filter?: {branch: string, remote: string}): Observable<Application> {
        let store = this._application.getValue();
        let appKey = key + '-' + appName;

        if (store.size === 0 || !store.get(appKey)) {
            return this._applicationService.getApplication(key, appName, filter).map( res => {
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
        let navbarRecentData = new NavbarRecentData();
        navbarRecentData.project_key = key;
        navbarRecentData.name = application.name;
        let currentRecentApps: Array<NavbarRecentData> = JSON.parse(localStorage.getItem(ApplicationStore.RECENT_APPLICATIONS_KEY));
        if (currentRecentApps) {
            let index: number = currentRecentApps.findIndex(app =>
                app.name === navbarRecentData.name && app.project_key === navbarRecentData.project_key
            );
            if (index >= 0) {
                currentRecentApps.splice(index, 1);
            }
        } else {
            currentRecentApps = new Array<NavbarRecentData>();
        }
        currentRecentApps.splice(0, 0, navbarRecentData);
        currentRecentApps = currentRecentApps.splice(0, 15);
        localStorage.setItem(ApplicationStore.RECENT_APPLICATIONS_KEY, JSON.stringify(currentRecentApps));
        this._recentApplications.next(List(currentRecentApps));
    }

    externalModification(appKey: string) {
        let cache = this._application.getValue();
        let appToUpdate = cache.get(appKey);
        if (appToUpdate) {
            appToUpdate.externalChange = true;
            this._application.next(cache.set(appKey, appToUpdate));
        }
    }

    /**
     * Get an Application
     * @returns {Observable<Application>}
     */
    getApplications(key: string, appName: string, filter?: {branch: string, remote: string}): Observable<Map<string, Application>> {
        let store = this._application.getValue();
        let appKey = key + '-' + appName;
        if (appName && !store.get(appKey)) {
            this.resync(key, appName, filter);
        }
        return new Observable<Map<string, Application>>(fn => this._application.subscribe(fn));
    }

    resync(key: string, appName: string, filter?: {branch: string, remote: string}) {
        let store = this._application.getValue();
        let appKey = key + '-' + appName;
        this._applicationService.getApplication(key, appName, filter).subscribe(res => {
            this._application.next(store.set(appKey, res));
        }, err => {
            this._application.error(err);
        });
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
     * Create application
     * @param key Project unique key
     * @param app New application data
     * @returns Observable<Application>
     */
    createApplication(key: string, app: Application): Observable<Application> {
        return this._applicationService.createApplication(key, app).map(appResult => appResult);
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

            // Remove from application cache
            let appKey = key + '-' + appName;
            this.removeFromStore(appKey);

            // Remove from recent application
            let recentApp = this._recentApplications
                .getValue()
                .toArray()
                .filter((app) => !(app.name === appName && app.project_key === key));

            this._recentApplications.next(List(recentApp));

            return res;
        });
    }

    removeFromStore(appKey: string) {
        let cache = this._application.getValue();
        this._application.next(cache.delete(appKey));
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
                    appToUpdate.vcs_server = app.vcs_server;
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
                    delete pToUpdate.vcs_server;
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
            return this.refreshApplicationWorkflowCache(key, appName, app);
        });
    }

    /**
     * Update given poller
     * @param key Project key
     * @param appName Application name
     * @param poller Poller to update
     * @returns {Observable<Application>}
     */
    updatePoller(key: string, appName: string, pipName: string, poller: RepositoryPoller): Observable<Application> {
        return this._applicationService.updatePoller(key, appName, pipName, poller).map(app => {
            return this.refreshApplicationWorkflowCache(key, appName, app);
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
           return this.refreshApplicationWorkflowCache(key, appName, app);
        });
    }

    /**
     * Add a hook for the current application on the given pipeline name
     * @param p Project
     * @param a Application
     * @param hook Hook to add
     * @returns {Observable<Application>}
     */
    addHook(p: Project, a: Application, hook: Hook): Observable<Application> {
        return this._applicationService.addHook(p.key, a.name, a.vcs_server, a.repository_fullname, hook.pipeline.name)
            .map( app => {
                return this.refreshApplicationWorkflowCache(p.key, a.name, app);
        });
    }

    /**
     * Update the given hook
     * @param p Project
     * @param a Application
     * @param h Hook to update
     * @returns {Observable<Application>}
     */
    updateHook(p: Project, a: Application, pipName: string, h: Hook): Observable<Application> {
        return this._applicationService.updateHook(p.key, a.name, pipName, h)
            .map( app => {
                return this.refreshApplicationWorkflowCache(p.key, a.name, app);
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
        return this._applicationService.deleteHook(p.key, a.name, h.id).map(app => {
            return this.refreshApplicationWorkflowCache(p.key, a.name, app);
        });
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

    /**
     * Add a lit of notification on the application
     * @param key Project unique key
     * @param appName Application name
     * @param notifications List of notifications to add
     * @returns {Uint8Array|Uint8ClampedArray|Application[]|Int8Array|Promise<any[]>|Int32Array|any}
     */
    addNotifications(key: string, appName: string, notifications: Array<Notification>): Observable<Application> {
        return this._applicationService.addNotifications(key, appName, notifications).map( app => {
           return this.refreshApplicationNotificationsCache(key, appName, app);
        });
    }

    updateNotification(key: string, appName: string, pipName: string, notification: Notification): Observable<Application> {
        return this._applicationService.updateNotification(key, appName, pipName, notification).map( app => {
            return this.refreshApplicationNotificationsCache(key, appName, app);
        });
    }

    /**
     * Update a notification
     * @param key Project unique key
     * @param appName Application name
     * @param pipName Pipeline name
     */
    deleteNotification(key: string, appName: string, pipName: string, envName?: string): Observable<Application> {
        return this._applicationService.deleteNotification(key, appName, pipName, envName).map( app => {
            return this.refreshApplicationNotificationsCache(key, appName, app);
        });
    }

    /**
     * Refresh application cache
     * @param key Project unique key
     * @param appName Application Name
     * @param application updated workflow application
     * @returns {: Application}
     */
    refreshApplicationNotificationsCache(key: string, appName: string, application: Application): Application {
        let cache = this._application.getValue();
        let appKey = key + '-' + appName;
        let appToUpdate = cache.get(appKey);
        if (appToUpdate) {
            appToUpdate.last_modified = application.last_modified;
            appToUpdate.notifications = application.notifications;
            this._application.next(cache.set(appKey, appToUpdate));
            return appToUpdate;
        }
        return application;
    }

    /**
     * Attach pipelines to application
     * @param key Project unique key
     * @param appName Application name
     * @param pipelines Array of pipeline name
     * @returns {Observable<Application>}
     */
    attachPipelines(key: string, appName: string, pipelines: Array<string>): Observable<Application> {
        return this._applicationService.attachPipelines(key, appName, pipelines).map( app => {
            return this.refreshApplicationPipelineCache(key, appName, app);
        });
    }

    /**
     * Refresh application cache
     * @param key Project unique key
     * @param appName Application Name
     * @param application updated workflow application
     * @returns {: Application}
     */
    refreshApplicationPipelineCache(key: string, appName: string, application: Application): Application {
        let cache = this._application.getValue();
        let appKey = key + '-' + appName;
        let appToUpdate = cache.get(appKey);
        if (appToUpdate) {
            appToUpdate.last_modified = application.last_modified;
            appToUpdate.pipelines = application.pipelines;
            appToUpdate.workflows = application.workflows;
            this._application.next(cache.set(appKey, appToUpdate));
            return appToUpdate;
        }
        return application;
    }

    /**
     * Detach a pipeline from application
     * @param key Project unique key
     * @param appName Application name
     * @param pipName Pipeline name to detach
     * @returns {Observable<Application>}
     */
    detachPipeline(key: string, appName: string, pipName: string): Observable<Application> {
        return this._applicationService.detachPipelines(key, appName, pipName).map( app => {
            return this.refreshApplicationPipelineCache(key, appName, app);
        });
    }

    /**
     * Add a scheduler
     * @param key Project unique key
     * @param appName Application name
     * @param pipName Pipeline Name
     * @param scheduler Scheduler to add
     * @returns {Observable<Application>}
     */
    addScheduler(key: string, appName: string, pipName: string, scheduler: Scheduler): Observable<Application> {
        return this._applicationService.addScheduler(key, appName, pipName, scheduler).map( app => {
            return this.refreshApplicationWorkflowCache(key, appName, app);
        });
    }

    /**
     * Update a scheduler
     * @param key Project unique key
     * @param appName Application name
     * @param pipName Pipeline Name
     * @param scheduler Scheduler to update
     * @returns {Observable<Application>}
     */
    updateScheduler(key: string, appName: string, pipName: string, scheduler: Scheduler): Observable<Application> {
        return this._applicationService.updateScheduler(key, appName, pipName, scheduler).map( app => {
            return this.refreshApplicationWorkflowCache(key, appName, app);
        });
    }

    /**
     * Delete a scheduler
     * @param key Project unique key
     * @param appName Application name
     * @param pipName Pipeline Name
     * @param scheduler Scheduler to update
     * @returns {Observable<Application>}
     */
    deleteScheduler(key: string, appName: string, pipName: string, scheduler: Scheduler): Observable<Application> {
        return this._applicationService.deleteScheduler(key, appName, pipName, scheduler).map( app => {
            return this.refreshApplicationWorkflowCache(key, appName, app);
        });
    }
}
