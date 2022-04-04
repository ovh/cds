import { Injectable } from '@angular/core';
import { BehaviorSubject, Observable } from 'rxjs';
import { NzConfigService } from 'ng-zorro-antd/core/config';

@Injectable()
export class ThemeStore {
    private _theme: BehaviorSubject<string> = new BehaviorSubject(null);

    constructor(private _nzConfigService: NzConfigService) {
        let previousTheme = localStorage.getItem('CDS-Theme');
        if (previousTheme) {
            this.set(previousTheme);
        }
    }

    get() {
        return new Observable<string>(fn => this._theme.subscribe(fn));
    }

    set(t: string) {
        localStorage.setItem('CDS-Theme', t);
        this._theme.next(t);
        if (t === 'night') {
            const style = document.createElement('link');
            style.type = 'text/css';
            style.rel = 'stylesheet';
            style.id = 'dark-theme';
            style.href = 'assets/ng-zorro-antd.dark.min.css';
            document.body.appendChild(style);
        } else {
            const dom = document.getElementById('dark-theme');
            if (dom) {
                dom.remove();
            }
        }

    }
}
