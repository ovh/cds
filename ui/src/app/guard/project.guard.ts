import { Injectable } from '@angular/core';
import { ActivatedRouteSnapshot, Router, RouterStateSnapshot } from '@angular/router';
import { Store } from '@ngxs/store';
import { LoadOpts } from 'app/model/project.model';
import { V2ProjectService } from 'app/service/projectv2/project.service';
import { RouterService } from 'app/service/services.module';
import { SetCurrentProjectV2 } from 'app/store/project-v2.action';
import { FetchProject } from 'app/store/project.action';
import { ProjectState } from 'app/store/project.state';
import { Observable, switchMap } from 'rxjs';
import { map, filter, first } from 'rxjs/operators';

@Injectable()
export class ProjectGuard {

    constructor(
        private _store: Store,
        private _router: Router,
        private _routerService: RouterService
    ) { }

    loadProject(state: RouterStateSnapshot): Observable<boolean> {
        const params = this._routerService.getRouteSnapshotParams({}, state.root);
        const opts = [
            new LoadOpts('withApplicationNames', 'application_names'),
            new LoadOpts('withPipelineNames', 'pipeline_names'),
            new LoadOpts('withWorkflowNames', 'workflow_names'),
            new LoadOpts('withEnvironmentNames', 'environment_names'),
            new LoadOpts('withLabels', 'labels')
        ];

        return this._store.dispatch(new FetchProject({
            projectKey: params['key'],
            opts
        })).pipe(
            switchMap(() => this._store.selectOnce(ProjectState.projectSnapshot)),
            map(p => {
                if (!p) {
                    this._router.navigate([`/project/${params['key']}/explore`]);
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
                switchMap((p) => this._store.dispatch(new SetCurrentProjectV2(p))),
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
