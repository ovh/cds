import {Injectable, NgZone} from '@angular/core';
import {Observable} from 'rxjs/Observable';
import {environment} from '../../../environments/environment';
import {EventSourcePolyfill} from 'ng-event-source';
import {AuthentificationStore} from '../auth/authentification.store';

@Injectable()
export class LastUpdateService {

    zone: NgZone;

    constructor(private _authStore: AuthentificationStore) {
        this.zone = new NgZone({enableLongStackTrace: false});
    }

    getLastUpdate(): Observable<string> {
        let authHeader = {};
        // ADD user AUTH
        let sessionToken = this._authStore.getSessionToken();
        if (sessionToken) {
            authHeader[this._authStore.localStorageSessionKey] = sessionToken;
        } else {
            authHeader['Authorization'] = 'Basic ' + this._authStore.getUser().token;
        }

        return Observable.create((observer) => {
            let eventSource = new EventSourcePolyfill(environment.apiURL + '/mon/lastupdates/events',
                {headers: authHeader, errorOnTimeout: false, checkActivity: false, connectionTimeout: 5000});
            eventSource.onmessage = (data => {
                this.zone.run(() => {
                    observer.next(data.data);
                });
            });
        });
    }
}
