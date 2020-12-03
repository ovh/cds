import { HttpClient, HttpHeaders, HttpParams } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Action, createSelector, State, StateContext } from '@ngxs/store';
import { Application, Overview } from 'app/model/application.model';
import { IntegrationModel, ProjectIntegration } from 'app/model/integration.model';
import { Key } from 'app/model/keys.model';
import { Variable } from 'app/model/variable.model';
import { ApplicationService } from 'app/service/application/application.service';
import { cloneDeep } from 'lodash-es';
import { tap } from 'rxjs/operators';
import * as ActionApplication from './applications.action';
import { ClearCacheApplication } from './applications.action';
import * as ActionProject from './project.action';

export class ApplicationStateModel {
    public application: Application;
    public editApplication: Application;
    public overview: Overview;
    public currentProjectKey: string;
    public loading: boolean;
    public editMode: boolean;
}

export function getInitialApplicationsState(): ApplicationStateModel {
    return {
        application: null,
        editApplication: null,
        overview: null,
        currentProjectKey: null,
        loading: true,
        editMode: false
    };
}

@State<ApplicationStateModel>({
    name: 'application',
    defaults: getInitialApplicationsState()
})
@Injectable()
export class ApplicationsState {

    constructor(private _http: HttpClient, private _appService: ApplicationService) { }

    static currentState() {
        return createSelector(
            [ApplicationsState],
            (state: ApplicationStateModel) => state
        );
    }

    static selectOverview() {
        return createSelector(
            [ApplicationsState],
            (state: ApplicationStateModel) => state.overview
        );
    }

    @Action(ActionApplication.AddApplication)
    add(ctx: StateContext<ApplicationStateModel>, action: ActionApplication.AddApplication) {
        return this._http.post<Application>(
            `/project/${action.payload.projectKey}/applications`,
            action.payload.application
        ).pipe(tap((app) => {
            const state = ctx.getState();
            ctx.setState({
                ...state,
                currentProjectKey: action.payload.projectKey,
                editMode: false,
                editApplication: null,
                application: app,
                loading: false
            });
            ctx.dispatch(new ActionProject.AddApplicationInProject(app));
        }));

    }

    @Action(ActionApplication.CloneApplication)
    clone(ctx: StateContext<ApplicationStateModel>, action: ActionApplication.CloneApplication) {
        return this._http.post<Application>(
            `/project/${action.payload.projectKey}/application/${action.payload.clonedAppName}/clone`,
            action.payload.newApplication
        ).pipe(tap((app) => {
            const state = ctx.getState();
            ctx.setState({
                ...state,
                currentProjectKey: action.payload.projectKey,
                application: app,
                loading: false,
            });
            ctx.dispatch(new ActionProject.AddApplicationInProject(app));
        }));
    }

    @Action(ActionApplication.LoadApplication)
    load(ctx: StateContext<ApplicationStateModel>, action: ActionApplication.LoadApplication) {
        const state = ctx.getState();

        let editMode = false;
        if (action.payload.from_repository) {
            editMode = true;
        }
        ctx.setState({
            ...state,
            currentProjectKey: action.payload.project_key,
            application: action.payload,
            editApplication: cloneDeep(action.payload),
            editMode,
            loading: false,
        });
    }

    @Action(ActionApplication.FetchApplication)
    fetch(ctx: StateContext<ApplicationStateModel>, action: ActionApplication.FetchApplication) {
        const state = ctx.getState();
        if (state.application && state.application.name === action.payload.applicationName
            && state.currentProjectKey === action.payload.projectKey) {
            return ctx.dispatch(new ActionApplication.LoadApplication(state.application));
        }
        return ctx.dispatch(new ActionApplication.ResyncApplication({ ...action.payload }));
    }

    @Action(ActionApplication.UpdateApplication)
    update(ctx: StateContext<ApplicationStateModel>, action: ActionApplication.UpdateApplication) {
        const stateEditMode = ctx.getState();
        if (stateEditMode.editMode) {
            let appToUpdate = cloneDeep(stateEditMode.editApplication);
            appToUpdate.name = action.payload.changes.name;
            appToUpdate.description = action.payload.changes.description;
            appToUpdate.deployment_strategies = action.payload.changes.deployment_strategies;
            appToUpdate.vcs_strategy = action.payload.changes.vcs_strategy;
            appToUpdate.editModeChanged = true;
            return ctx.setState({
                ...stateEditMode,
                editApplication: appToUpdate,
            });
            return;
        }


        return this._http.put<Application>(
            `/project/${action.payload.projectKey}/application/${action.payload.applicationName}`,
            action.payload.changes
        ).pipe(tap((app) => {
            if (app.vcs_strategy) {
                app.vcs_strategy.password = '**********';
            }
            const state = ctx.getState();

            if (app.name !== action.payload.applicationName) {
                let application = app;
                ctx.setState({
                    ...state,
                    application,
                });

                ctx.dispatch(new ActionProject.UpdateApplicationInProject({
                    previousAppName: action.payload.applicationName,
                    changes: app
                }));
            } else {
                let applicationUpdated = {
                    ...state.application,
                    ...app
                };

                ctx.setState({
                    ...state,
                    application: Object.assign({}, state.application, applicationUpdated),
                });
            }
        }));
    }

    @Action(ActionApplication.DeleteApplication)
    delete(ctx: StateContext<ApplicationStateModel>, action: ActionApplication.DeleteApplication) {
        return this._http.delete(
            `/project/${action.payload.projectKey}/application/${action.payload.applicationName}`
        ).pipe(tap(() => {
            ctx.dispatch(new ClearCacheApplication());
            ctx.dispatch(new ActionProject.DeleteApplicationInProject({ applicationName: action.payload.applicationName }));
        }));
    }

    @Action(ActionApplication.FetchApplicationOverview)
    fetchOverview(ctx: StateContext<ApplicationStateModel>, action: ActionApplication.FetchApplicationOverview) {
        const state = ctx.getState();

        return this._http.get<Overview>(
            `/ui/project/${action.payload.projectKey}/application/${action.payload.applicationName}/overview`
        ).pipe(tap((overview) => {
            ctx.setState({
                ...state,
                overview
            });
        }));
    }

    //  ------- Variables --------- //
    @Action(ActionApplication.AddApplicationVariable)
    addVariable(ctx: StateContext<ApplicationStateModel>, action: ActionApplication.AddApplicationVariable) {
        let variable = action.payload.variable;

        const stateEditMode = ctx.getState();
        if (stateEditMode.editMode) {
            let appToupdate = cloneDeep(stateEditMode.editApplication);
            if (!appToupdate.variables) {
                appToupdate.variables = new Array<Variable>();
            }
            action.payload.variable.hasChanged = false;
            action.payload.variable.updating = false;
            appToupdate.variables.push(cloneDeep(action.payload.variable));
            appToupdate.editModeChanged = true;
            return ctx.setState({
                ...stateEditMode,
                editApplication: appToupdate,
            });
        }


        let url = '/project/' + action.payload.projectKey + '/application/' + action.payload.applicationName + '/variable/' + variable.name;
        return this._http.post<Variable>(url, variable)
            .pipe(tap((v) => {
                const state = ctx.getState();
                let applicationUpdated = cloneDeep(state.application);
                if (!applicationUpdated.variables) {
                    applicationUpdated.variables = new Array<Variable>();
                }
                applicationUpdated.variables.push(v);
                ctx.setState({
                    ...state,
                    application: Object.assign({}, state.application, applicationUpdated),
                });
            }));
    }

    @Action(ActionApplication.UpdateApplicationVariable)
    updateVariable(ctx: StateContext<ApplicationStateModel>, action: ActionApplication.UpdateApplicationVariable) {
        let variable = action.payload.variable;

        const stateEditMode = ctx.getState();
        if (stateEditMode.editMode) {
            let appToupdate = cloneDeep(stateEditMode.editApplication);
            let varIndex = appToupdate.variables.findIndex(app => action.payload.variableName === app.name);
            action.payload.variable.hasChanged = false;
            action.payload.variable.updating = false;
            appToupdate.variables[varIndex] = action.payload.variable;
            appToupdate.editModeChanged = true;
            ctx.setState({
                ...stateEditMode,
                editApplication: appToupdate,
            });
            return
        }


        let url = '/project/' + action.payload.projectKey +
            '/application/' + action.payload.applicationName +
            '/variable/' + action.payload.variableName;

        return this._http.put<Variable>(url, variable)
            .pipe(tap((updatedVar) => {
                const state = ctx.getState();
                let applicationUpdated = cloneDeep(state.application);
                applicationUpdated.variables = applicationUpdated.variables.map(v => {
                   if (v.name !== action.payload.variableName) {
                       return v;
                   }
                   return updatedVar;
                });
                ctx.setState({
                    ...state,
                    application: Object.assign({}, state.application, applicationUpdated),
                });
            }));
    }

    @Action(ActionApplication.DeleteApplicationVariable)
    deleteVariable(ctx: StateContext<ApplicationStateModel>, action: ActionApplication.DeleteApplicationVariable) {
        let variable = action.payload.variable;

        const stateEditMode = ctx.getState();
        if (stateEditMode.editMode) {
            let appToupdate = cloneDeep(stateEditMode.editApplication);
            action.payload.variable.hasChanged = false;
            action.payload.variable.updating = false;
            appToupdate.variables = appToupdate.variables.filter(app => app.name !== action.payload.variable.name);
            appToupdate.editModeChanged = true;
            ctx.setState({
                ...stateEditMode,
                editApplication: appToupdate,
            });
            return
        }
        let url = `/project/${action.payload.projectKey}/application/${action.payload.applicationName}/variable/${variable.name}`;
        return this._http.delete<any>(url)
            .pipe(tap(() => {
                const state = ctx.getState();
                let applicationUpdated = state.application;
                applicationUpdated.variables = applicationUpdated.variables.filter(v => v.name !== action.payload.variable.name);

                ctx.setState({
                    ...state,
                    application: Object.assign({}, state.application, applicationUpdated),
                });
            }));
    }

    //  ------- Keys --------- //
    @Action(ActionApplication.AddApplicationKey)
    addKey(ctx: StateContext<ApplicationStateModel>, action: ActionApplication.AddApplicationKey) {
        let key = action.payload.key;

        const stateEditMode = ctx.getState();
        if (stateEditMode.editMode) {
            let appToupdate = cloneDeep(stateEditMode.editApplication);
            if (!appToupdate.keys) {
                appToupdate.keys = new Array<Key>();
            }
            appToupdate.keys.push(action.payload.key);
            appToupdate.editModeChanged = true;
            ctx.setState({
                ...stateEditMode,
                editApplication: appToupdate,
            });
            return
        }

        let url = '/project/' + action.payload.projectKey + '/application/' + action.payload.applicationName + '/keys';
        return this._http.post<Key>(url, key)
            .pipe(tap((newKey) => {
                const state = ctx.getState();
                let keys = state.application.keys != null ? state.application.keys.concat([newKey]) : [newKey];
                let applicationUpdated = Object.assign({}, state.application, { keys });

                ctx.setState({
                    ...state,
                    application: Object.assign({}, state.application, applicationUpdated),
                });
            }));
    }

    @Action(ActionApplication.DeleteApplicationKey)
    deleteKey(ctx: StateContext<ApplicationStateModel>, action: ActionApplication.DeleteApplicationKey) {
        let key = action.payload.key;

        const stateEditMode = ctx.getState();
        if (stateEditMode.editMode) {
            let appToupdate = cloneDeep(stateEditMode.editApplication);
            appToupdate.keys = appToupdate.keys.filter(k => action.payload.key.name !== k.name);
            appToupdate.editModeChanged = true;
            ctx.setState({
                ...stateEditMode,
                editApplication: appToupdate,
            });
            return
        }

        let url = `/project/${action.payload.projectKey}/application/${action.payload.applicationName}/keys/${key.name}`;
        return this._http.delete(url)
            .pipe(tap(() => {
                const state = ctx.getState();
                let keys = state.application.keys.filter((currKey) => currKey.name !== key.name);
                let applicationUpdated = Object.assign({}, state.application, { keys });

                ctx.setState({
                    ...state,
                    application: Object.assign({}, state.application, applicationUpdated),
                });
            }));
    }

    //  ------- Deployment strategies --------- //
    @Action(ActionApplication.AddApplicationDeployment)
    addDeployment(ctx: StateContext<ApplicationStateModel>, action: ActionApplication.AddApplicationDeployment) {
        let integration = action.payload.integration;

        const stateEditMode = ctx.getState();
        if (stateEditMode.editMode) {
            let appToupdate = cloneDeep(stateEditMode.editApplication);
            if (!appToupdate.deployment_strategies) {
                appToupdate.deployment_strategies = {};
            }
            appToupdate.deployment_strategies[integration.name] = cloneDeep(integration.model.deployment_default_config);
            appToupdate.editModeChanged = true;
            ctx.setState({
                ...stateEditMode,
                editApplication: appToupdate,
            });
            return
        }

        let url = '/project/' + action.payload.projectKey +
            '/application/' + action.payload.applicationName + '/deployment/config/' + integration.name;
        return this._http.post<Application>(url, integration.model.deployment_default_config)
            .pipe(tap((app) => {
                const state = ctx.getState();
                let applicationUpdated = Object.assign({}, state.application, {
                    deployment_strategies: app.deployment_strategies
                });

                ctx.setState({
                    ...state,
                    application: Object.assign({}, state.application, applicationUpdated),
                });
            }));
    }

    @Action(ActionApplication.UpdateApplicationDeployment)
    updateDeployment(ctx: StateContext<ApplicationStateModel>, action: ActionApplication.UpdateApplicationDeployment) {
        let integration = new ProjectIntegration();
        integration.name = action.payload.deploymentName;
        integration.model = new IntegrationModel();
        integration.model.deployment_default_config = action.payload.config;

        const stateEditMode = ctx.getState();
        if (stateEditMode.editMode) {
            let appToupdate = cloneDeep(stateEditMode.editApplication);
            appToupdate.deployment_strategies[integration.name] = integration.model.deployment_default_config;
            appToupdate.editModeChanged = true;
            ctx.setState({
                ...stateEditMode,
                editApplication: appToupdate,
            });
            return
        }

        return ctx.dispatch(new ActionApplication.AddApplicationDeployment({
            projectKey: action.payload.projectKey,
            applicationName: action.payload.applicationName,
            integration
        }));
    }

    @Action(ActionApplication.DeleteApplicationDeployment)
    deleteDeployment(ctx: StateContext<ApplicationStateModel>, action: ActionApplication.DeleteApplicationDeployment) {

        const stateEditMode = ctx.getState();
        if (stateEditMode.editMode) {
            let appToupdate = cloneDeep(stateEditMode.editApplication);
            delete appToupdate.deployment_strategies[action.payload.integrationName];
            appToupdate.editModeChanged = true;
            ctx.setState({
                ...stateEditMode,
                editApplication: appToupdate,
            });
            return
        }

        let url = '/project/' + action.payload.projectKey +
            '/application/' + action.payload.applicationName + '/deployment/config/' + action.payload.integrationName;
        return this._http.delete<Application>(url)
            .pipe(tap((app) => {
                const state = ctx.getState();
                let applicationUpdated = Object.assign({}, state.application, {
                    deployment_strategies: app.deployment_strategies
                });

                ctx.setState({
                    ...state,
                    application: Object.assign({}, state.application, applicationUpdated),
                });
            }));
    }

    //  ------- VCS strategies --------- //
    @Action(ActionApplication.ConnectVcsRepoOnApplication)
    connectRepo(ctx: StateContext<ApplicationStateModel>, action: ActionApplication.ConnectVcsRepoOnApplication) {
        let repoManager = action.payload.repoManager;
        let repoFullname = action.payload.repoFullName;
        let url = '/project/' + action.payload.projectKey + '/repositories_manager/' +
            repoManager + '/application/' + action.payload.applicationName + '/attach';
        let headers = new HttpHeaders();
        headers.append('Content-Type', 'application/x-www-form-urlencoded');

        let params = new HttpParams();
        params = params.append('fullname', repoFullname);

        return this._http.post<Application>(url, params.toString(), { headers, params })
            .pipe(tap((app) => {
                const state = ctx.getState();
                let applicationUpdated = Object.assign({}, state.application, {
                    vcs_server: app.vcs_server,
                    repository_fullname: app.repository_fullname,
                    vcs_strategy: app.vcs_strategy
                });

                ctx.setState({
                    ...state,
                    application: Object.assign({}, state.application, applicationUpdated),
                });
            }));
    }

    @Action(ActionApplication.DeleteVcsRepoOnApplication)
    deleteRepo(ctx: StateContext<ApplicationStateModel>, action: ActionApplication.DeleteVcsRepoOnApplication) {
        let repoManager = action.payload.repoManager;
        let url = '/project/' + action.payload.projectKey + '/repositories_manager/' +
            repoManager + '/application/' + action.payload.applicationName + '/detach';

        return this._http.post<Application>(url, null)
            .pipe(tap((app) => {
                const state = ctx.getState();
                let applicationUpdated = Object.assign({}, state.application, <Application>{
                    vcs_server: app.vcs_server,
                    repository_fullname: app.repository_fullname,
                    vcs_strategy: app.vcs_strategy
                });

                ctx.setState({
                    ...state,
                    application: Object.assign({}, state.application, applicationUpdated),
                });
            }));
    }

    //  ------- Misc --------- //
    @Action(ActionApplication.ExternalChangeApplication)
    externalChange(ctx: StateContext<ApplicationStateModel>, action: ActionApplication.ExternalChangeApplication) {
        const state = ctx.getState();
        const applicationUpdated = Object.assign({}, state.application, { externalChange: true });

        ctx.setState({
            ...state,
            application: Object.assign({}, state.application, applicationUpdated),
        });
    }

    @Action(ActionApplication.ResyncApplication)
    resync(ctx: StateContext<ApplicationStateModel>, action: ActionApplication.ResyncApplication) {
        return this._appService.getApplication(action.payload.projectKey, action.payload.applicationName)
            .pipe(tap((app) => {
            if (app.vcs_strategy) {
                app.vcs_strategy.password = '**********';
            }
            ctx.dispatch(new ActionApplication.LoadApplication(app));
        }));
    }

    @Action(ActionApplication.ClearCacheApplication)
    clearCache(ctx: StateContext<ApplicationStateModel>, _: ActionApplication.ClearCacheApplication) {
        ctx.setState(getInitialApplicationsState());
    }

    @Action(ActionApplication.CancelApplicationEdition)
    cancelApplicationEdition(ctx: StateContext<ApplicationStateModel>, _: ActionApplication.CancelApplicationEdition) {
        const state = ctx.getState();
        let editMode = state.editMode;
        if (state.application.from_repository) {
            editMode = true;
        }
        let editApplication = cloneDeep(state.application);
        ctx.setState({
            ...state,
            editApplication,
            editMode,
        });
    }
}
