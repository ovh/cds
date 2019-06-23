import { Injectable } from '@angular/core';
import { ActivatedRouteSnapshot, CanActivate, CanActivateChild, RouterStateSnapshot } from '@angular/router';
import { Store } from '@ngxs/store';
import { User } from 'app/model/user.model';
import { FetchCurrentUser } from 'app/store/authentication.action';
import { AuthenticationState } from 'app/store/authentication.state';
import { Observable } from 'rxjs';

@Injectable()
export class AuthenticationGuard implements CanActivate, CanActivateChild {

    constructor(
        private _store: Store
    ) { }

    canActivate(
        route: ActivatedRouteSnapshot,
        state: RouterStateSnapshot
    ): Observable<boolean> | Promise<boolean> | boolean {
        const currentUser = this._store.selectSnapshot(AuthenticationState.user);
        if (currentUser) {
            return true;
        }

        return this._store.dispatch(new FetchCurrentUser()).take(1).map((me: User): boolean => {
            return !!me;
        })

        // if (this._authStore.isConnected()) {
        //     return true;
        // }
        // let navigationExtras: NavigationExtras = {
        //     queryParams: {
        //         redirect: state.url
        //     }
        // };
        // this._router.navigate(['auth/signin'], navigationExtras);
        // return false;
    }

    canActivateChild(
        route: ActivatedRouteSnapshot,
        state: RouterStateSnapshot
    ): Observable<boolean> | Promise<boolean> | boolean {
        return true;

        // if (this._authStore.isConnected()) {
        //     return true;
        // }
        // let navigationExtras: NavigationExtras = {
        //     queryParams: {}
        // }
        // if (state.url && state.url.indexOf('auth/signin') === -1) {
        //     navigationExtras.queryParams = { redirect: state.url };
        // }
        // this._router.navigate(['auth/signin'], navigationExtras);
        // return false;
    }
}
