
import {Observable, of as observableOf} from 'rxjs';

import {Injectable} from '@angular/core';
import {ActivatedRouteSnapshot, Resolve, RouterStateSnapshot} from '@angular/router';
import {catchError, map} from 'rxjs/operators';
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
