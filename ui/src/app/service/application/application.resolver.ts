import {Injectable} from '@angular/core';
import {Resolve, ActivatedRouteSnapshot, RouterStateSnapshot} from '@angular/router';
import {Observable} from 'rxjs/Observable';
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
            return this.appStore.getApplicationResolver(route.params['key'], route.queryParams['application']).map( app => {
                return app;
            }).catch(() => {
                return Observable.of(null);
            });
        } else {
            return Observable.of(null);
        }
    }

    constructor(private appStore: ApplicationStore) {}
}
