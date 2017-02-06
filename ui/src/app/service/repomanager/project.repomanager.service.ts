import {Injectable} from '@angular/core';
import {Http} from '@angular/http';
import {Observable} from 'rxjs/Rx';
import {RepositoriesManager, Repository} from '../../model/repositories.model';

/**
 * Service to access Repository Manager from API.
 */
@Injectable()
export class RepoManagerService {


    constructor(private _http: Http) {
    }

    /**
     * Get all available repositories manager
     * @returns {Observable<RepositoriesManager[]>}
     */
    getAll(): Observable<RepositoriesManager[]> {
        return this._http.get('/repositories_manager').map( res => res.json());
    }

    /**
     * Get all the repositories for the given repository manager.
     * @param key Project unique key
     * @param repoManName Repository manager name
     * @returns {Observable<Repository[]>}
     */
    getRepositories(key: string, repoManName: string): Observable<Repository[]> {
        return this._http.get('/project/' + key + '/repositories_manager/' + repoManName + '/repos').map( res => res.json());
    }
}
