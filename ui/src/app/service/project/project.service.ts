
import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { ProjectIntegration } from 'app/model/integration.model';
import { Schema } from 'app/model/json-schema.model';
import { Key } from 'app/model/keys.model';
import { LoadOpts, Project, ProjectRepository, RepositoryHookEvent } from 'app/model/project.model';
import { Observable } from 'rxjs';
import { map } from 'rxjs/operators';
import { RepositoryAnalysis } from "../../model/analysis.model";
import { Branch, Tag } from "../../model/repositories.model";
import { Entity, EntityType } from "../../model/entity.model";
import { VCSProject } from 'app/model/vcs.model';

@Injectable()
export class ProjectService {

    constructor(
        private _http: HttpClient
    ) { }

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
        opts.forEach((opt) => params = params.append(opt.queryParam, 'true'));

        return this._http.get<Project>('/project/' + key, { params });
    }

    getProjects(): Observable<Array<Project>> {
        return this._http.get<Array<Project>>('/project');
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

    addKey(projKey: string, key: Key): Observable<Key> {
        return this._http.post<Key>('/project/' + projKey + '/keys', key);
    }

    getIntegrations(key: string): Observable<Array<ProjectIntegration>> {
        return this._http.get<Array<ProjectIntegration>>(`/project/${key}/integrations`);
    }

    updateIntegration(key: string, integration: ProjectIntegration): Observable<ProjectIntegration> {
        return this._http.put<ProjectIntegration>('/project/' + key + '/integrations/' + integration.name, integration);
    }

    listVCSProject(key: string): Observable<Array<VCSProject>> {
        return this._http.get<Array<VCSProject>>(`/v2/project/${key}/vcs`);
    }

    getVCSProject(key: string, vcsName: string): Observable<VCSProject> {
        return this._http.get<VCSProject>(`/v2/project/${key}/vcs/${vcsName}`);
    }

    addVCSProject(key: string, vcsProject: VCSProject): Observable<VCSProject> {
        return this._http.post<VCSProject>(`/v2/project/${key}/vcs`, vcsProject);
    }

    saveVCSProject(key: string, vcsProject: VCSProject): Observable<VCSProject> {
        return this._http.put<VCSProject>(`/v2/project/${key}/vcs/${vcsProject.name}`, vcsProject);
    }

    deleteVCSProject(key: string, vcsName: string): Observable<any> {
        return this._http.delete(`/v2/project/${key}/vcs/${vcsName}`);
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

    getVCSRepositoryTags(key: string, vcsName: string, repoName: string): Observable<Array<Tag>> {
        let encodedRepo = encodeURIComponent(repoName);
        return this._http.get<Array<Tag>>(`/v2/project/${key}/vcs/${vcsName}/repository/${encodedRepo}/tags`);
    }

    getVCSRepositories(key: string, vcsName: string): Observable<Array<ProjectRepository>> {
        return this._http.get<Array<ProjectRepository>>(`/v2/project/${key}/vcs/${vcsName}/repository`);
    }

    listVCSRepositoryAnalysis(key: string, vcsName: string, repoName: string): Observable<Array<RepositoryAnalysis>> {
        let encodedRepo = encodeURIComponent(repoName);
        return this._http.get<Array<RepositoryAnalysis>>(`/v2/project/${key}/vcs/${vcsName}/repository/${encodedRepo}/analysis`)
    }

    getAnalysis(key: string, vcsName: string, repoName: string, id: string): Observable<RepositoryAnalysis> {
        let encodedRepo = encodeURIComponent(repoName);
        return this._http.get<RepositoryAnalysis>(`/v2/project/${key}/vcs/${vcsName}/repository/${encodedRepo}/analysis/${id}`)
    }

    getRepositoryEvent(key: string, vcsName: string, repoName: string, eventID: string): Observable<RepositoryHookEvent> {
        let encodedRepo = encodeURIComponent(repoName);
        return this._http.get<RepositoryHookEvent>(`/v2/project/${key}/vcs/${vcsName}/repository/${encodedRepo}/events/${eventID}`)
    }

    listRepositoryEvents(key: string, vcsName: string, repoName: string): Observable<Array<RepositoryHookEvent>> {
        let encodedRepo = encodeURIComponent(repoName);
        return this._http.get<Array<RepositoryHookEvent>>(`/v2/project/${key}/vcs/${vcsName}/repository/${encodedRepo}/events`)
    }

    addVCSRepository(key: string, vcsName: string, repo: ProjectRepository): Observable<ProjectRepository> {
        return this._http.post<ProjectRepository>(`/v2/project/${key}/vcs/${vcsName}/repository`, repo);
    }

    deleteVCSRepository(key: string, vcsName: string, repoName: string): Observable<any> {
        let encodedRepo = encodeURIComponent(repoName);
        return this._http.delete(`/v2/project/${key}/vcs/${vcsName}/repository/${encodedRepo}`);
    }

    getRepoEntities(key: string, vcsName: string, repo: string, ref?: string): Observable<Array<Entity>> {
        let encodedRepo = encodeURIComponent(repo);
        let params = new HttpParams();
        if (ref) {
            params = params.append('ref', ref);
        }
        return this._http.get<Array<Entity>>(`/v2/project/${key}/vcs/${vcsName}/repository/${encodedRepo}/entities`, { params });
    }

    getRepoEntity(key: string, vcsName: string, repoName: string, entityType: EntityType, entityName: string, ref?: string): Observable<Entity> {
        let encodedRepo = encodeURIComponent(repoName);
        let params = new HttpParams();
        if (ref) {
            params = params.append('ref', ref);
        }
        return this._http.get<Entity>(`/v2/project/${key}/vcs/${vcsName}/repository/${encodedRepo}/entities/${entityType}/${entityName}`, { params });
    }

    getJSONSchema(type: string): Observable<Schema> {
        return this._http.get<Schema>(`/v2/jsonschema/${type}`).pipe(
            map(s => Object.assign(new Schema(), s))
        );
    }
}
