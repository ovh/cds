import {Injectable} from '@angular/core';
import {Resolve, ActivatedRouteSnapshot, RouterStateSnapshot} from '@angular/router';
import {Observable} from 'rxjs/Rx';
import {Project} from '../../model/project.model';
import {ProjectStore, LoadOpts} from './project.store';
import {RouterService} from '../router/router.service';

@Injectable()
export class ProjectResolver implements Resolve<Project> {

    resolve(route: ActivatedRouteSnapshot, state: RouterStateSnapshot): Observable<any>|Promise<any>|any {
        let params = this.routerService.getRouteSnapshotParams({}, state.root);
        let opts = [
          new LoadOpts('withVariables', 'variables', this.projectStore.getProjectVariablesResolver),
          new LoadOpts('withPipelines', 'pipelines', this.projectStore.getProjectPipelinesResolver),
          new LoadOpts('withEnvironments', 'environments', this.projectStore.getProjectEnvironmentsResolver),
          new LoadOpts('withApplications', 'applications', null),
          new LoadOpts('withApplicationPipelines', 'applications.pipelines', null),
          new LoadOpts('withGroups', 'groups', null),
          new LoadOpts('withPermission', 'permissions', null),
          new LoadOpts('withWorkflows', 'workflows', null)
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
          new LoadOpts('withPipelines', 'pipelines', this.projectStore.getProjectPipelinesResolver),
          new LoadOpts('withEnvironments', 'environments', this.projectStore.getProjectEnvironmentsResolver),
          new LoadOpts('withApplicationPipelines', 'applications.pipelines', null),
          new LoadOpts('withGroups', 'groups', null),
          new LoadOpts('withPermission', 'permissions', null),
        ];

        return this.projectStore.getProjectResolver(params['key'], opts);
    }

    constructor(private projectStore: ProjectStore, private routerService: RouterService) {}
}
