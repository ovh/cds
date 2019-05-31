import { Injectable } from '@angular/core';
import { BehaviorSubject, Observable } from 'rxjs';

@Injectable()
export class ThemeStore {
    private _theme: BehaviorSubject<string> = new BehaviorSubject(null);

    constructor() {
        let previousTheme = localStorage.getItem('CDS-Theme');
        if (previousTheme) {
            this._theme.next(previousTheme);
        }
    }

    get() {
        return new Observable<string>(fn => this._theme.subscribe(fn));
    }

    set(t: string) {
        localStorage.setItem('CDS-Theme', t);
        this._theme.next(t);
    }
}
