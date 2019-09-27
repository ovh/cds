import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { NavbarProjectData } from 'app/model/navbar.model';
import { BehaviorSubject } from 'rxjs';
import { Observable } from 'rxjs/Rx';

/**
 * Service to access Navbar from API.
 */
@Injectable()
export class NavbarService {

  private _navbar: BehaviorSubject<Array<NavbarProjectData>> = new BehaviorSubject(null);

  constructor(private _http: HttpClient) { }

  /**
   * Get the navbar data from API.
   * @returns {Observable<Array<NavbarProjectData>>}
   */
  getData(fromCache?: boolean): Observable<Array<NavbarProjectData>> {
    if (!fromCache) {
      this._http.get<Array<NavbarProjectData>>('/ui/navbar')
        .subscribe((data) => {
          this._navbar.next(data);
        });
    }

    return new Observable<Array<NavbarProjectData>>(fn => this._navbar.subscribe(fn));
  }
}
