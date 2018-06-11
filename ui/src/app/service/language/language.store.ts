import {Injectable} from '@angular/core';
import {BehaviorSubject, Observable} from 'rxjs';
import * as moment from 'moment';

@Injectable()
export class LanguageStore {

    private _language: BehaviorSubject<string> = new BehaviorSubject(null);

    constructor() {
        let previousLanguage = localStorage.getItem('CDS-Language');
        if (previousLanguage) {
            this._language.next(previousLanguage);
        }
    }

    get() {
        return new Observable<string>(fn => this._language.subscribe(fn));
    }

    set(l: string) {
        moment.locale(l);
        localStorage.setItem('CDS-Language', l);

        this._language.next(l);
    }
}
