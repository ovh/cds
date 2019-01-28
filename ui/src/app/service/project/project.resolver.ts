import {Injectable} from '@angular/core';
import {ActivatedRouteSnapshot, Resolve, RouterStateSnapshot} from '@angular/router';
import {Observable} from 'rxjs';
import {first} from 'rxjs/operators';
import {LoadOpts, Project} from '../../model/project.model';
import {RouterService} from '../router/router.service';
import {ProjectStore} from './project.store';

@Injectable()
export class ProjectResolver implements Resolve<Project> {

    resolve(route: ActivatedRouteSnapshot, state: RouterStateSnapshot): Observable<any>|Promise<any>|any {
        let params = this.routerService.getRouteSnapshotParams({}, state.root);
        let opts = [
            new LoadOpts('withApplicationNames', 'application_names'),
            new LoadOpts('withPipelineNames', 'pipeline_names'),
            new LoadOpts('withWorkflowNames', 'workflow_names'),
            new LoadOpts('withLabels', 'labels')
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
            new LoadOpts('withIntegrations', 'integrations'),
            new LoadOpts('withLabels', 'labels'),
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
          new LoadOpts('withLabels', 'labels'),
          new LoadOpts('withEnvironments', 'environments'),
        ];

        return this.projectStore.getProjectResolver(params['key'], opts).pipe(first());
    }

    constructor(private projectStore: ProjectStore, private routerService: RouterService) {}
}
