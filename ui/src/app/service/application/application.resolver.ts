
import {of as observableOf, Observable} from 'rxjs';

import {catchError, map} from 'rxjs/operators';
import {Injectable} from '@angular/core';
import {Resolve, ActivatedRouteSnapshot, RouterStateSnapshot} from '@angular/router';
import {Application} from '../../model/application.model';
import {ApplicationStore} from './application.store';


@Injectable()
export class ApplicationResolver implements Resolve<Application> {

    resolve(route: ActivatedRouteSnapshot, state: RouterStateSnapshot): Observable<any>|Promise<any>|any {
        return this.appStore.getApplicationResolver(route.params['key'], route.params['appName']);
    }

    constructor(private appStore: ApplicationStore) {}
}

@Injectable()
export class ApplicationQueryParamResolver implements Resolve<Application> {

    resolve(route: ActivatedRouteSnapshot, state: RouterStateSnapshot): Observable<any>|Promise<any>|any {
        if (route.queryParams['application']) {
            return this.appStore.getApplicationResolver(route.params['key'], route.queryParams['application']).pipe(map( app => {
                return app;
            }), catchError(() => {
                return observableOf(null);
            }));
        } else {
            return observableOf(null);
        }
    }

    constructor(private appStore: ApplicationStore) {}
}
