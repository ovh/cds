import {Injectable} from '@angular/core';
import {ActivatedRouteSnapshot, CanActivate, CanActivateChild, NavigationExtras, Router, RouterStateSnapshot} from '@angular/router';
import {Observable} from 'rxjs';
import {AuthentificationStore} from './authentification.store';


@Injectable()
export class CanActivateAuthRoute implements CanActivate , CanActivateChild {

    constructor(private _router: Router, private _authStore: AuthentificationStore) {}

    canActivate(
        route: ActivatedRouteSnapshot,
        state: RouterStateSnapshot
    ): Observable<boolean>|Promise<boolean>|boolean {
        if (this._authStore.isConnected()) {
            return true;
        }

        let navigationExtras: NavigationExtras = {
            queryParams: {
                redirect: state.url
            }
        };

        this._router.navigate(['account/login'], navigationExtras);
        return false;
    }

    canActivateChild(
        route: ActivatedRouteSnapshot,
        state: RouterStateSnapshot
    ): Observable<boolean>|Promise<boolean>|boolean {
        if (this._authStore.isConnected()) {
            return true;
        }
        let navigationExtras: NavigationExtras = {
            queryParams: {}
        };

        if (state.url && state.url.indexOf('account/login') === -1) {
          navigationExtras.queryParams = {redirect: state.url};
        }

        this._router.navigate(['account/login'], navigationExtras);
        return false;
    }
}
