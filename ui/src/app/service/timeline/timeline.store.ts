import { Injectable } from '@angular/core';
import { Event } from 'app/model/event.model';
import { TimelineFilter } from 'app/model/timeline.model';
import { List } from 'immutable';
import { BehaviorSubject, Observable } from 'rxjs';
import { map } from 'rxjs/operators';
import { TimelineService } from './timeline.service';

@Injectable()
export class TimelineStore {

    // List of all project. Use by Navbar
    private _alltimeline: BehaviorSubject<List<Event>> = new BehaviorSubject(List([]));
    private _filter: BehaviorSubject<TimelineFilter> = new BehaviorSubject(null);

    constructor(private _timelineService: TimelineService) {}

    alltimeline(): Observable<List<Event>> {
        this.getMore(0, true);
        return new Observable<List<Event>>(fn => this._alltimeline.subscribe(fn));
    }

    getMore(currentItem: number, erase: boolean): void {
        this._timelineService.get(currentItem).subscribe(events => {
            if (erase) {
                this._alltimeline.next(List.of(...events));
            } else {
                this._alltimeline.next(this._alltimeline.getValue().push(...events));
            }
        });
    }

    add(e: Event): void {
        this._alltimeline.next(this._alltimeline.getValue().unshift(e));
    }

    getFilter(): Observable<TimelineFilter> {
        if (!this._filter.getValue()) {
            this._timelineService.getFilter().subscribe(f => {
                this._filter.next(f);
            });
        }
        return new Observable<TimelineFilter>(fn => this._filter.subscribe(fn));
    }

    saveFilter(filter: TimelineFilter): Observable<boolean> {
        return this._timelineService.saveFilter(filter).pipe(map(() => {
            this._filter.next(filter);
            return true;
        }));
    }
}
