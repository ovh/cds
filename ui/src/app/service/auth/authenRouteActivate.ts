import {CanActivate, Router, ActivatedRouteSnapshot, RouterStateSnapshot, CanActivateChild} from '@angular/router';
import {Injectable} from '@angular/core';
import {Observable} from 'rxjs/Rx';
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
        this._router.navigate(['account/login']);
        return false;
    }

    canActivateChild(
        route: ActivatedRouteSnapshot,
        state: RouterStateSnapshot
    ): Observable<boolean>|Promise<boolean>|boolean {
        if (this._authStore.isConnected()) {
            return true;
        }
        this._router.navigate(['account/login']);
        return false;
    }
}
