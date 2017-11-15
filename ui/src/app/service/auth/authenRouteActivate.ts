import {CanActivate, Router, ActivatedRouteSnapshot, RouterStateSnapshot, CanActivateChild, NavigationExtras} from '@angular/router';
import {Injectable} from '@angular/core';
import {Observable} from 'rxjs/Observable';
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
            queryParams: {
                redirect: state.url
            }
        };

        this._router.navigate(['account/login'], navigationExtras);
        return false;
    }
}
