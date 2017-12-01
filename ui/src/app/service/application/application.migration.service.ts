import {Injectable} from '@angular/core';
import {HttpClient, HttpParams} from '@angular/common/http';
import {Observable} from 'rxjs/Observable';

@Injectable()
export class ApplicationMigrateService {

    constructor(private _http: HttpClient) {
    }

    migrateApplicationToWorkflow(key: string, appName: string, f: boolean): Observable<any> {
        let p = new HttpParams();
        p = p.append('force', f.toString());
        return this._http.post('/project/' + key + '/application/' + appName + '/workflow/migrate', null, {params: p});
    }

    cleanWorkflow(key: string, appName: string): Observable<any> {
        return this._http.post('/project/' + key + '/application/' + appName + '/workflow/clean', null);
    }
}
