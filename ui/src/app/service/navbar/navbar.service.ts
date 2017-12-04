import {Injectable} from '@angular/core';
import {Observable} from 'rxjs/Rx';
import {HttpClient, HttpParams} from '@angular/common/http';
import {NavbarData} from '../../model/navbar.model';

/**
 * Service to access Navbar from API.
 */
@Injectable()
export class NavbarService {

    constructor(private _http: HttpClient) {
    }

    /**
     * Get the navbar data from API.
     * @returns {Observable<NavbarData>}
     */
    getData(): Observable<NavbarData> {
        let params = new HttpParams();
        return this._http.get<NavbarData>('/ui/navbar');
    }
}
