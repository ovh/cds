
import {map} from 'rxjs/operators';
import {Injectable} from '@angular/core';
import {HttpClient, HttpParams} from '@angular/common/http';
import {Observable} from 'rxjs';
import {AllKeys, KeyType, Keys, Key} from '../../model/keys.model';

@Injectable()
export class KeyService {

    constructor(private _http: HttpClient) {
    }

    /**
     * Get all keys (project/application/env) from the given project
     * @param projectKey Project unique key
     * @returns {Observable<Keys>}
     */
    getAllKeys(projectKey: string, appName?: string): Observable<AllKeys> {
        let p = new HttpParams();
        if (appName) {
            p = p.append('appName', appName);
        }

        return this._http.get<Keys>('/project/' + projectKey + '/all/keys', {params: p}).pipe(map(keys => {
            return Keys.formatForSelect(keys);
        }));
    }

    /**
     * Get project keys from the given project
     * @param projectKey Project unique key
     * @returns {Observable<Keys>}
     */
    getProjectKeys(projectKey: string): Observable<AllKeys> {
        return this._http.get<Array<Key>>('/project/' + projectKey + '/keys').pipe(map(keys => {
            let k = new AllKeys();
            k.ssh.push(...keys.filter(key => key.type === KeyType.SSH).map(key => key.name));
            k.pgp.push(...keys.filter(key => key.type === KeyType.PGP).map(key => key.name));
            return k;
        }));
    }
}
