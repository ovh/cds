
import {HttpClient} from '@angular/common/http';
import {Injectable} from '@angular/core';
import {Service} from 'app/model/service.model';
import {Observable} from 'rxjs';

/**
 * Service to access Service from API.
 */
@Injectable()
export class ServiceService {

    constructor(private _http: HttpClient) {
    }

    /**
     * Get one specific service from API.
     *
     * @param name name of the service
     * @returns
     */
    getService(name: string): Observable<Service> {
        return this._http.get<Service>('/admin/service/' + name);
    }

    /**
     * Get all services that the user can access.
     *
     * @returns
     */
    getServices(): Observable<Service[]> {
        return this._http.get<Service[]>('/admin/services');
    }
}
