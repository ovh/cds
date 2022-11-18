import { Injectable } from '@angular/core';
import { ActivatedRouteSnapshot, CanActivate, Router, RouterStateSnapshot } from '@angular/router';
import { FeatureEnabledResponse } from 'app/model/feature.model';
import { FeatureNames, FeatureService } from 'app/service/feature/feature.service';
import { Observable } from 'rxjs';
import { filter, first, map } from 'rxjs/operators';

@Injectable()
export class FeatureGuard implements CanActivate {
    constructor(
        private _router: Router,
        private _featureService: FeatureService
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

    isWorkflowV3Active(projectKey: string): Observable<boolean> | boolean {
        if (!projectKey) {
            return false;
        }
        return this._featureService.isEnabled(FeatureNames.WorkflowV3, { project_key: projectKey })
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
        switch (route.data.feature) {
            case FeatureNames.WorkflowV3:
                return this.isWorkflowV3Active(route.params.key);
            case FeatureNames.AllAsCode:
                return this.isAllAsCodeActive(route.params.key);
            default:
                return false;
        }
    }
}
