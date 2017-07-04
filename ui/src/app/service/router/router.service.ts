import {Injectable} from '@angular/core';
import {ActivatedRoute, ActivatedRouteSnapshot} from '@angular/router';

@Injectable()
export class RouterService {

    getRouteParams(params: {}, activatedRoute: ActivatedRoute): {} {
        if (activatedRoute) {
            if (activatedRoute.snapshot.params) {
                if (activatedRoute.snapshot.params['key']) {
                    params['key'] = activatedRoute.snapshot.params['key'];
                }
                if (activatedRoute.snapshot.params['pipName']) {
                    params['pipName'] = activatedRoute.snapshot.params['pipName'];
                }
                if (activatedRoute.snapshot.params['appName']) {
                    params['appName'] = activatedRoute.snapshot.params['appName'];
                }
                if (activatedRoute.snapshot.params['buildNumber']) {
                    params['buildNumber'] = activatedRoute.snapshot.params['buildNumber'];
                }
            }
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
            if (activatedRoute.params) {
                if (activatedRoute.params['key']) {
                    params['key'] = activatedRoute.params['key'];
                }
                if (activatedRoute.params['pipName']) {
                    params['pipName'] = activatedRoute.params['pipName'];
                }
                if (activatedRoute.params['appName']) {
                    params['appName'] = activatedRoute.params['appName'];
                }
                if (activatedRoute.params['buildNumber']) {
                    params['buildNumber'] = activatedRoute.params['buildNumber'];
                }
            }
            if (activatedRoute.children) {
                activatedRoute.children.forEach(c => {
                    params = this.getRouteSnapshotParams(params, c);
                });
            }
        }
        return params;
    }
}
