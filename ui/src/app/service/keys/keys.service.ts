
import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { AllKeys, Key, Keys, KeyType } from 'app/model/keys.model';
import { forkJoin, Observable } from 'rxjs';
import { map } from 'rxjs/operators';

@Injectable()
export class KeyService {

    constructor(private _http: HttpClient) { }

    /**
     * Get all keys (project/application) from the given project
     * @param projectKey Project unique key
     * @returns {Observable<Keys>}
     */
    getAllKeys(projectKey: string, appName?: string): Observable<AllKeys> {
        if (!appName) {
            return this._http.get<Keys>('/project/' + projectKey + '/keys').pipe(map(keys => {
                return Keys.formatForSelect(keys);
            }));
        }

        forkJoin<Keys, Keys> ([
            this._http.get<Keys>('/project/' + projectKey + '/keys'),
            this._http.get<Keys>('/project/' + projectKey + '/application/' + appName + '/keys')
        ]).pipe(map((k1, k2) => {
            return Keys.formatForSelect(k1, k2);
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
            k.ssh.push(...keys.filter(key => key.type === KeyType.SSH));
            k.pgp.push(...keys.filter(key => key.type === KeyType.PGP));
            return k;
        }));
    }
}
