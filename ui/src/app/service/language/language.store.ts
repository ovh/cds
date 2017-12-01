import {Injectable} from '@angular/core';
import {BehaviorSubject} from 'rxjs/BehaviorSubject';
import {Observable} from 'rxjs/Observable';

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
        localStorage.setItem('CDS-Language', l);
        this._language.next(l);
    }
}
