import {Injectable} from '@angular/core';
import {Resolve, ActivatedRouteSnapshot, RouterStateSnapshot} from '@angular/router';
import {Observable} from 'rxjs/Rx';
import {Project} from '../../model/project.model';
import {ProjectStore} from './project.store';

@Injectable()
export class ProjectResolver implements Resolve<Project> {

    resolve(route: ActivatedRouteSnapshot, state: RouterStateSnapshot): Observable<any>|Promise<any>|any {
        return this.projectStore.getProjectResolver(route.params['key']).map( p => {
            return p;
        });
    }

    constructor(private projectStore: ProjectStore) {}
}
