import {Injectable} from '@angular/core';
import {Project, LoadOpts} from '../../model/project.model';
import {Application} from '../../model/application.model';
import {Observable} from 'rxjs/Observable';
import {Variable} from '../../model/variable.model';
import {GroupPermission} from '../../model/group.model';
import {Environment} from '../../model/environment.model';
import {Notification} from '../../model/notification.model';
import {HttpClient, HttpParams} from '@angular/common/http';
import {Key} from '../../model/keys.model';
import {ProjectPlatform} from '../../model/platform.model';

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
    getProject(key: string, opts: LoadOpts[]): Observable<Project> {
        let params = new HttpParams();

        if (Array.isArray(opts) && opts.length) {
            opts = opts.concat([
                new LoadOpts('withGroups', 'groups'),
                new LoadOpts('withPermission', 'permission')
            ]);
        } else {
            opts = [
                new LoadOpts('withGroups', 'groups'),
                new LoadOpts('withPermission', 'permission')
            ];
        }

        opts.forEach((opt) => params = params.append(opt.queryParam, 'true'));

        return this._http.get<Project>('/project/' + key, {params: params});
    }

    /**
     * Get all projects that the user can access.
     * @returns {Observable<Project[]>}
     */
    getProjects(withApplication: boolean): Observable<Project[]> {
        let params = new HttpParams();
        if (withApplication) {
          params = params.append('application', 'true');
        }
        return this._http.get<Project[]>('/project', {params: params});
    }

    /**
     * Create a new project
     * @param project Project to create
     * @returns {Observable<Project>}
     */
    addProject(project: Project): Observable<Project> {
        return this._http.post<Project>('/project', project);
    }

    /**
     * Update the given project.
     * @param project Project updated
     * @returns {Observable<Project>}
     */
    updateProject(project: Project): Observable<Project> {
        return this._http.put<Project>('/project/' + project.key, project);
    }

    /**
     * Update favorite project.
     * @param project Project updated
     * @returns {Observable<Project>}
     */
    updateFavorite(projectKey: string): Observable<Project> {
        return this._http.post<Project>('/project/' + projectKey + '/favorite', {});
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
        return this._http.post<Project>('/project/' + key + '/variable/' + v.name, v);
    }

    /**
     * Update a project variable.
     * @param key Project unique key
     * @param v Variable to update
     * @returns {Observable<Project>}
     */
    updateVariable(key: string, v: Variable): Observable<Variable> {
        return this._http.put<Variable>('/project/' + key + '/variable/' + v.name, v);
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
        return this._http.post<Array<GroupPermission>>('/project/' + key + '/group', gp);
    }

    /**
     * Update a permission.
     * @param key Project unique key
     * @param gp Permission to update
     * @returns {Observable<Project>}
     */
    updatePermission(key: string, gp: GroupPermission): Observable<GroupPermission> {
        return this._http.put<GroupPermission>('/project/' + key + '/group/' + gp.group.name, gp);
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
        return this._http.delete<Project>('/project/' + key + '/repositories_manager/' + repoName);
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
        return this._http.post<Project>(url, request);
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

        return this._http.get<Environment>('/project/' + key + '/environment/' + envName, {params});
    }

    /**
     * Add a new environment in the given project
     * @param key Project unique key
     * @param environment Environment to add
     * @returns {Observable<Project>}
     */
    addEnvironment(key: string, environment: Environment): Observable<Project> {
        return this._http.post<Project>('/project/' + key + '/environment', environment);
    }

    /**
     * Rename an environment in the given project
     * @param key Project unique key
     * @param environment Environment to rename
     * @returns {Observable<Project>}
     */
    renameEnvironment(key: string, oldName: string, environment: Environment): Observable<Project> {
        return this._http.put<Project>('/project/' + key + '/environment/' + oldName, environment);
    }

    /**
     * Clone an environment in the given project
     * @param key Project unique key
     * @param environment Environment to clone
     * @param cloneName for the new environment cloned
     * @returns {Observable<Project>}
     */
    cloneEnvironment(key: string, environment: Environment, cloneName: string): Observable<Project> {
        return this._http.post<Project>(`/project/${key}/environment/${environment.name}/clone/${cloneName}`, {});
    }

    /**
     * Remove an environment in the given project
     * @param key Project unique key
     * @param environment Environment to remove
     * @returns {Observable<Project>}
     */
    removeEnvironment(key: string, environment: Environment): Observable<Project> {
        return this._http.delete<Project>('/project/' + key + '/environment/' + environment.name);
    }

    /**
     * Add a variable in the given environment
     * @param key Project unique key
     * @param envName Environment name
     * @param v Variable to add
     * @returns {Observable<Project>}
     */
    addEnvironmentVariable(key: string, envName: string, v: Variable): Observable<Project> {
        return this._http.post<Project>('/project/' + key + '/environment/' + envName + '/variable/' + v.name, v);
    }

    /**
     * Update a variable in the given environment
     * @param key Project unique key
     * @param envName Environment name
     * @param v Variable to update
     * @returns {Observable<Project>}
     */
    updateEnvironmentVariable(key: string, envName: string, v: Variable): Observable<Project> {
        return this._http.put<Project>('/project/' + key + '/environment/' + envName + '/variable/' + v.name, v);
    }

    /**
     * Remove a variable from the given environment
     * @param key Project unique key
     * @param envName Environment name
     * @param v Variable to update
     * @returns {Observable<Project>}
     */
    removeEnvironmentVariable(key: string, envName: string, v: Variable): Observable<Project> {
        return this._http.delete<Project>('/project/' + key + '/environment/' + envName + '/variable/' + v.name);
    }

    /**
     * Add permission on environments
     * @param key Project unique key
     * @param envName environment name
     * @param gps New group permission to add
     * @returns {Observable<Environment>}
     */
    addEnvironmentPermission(key: string, envName: string, gps: Array<GroupPermission>): Observable<Environment> {
        return this._http.post<Environment>('/project/' + key + '/environment/' + envName + '/groups', gps);
    }

    /**
     * Update a permission on an environment
     * @param key Project unique key
     * @param envName Environmenet name
     * @param gp Group Permission to update
     * @returns {Observable<Environment>}
     */
    updateEnvironmentPermission(key: string, envName: string, gp: GroupPermission): Observable<Environment> {
        return this._http.put<Environment>('/project/' + key + '/environment/' + envName + '/group/' + gp.group.name, gp);
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
        return this._http.get<Array<Notification>>('/project/' + key + '/notifications');
    }

    /**
     * Get all projects keys
     * @param key
     * @returns {Observable<Array<Key>>}
     */
    getKeys(key: string): Observable<Array<Key>> {
        return this._http.get<Array<Key>>('/project/' + key + '/keys');
    }

    /**
     * Add a project key
     * @param projKey Project unique key
     * @param key Key to add
     * @returns {Observable<Key>}
     */
    addKey(projKey: string, key: Key): Observable<Key> {
        return this._http.post<Key>('/project/' + projKey + '/keys', key);
    }

    /**
     * Remove a key from the project
     * @param key project unique key
     * @param name key name
     * @returns {Observable<any>}
     */
    removeKey(key: string, name: string): Observable<any> {
        return this._http.delete('/project/' + key + '/keys/' + name)
    }

    /**
     * Get all applications in project
     * @param key Project unique key
     */
    getApplications(key: string): Observable<Array<Application>> {
        return this._http.get<Array<Application>>('/project/' + key + '/applications');
    }

    /**
     * Get all platforms in project
     * @param key Project unique key
     * @returns {Observable<Object>}
     */
    getPlatforms(key: string): Observable<Array<ProjectPlatform>> {
        return this._http.get<Array<ProjectPlatform>>('/project/' + key + '/platforms');
    }

    /**
     * Add a platform to a project
     * @param key Project unique key
     * @param p Platform to add
     * @returns {Observable<ProjectPlatform>}
     */
    addPlatform(key: string, p: ProjectPlatform): Observable<ProjectPlatform> {
        return this._http.post<ProjectPlatform>('/project/' + key + '/platforms', p);
    }

    /**
     * Remove a project platform
     * @param key project unique key
     * @param name platform name
     * @returns {Observable<Object>}
     */
    removePlatform(key: string, name: string): Observable<any> {
        return this._http.delete('/project/' + key + '/platforms/' + name);
    }

    /**
     * Update project platform configuration
     * @param key Project unique key
     * @param platform Platform to update
     * @returns {Observable<ProjectPlatform>}
     */
    updatePlatform(key: string, platform: ProjectPlatform): Observable<ProjectPlatform> {
        return this._http.put<ProjectPlatform>('/project/' + key + '/platforms/' + platform.name, platform);
    }
}
