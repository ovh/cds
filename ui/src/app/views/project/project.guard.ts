import { HttpErrorResponse } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { ActivatedRouteSnapshot, Router, RouterStateSnapshot } from '@angular/router';
import { Store } from '@ngxs/store';
import { LoadOpts } from 'app/model/project.model';
import { V2ProjectService } from 'app/service/projectv2/project.service';
import { RouterService } from 'app/service/services.module';
import { SetCurrentProjectV2 } from 'app/store/project-v2.action';
import { FetchProject } from 'app/store/project.action';
import { ProjectState } from 'app/store/project.state';
import { firstValueFrom, lastValueFrom, Observable, switchMap } from 'rxjs';
import { map, filter, first, tap } from 'rxjs/operators';

@Injectable()
export class ProjectGuard {

    constructor(
        private _store: Store,
        private _routerService: RouterService
    ) { }

    async loadProject(state: RouterStateSnapshot) {
        const params = this._routerService.getRouteSnapshotParams({}, state.root);
        const opts = [
            new LoadOpts('withApplicationNames', 'application_names'),
            new LoadOpts('withPipelineNames', 'pipeline_names'),
            new LoadOpts('withWorkflowNames', 'workflow_names'),
            new LoadOpts('withEnvironmentNames', 'environment_names'),
            new LoadOpts('withLabels', 'labels')
        ];

        try {
            await lastValueFrom(this._store.dispatch(new FetchProject({
                projectKey: params['key'],
                opts
            })));

            await firstValueFrom(this._store.selectOnce(ProjectState.projectSnapshot));
        } catch (e) {
            if (e instanceof HttpErrorResponse) {
                if (e?.error?.id === 194) {
                    // MFA required error, let the ErrorInterceptor handle it
                    return false;
                }
            }
        }

        return true; // Always try to load the project using v1 to detect if user have the permission to access it
    }

    canActivate(
        route: ActivatedRouteSnapshot,
        state: RouterStateSnapshot
    ): Observable<boolean> | Promise<boolean> | boolean {
        return this.loadProject(state);
    }
}

@Injectable()
export class ProjectExistsGuard {

    constructor(
        private _store: Store,
        private _router: Router,
        private _routerService: RouterService
    ) { }

    loadProject(state: RouterStateSnapshot): boolean {
        const p = this._store.selectSnapshot(ProjectState.projectSnapshot);
        if (!p) {
            const params = this._routerService.getRouteSnapshotParams({}, state.root);
            this._router.navigate([`/project/${params['key']}/explore`]);
            return false;
        }
        return true;  // Always return true if project is loaded
    }

    canActivate(
        route: ActivatedRouteSnapshot,
        state: RouterStateSnapshot
    ): Observable<boolean> | Promise<boolean> | boolean {
        return this.loadProject(state);
    }

    canActivateChild(
        route: ActivatedRouteSnapshot,
        state: RouterStateSnapshot
    ): Observable<boolean> | Promise<boolean> | boolean {
        return this.loadProject(state);
    }
}

@Injectable()
export class ProjectV2Guard {

    constructor(
        private _store: Store,
        private _router: Router,
        private _routerService: RouterService,
        private _v2ProjectService: V2ProjectService
    ) { }

    loadProject(state: RouterStateSnapshot): Observable<boolean> {
        const params = this._routerService.getRouteSnapshotParams({}, state.root);

        return this._v2ProjectService.get(params['key'])
            .pipe(
                tap((p) => this._store.dispatch(new SetCurrentProjectV2(p))),
                map(p => {
                    if (!p) {
                        this._router.navigate([`/project/${params['key']}`]);
                        return null;
                    }
                    return true;
                }),
                filter(exists => exists),
                first()
            );
    }

    canActivate(
        route: ActivatedRouteSnapshot,
        state: RouterStateSnapshot
    ): Observable<boolean> | Promise<boolean> | boolean {
        return this.loadProject(state);
    }

    canActivateChild(
        route: ActivatedRouteSnapshot,
        state: RouterStateSnapshot
    ): Observable<boolean> | Promise<boolean> | boolean {
        return this.loadProject(state);
    }
}
