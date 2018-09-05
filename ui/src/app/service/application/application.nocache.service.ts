import {HttpClient} from '@angular/common/http';
import {Injectable} from '@angular/core';
import {Observable} from 'rxjs';
import {Metric} from '../../model/metric.model';

@Injectable()
export class ApplicationNoCacheService {

    constructor(private _http: HttpClient) {
    }

    getMetrics(key: string, appName: string, metric: string): Observable<Array<Metric>> {
        return this._http.get<Array<Metric>>('/project/' + key + '/application/' + appName + '/metrics/' + metric);
    }
}
