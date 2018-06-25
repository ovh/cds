import {HttpClient, HttpParams} from '@angular/common/http';
import {Injectable} from '@angular/core';
import {Observable} from 'rxjs/Observable';
import {Event} from '../../model/event.model';

@Injectable()
export class TimelineService {

    constructor(private _http: HttpClient) {}

    get(currentItem): Observable<Array<Event>> {
        let hp = new HttpParams();
        hp = hp.append('currentItem', currentItem.toString());
        return this._http.get<Array<Event>>('/timeline', {params: hp });
    }
}
