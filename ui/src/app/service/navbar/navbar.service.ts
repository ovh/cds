import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { NavbarProjectData } from 'app/model/navbar.model';
import { BehaviorSubject, Observable } from 'rxjs';


/**
 * Service to access Navbar from API.
 */
@Injectable()
export class NavbarService {

  private _navbar: BehaviorSubject<Array<NavbarProjectData>> = new BehaviorSubject(null);

  constructor(private _http: HttpClient) { }

  getObservable(): Observable<Array<NavbarProjectData>> {
    return new Observable<Array<NavbarProjectData>>(fn => this._navbar.subscribe(fn));
  }

  /**
   * Get the navbar data from API.
   *
   * @returns
   */
  refreshData(): void {
    this._http.get<Array<NavbarProjectData>>('/ui/navbar')
      .subscribe((data) => {
        this._navbar.next(data);
      });
  }
}
