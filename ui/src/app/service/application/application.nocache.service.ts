import {HttpClient} from '@angular/common/http';
import {Injectable} from '@angular/core';
import {Observable} from 'rxjs';
import {Metric} from '../../model/metric.model';
import {Overview} from '../../model/application.model';

@Injectable()
export class ApplicationNoCacheService {

    constructor(private _http: HttpClient) {
    }

    getMetrics(key: string, appName: string, metric: string): Observable<Array<Metric>> {
        return this._http.get<Array<Metric>>('/project/' + key + '/application/' + appName + '/metrics/' + metric);
    }

    getOverview(key: string, appName: string): Observable<Overview> {
        return this._http.get<Overview>('/ui/project/' + key + '/application/' + appName + '/overview');
    }
}
