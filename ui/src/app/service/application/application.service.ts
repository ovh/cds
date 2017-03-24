import {Injectable} from '@angular/core';
import {Http, RequestOptions, Headers, URLSearchParams} from '@angular/http';
import {Observable} from 'rxjs/Rx';
import {Application} from '../../model/application.model';
import {PipelineBuild, Pipeline} from '../../model/pipeline.model';
import {Variable} from '../../model/variable.model';
import {RepositoryPoller} from '../../model/polling.model';
import {GroupPermission} from '../../model/group.model';
import {Trigger} from '../../model/trigger.model';
import {ApplyTemplateRequest} from '../../model/template.model';
import {Project} from '../../model/project.model';
import {Notification} from '../../model/notification.model';
import {Scheduler} from '../../model/scheduler.model';
import {Hook} from '../../model/hook.model';

@Injectable()
export class ApplicationService {

    constructor(private _http: Http) {
    }

    /**
     * Get the given application from API
     * @param key Project unique key
     * @param appName Application Name
     */
    getApplication(key: string, appName: string): Observable<Application> {
        let options = new RequestOptions();
        options.search = new URLSearchParams();
        options.search.set('withPollers', 'true');
        options.search.set('withHooks', 'true');
        options.search.set('withWorkflow', 'true');
        options.search.set('withNotifs', 'true');
        options.search.set('withRepoMan', 'true');
        return this._http.get('/project/' + key + '/application/' + appName, options).map(res => res.json());
    }

    /**
     * Update the given application
     * @param key Project unique key
     * @param application Application to update
     * @returns {Observable<Application>}
     */
    renameApplication(key: string, appOldName: string, appNewName: string): Observable<Application> {
        let appRenamed = new Application();
        appRenamed.name = appNewName;
        return this._http.put('/project/' + key + '/application/' + appOldName, appRenamed).map(res => res.json());
    }

    /**
     * Clone application
     * @param key Project unique key
     * @param appName Application to clone
     * @param application Application data
     * @returns {Observable<Application>}
     */
    cloneApplication(key: string, appName: string, application: Application): Observable<Application> {
        return this._http.post('/project/' + key + '/application/' + appName + '/clone', application).map(res => res.json());
    }

    /***
     * Apply a template to a project to create an application
     * @param key Project unique key
     */
    applyTemplate(key: string, request: ApplyTemplateRequest): Observable<Project> {
        return this._http.post('/project/' + key + '/template', request).map(res => res.json());
    }

    /**
     * Delete an application
     * @param key Project unique key
     * @param appName Application name to delete
     * @returns {Observable<boolean>}
     */
    deleteApplication(key: string, appName: string): Observable<boolean> {
        return this._http.delete('/project/' + key + '/application/' + appName).map(res => {
            return true;
        });
    }

    /**
     * Remove the given repository from the given application
     * @param key Project unique key
     * @param appName Application name
     * @param repoManName Repo manager name
     * @returns {Observable<Application>}
     */
    removeRepository(key: string, appName: string, repoManName: string): Observable<Application> {
        let url = '/project/' + key + '/repositories_manager/' + repoManName + '/application/' + appName + '/detach';
        return this._http.post(url, null).map(res => res.json());
    }

    /**
     * Connect the given repository to the application
     * @param key Project unique key
     * @param appName Application name
     * @param repoManName Repository manager name
     * @param repoFullName Repository fullname
     * @returns {Observable<Application>}
     */
    connectRepository(key: string, appName: string, repoManName: string, repoFullName: string): Observable<Application> {
        let url = '/project/' + key + '/repositories_manager/' + repoManName + '/application/' + appName + '/attach';

        let headers = new Headers({ 'Content-Type': 'application/x-www-form-urlencoded' });
        let options = new RequestOptions({ headers: headers });
        let params: URLSearchParams = new URLSearchParams();
        params.set('fullname', repoFullName);
        return this._http.post(url, params.toString(), options).map(res => res.json());
    }

    /**
     * Add a poller on the application for the given pipeline
     * @param key Project unique key
     * @param appName Application name
     * @param pipName Pipeline name
     * @param poller Poller to add
     * @returns {Observable<Application>}
     */
    addPoller(key: string, appName: string, pipName: string, poller: RepositoryPoller): Observable<Application> {
        let url = '/project/' + key + '/application/' + appName + '/pipeline/' + pipName + '/polling';
        return this._http.post(url, poller).map(res => res.json());
    }

    /**
     * Update the given poller
     * @param key Project unique key
     * @param appName Application name
     * @param poller Poller to update
     * @returns {Observable<Application>}
     */
    updatePoller(key: string, appName: string, pipName: string, poller: RepositoryPoller): Observable<Application> {
        let url = '/project/' + key + '/application/' + appName + '/pipeline/' + pipName + '/polling';
        return this._http.put(url, poller).map(res => res.json());
    }

    /**
     * Delete the poller from the given application
     * @param key Project unique key
     * @param appName Application name
     * @param pipName Pipeline name
     * @returns {Observable<Application>}
     */
    deletePoller(key: string, appName: string, pipName: string): Observable<Application> {
        let url = '/project/' + key + '/application/' + appName + '/pipeline/' + pipName + '/polling';
        return this._http.delete(url, null).map(res => res.json());
    }

    /**
     * Add a hook for the current application on the given pipeline name
     * @param key Project unique key
     * @param appName Applicatio name
     * @param repoManName Repository manager name
     * @param repoFullName Repository fullname
     * @param pipName Pipeline name
     * @returns {Observable<Application>}
     */
    addHook(key: string, appName: string, repoManName: string, repoFullName: string, pipName: string): Observable<Application> {
        let url = '/project/' + key + '/application/' + appName + '/repositories_manager/' + repoManName + '/hook';
        let request = {
            'repository_fullname': repoFullName,
            'pipeline_name' : pipName
        };
        return this._http.post(url, request).map(res => res.json());
    }

    /**
     * Update the given hook
     * @param key Project key
     * @param appName Application Name
     * @param pipName Pipeline name
     * @param hook Hook to update
     */
    updateHook(key: string, appName: string, pipName: string, hook: Hook) {
        let url = '/project/' + key + '/application/' + appName + '/pipeline/' + pipName + '/hook/' + hook.id;
        return this._http.put(url, hook).map(res => res.json());
    }

    /**
     * Delete a hook from the given application/repository manager
     * @param key Project unique key
     * @param appName Applicatio name
     * @param repoManName Repository manager name
     * @param hookId Hook id to delete
     * @returns {Observable<R>}
     */
    deleteHook(key: string, appName: string, repoManName: string, hookId: number): Observable<Application> {
        let url = '/project/' + key + '/application/' + appName + '/repositories_manager/' + repoManName + '/hook/' + hookId;
        return this._http.delete(url, null).map(res => res.json());
    }

    /**
     * Add a variable in the application
     * @param key Project unique key
     * @param appName Application name
     * @param v Variable to add
     * @returns {Observable<Application>}
     */
    addVariable(key: string, appName: string, v: Variable): Observable<Application> {
        let url = '/project/' + key + '/application/' + appName + '/variable/' + v.name;
        return this._http.post(url, v).map(res => res.json());
    }

    /**
     * Update a variable in the application
     * @param key Project unique key
     * @param appName Application name
     * @param v Variable to update
     * @returns {Observable<Application>}
     */
    updateVariable(key: string, appName: string, v: Variable): Observable<Application> {
        let url = '/project/' + key + '/application/' + appName + '/variable/' + v.name;
        return this._http.put(url, v).map(res => res.json());
    }

    /**
     * Delete a variable from the application
     * @param key Project unique key
     * @param appName Application name
     * @param v Variable to delete
     * @returns {Observable<Application>}
     */
    removeVariable(key: string, appName: string, v: Variable): Observable<Application> {
        let url = '/project/' + key + '/application/' + appName + '/variable/' + v.name;
        return this._http.delete(url).map(res => res.json());
    }

    /**
     * Add a permission on the application.
     * @param key Project unique key
     * @param appName Application name
     * @param gp Permission to add
     * @returns {Observable<Application>}
     */
    addPermission(key: string, appName: string, gp: GroupPermission): Observable<Application> {
        return this._http.post('/project/' + key + '/application/' + appName + '/group', gp).map(res => res.json());
    }

    /**
     * Update a permission.
     * @param key Project unique key
     * @param appName Application name
     * @param gp Permission to update
     * @returns {Observable<Application>}
     */
    updatePermission(key: string, appName: string, gp: GroupPermission): Observable<Application> {
        return this._http.put('/project/' + key + '/application/' + appName + '/group/' + gp.group.name, gp).map(res => res.json());
    }

    /**
     * Delete a permission.
     * @param key Project unique key
     * @param appName Application name
     * @param gp Permission to delete
     * @returns {Observable<Application>}
     */
    removePermission(key: string, appName: string, gp: GroupPermission): Observable<Application> {
        return this._http.delete('/project/' + key + '/application/' + appName + '/group/' + gp.group.name).map(res => res.json());
    }

    /**
     * Add a trigger on the given application/pipeline
     * @param key Project unique key
     * @param appName Application name
     * @param pipName Pipeline Name
     * @param t Trigger to add
     * @returns {Observable<Application>}
     */
    addTrigger(key: string, appName: string, pipName: string, t: Trigger): Observable<Application> {
        let url = '/project/' + key + '/application/' + appName + '/pipeline/' + pipName + '/trigger';
        return this._http.post(url, t).map(res => res.json());
    }

    /**
     * Update the given trigger
     * @param key Project unique key
     * @param appName Application name
     * @param pipName Pipeline Name
     * @param t Trigger to update
     * @returns {Observable<Application>}
     */
    updateTrigger(key: string, appName: string, pipName: string, t: Trigger): Observable<Application> {
        let url = '/project/' + key + '/application/' + appName + '/pipeline/' + pipName + '/trigger/' + t.id;
        return this._http.put(url, t).map(res => res.json());
    }

    /**
     * Delete the given trigger
     * @param key Project unique key
     * @param appName Application name
     * @param pipName Pipeline Name
     * @param t Trigger to delete
     * @returns {Observable<Application>}
     */
    removeTrigger(key: string, appName: string, pipName: string, t: Trigger): Observable<Application> {
        let url = '/project/' + key + '/application/' + appName + '/pipeline/' + pipName + '/trigger/' + t.id;
        return this._http.delete(url).map(res => res.json());
    }

    /**
     * Add notifications to the application
     * @param key Project unique key
     * @param appName Application name
     * @param notifications List of notification
     */
    addNotifications(key: string, appName: string, notifications: Array<Notification>): Observable<Application> {
        return this._http.post('/project/' + key + '/application/' + appName + '/notifications', notifications).map(res => res.json());
    }

    /**
     * Update a notification
     * @param key Project unique key
     * @param appName Application name
     * @param pipName Pipeline name
     * @param notification Notification data
     * @returns {Observable<Notification>}
     */
    updateNotification(key: string, appName: string, pipName: string, notification: Notification) {
        let url = '/project/' + key + '/application/' + appName + '/pipeline/' + pipName + '/notification';
        return this._http.put(url, notification).map(res => res.json());
    }

    /**
     * Delete all notifications on appliation/pipeline
     * @param key Project unique key
     * @param appName Application name
     * @param pipName Pipeline name
     * @returns {Observable<Application>}
     */
    deleteNotification(key: string, appName: string, pipName: string, envName?: string): Observable<Application> {
        let options = new RequestOptions();
        options.search = new URLSearchParams();
        options.search.set('envName', envName);
        let url = '/project/' + key + '/application/' + appName + '/pipeline/' + pipName + '/notification';
        return this._http.delete(url, options).map(res => res.json());
    }

    /**
     * Attach liste of pipeline to the application
     * @param key Project unique key
     * @param appName Application name
     * @param pipelines Array of pipeline name to attach
     */
    attachPipelines(key: string, appName: string, pipelines: Array<string>): Observable<Application> {
        return this._http.post('/project/' + key + '/application/' + appName + '/pipeline/attach', pipelines).map(res => res.json());
    }

    /**
     * Detach liste of pipeline to the application
     * @param key Project unique key
     * @param appName Application name
     * @param pipName Pipeline name to detach
     */
    detachPipelines(key: string, appName: string, pipName: string): Observable<Application> {
        return this._http.delete('/project/' + key + '/application/' + appName + '/pipeline/' + pipName).map(res => res.json());
    }


    /**
     * Add a scheduler on the couple application/pipeline
     * @param key Project unique key
     * @param appName Application name
     * @param pipName Pipeline name
     * @param scheduler Scheduler
     */
    addScheduler(key: string, appName: string, pipName: string, scheduler: Scheduler): Observable<Application> {
        let options = new RequestOptions();
        options.search = new URLSearchParams();
        options.search.set('envName', scheduler.environment_name);
        let url = '/project/' + key + '/application/' + appName + '/pipeline/' + pipName + '/scheduler';
        return this._http.post(url, scheduler, options).map(res => res.json());

    }

    /**
     * Update a scheduler
     * @param key Project unique key
     * @param appName Application name
     * @param pipName Pipeline name
     * @param scheduler Scheduler
     */
    updateScheduler(key: string, appName: string, pipName: string, scheduler: Scheduler): Observable<Application> {
        let url = '/project/' + key + '/application/' + appName + '/pipeline/' + pipName + '/scheduler';
        return this._http.put(url, scheduler).map(res => res.json());

    }

    /**
     * Delete a scheduler
     * @param key Project unique key
     * @param appName Application name
     * @param pipName Pipeline name
     * @param scheduler Scheduler
     */
    deleteScheduler(key: string, appName: string, pipName: string, scheduler: Scheduler): Observable<Application> {
        let url = '/project/' + key + '/application/' + appName + '/pipeline/' + pipName + '/scheduler/' + scheduler.id;
        return this._http.delete(url).map(res => res.json());

    }
}
