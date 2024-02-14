import { Injectable } from '@angular/core';
import { Action, createSelector, Selector, State, StateContext } from '@ngxs/store';
import * as actionPreferences from './preferences.action';

export class PreferencesStateModel {
    panel: {
        resizing: boolean;
        sizes: { [key: string]: number };
    };
    theme: string;
    searches: Array<{
        name: string;
        value: string;
    }>;
}

@State<PreferencesStateModel>({
    name: 'preferences',
    defaults: {
        panel: {
            resizing: false,
            sizes: {}
        },
        theme: 'light',
        searches: []
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

    @Selector()
    static resizing(state: PreferencesStateModel) {
        return state.panel.resizing;
    }

    @Selector()
    static searches(state: PreferencesStateModel) {
        return state.searches;
    }

    @Action(actionPreferences.SetPanelResize)
    setPanelResize(ctx: StateContext<PreferencesStateModel>, action: actionPreferences.SetPanelResize) {
        const state = ctx.getState();
        ctx.setState({
            ...state,
            panel: {
                ...state.panel,
                resizing: action.payload.resizing
            }
        });
    }

    @Action(actionPreferences.SavePanelSize)
    savePanelSive(ctx: StateContext<PreferencesStateModel>, action: actionPreferences.SavePanelSize) {
        const state = ctx.getState();
        let sizes = { ...state.panel.sizes };
        sizes[action.payload.panelKey] = action.payload.size;
        ctx.setState({
            ...state,
            panel: {
                ...state.panel,
                sizes
            }
        });
    }

    @Action(actionPreferences.SaveWorkflowRunSearch)
    saveWorkflowRunSearch(ctx: StateContext<PreferencesStateModel>, action: actionPreferences.SaveWorkflowRunSearch) {
        const state = ctx.getState();
        let searches = (state.searches ?? []).filter(s => s.name !== action.payload.name);
        searches.push({
            name: action.payload.name,
            value: action.payload.value
        });
        ctx.setState({
            ...state,
            searches
        });
    }

    @Action(actionPreferences.DeleteWorkflowRunSearch)
    deleteWorkflowRunSearch(ctx: StateContext<PreferencesStateModel>, action: actionPreferences.DeleteWorkflowRunSearch) {
        const state = ctx.getState();
        ctx.setState({
            ...state,
            searches: (state.searches ?? []).filter(s => s.name !== action.payload.name)
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
