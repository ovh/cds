import {Injectable} from '@angular/core';
import {ActivatedRoute} from '@angular/router';

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
}
