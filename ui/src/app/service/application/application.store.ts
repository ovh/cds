import { Injectable } from '@angular/core';
import * as immutable from 'immutable';
import { BehaviorSubject, Observable, of as observableOf } from 'rxjs';
import { map } from 'rxjs/operators';
import { Application, Vulnerability } from '../../model/application.model';
import { GroupPermission } from '../../model/group.model';
import { Hook } from '../../model/hook.model';
import { Key } from '../../model/keys.model';
import { NavbarRecentData } from '../../model/navbar.model';
import { Notification } from '../../model/notification.model';
import { RepositoryPoller } from '../../model/polling.model';
import { Project } from '../../model/project.model';
import { Scheduler } from '../../model/scheduler.model';
import { Trigger } from '../../model/trigger.model';
import { Variable } from '../../model/variable.model';
import { ProjectStore } from '../project/project.store';
import { WorkflowStore } from '../workflow/workflow.store';
import { ApplicationService } from './application.service';




@Injectable()
export class ApplicationStore {

    static RECENT_APPLICATIONS_KEY = 'CDS-RECENT-APPLICATIONS';

    // immutable.List of all applications.
    private _application: BehaviorSubject<immutable.Map<string, Application>> =
        new BehaviorSubject(immutable.Map<string, Application>());

    private _recentApplications: BehaviorSubject<immutable.List<NavbarRecentData>> =
        new BehaviorSubject(immutable.List<NavbarRecentData>());


    constructor(
      private _applicationService: ApplicationService,
      private _projectStore: ProjectStore,
      private _workflowStore: WorkflowStore) {
        this.loadRecentApplication();

    }

    loadRecentApplication(): void {
        let arrayApp = JSON.parse(localStorage.getItem(ApplicationStore.RECENT_APPLICATIONS_KEY));
        this._recentApplications.next(immutable.List.of(...arrayApp));
    }

    /**
     * Get recent application.
     * @returns {Observable<immutable.List<Application>>}
     */
    getRecentApplications(): Observable<immutable.List<Application>> {
        return new Observable<immutable.List<Application>>(fn => this._recentApplications.subscribe(fn));
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
            return this._applicationService.getApplication(key, appName).pipe(map( res => {
                this._application.next(store.set(appKey, res));
                return res;
            }));
        } else {
            return observableOf(store.get(appKey));
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
        this._recentApplications.next(immutable.List(currentRecentApps));
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
    getApplications(key: string, appName: string):
        Observable<immutable.Map<string, Application>> {

        let store = this._application.getValue();
        let appKey = key + '-' + appName;
        if (appName && !store.get(appKey)) {
            this.resync(key, appName);
        }
        return new Observable<immutable.Map<string, Application>>(fn => this._application.subscribe(fn));
    }

    resync(key: string, appName: string) {
        let store = this._application.getValue();
        let appKey = key + '-' + appName;
        this._applicationService.getApplication(key, appName).subscribe(res => {
            this._application.next(store.set(appKey, res));
        }, err => {
            this._application.error(err);
            this._application = new BehaviorSubject(immutable.Map<string, Application>());
            this._application.next(store);
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
        return this._applicationService.cloneApplication(key, appName, app).pipe(map(appResult => appResult));
    }

    /**
     * Create application
     * @param key Project unique key
     * @param app New application data
     * @returns Observable<Application>
     */
    createApplication(key: string, app: Application): Observable<Application> {
        return this._applicationService.createApplication(key, app).pipe(map(appResult => appResult));
    }

    /**
     * Update the given application
     * @param key Project unique key
     * @param oldName Old application name
     * @param application Application to update
     * @returns {Observable<Application>}
     */
    updateApplication(key: string, oldName: string, appli: Application): Observable<Application> {
        return this._applicationService.updateApplication(key, oldName, appli).pipe(map(app => {
            let cache = this._application.getValue();
            let appKey = key + '-' + oldName;
            if (cache.get(appKey)) {
                let pToUpdate = cache.get(appKey);
                pToUpdate.last_modified = app.last_modified;
                pToUpdate.name = app.name;
                pToUpdate.description = app.description;
                this._application.next(cache.set(key + '-' + app.name, pToUpdate).remove(appKey));
            }
            if (oldName !== appli.name) {
                this._projectStore.updateApplicationName(key, oldName, appli.name);
            }
            return app;
        }));
    }

    /**
     * Delete an application
     * @param key Project unique key
     * @param appName Application name to delete
     * @returns {Observable<boolean>}
     */
    deleteApplication(key: string, appName: string): Observable<boolean> {
        return this._applicationService.deleteApplication(key, appName).pipe(map(res => {

            // Remove from application cache
            let appKey = key + '-' + appName;
            this.removeFromStore(appKey);

            // Remove from recent application
            let recentApp = this._recentApplications
                .getValue()
                .toArray()
                .filter((app) => !(app.name === appName && app.project_key === key));

            this._recentApplications.next(immutable.List(recentApp));

            return res;
        }));
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
        return this._applicationService.connectRepository(key, currentName, repoManName, repoFullname).pipe(
            map(app => {
                let cache = this._application.getValue();
                let appKey = key + '-' + currentName;
                let appToUpdate = cache.get(appKey);
                if (appToUpdate) {
                    appToUpdate.last_modified = app.last_modified;
                    appToUpdate.vcs_server = app.vcs_server;
                    appToUpdate.repository_fullname = app.repository_fullname;
                    this._application.next(cache.set(appKey, appToUpdate));
                    if (appToUpdate.usage && Array.isArray(appToUpdate.usage.workflows)) {
                        appToUpdate.usage.workflows.forEach((wf) => this._workflowStore.removeFromStore(key + '-' + wf.name));
                    }
                }
                return app;
            }));
    }

    /**
     * Remove the attached repository
     * @param application Application
     * @returns {Observable<Application>}
     */
    removeRepository(key: string, currentName: string, repoManName: string): Observable<Application> {
        return this._applicationService.removeRepository(key, currentName, repoManName).pipe(
            map(app => {
                let cache = this._application.getValue();
                let appKey = key + '-' + currentName;
                if (cache.get(appKey)) {
                    let pToUpdate = cache.get(appKey);
                    pToUpdate.last_modified = app.last_modified;
                    delete pToUpdate.vcs_server;
                    delete pToUpdate.repository_fullname;
                    this._application.next(cache.set(appKey, pToUpdate));
                    if (pToUpdate.usage && Array.isArray(pToUpdate.usage.workflows)) {
                        pToUpdate.usage.workflows.forEach((wf) => this._workflowStore.removeFromStore(key + '-' + wf.name));
                    }
                }
                return app;
            }));
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
        return this._applicationService.addPoller(key, appName, pipName, poller).pipe(map(app => {
            return this.refreshApplicationWorkflowCache(key, appName, app);
        }));
    }

    /**
     * Update given poller
     * @param key Project key
     * @param appName Application name
     * @param poller Poller to update
     * @returns {Observable<Application>}
     */
    updatePoller(key: string, appName: string, pipName: string, poller: RepositoryPoller): Observable<Application> {
        return this._applicationService.updatePoller(key, appName, pipName, poller).pipe(map(app => {
            return this.refreshApplicationWorkflowCache(key, appName, app);
        }));
    }

    /**
     * Delete the given poller from the application
     * @param key Project unique key
     * @param appName Application name
     * @param poller Poller to delete
     * @returns {Observable<Application>}
     */
    deletePoller(key: string, appName: string, poller: RepositoryPoller): Observable<Application> {
        return this._applicationService.deletePoller(key, appName, poller.pipeline.name).pipe(map( app => {
           return this.refreshApplicationWorkflowCache(key, appName, app);
        }));
    }

    /**
     * Add a hook for the current application on the given pipeline name
     * @param p Project
     * @param a Application
     * @param hook Hook to add
     * @returns {Observable<Application>}
     */
    addHook(p: Project, a: Application, hook: Hook): Observable<Application> {
        return this._applicationService.addHook(p.key, a.name, a.vcs_server, a.repository_fullname, hook.pipeline.name).pipe(
            map( app => {
                return this.refreshApplicationWorkflowCache(p.key, a.name, app);
        }));
    }

    /**
     * Update the given hook
     * @param p Project
     * @param a Application
     * @param h Hook to update
     * @returns {Observable<Application>}
     */
    updateHook(p: Project, a: Application, pipName: string, h: Hook): Observable<Application> {
        return this._applicationService.updateHook(p.key, a.name, pipName, h).pipe(
            map( app => {
                return this.refreshApplicationWorkflowCache(p.key, a.name, app);
            }));
    }

    /**
     * Remove a hook from an application
     * @param p Project
     * @param a Application
     * @param h Hook to delete
     * @returns {Observable<Application>}
     */
    removeHook(p: Project, a: Application, h: Hook): Observable<Application> {
        return this._applicationService.deleteHook(p.key, a.name, h.id).pipe(map(app => {
            return this.refreshApplicationWorkflowCache(p.key, a.name, app);
        }));
    }

    /**
     * Add a variable in the application
     * @param key Project unique key
     * @param appName Application name
     * @param v Variable to add
     * @returns {Observable<Application>}
     */
    addVariable(key: string, appName: string, v: Variable): Observable<Application> {
        return this._applicationService.addVariable(key, appName, v).pipe(map(app => {
            return this.refreshApplicationVariableCache(key, appName, app);
        }));
    }

    /**
     * Update a variable in the application
     * @param key Project unique key
     * @param appName Application name
     * @param v Variable to update
     * @returns {Observable<Application>}
     */
    updateVariable(key: string, appName: string, v: Variable): Observable<Application> {
        return this._applicationService.updateVariable(key, appName, v).pipe(map(app => {
            return this.refreshApplicationVariableCache(key, appName, app);
        }));
    }

    /**
     * Delete a variable from the application
     * @param key Project unique key
     * @param appName Application name
     * @param v Variable to delete
     * @returns {Observable<Application>}
     */
    removeVariable(key: string, appName: string, v: Variable): Observable<Application> {
        return this._applicationService.removeVariable(key, appName, v).pipe(map(app => {
            return this.refreshApplicationVariableCache(key, appName, app);
        }));
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
        return this._applicationService.addPermission(key, appName, gp).pipe(map(app => {
            return this.refreshApplicationPermissionCache(key, appName, app);
        }));
    }

    /**
     * Update a permission on the given application
     * @param key Project unique key
     * @param appName Application name
     * @param gp Permission to update
     * @returns {Observable<Application>}
     */
    updatePermission(key: string, appName: string, gp: GroupPermission): Observable<Application> {
        return this._applicationService.updatePermission(key, appName, gp).pipe(map(app => {
            return this.refreshApplicationPermissionCache(key, appName, app);
        }));
    }

    /**
     * Remove a permission from the given application
     * @param key Project unique key
     * @param appName Application name
     * @param gp Permission to remove
     * @returns {Observable<Application>}
     */
    removePermission(key: string, appName: string, gp: GroupPermission): Observable<Application> {
        return this._applicationService.removePermission(key, appName, gp).pipe(map(app => {
            return this.refreshApplicationPermissionCache(key, appName, app);
        }));
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
        return this._applicationService.addTrigger(key, appName, pipName, t).pipe(map( app => {
            return this.refreshApplicationWorkflowCache(key, appName, app);
        }));
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
        return this._applicationService.updateTrigger(key, appName, pipName, t).pipe(map( app => {
            return this.refreshApplicationWorkflowCache(key, appName, app);
        }));
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
        return this._applicationService.removeTrigger(key, appName, pipName, t).pipe(map( app => {
            return this.refreshApplicationWorkflowCache(key, appName, app);
        }));
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
            this._application.next(cache.set(appKey, appToUpdate));
            return appToUpdate;
        }
        return application;
    }

    /**
     * Add a lit of notification on the application
     * @param key Project unique key
     * @param appName Application name
     * @param notifications immutable.List of notifications to add
     * @returns {Uint8Array|Uint8ClampedArray|Application[]|Int8Array|Promise<any[]>|Int32Array|any}
     */
    addNotifications(key: string, appName: string, notifications: Array<Notification>): Observable<Application> {
        return this._applicationService.addNotifications(key, appName, notifications).pipe(map( app => {
           return this.refreshApplicationNotificationsCache(key, appName, app);
        }));
    }

    updateNotification(key: string, appName: string, pipName: string, notification: Notification): Observable<Application> {
        return this._applicationService.updateNotification(key, appName, pipName, notification).pipe(map( app => {
            return this.refreshApplicationNotificationsCache(key, appName, app);
        }));
    }

    /**
     * Update a notification
     * @param key Project unique key
     * @param appName Application name
     * @param pipName Pipeline name
     */
    deleteNotification(key: string, appName: string, pipName: string, envName?: string): Observable<Application> {
        return this._applicationService.deleteNotification(key, appName, pipName, envName).pipe(map( app => {
            return this.refreshApplicationNotificationsCache(key, appName, app);
        }));
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
        return this._applicationService.attachPipelines(key, appName, pipelines).pipe(map( app => {
            return this.refreshApplicationPipelineCache(key, appName, app);
        }));
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
        return this._applicationService.detachPipelines(key, appName, pipName).pipe(map( app => {
            return this.refreshApplicationPipelineCache(key, appName, app);
        }));
    }

    /**
     * Add a key on the application
     * @param key Project unique key
     * @param appName Application name
     * @param k Key to add
     * @returns {OperatorFunction<Key>}
     */
    addKey(key: string, appName: string, k: Key): Observable<Key> {
        return this._applicationService.addKey(key, appName, k).pipe(map( () => {
            let cache = this._application.getValue();
            let appKey = key + '-' + appName;
            let appToUpdate = cache.get(appKey);
            if (appToUpdate) {
                if (!appToUpdate.keys) {
                    appToUpdate.keys = new Array<Key>();
                }
                appToUpdate.keys.push(k);
                this._application.next(cache.set(appKey, appToUpdate));
            }
            return k;
        }));
    }

    /**
     * Remove a key from the application
     * @param key Project unique key
     * @param appName Application name
     * @param k Key to remove
     * @returns {OperatorFunction<boolean>}
     */
    removeKey(key: string, appName: string, k: Key): Observable<boolean> {
        return this._applicationService.removeKey(key, appName, k.name).pipe(map( () => {
            let cache = this._application.getValue();
            let appKey = key + '-' + appName;
            let appToUpdate = cache.get(appKey);
            if (appToUpdate) {
                let i = appToUpdate.keys.findIndex(kkey => kkey.name === k.name);
                if (i > -1) {
                    appToUpdate.keys.splice(i, 1);
                }
                this._application.next(cache.set(appKey, appToUpdate));
            }
            return true;
        }));
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
        return this._applicationService.addScheduler(key, appName, pipName, scheduler).pipe(map( app => {
            return this.refreshApplicationWorkflowCache(key, appName, app);
        }));
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
        return this._applicationService.updateScheduler(key, appName, pipName, scheduler).pipe(map( app => {
            return this.refreshApplicationWorkflowCache(key, appName, app);
        }));
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
        return this._applicationService.deleteScheduler(key, appName, pipName, scheduler).pipe(map( app => {
            return this.refreshApplicationWorkflowCache(key, appName, app);
        }));
    }

    /**
     * Delete a deployment strategy
     * @param key Project unique key
     * @param appName Application name
     * @param pfName Platform Name
     * @param pfConfig Platform config
     * @returns {Observable<Application>}
     */
    saveDeploymentStrategy(key: string, appName: string, pfName: string, pfConfig: any): Observable<Application> {
        return this._applicationService.saveDeploymentStrategy(key, appName, pfName, pfConfig).pipe(map(app => {
            return this.refreshApplicationDeploymentCache(key, appName, app);
        }));
    }

    /**
     * Delete a deployment strategy
     * @param key Project unique key
     * @param appName Application name
     * @param pfName Platform Name
     * @returns {Observable<Application>}
     */
    deleteDeploymentStrategy(key: string, appName: string, pfName: string): Observable<Application> {
        return this._applicationService.deleteDeploymentStrategy(key, appName, pfName).pipe(map(app => {
            return this.refreshApplicationDeploymentCache(key, appName, app);
        }));
    }

    /**
     * Get deployment strategies map for an application
     * @param key Project unique key
     * @param appName Application name
     * @returns {Observable<Map<string, any>>}
     */
    getDeploymentStrategies(key: string, appName: string): Observable<Map<string, any>> {
        return this._applicationService.getDeploymentStrategies(key, appName);
    }

    /**
     * Refresh application cache
     * @param key Project unique key
     * @param appName Application Name
     * @param application updated workflow application
     * @returns {: Application}
     */
    refreshApplicationDeploymentCache(key: string, appName: string, application: Application): Application {
        let cache = this._application.getValue();
        let appKey = key + '-' + appName;
        let appToUpdate = cache.get(appKey);
        if (appToUpdate) {
            appToUpdate.last_modified = application.last_modified;
            appToUpdate.deployment_strategies = application.deployment_strategies;
            this._application.next(cache.set(appKey, appToUpdate));
            return appToUpdate;
        }
        return application;
    }

    ignoreVulnerability(key: string, appName: string, vulnerability: Vulnerability): Observable<Vulnerability> {
        return this._applicationService.ignoreVulnerability(key, appName, vulnerability);
    }
}
