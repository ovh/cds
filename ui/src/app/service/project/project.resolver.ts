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
        let opts = [
            new LoadOpts('withApplicationNames', 'application_names'),
            new LoadOpts('withPipelineNames', 'pipeline_names'),
            new LoadOpts('withWorkflowNames', 'workflow_names')
        ];

        return this.projectStore.getProjectResolver(params['key'], opts).pipe(first());
    }

    constructor(private projectStore: ProjectStore, private routerService: RouterService) {}
}

@Injectable()
export class ProjectForWorkflowResolver implements Resolve<Project> {

    resolve(route: ActivatedRouteSnapshot, state: RouterStateSnapshot): Observable<any>|Promise<any>|any {
        let params = this.routerService.getRouteSnapshotParams({}, state.root);
        let opts = [
            new LoadOpts('withWorkflowNames', 'workflow_names'),
            new LoadOpts('withPipelineNames', 'pipeline_names'),
            new LoadOpts('withApplicationNames', 'application_names'),
            new LoadOpts('withEnvironments', 'environments'),
            new LoadOpts('withPlatforms', 'platforms'),
            new LoadOpts('withKeys', 'keys')
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
          new LoadOpts('withWorkflowNames', 'workflow_names'),
          new LoadOpts('withPipelineNames', 'pipeline_names'),
          new LoadOpts('withApplicationNames', 'application_names'),
          new LoadOpts('withEnvironments', 'environments'),
        ];

        return this.projectStore.getProjectResolver(params['key'], opts).pipe(first());
    }

    constructor(private projectStore: ProjectStore, private routerService: RouterService) {}
}
