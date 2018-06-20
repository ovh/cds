import {Injectable} from '@angular/core';
import {List} from 'immutable';
import {BehaviorSubject, Observable} from 'rxjs/index';
import {Event} from '../../model/event.model';
import {TimelineService} from './timeline.service';

@Injectable()
export class TimelineStore {

    // List of all project. Use by Navbar
    private _alltimeline: BehaviorSubject<List<Event>> = new BehaviorSubject(List([]));

    constructor(private _timelineService: TimelineService) {}

    alltimeline(): Observable<List<Event>> {
        if (this._alltimeline.getValue().size === 0) {
            this._timelineService.get().subscribe(events => {
               this._alltimeline.next(this._alltimeline.getValue().push(...events));
            });
        }
        return new Observable<List<Event>>(fn => this._alltimeline.subscribe(fn));
    }
}
