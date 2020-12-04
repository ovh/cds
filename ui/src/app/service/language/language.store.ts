import { Injectable } from '@angular/core';
import * as moment from 'moment';
import { BehaviorSubject, Observable } from 'rxjs';

@Injectable()
export class LanguageStore {
    private _language: BehaviorSubject<string> = new BehaviorSubject(null);

    constructor() {
        this._language.next('en')
        moment.locale('en');
    }

    get() {
        return new Observable<string>(fn => this._language.subscribe(fn));
    }
}
