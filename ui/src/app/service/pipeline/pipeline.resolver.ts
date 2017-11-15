import {Injectable} from '@angular/core';
import {Resolve, ActivatedRouteSnapshot, RouterStateSnapshot} from '@angular/router';
import {Observable} from 'rxjs/Observable';
import {Pipeline} from '../../model/pipeline.model';
import {PipelineStore} from './pipeline.store';

@Injectable()
export class PipelineResolver implements Resolve<Array<Pipeline>> {

    resolve(route: ActivatedRouteSnapshot, state: RouterStateSnapshot): Observable<any>|Promise<any>|any {
        return this.pipStore.getPipelineResolver(route.params['key'], route.params['pipName']);
    }

    constructor(private pipStore: PipelineStore) {}
}
