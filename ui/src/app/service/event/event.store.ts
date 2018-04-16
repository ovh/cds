import {Injectable} from '@angular/core';
import {BehaviorSubject} from 'rxjs/BehaviorSubject';
import {EventSubscription} from '../../model/event.model';
import {HttpClient} from '@angular/common/http';
import {Observable} from 'rxjs/Observable';

@Injectable()
export class EventStore {

    _eventFilter: BehaviorSubject<EventSubscription> = new BehaviorSubject(null);

    constructor(private _http: HttpClient) {
    }

    setUUID(uuid: string): void {
        let f = this._eventFilter.getValue();
        if (!f) {
            f = new EventSubscription();
            f.uuid = uuid;
        }
        this._eventFilter.next(f);
    }

    changeFilter(filter: EventSubscription): Observable<boolean> {
        filter.uuid = this._eventFilter.getValue().uuid;
        return this._http.post('/events/subscribe', filter).map(() => {
            this._eventFilter.next(filter);
            return true;
        });
    }
}
