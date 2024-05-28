import { Injectable } from '@angular/core';
import { Action, createSelector, Selector, State, StateContext } from '@ngxs/store';
import * as actionPreferences from './preferences.action';

export class PreferencesStateModel {
    panel: {
        resizing: boolean;
        sizes: { [key: string]: string };
    };
    theme: string;
    projectRunFilters: {
        [projectKey: string]: Array<{
            name: string;
            value: string;
            sort: string;
        }>
    };
    projectTreeExpandState: {
        [projectKey: string]: { [key: string]: boolean };
    };
    messages: { [projectKey: string]: boolean };
}

@State<PreferencesStateModel>({
    name: 'preferences',
    defaults: {
        panel: {
            resizing: false,
            sizes: {}
        },
        theme: 'light',
        projectRunFilters: {},
        projectTreeExpandState: {},
        messages: {}
    }
})
@Injectable()
export class PreferencesState {
    constructor() { }

    static panelSize(key: string) {
        return createSelector(
            [PreferencesState],
            (state: PreferencesStateModel): string => {
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

    static selectProjectRunFilters(projectKey: string) {
        return createSelector(
            [PreferencesState],
            (state: PreferencesStateModel) => {
                return state.projectRunFilters[projectKey] ?? [];
            }
        );
    }

    static selectMessageState(messageKey: string) {
        return createSelector(
            [PreferencesState],
            (state: PreferencesStateModel) => {
                return state.messages[messageKey] ?? false;
            }
        );
    }

    static selectProjectTreeExpandState(projectKey: string) {
        return createSelector(
            [PreferencesState],
            (state: PreferencesStateModel) => {
                return Object.assign({}, state.projectTreeExpandState[projectKey]);
            }
        );
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

    @Action(actionPreferences.SaveProjectWorkflowRunFilter)
    saveProjectWorkflowRunFilter(ctx: StateContext<PreferencesStateModel>, action: actionPreferences.SaveProjectWorkflowRunFilter) {
        const state = ctx.getState();
        let projects = { ...state.projectRunFilters };
        if (!projects[action.payload.projectKey]) { projects[action.payload.projectKey] = []; }
        let searches = (projects[action.payload.projectKey] ?? []).filter(s => s.name !== action.payload.name);
        searches.push({
            name: action.payload.name,
            value: action.payload.value,
            sort: action.payload.sort
        });
        projects[action.payload.projectKey] = searches;
        ctx.setState({
            ...state,
            projectRunFilters: projects
        });
    }

    @Action(actionPreferences.SaveProjectTreeExpandState)
    saveProjectTreeExpandState(ctx: StateContext<PreferencesStateModel>, action: actionPreferences.SaveProjectTreeExpandState) {
        const state = ctx.getState();
        let projects = { ...state.projectTreeExpandState };
        projects[action.payload.projectKey] = action.payload.state;
        ctx.setState({
            ...state,
            projectTreeExpandState: projects
        });
    }

    @Action(actionPreferences.DeleteProjectWorkflowRunFilter)
    deleteWorkflowRunSearch(ctx: StateContext<PreferencesStateModel>, action: actionPreferences.DeleteProjectWorkflowRunFilter) {
        const state = ctx.getState();
        let projects = { ...state.projectRunFilters };
        if (projects[action.payload.projectKey]) {
            projects[action.payload.projectKey] = projects[action.payload.projectKey].filter(s => s.name !== action.payload.name);
        }
        ctx.setState({
            ...state,
            projectRunFilters: projects
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

    @Action(actionPreferences.SaveMessageState)
    saveMessageState(ctx: StateContext<PreferencesStateModel>, action: actionPreferences.SaveMessageState) {
        const state = ctx.getState();
        let messages = { ...state.messages };
        messages[action.payload.messageKey] = action.payload.value;
        ctx.setState({ ...state, messages });
    }
}
