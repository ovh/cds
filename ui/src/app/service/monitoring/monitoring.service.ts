import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { MonitoringStatus, MonitoringVersion } from 'app/model/monitoring.model';
import { Observable } from 'rxjs';

@Injectable()
export class MonitoringService {

    constructor(private _http: HttpClient) { }

    getVersion(): Observable<MonitoringVersion> {
        let params = new HttpParams();
        params = params.append('ts', new Date().getTime().toString());
        return this._http.get<MonitoringVersion>('./mon/version', { params });
    }

    getStatus(): Observable<MonitoringStatus> {
        return this._http.get<MonitoringStatus>('/mon/status');
    }

    getDebugProfiles(): Observable<any> {
        return this._http.get<any>('/admin/debug/profiles');
    }

    getGoroutines(): Observable<any> {
        return this._http.get<any>('/admin/debug/goroutines');
    }
}
