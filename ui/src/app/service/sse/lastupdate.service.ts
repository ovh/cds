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
        return Observable.create((observer) => {
            let eventSource = new EventSourcePolyfill('/cdsapi/mon/lastupdates/events',
                { errorOnTimeout: false, checkActivity: false, connectionTimeout: 5000 });
            eventSource.onmessage = (data => {
                this.zone.run(() => {
                    observer.next(data.data);
                });
            });
        });
    }
}
