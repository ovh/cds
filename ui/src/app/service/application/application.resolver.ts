
import { Injectable } from '@angular/core';
import { ActivatedRouteSnapshot, RouterStateSnapshot } from '@angular/router';
import { Store } from '@ngxs/store';
import { FetchApplication } from 'app/store/applications.action';
import { ApplicationsState } from 'app/store/applications.state';
import { Observable, of as observableOf } from 'rxjs';
import { catchError, switchMap } from 'rxjs/operators';



@Injectable()
export class ApplicationResolver  {

    constructor(private store: Store) { }

    resolve(route: ActivatedRouteSnapshot, state: RouterStateSnapshot): Observable<any> | Promise<any> | any {
        return this.store.dispatch(new FetchApplication({
            projectKey: route.params['key'],
            applicationName: route.queryParams['application']
        })).pipe(
            switchMap(() => this.store.selectOnce(ApplicationsState.current))
        );
    }
}

@Injectable()
export class ApplicationQueryParamResolver  {

    constructor(private store: Store) { }

    resolve(route: ActivatedRouteSnapshot, state: RouterStateSnapshot): Observable<any> | Promise<any> | any {
        if (route.queryParams['application']) {
            return this.store.dispatch(new FetchApplication({
                projectKey: route.params['key'],
                applicationName: route.queryParams['application']
            })).pipe(
                switchMap(() => this.store.selectOnce(ApplicationsState.current)),
                catchError(() => observableOf(null))
            );
        } else {
            return observableOf(null);
        }
    }
}
