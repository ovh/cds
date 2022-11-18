
import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { ProjectIntegration } from 'app/model/integration.model';
import { Key } from 'app/model/keys.model';
import { Entity, LoadOpts, Project, Repository, VCSProject } from 'app/model/project.model';
import { Observable } from 'rxjs';

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
     *
     * @param key Unique key of the project
     * @returns
     */
    getProject(key: string, opts: LoadOpts[]): Observable<Project> {
        let params = new HttpParams();

        if (Array.isArray(opts) && opts.length) {
            opts = opts.concat([
                new LoadOpts('withGroups', 'groups')
            ]);
        } else {
            opts = [
                new LoadOpts('withGroups', 'groups')
            ];
        }
        opts.push(new LoadOpts('withFeatures', 'features'));
        opts.push(new LoadOpts('withIntegrations', 'integrations'));
        opts.forEach((opt) => params = params.append(opt.queryParam, 'true'));

        return this._http.get<Project>('/project/' + key, { params });
    }

    /**
     * Get all projects that the user can access.
     *
     * @returns
     */
    getProjects(): Observable<Project[]> {
        let params = new HttpParams();
        params = params.append('withIcon', 'true');
        return this._http.get<Project[]>('/project', { params });
    }

    /**
     * Send verifier code to link repomanager to project.
     *
     * @param key Project unique key
     * @param repoName Repository manager name
     * @param token access token
     * @param verifier code verifier
     * @returns
     */
    callback(key: string, repoName: string, token: string, verifier: string): Observable<Project> {
        let request = {
            request_token: token,
            verifier
        };
        let url = '/project/' + key + '/repositories_manager/' + repoName + '/authorize/callback';
        return this._http.post<Project>(url, request);
    }

    /**
     * Add a project key
     *
     * @param projKey Project unique key
     * @param key Key to add
     * @returns
     */
    addKey(projKey: string, key: Key): Observable<Key> {
        return this._http.post<Key>('/project/' + projKey + '/keys', key);
    }

    /**
     * Update project integration configuration
     *
     * @param key Project unique key
     * @param integration Integration to update
     * @returns
     */
    updateIntegration(key: string, integration: ProjectIntegration): Observable<ProjectIntegration> {
        return this._http.put<ProjectIntegration>('/project/' + key + '/integrations/' + integration.name, integration);
    }

    /**
     * Get the list of VCS attached to the given project from the API
     * @param key
     */
    getVCSProject(key: string): Observable<Array<VCSProject>> {
        return this._http.get<Array<VCSProject>>(`/v2/project/${key}/vcs`);
    }

    getVCSRepositories(key: string, vcsName: string): Observable<Array<Repository>> {
        return this._http.get<Array<Repository>>(`/v2/project/${key}/vcs/${vcsName}/repository`);
    }

    getRepoEntities(key: string, vcsName: string, repo: string): Observable<Array<Entity>> {
        let encodedRepo = encodeURIComponent(repo);
        let url = `/v2/project/${key}/vcs/${vcsName}/repository/${encodedRepo}/entities`;
        console.log(url);
        return this._http.get<Array<Entity>>(url);
    }
}
