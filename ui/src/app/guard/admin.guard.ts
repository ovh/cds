import { Injectable } from '@angular/core';
import { ActivatedRouteSnapshot, CanActivate, CanActivateChild, Router, RouterStateSnapshot } from '@angular/router';
import { Store } from '@ngxs/store';
import { AuthentifiedUser } from 'app/model/user.model';
import { AuthenticationState } from 'app/store/authentication.state';
import { Observable } from 'rxjs';

@Injectable()
export class AdminGuard implements CanActivate, CanActivateChild {

    constructor(
        private _store: Store,
        private _router: Router
    ) { }

    isAdmin(): Observable<boolean> {
        return this._store.select(AuthenticationState.user)
            .map((u: AuthentifiedUser): boolean => {
                if (!u) {
                    return null;
                }
                if (!u.isAdmin()) {
                    this._router.navigate(['/']);
                    return null;
                }
                return true;
            })
            .filter(exists => exists !== null)
            .first();
    }

    canActivate(
        route: ActivatedRouteSnapshot,
        state: RouterStateSnapshot
    ): Observable<boolean> | Promise<boolean> | boolean {
        return this.isAdmin();
    }

    canActivateChild(
        route: ActivatedRouteSnapshot,
        state: RouterStateSnapshot
    ): Observable<boolean> | Promise<boolean> | boolean {
        return this.isAdmin();
    }
}
