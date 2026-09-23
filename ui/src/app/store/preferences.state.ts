import { Injectable } from '@angular/core';
import { Action, createSelector, NgxsOnInit, Selector, State, StateContext } from '@ngxs/store';
import * as actionPreferences from './preferences.action';

export const THEME_AUTO = 'auto';
export const THEME_LIGHT = 'light';
export const THEME_NIGHT = 'night';

export class PreferencesStateModel {
    panel: {
        resizing: boolean;
        sizes: { [key: string]: string };
    };
    // Kept only to migrate a theme forced before the auto mode existed, see ngxsOnInit
    theme?: string;
    themeMode: string;
    systemTheme: string;
    projectRunFilters: {
        [projectKey: string]: Array<{
            name: string;
            value: string;
            sort: string;
            order: number;
        }>
    };
    projectTreeExpandState: {
        [projectKey: string]: { [key: string]: boolean };
    };
    projectRefSelectState: {
        [projectKey: string]: { [key: string]: string };
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
        themeMode: THEME_AUTO,
        systemTheme: THEME_LIGHT,
        projectRunFilters: {},
        projectTreeExpandState: {},
        projectRefSelectState: {},
        messages: {}
    }
})
@Injectable()
export class PreferencesState implements NgxsOnInit {
    constructor() { }

    ngxsOnInit(ctx: StateContext<PreferencesStateModel>) {
        const state = ctx.getState();
        if (state.themeMode) { return; }
        ctx.setState({
            ...state,
            themeMode: state.theme === THEME_NIGHT ? THEME_NIGHT : THEME_AUTO,
            theme: undefined
        });
    }

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
        const mode = state.themeMode ?? THEME_AUTO;
        const resolved = mode === THEME_AUTO ? state.systemTheme : mode;
        return resolved === THEME_NIGHT ? THEME_NIGHT : THEME_LIGHT;
    }

    @Selector()
    static themeMode(state: PreferencesStateModel) {
        return state.themeMode ?? THEME_AUTO;
    }

    @Selector()
    static resizing(state: PreferencesStateModel) {
        return state.panel.resizing;
    }

    static selectProjectRunFilters(projectKey: string) {
        return createSelector(
            [PreferencesState],
            (state: PreferencesStateModel) => {
                return state.projectRunFilters?.[projectKey] ?? [];
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
                return Object.assign({}, state.projectTreeExpandState ? state.projectTreeExpandState[projectKey] : {});
            }
        );
    }

    static selectProjectRefSelectState(projectKey: string) {
        return createSelector(
            [PreferencesState],
            (state: PreferencesStateModel) => {
                return Object.assign({}, state.projectRefSelectState ? state.projectRefSelectState[projectKey] : {});
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
        let projects = { ...(state.projectRunFilters || {}) };
        if (!projects[action.payload.projectKey]) { projects[action.payload.projectKey] = []; }
        let searches = (projects[action.payload.projectKey] ?? []).filter(s => s.name !== action.payload.name);
        
        // Calculate order for new filter
        const maxOrder = searches.length > 0 ? Math.max(...searches.map(s => s.order || 0)) : -1;
        
        searches.push({
            name: action.payload.name,
            value: action.payload.value,
            sort: action.payload.sort,
            order: maxOrder + 1
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

    @Action(actionPreferences.SaveProjectRefSelectState)
    saveProjectRefSelectState(ctx: StateContext<PreferencesStateModel>, action: actionPreferences.SaveProjectRefSelectState) {
        const state = ctx.getState();
        let projects = { ...state.projectRefSelectState };
        projects[action.payload.projectKey] = action.payload.state;
        ctx.setState({
            ...state,
            projectRefSelectState: projects
        });
    }

    @Action(actionPreferences.DeleteProjectWorkflowRunFilter)
    deleteWorkflowRunSearch(ctx: StateContext<PreferencesStateModel>, action: actionPreferences.DeleteProjectWorkflowRunFilter) {
        const state = ctx.getState();
        let projects = { ...(state.projectRunFilters || {}) };
        if (projects[action.payload.projectKey]) {
            projects[action.payload.projectKey] = projects[action.payload.projectKey].filter(s => s.name !== action.payload.name);
        }
        ctx.setState({
            ...state,
            projectRunFilters: projects
        });
    }

    @Action(actionPreferences.ReorderProjectWorkflowRunFilters)
    reorderProjectWorkflowRunFilters(ctx: StateContext<PreferencesStateModel>, action: actionPreferences.ReorderProjectWorkflowRunFilters) {
        const state = ctx.getState();
        let projects = { ...(state.projectRunFilters || {}) };
        projects[action.payload.projectKey] = action.payload.filters;
        ctx.setState({
            ...state,
            projectRunFilters: projects
        });
    }

    @Action(actionPreferences.SetTheme)
    setTheme(ctx: StateContext<PreferencesStateModel>, action: actionPreferences.SetTheme) {
        const state = ctx.getState();
        const mode = action.payload.theme;
        ctx.setState({
            ...state,
            themeMode: mode === THEME_NIGHT || mode === THEME_LIGHT ? mode : THEME_AUTO
        });
    }

    @Action(actionPreferences.SetSystemTheme)
    setSystemTheme(ctx: StateContext<PreferencesStateModel>, action: actionPreferences.SetSystemTheme) {
        const state = ctx.getState();
        ctx.setState({
            ...state,
            systemTheme: action.payload.theme === THEME_NIGHT ? THEME_NIGHT : THEME_LIGHT
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
