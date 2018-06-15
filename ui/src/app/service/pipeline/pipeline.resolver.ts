import {Injectable} from '@angular/core';
import {ActivatedRouteSnapshot, Resolve, RouterStateSnapshot} from '@angular/router';
import {Observable} from 'rxjs';
import {Pipeline} from '../../model/pipeline.model';
import {PipelineStore} from './pipeline.store';

@Injectable()
export class PipelineResolver implements Resolve<Array<Pipeline>> {

    resolve(route: ActivatedRouteSnapshot, state: RouterStateSnapshot): Observable<any>|Promise<any>|any {
        return this.pipStore.getPipelineResolver(route.params['key'], route.params['pipName']);
    }

    constructor(private pipStore: PipelineStore) {}
}
