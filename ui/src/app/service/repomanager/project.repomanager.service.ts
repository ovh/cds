import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { IdName } from 'app/model/project.model';
import { Repository } from 'app/model/repositories.model';
import { Observable } from 'rxjs';

/**
 * Service to access Repository Manager from API.
 */
@Injectable()
export class RepoManagerService {


    constructor(private _http: HttpClient) {
    }

    /**
     * Get all available repositories manager
     *
     * @returns
     */
    getAll(): Observable<string[]> {
        return this._http.get<string[]>('/repositories_manager');
    }

    /**
     * Get all the repositories for the given repository manager.
     *
     * @param key Project unique key
     * @param repoManName Repository manager name
     * @returns
     */
    getRepositories(key: string, repoManName: string, sync: boolean): Observable<Repository[]> {
        let params = new HttpParams();
        params = params.append('synchronize', sync.toString());
        return this._http.get<Repository[]>('/project/' + key + '/repositories_manager/' + repoManName + '/repos', { params });
    }

    getDependencies(key: string, repoManName: string): Observable<IdName[]> {
        return this._http.get<IdName[]>('/project/' + key + '/repositories_manager/' + repoManName + '/applications');
    }
}
