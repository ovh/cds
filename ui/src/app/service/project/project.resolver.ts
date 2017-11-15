import {Injectable} from '@angular/core';
import {Resolve, ActivatedRouteSnapshot, RouterStateSnapshot} from '@angular/router';
import {Observable} from 'rxjs/Observable';
import {Project} from '../../model/project.model';
import {ProjectStore} from './project.store';
import {RouterService} from '../router/router.service';

@Injectable()
export class ProjectResolver implements Resolve<Project> {

    resolve(route: ActivatedRouteSnapshot, state: RouterStateSnapshot): Observable<any>|Promise<any>|any {
        let params = this.routerService.getRouteSnapshotParams({}, state.root);
        return this.projectStore.getProjectResolver(params['key']);
    }

    constructor(private projectStore: ProjectStore, private routerService: RouterService) {}
}
