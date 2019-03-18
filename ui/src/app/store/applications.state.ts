import { HttpClient, HttpHeaders, HttpParams } from '@angular/common/http';
import { Action, createSelector, State, StateContext } from '@ngxs/store';
import { Application, Overview } from 'app/model/application.model';
import { IntegrationModel, ProjectIntegration } from 'app/model/integration.model';
import { Key } from 'app/model/keys.model';
import { tap } from 'rxjs/operators';
import * as ActionApplication from './applications.action';
import * as ActionProject from './project.action';
import { DeleteFromCacheWorkflow } from './workflows.action';

export class ApplicationsStateModel {
    public applications: { [key: string]: Application };
    public overviews: { [key: string]: Overview };
    public currentProjectKey: string;
    public loading: boolean;
}

export function getInitialApplicationsState(): ApplicationsStateModel {
    return {
        applications: {},
        overviews: {},
        currentProjectKey: null,
        loading: true,
    };
}

@State<ApplicationsStateModel>({
    name: 'applications',
    defaults: getInitialApplicationsState()
})
export class ApplicationsState {

    static selectApplication(projectKey: string, applicationName: string) {
        return createSelector(
            [ApplicationsState],
            (state: ApplicationsStateModel) => state.applications[projectKey + '/' + applicationName]
        );
    }

    static selectOverview(projectKey: string, applicationName: string) {
        return createSelector(
            [ApplicationsState],
            (state: ApplicationsStateModel) => state.overviews[projectKey + '/' + applicationName]
        );
    }

    constructor(private _http: HttpClient) { }

    @Action(ActionApplication.AddApplication)
    add(ctx: StateContext<ApplicationsStateModel>, action: ActionApplication.AddApplication) {
        const state = ctx.getState();
        let appKey = `${action.payload.projectKey}/${action.payload.application.name}`;
        let applications = state.applications;

        // Refresh when change project
        if (state.currentProjectKey !== action.payload.projectKey) {
            applications = {};
        }

        return this._http.post<Application>(
            `/project/${action.payload.projectKey}/applications`,
            action.payload.application
        ).pipe(tap((app) => {
            ctx.setState({
                ...state,
                currentProjectKey: action.payload.projectKey,
                applications: Object.assign({}, applications, { [appKey]: app }),
                loading: false,
            });
            ctx.dispatch(new ActionProject.AddApplicationInProject(app));
        }));

    }

    @Action(ActionApplication.CloneApplication)
    clone(ctx: StateContext<ApplicationsStateModel>, action: ActionApplication.CloneApplication) {
        const state = ctx.getState();
        let appKey = `${action.payload.projectKey}/${action.payload.newApplication.name}`;
        let applications = state.applications;

        // Refresh when change project
        if (state.currentProjectKey !== action.payload.projectKey) {
            applications = {};
        }

        return this._http.post<Application>(
            `/project/${action.payload.projectKey}/application/${action.payload.clonedAppName}/clone`,
            action.payload.newApplication
        ).pipe(tap((app) => {
            ctx.setState({
                ...state,
                currentProjectKey: action.payload.projectKey,
                applications: Object.assign({}, applications, { [appKey]: app }),
                loading: false,
            });
            ctx.dispatch(new ActionProject.AddApplicationInProject(app));
        }));
    }

    @Action(ActionApplication.LoadApplication)
    load(ctx: StateContext<ApplicationsStateModel>, action: ActionApplication.LoadApplication) {
        const state = ctx.getState();
        let appKey = `${action.payload.project_key}/${action.payload.name}`;
        let applications = state.applications;

        // Refresh when change project
        if (state.currentProjectKey !== action.payload.project_key) {
            applications = {};
        }

        ctx.setState({
            ...state,
            currentProjectKey: action.payload.project_key,
            applications: Object.assign({}, applications, { [appKey]: action.payload }),
            loading: false,
        });
    }

    @Action(ActionApplication.FetchApplication)
    fetch(ctx: StateContext<ApplicationsStateModel>, action: ActionApplication.FetchApplication) {
        const state = ctx.getState();
        const appKey = action.payload.projectKey + '/' + action.payload.applicationName;

        if (state.applications[appKey]) {
            return ctx.dispatch(new ActionApplication.LoadApplication(state.applications[appKey]));
        }

        return ctx.dispatch(new ActionApplication.ResyncApplication({ ...action.payload }));
    }

    @Action(ActionApplication.UpdateApplication)
    update(ctx: StateContext<ApplicationsStateModel>, action: ActionApplication.UpdateApplication) {
        return this._http.put<Application>(
            `/project/${action.payload.projectKey}/application/${action.payload.applicationName}`,
            action.payload.changes
        ).pipe(tap((app) => {
            if (app.vcs_strategy) {
                app.vcs_strategy.password = '**********';
            }
            const state = ctx.getState();

            let appKey = action.payload.projectKey + '/' + action.payload.applicationName;
            if (app.name !== action.payload.applicationName) {
                let applications = Object.assign({}, state.applications, {
                    [action.payload.projectKey + '/' + app.name]: app,
                });
                delete applications[appKey];

                ctx.setState({
                    ...state,
                    applications,
                });

                if (state.applications[appKey].usage && Array.isArray(state.applications[appKey].usage.workflows)) {
                    state.applications[appKey].usage.workflows.forEach((wf) => {
                        ctx.dispatch(new DeleteFromCacheWorkflow({
                            projectKey: action.payload.projectKey,
                            workflowName: wf.name
                        }));
                    });
                }

                ctx.dispatch(new ActionProject.UpdateApplicationInProject({
                    previousAppName: action.payload.applicationName,
                    changes: app
                }));
            } else {
                let applicationUpdated = {
                    ...state.applications[appKey],
                    ...app
                };

                ctx.setState({
                    ...state,
                    applications: Object.assign({}, state.applications, { [appKey]: applicationUpdated }),
                });
            }
        }));
    }

    @Action(ActionApplication.DeleteApplication)
    delete(ctx: StateContext<ApplicationsStateModel>, action: ActionApplication.DeleteApplication) {
        return this._http.delete(
            `/project/${action.payload.projectKey}/application/${action.payload.applicationName}`
        ).pipe(tap(() => {
            const state = ctx.getState();
            let appKey = action.payload.projectKey + '/' + action.payload.applicationName;
            let applications = Object.assign({}, state.applications);
            delete applications[appKey];

            ctx.setState({
                ...state,
                applications
            });

            ctx.dispatch(new ActionProject.DeleteApplicationInProject({ applicationName: action.payload.applicationName }));
        }));
    }

    @Action(ActionApplication.FetchApplicationOverview)
    fetchOverview(ctx: StateContext<ApplicationsStateModel>, action: ActionApplication.FetchApplicationOverview) {
        const state = ctx.getState();
        const appKey = action.payload.projectKey + '/' + action.payload.applicationName;

        return this._http.get<Overview>(
            `/ui/project/${action.payload.projectKey}/application/${action.payload.applicationName}/overview`
        ).pipe(tap((overview) => {
            ctx.setState({
                ...state,
                overviews: {
                    ...state.overviews,
                    [appKey]: overview
                }
            });
        }));
    }

    //  ------- Variables --------- //
    @Action(ActionApplication.AddApplicationVariable)
    addVariable(ctx: StateContext<ApplicationsStateModel>, action: ActionApplication.AddApplicationVariable) {
        let variable = action.payload.variable;
        let url = '/project/' + action.payload.projectKey + '/application/' + action.payload.applicationName + '/variable/' + variable.name;
        return this._http.post<Application>(url, variable)
            .pipe(tap((app) => {
                const state = ctx.getState();
                let appKey = app.project_key + '/' + app.name;
                let applicationUpdated = Object.assign({}, state.applications[appKey], { variables: app.variables });

                ctx.setState({
                    ...state,
                    applications: Object.assign({}, state.applications, { [appKey]: applicationUpdated }),
                });
            }));
    }

    @Action(ActionApplication.UpdateApplicationVariable)
    updateVariable(ctx: StateContext<ApplicationsStateModel>, action: ActionApplication.UpdateApplicationVariable) {
        let variable = action.payload.variable;
        let url = '/project/' + action.payload.projectKey +
            '/application/' + action.payload.applicationName +
            '/variable/' + action.payload.variableName;

        return this._http.put<Application>(url, variable)
            .pipe(tap((app) => {
                const state = ctx.getState();
                let appKey = app.project_key + '/' + app.name;
                let applicationUpdated = Object.assign({}, state.applications[appKey], { variables: app.variables });

                ctx.setState({
                    ...state,
                    applications: Object.assign({}, state.applications, { [appKey]: applicationUpdated }),
                });
            }));
    }

    @Action(ActionApplication.DeleteApplicationVariable)
    deleteVariable(ctx: StateContext<ApplicationsStateModel>, action: ActionApplication.DeleteApplicationVariable) {
        let variable = action.payload.variable;
        let url = `/project/${action.payload.projectKey}/application/${action.payload.applicationName}/variable/${variable.name}`;
        return this._http.delete<Application>(url)
            .pipe(tap((app) => {
                const state = ctx.getState();
                let appKey = action.payload.projectKey + '/' + action.payload.applicationName;
                let applicationUpdated = Object.assign({}, state.applications[appKey], { variables: app.variables });

                ctx.setState({
                    ...state,
                    applications: Object.assign({}, state.applications, { [appKey]: applicationUpdated }),
                });
            }));
    }

    //  ------- Keys --------- //
    @Action(ActionApplication.AddApplicationKey)
    addKey(ctx: StateContext<ApplicationsStateModel>, action: ActionApplication.AddApplicationKey) {
        let key = action.payload.key;
        let url = '/project/' + action.payload.projectKey + '/application/' + action.payload.applicationName + '/keys';
        return this._http.post<Key>(url, key)
            .pipe(tap((newKey) => {
                const state = ctx.getState();
                let appKey = action.payload.projectKey + '/' + action.payload.applicationName;
                let keys = state.applications[appKey].keys != null ? state.applications[appKey].keys.concat([newKey]) : [newKey];
                let applicationUpdated = Object.assign({}, state.applications[appKey], { keys });

                ctx.setState({
                    ...state,
                    applications: Object.assign({}, state.applications, { [appKey]: applicationUpdated }),
                });
            }));
    }

    @Action(ActionApplication.DeleteApplicationKey)
    deleteKey(ctx: StateContext<ApplicationsStateModel>, action: ActionApplication.DeleteApplicationKey) {
        let key = action.payload.key;
        let url = `/project/${action.payload.projectKey}/application/${action.payload.applicationName}/keys/${key.name}`;
        return this._http.delete(url)
            .pipe(tap(() => {
                const state = ctx.getState();
                let appKey = action.payload.projectKey + '/' + action.payload.applicationName;
                let keys = state.applications[appKey].keys.filter((currKey) => currKey.name !== key.name);
                let applicationUpdated = Object.assign({}, state.applications[appKey], { keys });

                ctx.setState({
                    ...state,
                    applications: Object.assign({}, state.applications, { [appKey]: applicationUpdated }),
                });
            }));
    }

    //  ------- Deployment strategies --------- //
    @Action(ActionApplication.AddApplicationDeployment)
    addDeployment(ctx: StateContext<ApplicationsStateModel>, action: ActionApplication.AddApplicationDeployment) {
        let integration = action.payload.integration;
        let url = '/project/' + action.payload.projectKey +
            '/application/' + action.payload.applicationName + '/deployment/config/' + integration.name;
        return this._http.post<Application>(url, integration.model.deployment_default_config)
            .pipe(tap((app) => {
                const state = ctx.getState();
                let appKey = action.payload.projectKey + '/' + action.payload.applicationName;
                let applicationUpdated = Object.assign({}, state.applications[appKey], {
                    deployment_strategies: app.deployment_strategies
                });

                ctx.setState({
                    ...state,
                    applications: Object.assign({}, state.applications, { [appKey]: applicationUpdated }),
                });
            }));
    }

    @Action(ActionApplication.UpdateApplicationDeployment)
    updateDeployment(ctx: StateContext<ApplicationsStateModel>, action: ActionApplication.UpdateApplicationDeployment) {
        let integration = new ProjectIntegration();
        integration.name = action.payload.deploymentName;
        integration.model = new IntegrationModel();
        integration.model.deployment_default_config = action.payload.config;

        return ctx.dispatch(new ActionApplication.AddApplicationDeployment({
            projectKey: action.payload.projectKey,
            applicationName: action.payload.applicationName,
            integration
        }));
    }

    @Action(ActionApplication.DeleteApplicationDeployment)
    deleteDeployment(ctx: StateContext<ApplicationsStateModel>, action: ActionApplication.DeleteApplicationDeployment) {
        let url = '/project/' + action.payload.projectKey +
            '/application/' + action.payload.applicationName + '/deployment/config/' + action.payload.integrationName;
        return this._http.delete<Application>(url)
            .pipe(tap((app) => {
                const state = ctx.getState();
                let appKey = action.payload.projectKey + '/' + action.payload.applicationName;
                let applicationUpdated = Object.assign({}, state.applications[appKey], {
                    deployment_strategies: app.deployment_strategies
                });

                ctx.setState({
                    ...state,
                    applications: Object.assign({}, state.applications, { [appKey]: applicationUpdated }),
                });
            }));
    }

    //  ------- VCS strategies --------- //
    @Action(ActionApplication.ConnectVcsRepoOnApplication)
    connectRepo(ctx: StateContext<ApplicationsStateModel>, action: ActionApplication.ConnectVcsRepoOnApplication) {
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
                let appKey = action.payload.projectKey + '/' + action.payload.applicationName;
                let applicationUpdated = Object.assign({}, state.applications[appKey], {
                    vcs_server: app.vcs_server,
                    repository_fullname: app.repository_fullname,
                    vcs_strategy: app.vcs_strategy
                });

                ctx.setState({
                    ...state,
                    applications: Object.assign({}, state.applications, { [appKey]: applicationUpdated }),
                });
            }));
    }

    @Action(ActionApplication.DeleteVcsRepoOnApplication)
    deleteRepo(ctx: StateContext<ApplicationsStateModel>, action: ActionApplication.DeleteVcsRepoOnApplication) {
        let repoManager = action.payload.repoManager;
        let url = '/project/' + action.payload.projectKey + '/repositories_manager/' +
            repoManager + '/application/' + action.payload.applicationName + '/detach';

        return this._http.post<Application>(url, null)
            .pipe(tap((app) => {
                const state = ctx.getState();
                let appKey = action.payload.projectKey + '/' + action.payload.applicationName;
                let applicationUpdated = Object.assign({}, state.applications[appKey], <Application>{
                    vcs_server: app.vcs_server,
                    repository_fullname: app.repository_fullname,
                    vcs_strategy: app.vcs_strategy
                });

                ctx.setState({
                    ...state,
                    applications: Object.assign({}, state.applications, { [appKey]: applicationUpdated }),
                });
            }));
    }

    //  ------- Misc --------- //
    @Action(ActionApplication.ExternalChangeApplication)
    externalChange(ctx: StateContext<ApplicationsStateModel>, action: ActionApplication.ExternalChangeApplication) {
        const state = ctx.getState();
        const appKey = action.payload.projectKey + '/' + action.payload.applicationName;
        const applicationUpdated = Object.assign({}, state.applications[appKey], { externalChange: true });

        ctx.setState({
            ...state,
            applications: Object.assign({}, state.applications, { [appKey]: applicationUpdated }),
        });
    }

    @Action(ActionApplication.DeleteFromCacheApplication)
    deleteFromCache(ctx: StateContext<ApplicationsStateModel>, action: ActionApplication.DeleteFromCacheApplication) {
        const state = ctx.getState();
        const appKey = action.payload.projectKey + '/' + action.payload.applicationName;
        let applications = Object.assign({}, state.applications);
        delete applications[appKey];

        ctx.setState({
            ...state,
            applications,
        });
    }

    @Action(ActionApplication.ResyncApplication)
    resync(ctx: StateContext<ApplicationsStateModel>, action: ActionApplication.ResyncApplication) {
        let params = new HttpParams();
        params = params.append('withNotifs', 'true');
        params = params.append('withUsage', 'true');
        params = params.append('withIcon', 'true');
        params = params.append('withKeys', 'true');
        params = params.append('withDeploymentStrategies', 'true');
        params = params.append('withVulnerabilities', 'true');

        return this._http.get<Application>(
            `/project/${action.payload.projectKey}/application/${action.payload.applicationName}`,
            { params }
        ).pipe(tap((app) => {
            if (app.vcs_strategy) {
                app.vcs_strategy.password = '**********';
            }
            ctx.dispatch(new ActionApplication.LoadApplication(app));
        }));
    }

    @Action(ActionApplication.ClearCacheApplication)
    clearCache(ctx: StateContext<ApplicationsStateModel>, _: ActionApplication.ClearCacheApplication) {
        ctx.setState(getInitialApplicationsState());
    }
}
