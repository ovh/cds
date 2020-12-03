import { Injectable } from '@angular/core';
import { ActivatedRouteSnapshot, CanActivate, Router, RouterStateSnapshot } from '@angular/router';
import { Store } from '@ngxs/store';
import { FetchCurrentAuth } from 'app/store/authentication.action';
import { AuthenticationState } from 'app/store/authentication.state';
import { Observable } from 'rxjs';
import { filter, first, map } from 'rxjs/operators';

@Injectable()
export class NoAuthenticationGuard implements CanActivate {

    constructor(
        private _store: Store,
        private _router: Router
    ) { }

    getCurrentAuth(): Observable<boolean> {
        return this._store.select(AuthenticationState.error)
            .pipe(
                map((e: any): boolean => {
                    if (!e) {
                        this._store.dispatch(new FetchCurrentAuth()).subscribe(
                            () => {
                                this._router.navigate(['/']);
                            }
                        );
                        return null;
                    }
                    return true;
                }),
                filter(exists => exists !== null),
                first()
            );
    }

    canActivate(
        route: ActivatedRouteSnapshot,
        state: RouterStateSnapshot
    ): Observable<boolean> | Promise<boolean> | boolean {
        return this.getCurrentAuth();
    }
}
