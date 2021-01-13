import { Injectable } from '@angular/core';
import { ActivatedRouteSnapshot, CanActivate, CanActivateChild, NavigationExtras, Router, RouterStateSnapshot } from '@angular/router';
import { Store } from '@ngxs/store';
import { FetchCurrentAuth } from 'app/store/authentication.action';
import { AuthenticationState } from 'app/store/authentication.state';
import { Observable } from 'rxjs';
import { filter, first, map } from 'rxjs/operators';

@Injectable()
export class AuthenticationGuard implements CanActivate, CanActivateChild {

    constructor(
        private _store: Store,
        private _router: Router
    ) { }

    redirectSignin(url: string): void {
        this._router.navigate(['/auth/signin'], <NavigationExtras>{
            queryParams: {
                redirect: url
            }
        });
    }

    getCurrentAuth(state: RouterStateSnapshot): Observable<boolean> {
        return this._store.select(AuthenticationState.summary)
            .pipe(
                map(s => {
                    if (s) {
                        return true;
                    }
                    this._store.dispatch(new FetchCurrentAuth()).subscribe(
                        _ => { },
                        _ => {
                            this.redirectSignin(state.url);
                        });
                    return null;
                }),
                filter(exists => exists !== null),
                first()
            );
    }

    canActivate(
        route: ActivatedRouteSnapshot,
        state: RouterStateSnapshot
    ): Observable<boolean> | Promise<boolean> | boolean {
        return this.getCurrentAuth(state);
    }

    canActivateChild(
        route: ActivatedRouteSnapshot,
        state: RouterStateSnapshot
    ): Observable<boolean> | Promise<boolean> | boolean {
        return this.getCurrentAuth(state);
    }
}
