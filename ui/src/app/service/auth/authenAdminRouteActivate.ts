import {Injectable} from '@angular/core';
import {ActivatedRouteSnapshot, CanActivate, CanActivateChild, NavigationExtras, Router, RouterStateSnapshot} from '@angular/router';
import {Observable} from 'rxjs';
import {AuthentificationStore} from './authentification.store';


@Injectable()
export class CanActivateAuthAdminRoute implements CanActivate , CanActivateChild {

    constructor(private _router: Router, private _authStore: AuthentificationStore) {}

    canActivate(
        route: ActivatedRouteSnapshot,
        state: RouterStateSnapshot
    ): Observable<boolean>|Promise<boolean>|boolean {
        if (this._authStore.isConnected() && this._authStore.isAdmin()) {
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
        if (this._authStore.isConnected() && this._authStore.isAdmin()) {
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
