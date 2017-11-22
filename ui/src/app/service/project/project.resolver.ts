import {Injectable} from '@angular/core';
import {Resolve, ActivatedRouteSnapshot, RouterStateSnapshot} from '@angular/router';
import {Observable} from 'rxjs/Observable';
import {Project} from '../../model/project.model';
import {ProjectStore, LoadOpts} from './project.store';
import {RouterService} from '../router/router.service';

@Injectable()
export class ProjectResolver implements Resolve<Project> {

    resolve(route: ActivatedRouteSnapshot, state: RouterStateSnapshot): Observable<any>|Promise<any>|any {
        let params = this.routerService.getRouteSnapshotParams({}, state.root);
        let opts = [
          new LoadOpts('withVariables', 'variables'),
          new LoadOpts('withPipelines', 'pipelines'),
          new LoadOpts('withEnvironments', 'environments'),
          new LoadOpts('withApplications', 'applications'),
          new LoadOpts('withApplicationPipelines', 'applications.pipelines'),
          new LoadOpts('withGroups', 'groups'),
          new LoadOpts('withPermission', 'permissions'),
          new LoadOpts('withWorkflows', 'workflows')
        ];

        return this.projectStore.getProjectResolver(params['key'], opts);
    }

    constructor(private projectStore: ProjectStore, private routerService: RouterService) {}
}

@Injectable()
export class ProjectForApplicationResolver implements Resolve<Project> {

    resolve(route: ActivatedRouteSnapshot, state: RouterStateSnapshot): Observable<any>|Promise<any>|any {
        let params = this.routerService.getRouteSnapshotParams({}, state.root);
        let opts = [
          new LoadOpts('withPipelines', 'pipelines'),
          new LoadOpts('withEnvironments', 'environments'),
          new LoadOpts('withApplicationPipelines', 'applications.pipelines'),
          new LoadOpts('withGroups', 'groups'),
          new LoadOpts('withPermission', 'permissions'),
        ];

        return this.projectStore.getProjectResolver(params['key'], opts);
    }

    constructor(private projectStore: ProjectStore, private routerService: RouterService) {}
}
