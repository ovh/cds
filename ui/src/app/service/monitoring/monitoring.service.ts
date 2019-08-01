import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { MonitoringStatus } from 'app/model/monitoring.model';
import { Observable } from 'rxjs';

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
}
