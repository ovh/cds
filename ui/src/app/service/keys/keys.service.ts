
import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { AllKeys, formatKeysForSelect, Key} from 'app/model/keys.model';
import { forkJoin, Observable } from 'rxjs';
import { map } from 'rxjs/operators';

@Injectable()
export class KeyService {

    constructor(private _http: HttpClient) { }

    /**
     * Get all keys (project/application) from the given project
     *
     * @param projectKey Project unique key
     * @returns
     */
    getAllKeys(projectKey: string, appName?: string): Observable<AllKeys> {
        if (!appName) {
            return this._http.get<Key[]>('/project/' + projectKey + '/keys').pipe(map(keys => formatKeysForSelect(...keys)));
        }

        return forkJoin({
            projectKeys: this._http.get<Key[]>('/project/' + projectKey + '/keys'),
            appKeys: this._http.get<Key[]>('/project/' + projectKey + '/application/' + appName + '/keys')
        }).pipe(map(({projectKeys, appKeys}) => {
            let k = formatKeysForSelect(...projectKeys, ...appKeys);
            return k;
        }));
    }
}
