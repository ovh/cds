import {HttpClient, HttpParams} from '@angular/common/http';
import {Injectable} from '@angular/core';
import {Observable} from 'rxjs/Observable';
import {Event} from '../../model/event.model';
import {TimelineFilter} from '../../model/timeline.model';

@Injectable()
export class TimelineService {

    constructor(private _http: HttpClient) {}

    get(currentItem): Observable<Array<Event>> {
        let hp = new HttpParams();
        hp = hp.append('currentItem', currentItem.toString());
        return this._http.get<Array<Event>>('/user/timeline', {params: hp });
    }

    getFilter(): Observable<TimelineFilter> {
        return this._http.get<TimelineFilter>('/user/timeline/filter');
    }

    saveFilter(f: TimelineFilter): Observable<boolean> {
        return this._http.post<boolean>('/user/timeline/filter', f).map(() => true);
    }
}
