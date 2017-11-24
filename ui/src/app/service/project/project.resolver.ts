import {Injectable} from '@angular/core';
import {Resolve, ActivatedRouteSnapshot, RouterStateSnapshot} from '@angular/router';
import {Observable} from 'rxjs/Observable';
import {first} from 'rxjs/operators';
import {Project, LoadOpts} from '../../model/project.model';
import {ProjectStore} from './project.store';
import {RouterService} from '../router/router.service';

@Injectable()
export class ProjectResolver implements Resolve<Project> {

    resolve(route: ActivatedRouteSnapshot, state: RouterStateSnapshot): Observable<any>|Promise<any>|any {
        let params = this.routerService.getRouteSnapshotParams({}, state.root);
        let opts = [];

        return this.projectStore.getProjectResolver(params['key'], opts).pipe(first());
    }

    constructor(private projectStore: ProjectStore, private routerService: RouterService) {}
}

@Injectable()
export class ProjectForWorkflowResolver implements Resolve<Project> {

    resolve(route: ActivatedRouteSnapshot, state: RouterStateSnapshot): Observable<any>|Promise<any>|any {
        let params = this.routerService.getRouteSnapshotParams({}, state.root);
        let opts = [
            new LoadOpts('withPipelines', 'pipelines'),
            new LoadOpts('withApplications', 'applications'),
            new LoadOpts('withEnvironments', 'environments'),
        ];

        return this.projectStore.getProjectResolver(params['key'], opts).pipe(first());
    }

    constructor(private projectStore: ProjectStore, private routerService: RouterService) {}
}

@Injectable()
export class ProjectForPipelineCreateResolver implements Resolve<Project> {

    resolve(route: ActivatedRouteSnapshot, state: RouterStateSnapshot): Observable<any>|Promise<any>|any {
        let params = this.routerService.getRouteSnapshotParams({}, state.root);
        let opts = [
          new LoadOpts('withPipelineNames', 'pipeline_names'),
        ];

        return this.projectStore.getProjectResolver(params['key'], opts).pipe(first());
    }

    constructor(private projectStore: ProjectStore, private routerService: RouterService) {}
}

@Injectable()
export class ProjectForApplicationResolver implements Resolve<Project> {

    resolve(route: ActivatedRouteSnapshot, state: RouterStateSnapshot): Observable<any>|Promise<any>|any {
        let params = this.routerService.getRouteSnapshotParams({}, state.root);
        let opts = [
          new LoadOpts('withPipelineNames', 'pipeline_names'),
          new LoadOpts('withApplicationNames', 'application_names'),
          new LoadOpts('withEnvironments', 'environments'),
        ];

        return this.projectStore.getProjectResolver(params['key'], opts).pipe(first());
    }

    constructor(private projectStore: ProjectStore, private routerService: RouterService) {}
}
