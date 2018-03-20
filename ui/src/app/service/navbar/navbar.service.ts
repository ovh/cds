import {Injectable} from '@angular/core';
import {Observable} from 'rxjs/Rx';
import {HttpClient} from '@angular/common/http';
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
        return this._http.get<NavbarData>('/navbar');
    }
}
