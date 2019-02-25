import { HttpClient, HttpHeaders, HttpParams } from '@angular/common/http';
import { Action, createSelector, State, StateContext } from '@ngxs/store';
import { IntegrationModel, ProjectIntegration } from 'app/model/integration.model';
import { Key } from 'app/model/keys.model';
import { Pipeline } from 'app/model/pipeline.model';
import { tap } from 'rxjs/operators';
import * as actionPipeline from './pipelines.action';
import * as ActionProject from './project.action';

export class PipelinesStateModel {
    public pipelines: { [key: string]: Pipeline };
    public currentProjectKey: string;
    public loading: boolean;
}

export function getInitialPipelinesState(): PipelinesStateModel {
    return {
        pipelines: {},
        currentProjectKey: null,
        loading: true,
    };
}

@State<PipelinesStateModel>({
    name: 'pipelines',
    defaults: getInitialPipelinesState()
})
export class PipelinesState {

    static selectPipeline(projectKey: string, pipelineName: string) {
        return createSelector(
            [PipelinesState],
            (state: PipelinesStateModel) => state.pipelines[projectKey + '/' + pipelineName]
        );
    }

    constructor(private _http: HttpClient) { }

    @Action(actionPipeline.AddPipeline)
    add(ctx: StateContext<PipelinesStateModel>, action: actionPipeline.AddPipeline) {
        const state = ctx.getState();
        let pipKey = `${action.payload.projectKey}/${action.payload.pipeline.name}`;
        let pipelines = state.pipelines;

        // Refresh when change project
        if (state.currentProjectKey !== action.payload.projectKey) {
            pipelines = {};
        }

        return this._http.post<Pipeline>(
            `/project/${action.payload.projectKey}/pipelines`,
            action.payload.pipeline
        ).pipe(tap((pip) => {
            ctx.setState({
                ...state,
                currentProjectKey: action.payload.projectKey,
                pipelines: Object.assign({}, pipelines, { [pipKey]: pip }),
                loading: false,
            });
            ctx.dispatch(new ActionProject.AddPipelineInProject(pip));
        }));

    }

    @Action(actionPipeline.LoadPipeline)
    load(ctx: StateContext<PipelinesStateModel>, action: actionPipeline.LoadPipeline) {
        const state = ctx.getState();
        let pipKey = `${action.payload.projectKey}/${action.payload.pipeline.name}`;
        let pipelines = state.pipelines;

        // Refresh when change project
        if (state.currentProjectKey !== action.payload.projectKey) {
            pipelines = {};
        }

        ctx.setState({
            ...state,
            currentProjectKey: action.payload.projectKey,
            pipelines: Object.assign({}, pipelines, { [pipKey]: action.payload }),
            loading: false,
        });
    }

    @Action(actionPipeline.FetchPipeline)
    fetch(ctx: StateContext<PipelinesStateModel>, action: actionPipeline.FetchPipeline) {
        const state = ctx.getState();
        const pipKey = action.payload.projectKey + '/' + action.payload.pipelineName;

        if (state.pipelines[pipKey]) {
            return ctx.dispatch(new actionPipeline.LoadPipeline({
                projectKey: action.payload.projectKey,
                pipeline: state.pipelines[pipKey]
            }));
        }

        return ctx.dispatch(new actionPipeline.ResyncPipeline({ ...action.payload }));
    }

    @Action(actionPipeline.UpdatePipeline)
    update(ctx: StateContext<PipelinesStateModel>, action: actionPipeline.UpdatePipeline) {
        return this._http.put<Pipeline>(
            `/project/${action.payload.projectKey}/application/${action.payload.pipelineName}`,
            action.payload.changes
        ).pipe(tap((pip) => {
            const state = ctx.getState();

            let pipKey = action.payload.projectKey + '/' + action.payload.pipelineName;
            if (pip.name !== action.payload.pipelineName) {
                let pipelines = Object.assign({}, state.pipelines, {
                    [action.payload.projectKey + '/' + pip.name]: pip,
                });
                delete pipelines[pipKey];

                ctx.setState({
                    ...state,
                    pipelines,
                });
                ctx.dispatch(new ActionProject.UpdatePipelineInProject({
                    previousPipName: action.payload.pipelineName,
                    changes: pip
                }));
            } else {
                let pipUpdated = {
                    ...state.pipelines[pipKey],
                    ...pip
                };

                ctx.setState({
                    ...state,
                    pipelines: Object.assign({}, state.pipelines, { [pipKey]: pipUpdated }),
                });
            }
        }));
    }

    @Action(actionPipeline.DeletePipeline)
    delete(ctx: StateContext<PipelinesStateModel>, action: actionPipeline.DeletePipeline) {
        return this._http.delete(
            `/project/${action.payload.projectKey}/application/${action.payload.pipelineName}`
        ).pipe(tap(() => {
            const state = ctx.getState();
            let pipKey = action.payload.projectKey + '/' + action.payload.pipelineName;
            let pipelines = Object.assign({}, state.pipelines);
            delete pipelines[pipKey];

            ctx.setState({
                ...state,
                pipelines
            });

            ctx.dispatch(new ActionProject.DeletePipelineInProject({ pipelineName: action.payload.pipelineName }));
        }));
    }

    //  ------- Parameter --------- //
    @Action(actionPipeline.AddPipelineParameter)
    addParameter(ctx: StateContext<PipelinesStateModel>, action: actionPipeline.AddPipelineParameter) {
        let parameter = action.payload.parameter;
        let url = '/project/' + action.payload.projectKey + '/pipeline/' + action.payload.pipelineName + '/parameter/' + parameter.name;
        return this._http.post<Pipeline>(url, parameter)
            .pipe(tap((pip) => {
                const state = ctx.getState();
                let pipKey = pip.projectKey + '/' + pip.name;
                let pipUpdated = Object.assign({}, state.pipelines[pipKey], { parameters: pip.parameters });

                ctx.setState({
                    ...state,
                    pipelines: Object.assign({}, state.pipelines, { [pipKey]: pipUpdated }),
                });
            }));
    }

    @Action(actionPipeline.UpdatePipelineParameter)
    updateParameter(ctx: StateContext<PipelinesStateModel>, action: actionPipeline.UpdatePipelineParameter) {
        let parameter = action.payload.parameter;
        let url = '/project/' + action.payload.projectKey +
            '/pipeline/' + action.payload.pipelineName +
            '/parameter/' + action.payload.parameterName;

        return this._http.put<Pipeline>(url, parameter)
            .pipe(tap((pip) => {
                const state = ctx.getState();
                let pipKey = pip.projectKey + '/' + pip.name;
                let pipUpdated = Object.assign({}, state.pipelines[pipKey], { parameters: pip.parameters });

                ctx.setState({
                    ...state,
                    pipelines: Object.assign({}, state.pipelines, { [pipKey]: pipUpdated }),
                });
            }));
    }

    @Action(actionPipeline.DeletePipelineParameter)
    deleteParameter(ctx: StateContext<PipelinesStateModel>, action: actionPipeline.DeletePipelineParameter) {
        let parameter = action.payload.parameter;
        let url = `/project/${action.payload.projectKey}/application/${action.payload.pipelineName}/parameter/${parameter.name}`;
        return this._http.delete<Pipeline>(url)
            .pipe(tap((pip) => {
                const state = ctx.getState();
                let pipKey = action.payload.projectKey + '/' + action.payload.pipelineName;
                let pipUpdated = Object.assign({}, state.pipelines[pipKey], { parameters: pip.parameters });

                ctx.setState({
                    ...state,
                    pipelines: Object.assign({}, state.pipelines, { [pipKey]: pipUpdated }),
                });
            }));
    }

    //  ------- Keys --------- //
    @Action(actionPipeline.AddPipelineKey)
    addKey(ctx: StateContext<PipelinesStateModel>, action: actionPipeline.AddPipelineKey) {
        let key = action.payload.key;
        let url = '/project/' + action.payload.projectKey + '/application/' + action.payload.pipelineName + '/keys';
        return this._http.post<Key>(url, key)
            .pipe(tap((newKey) => {
                const state = ctx.getState();
                let pipKey = action.payload.projectKey + '/' + action.payload.pipelineName;
                let keys = state.pipelines[pipKey].keys != null ? state.pipelines[pipKey].keys.concat([newKey]) : [newKey];
                let pipUpdated = Object.assign({}, state.pipelines[pipKey], { keys });

                ctx.setState({
                    ...state,
                    pipelines: Object.assign({}, state.pipelines, { [pipKey]: pipUpdated }),
                });
            }));
    }

    @Action(actionPipeline.DeletePipelineKey)
    deleteKey(ctx: StateContext<PipelinesStateModel>, action: actionPipeline.DeletePipelineKey) {
        let key = action.payload.key;
        let url = `/project/${action.payload.projectKey}/application/${action.payload.pipelineName}/keys/${key.name}`;
        return this._http.delete(url)
            .pipe(tap(() => {
                const state = ctx.getState();
                let pipKey = action.payload.projectKey + '/' + action.payload.pipelineName;
                let keys = state.pipelines[pipKey].keys.filter((currKey) => currKey.name !== key.name);
                let pipUpdated = Object.assign({}, state.pipelines[pipKey], { keys });

                ctx.setState({
                    ...state,
                    pipelines: Object.assign({}, state.pipelines, { [pipKey]: pipUpdated }),
                });
            }));
    }

    //  ------- Deployment strategies --------- //
    @Action(actionPipeline.AddPipelineDeployment)
    addDeployment(ctx: StateContext<PipelinesStateModel>, action: actionPipeline.AddPipelineDeployment) {
        let integration = action.payload.integration;
        let url = '/project/' + action.payload.projectKey +
            '/application/' + action.payload.pipelineName + '/deployment/config/' + integration.name;
        return this._http.post<Pipeline>(url, integration.model.deployment_default_config)
            .pipe(tap((pip) => {
                const state = ctx.getState();
                let pipKey = action.payload.projectKey + '/' + action.payload.pipelineName;
                let pipUpdated = Object.assign({}, state.pipelines[pipKey], {
                    deployment_strategies: pip.deployment_strategies
                });

                ctx.setState({
                    ...state,
                    pipelines: Object.assign({}, state.pipelines, { [pipKey]: pipUpdated }),
                });
            }));
    }

    @Action(actionPipeline.UpdatePipelineDeployment)
    updateDeployment(ctx: StateContext<PipelinesStateModel>, action: actionPipeline.UpdatePipelineDeployment) {
        let integration = new ProjectIntegration();
        integration.name = action.payload.deploymentName;
        integration.model = new IntegrationModel();
        integration.model.deployment_default_config = action.payload.config;

        return ctx.dispatch(new actionPipeline.AddPipelineDeployment({
            projectKey: action.payload.projectKey,
            applicationName: action.payload.pipelineName,
            integration
        }));
    }

    @Action(actionPipeline.DeletePipelineDeployment)
    deleteDeployment(ctx: StateContext<PipelinesStateModel>, action: actionPipeline.DeletePipelineDeployment) {
        let url = '/project/' + action.payload.projectKey +
            '/application/' + action.payload.pipelineName + '/deployment/config/' + action.payload.integrationName;
        return this._http.delete<Pipeline>(url)
            .pipe(tap((pip) => {
                const state = ctx.getState();
                let pipKey = action.payload.projectKey + '/' + action.payload.pipelineName;
                let pipUpdated = Object.assign({}, state.pipelines[pipKey], {
                    deployment_strategies: pip.deployment_strategies
                });

                ctx.setState({
                    ...state,
                    pipelines: Object.assign({}, state.pipelines, { [pipKey]: pipUpdated }),
                });
            }));
    }

    //  ------- VCS strategies --------- //
    @Action(actionPipeline.ConnectVcsRepoOnApplication)
    connectRepo(ctx: StateContext<PipelinesStateModel>, action: actionPipeline.ConnectVcsRepoOnApplication) {
        let repoManager = action.payload.repoManager;
        let repoFullname = action.payload.repoFullName;
        let url = '/project/' + action.payload.projectKey + '/repositories_manager/' +
            repoManager + '/application/' + action.payload.pipelineName + '/attach';
        let headers = new HttpHeaders();
        headers.append('Content-Type', 'application/x-www-form-urlencoded');

        let params = new HttpParams();
        params = params.append('fullname', repoFullname);

        return this._http.post<Pipeline>(url, params.toString(), { headers, params })
            .pipe(tap((pip) => {
                const state = ctx.getState();
                let pipKey = action.payload.projectKey + '/' + action.payload.pipelineName;
                let pipUpdated = Object.assign({}, state.pipelines[pipKey], {
                    vcs_server: pip.vcs_server,
                    repository_fullname: pip.repository_fullname,
                    vcs_strategy: pip.vcs_strategy
                });

                ctx.setState({
                    ...state,
                    pipelines: Object.assign({}, state.pipelines, { [pipKey]: pipUpdated }),
                });
            }));
    }

    @Action(actionPipeline.DeleteVcsRepoOnApplication)
    deleteRepo(ctx: StateContext<PipelinesStateModel>, action: actionPipeline.DeleteVcsRepoOnApplication) {
        let repoManager = action.payload.repoManager;
        let url = '/project/' + action.payload.projectKey + '/repositories_manager/' +
            repoManager + '/application/' + action.payload.pipelineName + '/detach';

        return this._http.post<Pipeline>(url, null)
            .pipe(tap((pip) => {
                const state = ctx.getState();
                let pipKey = action.payload.projectKey + '/' + action.payload.pipelineName;
                let pipUpdated = Object.assign({}, state.pipelines[pipKey], <Pipeline>{
                    vcs_server: pip.vcs_server,
                    repository_fullname: pip.repository_fullname,
                    vcs_strategy: pip.vcs_strategy
                });

                ctx.setState({
                    ...state,
                    pipelines: Object.assign({}, state.pipelines, { [pipKey]: pipUpdated }),
                });
            }));
    }

    //  ------- Misc --------- //
    @Action(actionPipeline.ExternalChangeApplication)
    externalChange(ctx: StateContext<PipelinesStateModel>, action: actionPipeline.ExternalChangeApplication) {
        const state = ctx.getState();
        const pipKey = action.payload.projectKey + '/' + action.payload.pipelineName;
        const pipUpdated = Object.assign({}, state.pipelines[pipKey], { externalChange: true });

        ctx.setState({
            ...state,
            pipelines: Object.assign({}, state.pipelines, { [pipKey]: pipUpdated }),
        });
    }

    @Action(actionPipeline.DeleteFromCacheApplication)
    deleteFromCache(ctx: StateContext<PipelinesStateModel>, action: actionPipeline.DeleteFromCacheApplication) {
        const state = ctx.getState();
        const pipKey = action.payload.projectKey + '/' + action.payload.pipelineName;
        let pipelines = Object.assign({}, state.pipelines);
        delete pipelines[pipKey];

        ctx.setState({
            ...state,
            pipelines,
        });
    }

    @Action(actionPipeline.ResyncPipeline)
    resync(ctx: StateContext<PipelinesStateModel>, action: actionPipeline.ResyncPipeline) {
        let params = new HttpParams();
        params = params.append('withNotifs', 'true');
        params = params.append('withUsage', 'true');
        params = params.append('withIcon', 'true');
        params = params.append('withKeys', 'true');
        params = params.append('withDeploymentStrategies', 'true');
        params = params.append('withVulnerabilities', 'true');

        return this._http.get<Pipeline>(
            `/project/${action.payload.projectKey}/application/${action.payload.pipelineName}`,
            { params }
        ).pipe(tap((pip) => {
            if (pip.vcs_strategy) {
                pip.vcs_strategy.password = '**********';
            }
            ctx.dispatch(new actionPipeline.LoadPipeline(pip));
        }));
    }
}
