import { Injectable } from '@angular/core';
import { ActivatedRouteSnapshot, NavigationExtras, Router, RouterStateSnapshot } from '@angular/router';
import { Store } from '@ngxs/store';
import { FetchCurrentAuth } from 'app/store/authentication.action';
import { AuthenticationState } from 'app/store/authentication.state';
import { FetchAPIConfig } from 'app/store/config.action';
import { ConfigState } from 'app/store/config.state';
import { firstValueFrom, lastValueFrom, Observable } from 'rxjs';

@Injectable()
export class AuthenticationGuard {

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

    async getCurrentAuth(state: RouterStateSnapshot) {
        const summary = await firstValueFrom(this._store.select(AuthenticationState.summary));
        if (!summary) {
            try {
                await lastValueFrom(this._store.dispatch(new FetchCurrentAuth()));
            } catch (e) {
                this.redirectSignin(state.url);
                return false;
            }
        }

        const apiConfig = await firstValueFrom(this._store.select(ConfigState.api));
        if (!apiConfig) {
            try {
                await lastValueFrom(this._store.dispatch(new FetchAPIConfig()));
            } catch (e) {
                this.redirectSignin(state.url);
                return false;
            }
        }

        return true;
    }

    canActivateChild(
        route: ActivatedRouteSnapshot,
        state: RouterStateSnapshot
    ): Observable<boolean> | Promise<boolean> | boolean {
        return this.getCurrentAuth(state);
    }
}
