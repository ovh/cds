import { Injectable } from '@angular/core';
import { ActivatedRouteSnapshot, Resolve, RouterStateSnapshot } from '@angular/router';
import { Store } from '@ngxs/store';
import { FetchProject } from 'app/store/project.action';
import { ProjectState, ProjectStateModel } from 'app/store/project.state';
import { Observable } from 'rxjs';
import { flatMap, map } from 'rxjs/operators';
import { LoadOpts, Project } from '../../model/project.model';
import { RouterService } from '../router/router.service';

@Injectable()
export class ProjectResolver implements Resolve<Project> {

    resolve(route: ActivatedRouteSnapshot, state: RouterStateSnapshot): Observable<any> | Promise<any> | any {
        let params = this.routerService.getRouteSnapshotParams({}, state.root);
        let opts = [
            new LoadOpts('withApplicationNames', 'application_names'),
            new LoadOpts('withPipelineNames', 'pipeline_names'),
            new LoadOpts('withWorkflowNames', 'workflow_names'),
            new LoadOpts('withEnvironmentNames', 'environment_names'),
            new LoadOpts('withLabels', 'labels')
        ];

        return this.store.dispatch(new FetchProject({
            projectKey: params['key'],
            opts
        })).pipe(
            flatMap(() => this.store.selectOnce(ProjectState)),
            map((projectState: ProjectStateModel) => projectState.project)
        );
    }

    constructor(private store: Store, private routerService: RouterService) { }
}

@Injectable()
export class ProjectForWorkflowResolver implements Resolve<Project> {

    resolve(route: ActivatedRouteSnapshot, state: RouterStateSnapshot): Observable<any> | Promise<any> | any {
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

        return this.store.dispatch(new FetchProject({
            projectKey: params['key'],
            opts
        })).pipe(
            flatMap(() => this.store.selectOnce(ProjectState)),
            map((projectState: ProjectStateModel) => projectState.project)
        );
    }

    constructor(private store: Store, private routerService: RouterService) { }
}

@Injectable()
export class ProjectForApplicationResolver implements Resolve<Project> {

    resolve(route: ActivatedRouteSnapshot, state: RouterStateSnapshot): Observable<any> | Promise<any> | any {
        let params = this.routerService.getRouteSnapshotParams({}, state.root);
        let opts = [
            new LoadOpts('withWorkflowNames', 'workflow_names'),
            new LoadOpts('withPipelineNames', 'pipeline_names'),
            new LoadOpts('withApplicationNames', 'application_names'),
            new LoadOpts('withEnvironmentNames', 'environment_names'),
        ];

        return this.store.dispatch(new FetchProject({
            projectKey: params['key'],
            opts
        })).pipe(
            flatMap(() => this.store.selectOnce(ProjectState)),
            map((projectState: ProjectStateModel) => projectState.project)
        );
    }

    constructor(private store: Store, private routerService: RouterService) { }
}
