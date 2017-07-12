import {Injectable} from '@angular/core';
import {ActivatedRoute, ActivatedRouteSnapshot} from '@angular/router';

@Injectable()
export class RouterService {

    getRouteParams(params: {}, activatedRoute: ActivatedRoute): {} {
        if (activatedRoute) {
            params = Object.assign({}, params, activatedRoute.snapshot.params);
            if (activatedRoute.children) {
                activatedRoute.children.forEach(c => {
                    params = this.getRouteParams(params, c);
                });
            }
        }
        return params;
    }

    getRouteSnapshotParams(params: {}, activatedRoute: ActivatedRouteSnapshot): {} {
        if (activatedRoute) {
            params = Object.assign({}, params, activatedRoute.params);
            if (activatedRoute.children) {
                activatedRoute.children.forEach(c => {
                    params = this.getRouteSnapshotParams(params, c);
                });
            }
        }
        return params;
    }

    getRouteSnapshotQueryParams(params: {}, activatedRoute: ActivatedRouteSnapshot): {} {
        if (activatedRoute) {
            params = Object.assign({}, params, activatedRoute.queryParams);
            if (activatedRoute.children) {
                activatedRoute.children.forEach(c => {
                    params = this.getRouteSnapshotParams(params, c);
                });
            }
        }
        return params;
    }
}
