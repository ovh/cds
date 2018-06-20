import {HttpClient} from '@angular/common/http';
import {Injectable} from '@angular/core';
import {Observable} from 'rxjs/Observable';
import {Event} from '../../model/event.model';

@Injectable()
export class TimelineService {

    constructor(private _http: HttpClient) {}

    get(): Observable<Array<Event>> {
        return this._http.get<Array<Event>>('/timeline')
    }
}
