
import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { map } from 'rxjs/operators';
import { Application } from '../../model/application.model';
import { Environment } from '../../model/environment.model';
import { GroupPermission } from '../../model/group.model';
import { ProjectIntegration } from '../../model/integration.model';
import { Key } from '../../model/keys.model';
import { Notification } from '../../model/notification.model';
import { Label, LoadOpts, Project } from '../../model/project.model';
import { Variable } from '../../model/variable.model';

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
        opts.push(new LoadOpts('withFeatures', 'features'));
        opts.push(new LoadOpts('withIntegrations', 'integrations'));
        opts.forEach((opt) => params = params.append(opt.queryParam, 'true'));

        return this._http.get<Project>('/project/' + key, { params: params });
    }

    /**
     * Get all projects that the user can access.
     * @returns {Observable<Project[]>}
     */
    getProjects(): Observable<Project[]> {
        let params = new HttpParams();
        params = params.append('withIcon', 'true');
        return this._http.get<Project[]>('/project', { params });
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
        return this._http.post<Project>('/user/favorite', {
            type: 'project',
            project_key: projectKey
        });
    }

    /**
     * Delete the given project
     * @param key Project unique key
     * @returns {Observable<boolean>}
     */
    deleteProject(key: string): Observable<boolean> {
        return this._http.delete('/project/' + key).pipe(map(() => {
            return true;
        }));
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
        return this._http.delete('/project/' + key + '/variable/' + varName).pipe(map(res => true));
    }

    /**
     * Add a project permission.
     * @param key Project unique key
     * @param gp Permission to add
     * @returns {Observable<Project>}
     */
    addPermission(key: string, gp: GroupPermission, onlyForProject?: boolean): Observable<Array<GroupPermission>> {
        let params = new HttpParams();
        if (onlyForProject) {
            params = params.append('onlyProject', 'true');
        }
        return this._http.post<Array<GroupPermission>>('/project/' + key + '/group', gp, { params });
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
        return this._http.delete('/project/' + key + '/group/' + gp.group.name).pipe(map(res => true));
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

        return this._http.get<Environment>('/project/' + key + '/environment/' + envName, { params });
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
     * Get all integrations in project
     * @param key Project unique key
     * @returns {Observable<Object>}
     */
    getIntegrations(key: string): Observable<Array<ProjectIntegration>> {
        return this._http.get<Array<ProjectIntegration>>('/project/' + key + '/integrations');
    }

    /**
     * Add a integration to a project
     * @param key Project unique key
     * @param p Integration to add
     * @returns {Observable<ProjectIntegration>}
     */
    addIntegration(key: string, p: ProjectIntegration): Observable<ProjectIntegration> {
        return this._http.post<ProjectIntegration>('/project/' + key + '/integrations', p);
    }

    /**
     * Remove a project integration
     * @param key project unique key
     * @param name integration name
     * @returns {Observable<Object>}
     */
    removeIntegration(key: string, name: string): Observable<any> {
        return this._http.delete('/project/' + key + '/integrations/' + name);
    }

    /**
     * Update project integration configuration
     * @param key Project unique key
     * @param integration Integration to update
     * @returns {Observable<ProjectIntegration>}
     */
    updateIntegration(key: string, integration: ProjectIntegration): Observable<ProjectIntegration> {
        return this._http.put<ProjectIntegration>('/project/' + key + '/integrations/' + integration.name, integration);
    }

    /**
     * Update project labels
     * @param key Project unique key
     * @param labels Labels to update
     * @returns {Observable<Project>}
     */
    updateLabels(key: string, labels: Label[]): Observable<Project> {
        return this._http.put<Project>('/project/' + key + '/labels', labels);
    }
}
