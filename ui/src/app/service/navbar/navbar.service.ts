import {Injectable} from '@angular/core';
import {Observable} from 'rxjs/Rx';
import {BehaviorSubject} from 'rxjs/BehaviorSubject';
import {HttpClient} from '@angular/common/http';
import {NavbarProjectData} from '../../model/navbar.model';

/**
 * Service to access Navbar from API.
 */
@Injectable()
export class NavbarService {

    private _navbar: BehaviorSubject<Array<NavbarProjectData>> = new BehaviorSubject(null);

    constructor(private _http: HttpClient) {
    }

    /**
     * Get the navbar data from API.
     * @returns {Observable<Array<NavbarProjectData>>}
     */
    getData(fromCache?: boolean): Observable<Array<NavbarProjectData>> {
        if (!fromCache) {
          this._http.get<Array<NavbarProjectData>>('/navbar')
            .subscribe((data) => {
              this._navbar.next(data);
            });
        }

        return new Observable<Array<NavbarProjectData>>(fn => this._navbar.subscribe(fn));
    }
}
