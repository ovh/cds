import { Injectable } from '@angular/core';
import { ActivatedRouteSnapshot, Router, RouterStateSnapshot } from '@angular/router';
import { FeatureEnabledResponse } from 'app/model/feature.model';
import { FeatureNames, FeatureService } from 'app/service/feature/feature.service';
import { RouterService } from 'app/service/services.module';
import { Observable } from 'rxjs';
import { filter, first, map } from 'rxjs/operators';

@Injectable()
export class FeatureGuard {
    constructor(
        private _router: Router,
        private _featureService: FeatureService,
        private _routerService: RouterService
    ) { }

    isAllAsCodeActive(projectKey: string): Observable<boolean> | boolean {
        if (!projectKey) {
            return false;
        }
        return this._featureService.isEnabled(FeatureNames.AllAsCode, { project_key: projectKey })
            .pipe(
                map((r: FeatureEnabledResponse): boolean => {
                    if (!r.enabled) {
                        this._router.navigate(['/']);
                        return null;
                    }
                    return true;
                }),
                filter(active => active !== null),
                first()
            );
    }

    canActivate(
        route: ActivatedRouteSnapshot,
        state: RouterStateSnapshot
    ): Observable<boolean> | Promise<boolean> | boolean {
        const allParamsSnapshot = this._routerService.getRouteSnapshotParams({}, state.root);
        switch (route.data.feature) {
            case FeatureNames.AllAsCode:
                return this.isAllAsCodeActive(allParamsSnapshot['key']);
            default:
                return false;
        }
    }
}
