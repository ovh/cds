import { HttpClient, HttpHeaders, HttpParams } from '@angular/common/http';
import { Action, createSelector, State, StateContext } from '@ngxs/store';
import { Application, Overview } from 'app/model/application.model';
import { IntegrationModel, ProjectIntegration } from 'app/model/integration.model';
import { Key } from 'app/model/keys.model';
import { tap } from 'rxjs/operators';
import {
    AddApplicationDeployment,
    AddApplicationKey,
    AddApplicationVariable,
    ConnectVcsRepoOnApplication,
    DeleteApplication,
    DeleteApplicationDeployment,
    DeleteApplicationKey,
    DeleteApplicationVariable,
    DeleteVcsRepoOnApplication,
    FetchApplication,
    FetchApplicationOverview,
    LoadApplication,
    UpdateApplication,
    UpdateApplicationDeployment,
    UpdateApplicationVariable
} from './applications.action';

export class ApplicationsStateModel {
    public applications: { [key: string]: Application };
    public currentProjectKey: string;
    public loading: boolean;
}

export function getInitialApplicationsState(): ApplicationsStateModel {
    return {
        applications: {},
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

    constructor(private _http: HttpClient) { }

    @Action(LoadApplication)
    load(ctx: StateContext<ApplicationsStateModel>, action: LoadApplication) {
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

    @Action(FetchApplication)
    fetch(ctx: StateContext<ApplicationsStateModel>, action: FetchApplication) {
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
            app.vcs_strategy.password = '**********';
            ctx.dispatch(new LoadApplication(app));
        }));
    }

    @Action(UpdateApplication)
    update(ctx: StateContext<ApplicationsStateModel>, action: UpdateApplication) {
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
                ctx.setState({
                    ...state,
                    applications: Object.assign({}, state.applications, {
                        [appKey]: null,
                        [action.payload.projectKey + '/' + app.name]: app,
                    }),
                });
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

    @Action(DeleteApplication)
    delete(ctx: StateContext<ApplicationsStateModel>, action: DeleteApplication) {
        return this._http.delete(
            `/project/${action.payload.projectKey}/application/${action.payload.applicationName}`
        ).pipe(tap(() => {
            const state = ctx.getState();
            let appKey = action.payload.projectKey + '/' + action.payload.applicationName;
            ctx.setState({
                ...state,
                applications: Object.assign({}, state.applications, { [appKey]: null }),
            });

            // todo dispatch action on project state to delete from application_names
        }));
    }

    @Action(FetchApplicationOverview)
    fetchOverview(ctx: StateContext<ApplicationsStateModel>, action: FetchApplicationOverview) {
        return this._http.get<Overview>(
            `/ui/project/${action.payload.projectKey}/application/${action.payload.applicationName}/overview`
        ).pipe(tap((overview) => {
            const state = ctx.getState();
            let appKey = action.payload.projectKey + '/' + action.payload.applicationName;
            let applicationUpdated = Object.assign({}, state.applications[appKey], { overview });

            ctx.setState({
                ...state,
                applications: Object.assign({}, state.applications, { [appKey]: applicationUpdated }),
            });
        }));
    }

    //  ------- Variables --------- //
    @Action(AddApplicationVariable)
    addVariable(ctx: StateContext<ApplicationsStateModel>, action: AddApplicationVariable) {
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

    @Action(UpdateApplicationVariable)
    updateVariable(ctx: StateContext<ApplicationsStateModel>, action: UpdateApplicationVariable) {
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

    @Action(DeleteApplicationVariable)
    deleteVariable(ctx: StateContext<ApplicationsStateModel>, action: DeleteApplicationVariable) {
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
    @Action(AddApplicationKey)
    addKey(ctx: StateContext<ApplicationsStateModel>, action: AddApplicationKey) {
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

    @Action(DeleteApplicationKey)
    deleteKey(ctx: StateContext<ApplicationsStateModel>, action: DeleteApplicationKey) {
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
    @Action(AddApplicationDeployment)
    addDeployment(ctx: StateContext<ApplicationsStateModel>, action: AddApplicationDeployment) {
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

    @Action(UpdateApplicationDeployment)
    updateDeployment(ctx: StateContext<ApplicationsStateModel>, action: UpdateApplicationDeployment) {
        let integration = new ProjectIntegration();
        integration.name = action.payload.deploymentName;
        integration.model = new IntegrationModel();
        integration.model.deployment_default_config = action.payload.config;

        return ctx.dispatch(new AddApplicationDeployment({
            projectKey: action.payload.projectKey,
            applicationName: action.payload.applicationName,
            integration
        }));
    }

    @Action(DeleteApplicationDeployment)
    deleteDeployment(ctx: StateContext<ApplicationsStateModel>, action: DeleteApplicationDeployment) {
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
    @Action(ConnectVcsRepoOnApplication)
    connectRepo(ctx: StateContext<ApplicationsStateModel>, action: ConnectVcsRepoOnApplication) {
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
                    repository_fullname: app.repository_fullname
                });

                ctx.setState({
                    ...state,
                    applications: Object.assign({}, state.applications, { [appKey]: applicationUpdated }),
                });
            }));
    }

    @Action(DeleteVcsRepoOnApplication)
    deleteRepo(ctx: StateContext<ApplicationsStateModel>, action: DeleteVcsRepoOnApplication) {
        let repoManager = action.payload.repoManager;
        let url = '/project/' + action.payload.projectKey + '/repositories_manager/' +
            repoManager + '/application/' + action.payload.applicationName + '/detach';

        return this._http.post<Application>(url, null)
            .pipe(tap((app) => {
                const state = ctx.getState();
                let appKey = action.payload.projectKey + '/' + action.payload.applicationName;
                let applicationUpdated = Object.assign({}, state.applications[appKey], {
                    vcs_server: app.vcs_server,
                    repository_fullname: app.repository_fullname
                });

                ctx.setState({
                    ...state,
                    applications: Object.assign({}, state.applications, { [appKey]: applicationUpdated }),
                });
            }));
    }
}
