
import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { ProjectIntegration } from 'app/model/integration.model';
import { Schema } from 'app/model/json-schema.model';
import { Key } from 'app/model/keys.model';
import { LoadOpts, Project, ProjectRepository, VCSProject } from 'app/model/project.model';
import { Observable } from 'rxjs';
import { map } from 'rxjs/operators';
import { RepositoryAnalysis } from "../../model/analysis.model";
import { Branch } from "../../model/repositories.model";
import {Entity} from "../../model/entity.model";

/**
 * Service to access Project from API.
 * Only used by ProjectStore
 */
@Injectable()
export class ProjectService {

    constructor(
        private _http: HttpClient
    ) { }

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
    listVCSProject(key: string): Observable<Array<VCSProject>> {
        return this._http.get<Array<VCSProject>>(`/v2/project/${key}/vcs`);
    }

    getVCSProject(key: string, vcsName: string): Observable<VCSProject> {
        return this._http.get<VCSProject>(`/v2/project/${key}/vcs/${vcsName}`);
    }

    getVCSRepository(key: string, vcsName: string, repoName: string): Observable<ProjectRepository> {
        let encodedRepo = encodeURIComponent(repoName);
        return this._http.get<ProjectRepository>(`/v2/project/${key}/vcs/${vcsName}/repository/${encodedRepo}`);
    }

    getVCSRepositoryBranches(key: string, vcsName: string, repoName: string, limit: number): Observable<Array<Branch>> {
        let encodedRepo = encodeURIComponent(repoName);
        let params = new HttpParams();
        params = params.append('limit', limit);
        return this._http.get<Array<Branch>>(`/v2/project/${key}/vcs/${vcsName}/repository/${encodedRepo}/branches`);
    }

    getVCSRepositories(key: string, vcsName: string): Observable<Array<ProjectRepository>> {
        return this._http.get<Array<ProjectRepository>>(`/v2/project/${key}/vcs/${vcsName}/repository`);
    }

    listVCSRepositoryAnalysis(key: string, vcsName: string, repoName: string): Observable<Array<RepositoryAnalysis>> {
        let encodedRepo = encodeURIComponent(repoName);
        return this._http.get<Array<RepositoryAnalysis>>(`/v2/project/${key}/vcs/${vcsName}/repository/${encodedRepo}/analysis`)
    }

    addVCSRepository(key: string, vcsName: string, repo: ProjectRepository): Observable<ProjectRepository> {
        return this._http.post<ProjectRepository>(`/v2/project/${key}/vcs/${vcsName}/repository`, repo);
    }

    deleteVCSRepository(key: string, vcsName: string, repoName: string): Observable<any> {
        let encodedRepo = encodeURIComponent(repoName);
        return this._http.delete(`/v2/project/${key}/vcs/${vcsName}/repository/${encodedRepo}`);
    }

    getRepoEntities(key: string, vcsName: string, repo: string, branch?: string): Observable<Array<Entity>> {
        let encodedRepo = encodeURIComponent(repo);
        let params = new HttpParams();
        if (branch) {
            params = params.append('branch', branch);
        }
        return this._http.get<Array<Entity>>(`/v2/project/${key}/vcs/${vcsName}/repository/${encodedRepo}/entities`, { params });
    }

    getRepoEntity(key: string, vcsName: string, repoName: string, entityType: string, entityName: string, branch?: string): Observable<Entity> {
        let encodedRepo = encodeURIComponent(repoName);
        let params = new HttpParams();
        if (branch) {
            params = params.append('branch', branch);
        }
        return this._http.get<Entity>(`/v2/project/${key}/vcs/${vcsName}/repository/${encodedRepo}/entities/${entityType}/${entityName}`, { params });

    }

    getJSONSchema(type: string): Observable<Schema> {
        return this._http.get<Schema>(`/v2/jsonschema/${type}`).pipe(
            map(s => Object.assign(new Schema(), s))
        );
    }
}
