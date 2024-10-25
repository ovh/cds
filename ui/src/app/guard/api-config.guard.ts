import { Injectable } from '@angular/core';
import { ActivatedRouteSnapshot, RouterStateSnapshot, UrlTree } from '@angular/router';
import { Store } from '@ngxs/store';
import { FetchAPIConfig } from 'app/store/config.action';
import { ConfigState } from 'app/store/config.state';
import { Observable } from 'rxjs';
import { filter, first, map } from 'rxjs/operators';

@Injectable()
export class APIConfigGuard  {

    constructor(
        private _store: Store
    ) { }

    canActivateChild(childRoute: ActivatedRouteSnapshot, state: RouterStateSnapshot): boolean | UrlTree | Observable<boolean | UrlTree> | Promise<boolean | UrlTree> {
        return this.getAPIConfig(state);
    }

    getAPIConfig(state: RouterStateSnapshot): Observable<boolean> {
        return this._store.select(ConfigState.api)
            .pipe(
                map(s => {
                    if (s) {
                        return true;
                    }
                    this._store.dispatch(new FetchAPIConfig()).subscribe();
                    return null;
                }),
                filter(exists => exists !== null),
                first()
            );
    }
}
