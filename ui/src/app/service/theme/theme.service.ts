import { Injectable } from '@angular/core';
import { Store } from '@ngxs/store';
import { SetSystemTheme } from 'app/store/preferences.action';
import { THEME_LIGHT, THEME_NIGHT } from 'app/store/preferences.state';

@Injectable()
export class ThemeService {

    constructor(private _store: Store) {
        if (!window.matchMedia) {
            return;
        }
        const query = window.matchMedia('(prefers-color-scheme: dark)');
        this.publish(query.matches);
        query.addEventListener('change', e => this.publish(e.matches));
    }

    private publish(dark: boolean): void {
        this._store.dispatch(new SetSystemTheme({ theme: dark ? THEME_NIGHT : THEME_LIGHT }));
    }
}
