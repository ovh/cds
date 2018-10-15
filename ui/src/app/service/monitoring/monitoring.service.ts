import { HttpClient, HttpHeaders } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Observable } from 'rxjs';
import { MonitoringMetricsLine, MonitoringStatus } from '../../model/monitoring.model';

/**
 * Service about CDS Monitoring
 */
@Injectable()
export class MonitoringService {

    constructor(private _http: HttpClient) {
    }

    /**
     * Get the CDS API Status
     * @returns {Observable<MonitoringStatus>}
     */
    getStatus(): Observable<MonitoringStatus> {
        return this._http.get<MonitoringStatus>('/mon/status');
    }

    getMetrics(): Observable<MonitoringMetricsLine[]> {
        let headers = new HttpHeaders();
        headers = headers.set('Content-Type', 'application/json');
        return this._http.get<MonitoringMetricsLine[]>('/mon/metrics', {headers});
    }
}
