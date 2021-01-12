
import { Injectable } from '@angular/core';
import { ActivatedRouteSnapshot, Resolve, RouterStateSnapshot } from '@angular/router';
import { Store } from '@ngxs/store';
import { Application } from 'app/model/application.model';
import { FetchApplication } from 'app/store/applications.action';
import { ApplicationsState } from 'app/store/applications.state';
import { Observable, of as observableOf } from 'rxjs';
import { catchError, flatMap } from 'rxjs/operators';



@Injectable()
export class ApplicationResolver implements Resolve<Application> {

    constructor(private store: Store) { }

    resolve(route: ActivatedRouteSnapshot, state: RouterStateSnapshot): Observable<any> | Promise<any> | any {
        return this.store.dispatch(new FetchApplication({
            projectKey: route.params['key'],
            applicationName: route.queryParams['application']
        })).pipe(
            flatMap(() => this.store.selectOnce(ApplicationsState.currentState()))
        );
    }
}

@Injectable()
export class ApplicationQueryParamResolver implements Resolve<Application> {

    constructor(private store: Store) { }

    resolve(route: ActivatedRouteSnapshot, state: RouterStateSnapshot): Observable<any> | Promise<any> | any {
        if (route.queryParams['application']) {
            return this.store.dispatch(new FetchApplication({
                projectKey: route.params['key'],
                applicationName: route.queryParams['application']
            })).pipe(
                flatMap(() => this.store.selectOnce(ApplicationsState.currentState())),
                catchError(() => observableOf(null))
            );
        } else {
            return observableOf(null);
        }
    }
}
