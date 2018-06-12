import {Injectable} from '@angular/core';
import {Observable} from 'rxjs';
import {HttpClient} from '@angular/common/http';
import {MonitoringStatus} from '../../model/monitoring.model';

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
