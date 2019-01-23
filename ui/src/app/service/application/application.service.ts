
import { HttpClient, HttpHeaders, HttpParams } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { map } from 'rxjs/operators';
import { Application, Vulnerability } from '../../model/application.model';
import { GroupPermission } from '../../model/group.model';
import { Hook } from '../../model/hook.model';
import { Key } from '../../model/keys.model';
import { Notification, UserNotificationSettings } from '../../model/notification.model';
import { RepositoryPoller } from '../../model/polling.model';
import { Scheduler } from '../../model/scheduler.model';
import { Trigger } from '../../model/trigger.model';
import { Variable } from '../../model/variable.model';

@Injectable()
export class ApplicationService {

    constructor(private _http: HttpClient) {
    }

    /**
     * Get the given application from API
     * @param key Project unique key
     * @param appName Application Name
     */
    getApplication(key: string, appName: string): Observable<Application> {
        let params = new HttpParams();
        params = params.append('withNotifs', 'true');
        params = params.append('withUsage', 'true');
        params = params.append('withIcon', 'true');
        params = params.append('withKeys', 'true');
        params = params.append('withDeploymentStrategies', 'true');
        params = params.append('withVulnerabilities', 'true');

        return this._http.get<Application>('/project/' + key + '/application/' + appName, {params: params}).pipe(map(a => {
            a.vcs_strategy.password = '**********';
            return a;
        }));
    }

    /**
     * Update the given application
     * @param key Project unique key
     * @param application Application to update
     * @returns {Observable<Application>}
     */
    updateApplication(key: string, appOldName: string, app: Application): Observable<Application> {
        return this._http.put<Application>('/project/' + key + '/application/' + appOldName, app);
    }

    /**
     * Clone application
     * @param key Project unique key
     * @param appName Application to clone
     * @param application Application data
     * @returns {Observable<Application>}
     */
    cloneApplication(key: string, appName: string, application: Application): Observable<Application> {
        return this._http.post<Application>('/project/' + key + '/application/' + appName + '/clone', application);
    }

    /**
     * Create application
     * @param key Project unique key
     * @param application Application data
     * @returns {Observable<Application>}
     */
    createApplication(key: string, application: Application): Observable<Application> {
        return this._http.post<Application>('/project/' + key + '/applications', application);
    }


    /**
     * Delete an application
     * @param key Project unique key
     * @param appName Application name to delete
     * @returns {Observable<boolean>}
     */
    deleteApplication(key: string, appName: string): Observable<boolean> {
        return this._http.delete('/project/' + key + '/application/' + appName).pipe(map(res => {
            return true;
        }));
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
        return this._http.post<Application>(url, null);
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
        let headers = new HttpHeaders();
        headers.append('Content-Type', 'application/x-www-form-urlencoded');
        let params = new HttpParams();
        params = params.append('fullname', repoFullName);
        return this._http.post<Application>(url, params.toString(), {headers: headers, params: params});
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
        return this._http.post<Application>(url, poller);
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
        return this._http.put<Application>(url, poller);
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
        return this._http.delete<Application>(url);
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
            'pipeline_name': pipName
        };
        return this._http.post<Application>(url, request);
    }

    /**
     * Update the given hook
     * @param key Project key
     * @param appName Application Name
     * @param pipName Pipeline name
     * @param hook Hook to update
     */
    updateHook(key: string, appName: string, pipName: string, hook: Hook): Observable<Application> {
        let url = '/project/' + key + '/application/' + appName + '/pipeline/' + pipName + '/hook/' + hook.id;
        return this._http.put<Application>(url, hook);
    }

    /**
     * Delete a hook from the given application/repository manager
     * @param key Project unique key
     * @param appName Applicatio name
     * @param repoManName Repository manager name
     * @param hookId Hook id to delete
     * @returns {Observable<R>}
     */
    deleteHook(key: string, appName: string, hookId: number): Observable<Application> {
        let url = '/project/' + key + '/application/' + appName + '/repositories_manager/hook/' + hookId;
        return this._http.delete<Application>(url);
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
        return this._http.post<Application>(url, v);
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
        return this._http.put<Application>(url, v);
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
        return this._http.delete<Application>(url);
    }

    /**
     * Add a permission on the application.
     * @param key Project unique key
     * @param appName Application name
     * @param gp Permission to add
     * @returns {Observable<Application>}
     */
    addPermission(key: string, appName: string, gp: GroupPermission): Observable<Application> {
        return this._http.post<Application>('/project/' + key + '/application/' + appName + '/group', gp);
    }

    /**
     * Update a permission.
     * @param key Project unique key
     * @param appName Application name
     * @param gp Permission to update
     * @returns {Observable<Application>}
     */
    updatePermission(key: string, appName: string, gp: GroupPermission): Observable<Application> {
        return this._http.put<Application>('/project/' + key + '/application/' + appName + '/group/' + gp.group.name, gp);
    }

    /**
     * Delete a permission.
     * @param key Project unique key
     * @param appName Application name
     * @param gp Permission to delete
     * @returns {Observable<Application>}
     */
    removePermission(key: string, appName: string, gp: GroupPermission): Observable<Application> {
        return this._http.delete<Application>('/project/' + key + '/application/' + appName + '/group/' + gp.group.name);
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
        return this._http.post<Application>(url, t);
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
        return this._http.put<Application>(url, t);
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
        return this._http.delete<Application>(url);
    }

    /**
     * Add notifications to the application
     * @param key Project unique key
     * @param appName Application name
     * @param notifications List of notification
     */
    addNotifications(key: string, appName: string, notifications: Array<Notification>): Observable<Application> {
        return this._http.post<Application>('/project/' + key + '/application/' + appName + '/notifications', notifications);
    }

    /**
     * Update a notification
     * @param key Project unique key
     * @param appName Application name
     * @param pipName Pipeline name
     * @param notification Notification data
     * @returns {Observable<Notification>}
     */
    updateNotification(key: string, appName: string, pipName: string, notification: Notification): Observable<Application> {
        if (Array.isArray(notification.notifications)) {
            notification.notifications.forEach((n: UserNotificationSettings) => {
                if (Array.isArray(n.recipients)) {
                    n.recipients = n.recipients.map(r => r.trim());
                }
            });
        }

        let url = '/project/' + key + '/application/' + appName + '/pipeline/' + pipName + '/notification';
        return this._http.put<Application>(url, notification);
    }

    /**
     * Delete all notifications on appliation/pipeline
     * @param key Project unique key
     * @param appName Application name
     * @param pipName Pipeline name
     * @returns {Observable<Application>}
     */
    deleteNotification(key: string, appName: string, pipName: string, envName?: string): Observable<Application> {
        let params = new HttpParams();
        params = params.append('envName', envName);
        let url = '/project/' + key + '/application/' + appName + '/pipeline/' + pipName + '/notification';
        return this._http.delete<Application>(url, {params: params});
    }

    /**
     * Attach liste of pipeline to the application
     * @param key Project unique key
     * @param appName Application name
     * @param pipelines Array of pipeline name to attach
     */
    attachPipelines(key: string, appName: string, pipelines: Array<string>): Observable<Application> {
        return this._http.post<Application>('/project/' + key + '/application/' + appName + '/pipeline/attach', pipelines);
    }

    /**
     * Detach liste of pipeline to the application
     * @param key Project unique key
     * @param appName Application name
     * @param pipName Pipeline name to detach
     */
    detachPipelines(key: string, appName: string, pipName: string): Observable<Application> {
        return this._http.delete<Application>('/project/' + key + '/application/' + appName + '/pipeline/' + pipName);
    }

    /**
     * Add a key
     * @param key Project unique key
     * @param appName Application name
     * @param k Key to add
     */
    addKey(key: string, appName: string, k: Key): Observable<Key> {
        return this._http.post<Key>('/project/' + key + '/application/' + appName + '/keys', k);
    }

    /**
     * Remove a key
     * @param key Project unique key
     * @param appName Application name
     * @param kName Key to remove
     */
    removeKey(key: string, appName: string, kName: string): Observable<any> {
        return this._http.delete('/project/' + key + '/application/' + appName + '/keys/' + kName);
    }


    /**
     * Add a scheduler on the couple application/pipeline
     * @param key Project unique key
     * @param appName Application name
     * @param pipName Pipeline name
     * @param scheduler Scheduler
     */
    addScheduler(key: string, appName: string, pipName: string, scheduler: Scheduler): Observable<Application> {
        let params = new HttpParams();
        if (scheduler.environment_name) {
          params = params.append('envName', scheduler.environment_name);
        }
        let url = '/project/' + key + '/application/' + appName + '/pipeline/' + pipName + '/scheduler';
        return this._http.post<Application>(url, scheduler, {params: params});
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
        return this._http.put<Application>(url, scheduler);

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
        return this._http.delete<Application>(url);

    }

    /**
     * Save a deployment strategy
     * @param key Project unique key
     * @param appName Application name
     * @param pfName Platform name
     * @param pfConfig platform config
     */
    saveDeploymentStrategy(key: string, appName: string, pfName: string, pfConfig: any): Observable<Application> {
        let url = '/project/' + key + '/application/' + appName + '/deployment/config/' + pfName;
        return this._http.post<Application>(url, pfConfig);
    }

    /**
     * Delete a deployment strategy
     * @param key Project unique key
     * @param appName Application name
     * @param pfName Platform name
     */
    deleteDeploymentStrategy(key: string, appName: string, pfName: string): Observable<Application> {
        let url = '/project/' + key + '/application/' + appName + '/deployment/config/' + pfName;
        return this._http.delete<Application>(url);
    }

     /**
     * Get application deployment strategies
     * @param key Project unique key
     * @param appName Application name
     */
    getDeploymentStrategies(key: string, appName: string): Observable<Map<string, any>> {
        let url = '/project/' + key + '/application/' + appName + '/deployment/config';
        return this._http.get<Map<string, any>>(url);
    }

    /**
     * Ignore vulnerability
     * @param key project unique key
     * @param appName application name
     * @param id identifiant of the vulnerability
     */
    ignoreVulnerability(key: string, appName: string, v: Vulnerability): Observable<Vulnerability> {
        let url = '/project/' + key + '/application/' + appName + '/vulnerability/' + v.id;
        return this._http.post<Vulnerability>(url, v);
    }
}
