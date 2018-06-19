import {HttpClient, HttpParams} from '@angular/common/http';
import {Injectable} from '@angular/core';
import {Observable} from 'rxjs';

@Injectable()
export class ApplicationMigrateService {

    constructor(private _http: HttpClient) {
    }

    migrateApplicationToWorkflow(key: string, appName: string, force: boolean,
        disablePrefix: boolean, withRepositoryWebHook: boolean, withCurrentVersion: boolean): Observable<any> {
        let p = new HttpParams();
        p = p.append('force', force.toString());
        p = p.append('disablePrefix', disablePrefix.toString());
        p = p.append('withRepositoryWebHook', withRepositoryWebHook.toString());
        p = p.append('withCurrentVersion', withCurrentVersion.toString());
        return this._http.post('/project/' + key + '/application/' + appName + '/workflow/migrate', null, {params: p});
    }

    cleanWorkflow(key: string, appName: string): Observable<any> {
        return this._http.post('/project/' + key + '/application/' + appName + '/workflow/clean', null);
    }
}
