import {Injectable} from '@angular/core';
import {HttpClient, HttpParams} from '@angular/common/http';
import {Observable} from 'rxjs/Observable';
import {AllKeys, Keys} from '../../model/keys.model';

@Injectable()
export class KeyService {

    constructor(private _http: HttpClient) {
    }

    /**
     * Get all keys (project/application/env) from the given project
     * @param key Project unique key
     * @returns {Observable<Keys>}
     */
    getAllKeys(key: string, appName?: string): Observable<AllKeys> {
        let p = new HttpParams();
        if (appName) {
            p = p.append('appName', appName);
        }

        return this._http.get<Keys>('/project/' + key + '/all/keys', {params: p}).map(keys => {
            return Keys.formatForSelect(keys);
        });
    }
}
