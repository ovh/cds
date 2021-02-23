import { Injectable } from '@angular/core';
import { ActivatedRouteSnapshot, CanActivate, CanActivateChild, Router, RouterStateSnapshot } from '@angular/router';
import { Store } from '@ngxs/store';
import { AuthSummary } from 'app/model/user.model';
import { AuthenticationState } from 'app/store/authentication.state';
import { Observable } from 'rxjs';
import { filter, first, map } from 'rxjs/operators';

@Injectable()
export class MaintainerGuard implements CanActivate, CanActivateChild {

    constructor(
        private _store: Store,
        private _router: Router
    ) { }

    isMaintainer(): Observable<boolean> {
        return this._store.select(AuthenticationState.summary)
            .pipe(
                map((s: AuthSummary): boolean => {
                    if (!s) {
                        return null;
                    }
                    if (!s.isMaintainer()) {
                        this._router.navigate(['/']);
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
        return this.isMaintainer();
    }

    canActivateChild(
        route: ActivatedRouteSnapshot,
        state: RouterStateSnapshot
    ): Observable<boolean> | Promise<boolean> | boolean {
        return this.isMaintainer();
    }
}
