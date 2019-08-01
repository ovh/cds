import {Injectable} from '@angular/core';
import {ActivatedRoute, ActivatedRouteSnapshot} from '@angular/router';

@Injectable()
export class RouterService {

    getActivatedRoute(activatedRoute: ActivatedRoute): ActivatedRoute {
        let activeRoute = activatedRoute;
        if (activatedRoute) {
            if (activatedRoute.children) {
                activatedRoute.children.forEach(c => {
                    activeRoute = this.getActivatedRoute(c);
                });
            }
        }
        return activeRoute;
    }

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
}
