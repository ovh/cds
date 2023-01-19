import { Injectable } from '@angular/core';
import { Action, createSelector, Selector, State, StateContext } from '@ngxs/store';
import * as actionPreferences from './preferences.action';

export class PreferencesStateModel {
    panel: { sizes: { [key: string]: number } };
    theme: string;
}

@State<PreferencesStateModel>({
    name: 'preferences',
    defaults: {
        panel: {
            sizes: {}
        },
        theme: 'light'
    }
})
@Injectable()
export class PreferencesState {
    constructor() { }

    static panelSize(key: string) {
        return createSelector(
            [PreferencesState],
            (state: PreferencesStateModel): number => {
                return state.panel.sizes[key] ?? null;
            }
        );
    }

    @Selector()
    static theme(state: PreferencesStateModel) {
        return state.theme;
    }

    @Action(actionPreferences.SavePanelSize)
    savePanelSive(ctx: StateContext<PreferencesStateModel>, action: actionPreferences.SavePanelSize) {
        const state = ctx.getState();
        let sizes = { ...state.panel.sizes };
        sizes[action.payload.panelKey] = action.payload.size;
        ctx.setState({
            ...state,
            panel: { sizes }
        });
    }

    @Action(actionPreferences.SetTheme)
    setTheme(ctx: StateContext<PreferencesStateModel>, action: actionPreferences.SetTheme) {
        const state = ctx.getState();
        ctx.setState({
            ...state,
            theme: action.payload.theme === 'night' ? 'night' : 'light'
        });
    }
}
