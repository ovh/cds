import {Injectable} from '@angular/core';
import {Project} from '../../model/project.model';
import {Observable} from 'rxjs/Rx';
import {Variable} from '../../model/variable.model';
import {GroupPermission} from '../../model/group.model';
import {Environment} from '../../model/environment.model';
import {Notification} from '../../model/notification.model';
import {LoadOpts} from './project.store';
import {HttpClient, HttpParams} from '@angular/common/http';

/**
 * Service to access Project from API.
 * Only used by ProjectStore
 */
@Injectable()
export class ProjectService {

    constructor(private _http: HttpClient) {
    }

    /**
     * Get one specific project from API.
     * @param key Unique key of the project
     * @returns {Observable<Project>}
     */
    getProject(key: string, opts?: LoadOpts[]): Observable<Project> {
        let params = new HttpParams();

        if (Array.isArray(opts)) {
          opts.forEach((opt) => params = params.append(opt.queryParam, 'true'));
        }

        return this._http.get('/project/' + key, {params: params});
    }

    /**
     * Get all projects that the user can access.
     * @returns {Observable<Project[]>}
     */
    getProjects(): Observable<Project[]> {
        let params = new HttpParams();
        params = params.append('application', 'true');
        return this._http.get('/project', {params: params});
    }

    /**
     * Create a new project
     * @param project Project to create
     * @returns {Observable<Project>}
     */
    addProject(project: Project): Observable<Project> {
        return this._http.post('/project', project);
    }

    /**
     * Update the given project.
     * @param project Project updated
     * @returns {Observable<Project>}
     */
    updateProject(project: Project): Observable<Project> {
        return this._http.put('/project/' + project.key, project);
    }

    /**
     * Delete the given project
     * @param key Project unique key
     * @returns {Observable<boolean>}
     */
    deleteProject(key: string): Observable<boolean> {
        return this._http.delete('/project/' + key).map(() => {
            return true;
        });
    }

    /**
     * Add a project variables.
     * @param key Project unique key
     * @param v Variable to add
     * @returns {Observable<Project>}
     */
    addVariable(key: string, v: Variable): Observable<Project> {
        return this._http.post('/project/' + key + '/variable/' + v.name, v);
    }

    /**
     * Update a project variable.
     * @param key Project unique key
     * @param v Variable to update
     * @returns {Observable<Project>}
     */
    updateVariable(key: string, v: Variable): Observable<Variable> {
        return this._http.put('/project/' + key + '/variable/' + v.name, v);
    }

    /**
     * Delete a project variable.
     * @param key Project unique key
     * @param v Variable to delete
     * @returns {Observable<Project>}
     */
    removeVariable(key: string, varName: string): Observable<boolean> {
        return this._http.delete('/project/' + key + '/variable/' + varName).map(res => true);
    }

    /**
     * Add a project permission.
     * @param key Project unique key
     * @param gp Permission to add
     * @returns {Observable<Project>}
     */
    addPermission(key: string, gp: GroupPermission): Observable<Array<GroupPermission>> {
        return this._http.post('/project/' + key + '/group', gp);
    }

    /**
     * Update a permission.
     * @param key Project unique key
     * @param gp Permission to update
     * @returns {Observable<Project>}
     */
    updatePermission(key: string, gp: GroupPermission): Observable<GroupPermission> {
        return this._http.put('/project/' + key + '/group/' + gp.group.name, gp);
    }

    /**
     * Delete a permission.
     * @param key Project unique key
     * @param gp Permission to delete
     * @returns {Observable<Project>}
     */
    removePermission(key: string, gp: GroupPermission): Observable<boolean> {
        return this._http.delete('/project/' + key + '/group/' + gp.group.name).map(res => true);
    }

    /**
     * Connect the given repo manager to the given project.
     * @param key Project unique key
     * @param repoName Repo manager name to connect
     * @returns {Observable<any>}
     */
    connectRepoManager(key: string, repoName: string): Observable<any> {
        return this._http.post('/project/' + key + '/repositories_manager/' + repoName + '/authorize', null);
    }

    /**
     * Delete the given project
     * @param key Project unique key
     * @param repoName Repo manager name to delete
     * @returns {Observable<Project>}
     */
    disconnectRepoManager(key: string, repoName: string): Observable<Project> {
        return this._http.delete('/project/' + key + '/repositories_manager/' + repoName);
    }

    /**
     * Send verifier code to link repomanager to project.
     * @param key Project unique key
     * @param repoName Repository manager name
     * @param token access token
     * @param verifier code verifier
     * @returns {Observable<Project>}
     */
    callback(key: string, repoName: string, token: string, verifier: string): Observable<Project> {
        let request = {
            'request_token': token,
            'verifier': verifier
        };
        let url = '/project/' + key + '/repositories_manager/' + repoName + '/authorize/callback';
        return this._http.post(url, request);
    }

    /**
     * Get specific environment by his name in the given project
     * @param key Project unique key
     * @param environment name
     * @returns {Observable<Project>}
     */
    getEnvironment(key: string, envName: string): Observable<Environment> {
        let params = new HttpParams();
        params = params.append('withWorkflows', 'true');

        return this._http.get('/project/' + key + '/environment/' + envName, {params});
    }

    /**
     * Add a new environment in the given project
     * @param key Project unique key
     * @param environment Environment to add
     * @returns {Observable<Project>}
     */
    addEnvironment(key: string, environment: Environment): Observable<Project> {
        return this._http.post('/project/' + key + '/environment', environment);
    }

    /**
     * Rename an environment in the given project
     * @param key Project unique key
     * @param environment Environment to rename
     * @returns {Observable<Project>}
     */
    renameEnvironment(key: string, oldName: string, environment: Environment): Observable<Project> {
        return this._http.put('/project/' + key + '/environment/' + oldName, environment);
    }

    /**
     * Clone an environment in the given project
     * @param key Project unique key
     * @param environment Environment to clone
     * @param cloneName for the new environment cloned
     * @returns {Observable<Project>}
     */
    cloneEnvironment(key: string, environment: Environment, cloneName: string): Observable<Project> {
        return this._http.post(`/project/${key}/environment/${environment.name}/clone/${cloneName}`, {});
    }

    /**
     * Remove an environment in the given project
     * @param key Project unique key
     * @param environment Environment to remove
     * @returns {Observable<Project>}
     */
    removeEnvironment(key: string, environment: Environment): Observable<Project> {
        return this._http.delete('/project/' + key + '/environment/' + environment.name);
    }

    /**
     * Add a variable in the given environment
     * @param key Project unique key
     * @param envName Environment name
     * @param v Variable to add
     * @returns {Observable<Project>}
     */
    addEnvironmentVariable(key: string, envName: string, v: Variable): Observable<Project> {
        return this._http.post('/project/' + key + '/environment/' + envName + '/variable/' + v.name, v);
    }

    /**
     * Update a variable in the given environment
     * @param key Project unique key
     * @param envName Environment name
     * @param v Variable to update
     * @returns {Observable<Project>}
     */
    updateEnvironmentVariable(key: string, envName: string, v: Variable): Observable<Project> {
        return this._http.put('/project/' + key + '/environment/' + envName + '/variable/' + v.name, v);
    }

    /**
     * Remove a variable from the given environment
     * @param key Project unique key
     * @param envName Environment name
     * @param v Variable to update
     * @returns {Observable<Project>}
     */
    removeEnvironmentVariable(key: string, envName: string, v: Variable): Observable<Project> {
        return this._http.delete('/project/' + key + '/environment/' + envName + '/variable/' + v.name);
    }

    /**
     * Add permission on environments
     * @param key Project unique key
     * @param envName environment name
     * @param gps New group permission to add
     * @returns {Observable<Environment>}
     */
    addEnvironmentPermission(key: string, envName: string, gps: Array<GroupPermission>): Observable<Environment> {
        return this._http.post('/project/' + key + '/environment/' + envName + '/groups', gps);
    }

    /**
     * Update a permission on an environment
     * @param key Project unique key
     * @param envName Environmenet name
     * @param gp Group Permission to update
     * @returns {Observable<Environment>}
     */
    updateEnvironmentPermission(key: string, envName: string, gp: GroupPermission): Observable<Environment> {
        return this._http.put('/project/' + key + '/environment/' + envName + '/group/' + gp.group.name, gp);
    }

    /**
     * Remove a permission on an environment
     * @param key Project unique key
     * @param envName Environmenet name
     * @param gp Group Permission to update
     * @returns {Observable<boolean>}
     */
    removeEnvironmentPermission(key: string, envName: string, gp: GroupPermission): Observable<boolean> {
        return this._http.delete('/project/' + key + '/environment/' + envName + '/group/' + gp.group.name).map(res => true);
    }

    /**
     * Get all notification on project
     * @param key Project unique key
     */
    getAllNotifications(key: string): Observable<Array<Notification>> {
        return this._http.get('/project/' + key + '/notifications');
    }
}
