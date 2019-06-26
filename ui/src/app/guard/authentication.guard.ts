import { Injectable } from '@angular/core';
import { ActivatedRouteSnapshot, CanActivate, CanActivateChild, NavigationExtras, Router, RouterStateSnapshot } from '@angular/router';
import { Store } from '@ngxs/store';
import { User } from 'app/model/user.model';
import { FetchCurrentUser } from 'app/store/authentication.action';
import { AuthenticationState } from 'app/store/authentication.state';
import { Observable } from 'rxjs';

@Injectable()
export class AuthenticationGuard implements CanActivate, CanActivateChild {

    constructor(
        private _store: Store,
        private _router: Router
    ) { }

    getCurrentUser(state: RouterStateSnapshot): Observable<boolean> {
        return this._store.select(AuthenticationState.user)
            .map((u: User): boolean => {
                if (!u) {
                    this._store.dispatch(new FetchCurrentUser()).subscribe(
                        () => { },
                        () => {
                            this._router.navigate(['auth/signin'], <NavigationExtras>{
                                queryParams: {
                                    redirect: state.url
                                }
                            });
                        }
                    );
                    return null;
                }

                return true;
            })
            .filter(exists => {
                return exists !== null;
            })
            .first();
    }

    canActivate(
        route: ActivatedRouteSnapshot,
        state: RouterStateSnapshot
    ): Observable<boolean> | Promise<boolean> | boolean {
        return this.getCurrentUser(state);
    }

    canActivateChild(
        route: ActivatedRouteSnapshot,
        state: RouterStateSnapshot
    ): Observable<boolean> | Promise<boolean> | boolean {
        return this.getCurrentUser(state);
    }
}
