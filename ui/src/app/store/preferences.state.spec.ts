import { TestBed, waitForAsync } from '@angular/core/testing';
import { NgxsModule, Store } from '@ngxs/store';
import { SetSystemTheme, SetTheme } from 'app/store/preferences.action';
import { PreferencesState, THEME_AUTO, THEME_LIGHT, THEME_NIGHT } from 'app/store/preferences.state';

describe('Preferences', () => {
    let store: Store;

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            imports: [NgxsModule.forRoot([PreferencesState])]
        }).compileComponents();

        store = TestBed.inject(Store);
    }));

    it('should default to the auto mode', () => {
        expect(store.selectSnapshot(PreferencesState.themeMode)).toBe(THEME_AUTO);
    });

    it('should follow the system theme in auto mode', () => {
        store.dispatch(new SetTheme({ theme: THEME_AUTO }));

        store.dispatch(new SetSystemTheme({ theme: THEME_NIGHT }));
        expect(store.selectSnapshot(PreferencesState.theme)).toBe(THEME_NIGHT);

        store.dispatch(new SetSystemTheme({ theme: THEME_LIGHT }));
        expect(store.selectSnapshot(PreferencesState.theme)).toBe(THEME_LIGHT);
    });

    it('should ignore the system theme when a theme is forced', () => {
        store.dispatch(new SetSystemTheme({ theme: THEME_NIGHT }));
        store.dispatch(new SetTheme({ theme: THEME_LIGHT }));

        expect(store.selectSnapshot(PreferencesState.themeMode)).toBe(THEME_LIGHT);
        expect(store.selectSnapshot(PreferencesState.theme)).toBe(THEME_LIGHT);
    });

    it('should fall back to the auto mode for an unknown theme', () => {
        store.dispatch(new SetTheme({ theme: 'whatever' }));

        expect(store.selectSnapshot(PreferencesState.themeMode)).toBe(THEME_AUTO);
    });
});
