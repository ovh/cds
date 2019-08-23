import { Injectable, NgZone } from '@angular/core';
import { EventSourcePolyfill } from 'ng-event-source';
import { Observable } from 'rxjs';

@Injectable()
export class LastUpdateService {

    zone: NgZone;

    constructor() {
        this.zone = new NgZone({ enableLongStackTrace: false });
    }

    getLastUpdate(): Observable<string> {
        let authHeader = {};
        // ADD user AUTH
        // TODO refact
        // let sessionToken = this._authStore.getSessionToken();
        // if (sessionToken) {
        //     authHeader[this._authStore.localStorageSessionKey] = sessionToken;
        // } else {
        //     authHeader['Authorization'] = 'Basic ' + this._authStore.getUser().token;
        // }

        return Observable.create((observer) => {
            let eventSource = new EventSourcePolyfill('/cdsapi/mon/lastupdates/events',
                { headers: authHeader, errorOnTimeout: false, checkActivity: false, connectionTimeout: 5000 });
            eventSource.onmessage = (data => {
                this.zone.run(() => {
                    observer.next(data.data);
                });
            });
        });
    }
}
