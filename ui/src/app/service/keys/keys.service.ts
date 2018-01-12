import {Injectable} from '@angular/core';
import {HttpClient} from '@angular/common/http';
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
    getAllKeys(key: string): Observable<AllKeys> {
        return this._http.get<Keys>('/project/' + key + '/all/keys').map(keys => {
            return Keys.formatForSelect(keys);
        });
    }
}
