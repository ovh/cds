import { HttpClient, HttpHeaders, HttpParams } from '@angular/common/http';
import { Action, createSelector, State, StateContext } from '@ngxs/store';
import { Parameter } from 'app/model/parameter.model';
import { Pipeline, PipelineAudit } from 'app/model/pipeline.model';
import { cloneDeep } from 'lodash-es';
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
            (state: PipelinesStateModel): Pipeline => state.pipelines[projectKey + '/' + pipelineName]
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
            `/project/${action.payload.projectKey}/pipeline`,
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

    @Action(actionPipeline.ImportPipeline)
    import(ctx: StateContext<PipelinesStateModel>, action: actionPipeline.ImportPipeline) {
        const state = ctx.getState();

        // Refresh when change project
        if (state.currentProjectKey !== action.payload.projectKey) {
            ctx.setState({
                ...state,
                currentProjectKey: action.payload.projectKey,
                pipelines: {},
                loading: false,
            });
        }

        let headers = new HttpHeaders();
        headers = headers.append('Content-Type', 'application/x-yaml');
        let params = new HttpParams();
        params = params.append('format', 'yaml');

        let request = this._http.post<Array<string>>(
            `/project/${action.payload.projectKey}/import/pipeline`,
            action.payload.pipelineCode,
            { headers, params }
        );
        if (action.payload.pipName) {
            request = this._http.put<Array<string>>(
                `/project/${action.payload.projectKey}/import/pipeline/${action.payload.pipName}`,
                action.payload.pipelineCode,
                { headers, params }
            );
        }

        return request.pipe(tap(() => {
            if (action.payload.pipName) {
                return ctx.dispatch(new actionPipeline.ResyncPipeline({
                    projectKey: action.payload.projectKey,
                    pipelineName: action.payload.pipName
                }));
            }
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
            pipelines: Object.assign({}, pipelines, { [pipKey]: action.payload.pipeline }),
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
            `/project/${action.payload.projectKey}/pipeline/${action.payload.pipelineName}`,
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

                return ctx.dispatch(new actionPipeline.ResyncPipeline({
                    projectKey: action.payload.projectKey,
                    pipelineName: pip.name
                }));
            } else {
                let pipUpdated: Pipeline = {
                    ...state.pipelines[pipKey],
                    ...pip,
                    preview: null
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
            `/project/${action.payload.projectKey}/pipeline/${action.payload.pipelineName}`
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
        return this._http.post<Parameter>(url, parameter)
            .pipe(tap((param) => {
                const state = ctx.getState();
                let pipKey = action.payload.projectKey + '/' + action.payload.pipelineName;
                let pipToUpdate = cloneDeep(state.pipelines[pipKey]);
                if (!pipToUpdate.parameters) {
                    pipToUpdate.parameters = new Array<Parameter>();
                }
                pipToUpdate.parameters.push(param);
                ctx.setState({
                    ...state,
                    pipelines: Object.assign({}, state.pipelines, { [pipKey]: pipToUpdate }),
                });
            }));
    }

    @Action(actionPipeline.UpdatePipelineParameter)
    updateParameter(ctx: StateContext<PipelinesStateModel>, action: actionPipeline.UpdatePipelineParameter) {
        let parameter = action.payload.parameter;
        let url = '/project/' + action.payload.projectKey +
            '/pipeline/' + action.payload.pipelineName +
            '/parameter/' + action.payload.parameterName;

        return this._http.put<Parameter>(url, parameter)
            .pipe(tap((param) => {
                const state = ctx.getState();
                let pipKey = action.payload.projectKey + '/' + action.payload.pipelineName;
                let pipToUpdate = cloneDeep(state.pipelines[pipKey]);

                pipToUpdate.parameters = pipToUpdate.parameters.map(p => {
                   if (p.id === param.id) {
                       return param;
                   }
                   return p;
                });
                ctx.setState({
                    ...state,
                    pipelines: Object.assign({}, state.pipelines, { [pipKey]: pipToUpdate }),
                });
            }));
    }

    @Action(actionPipeline.DeletePipelineParameter)
    deleteParameter(ctx: StateContext<PipelinesStateModel>, action: actionPipeline.DeletePipelineParameter) {
        let parameter = action.payload.parameter;
        let url = `/project/${action.payload.projectKey}/pipeline/${action.payload.pipelineName}/parameter/${parameter.name}`;
        return this._http.delete<Parameter>(url)
            .pipe(tap((param) => {
                const state = ctx.getState();
                let pipKey = action.payload.projectKey + '/' + action.payload.pipelineName;
                let pipToUpdate = cloneDeep(state.pipelines[pipKey]);

                pipToUpdate.parameters = pipToUpdate.parameters.filter(p => p.id !== action.payload.parameter.id);
                ctx.setState({
                    ...state,
                    pipelines: Object.assign({}, state.pipelines, { [pipKey]: pipToUpdate }),
                });
            }));
    }

    //  ------- Audit --------- //
    @Action(actionPipeline.FetchPipelineAudits)
    fetchAudits(ctx: StateContext<PipelinesStateModel>, action: actionPipeline.FetchPipelineAudits) {
        return this._http.get<PipelineAudit[]>(
            `/project/${action.payload.projectKey}/pipeline/${action.payload.pipelineName}/audits`
        ).pipe(tap((audits: PipelineAudit[]) => {
            const state = ctx.getState();
            let pipKey = action.payload.projectKey + '/' + action.payload.pipelineName;

            ctx.dispatch(new actionPipeline.LoadPipeline({
                projectKey: action.payload.projectKey,
                pipeline: Object.assign({}, state.pipelines[pipKey], { audits })
            }));
        }));
    }

    @Action(actionPipeline.RollbackPipeline)
    rollback(ctx: StateContext<PipelinesStateModel>, action: actionPipeline.RollbackPipeline) {
        let auditId = action.payload.auditId;
        return this._http.post<Pipeline>(
            `/project/${action.payload.projectKey}/pipeline/${action.payload.pipelineName}/rollback/${auditId}`,
            {}
        ).pipe(tap((pip: Pipeline) => {
            ctx.dispatch(new actionPipeline.LoadPipeline({
                projectKey: action.payload.projectKey,
                pipeline: pip
            }));
        }));
    }

    //  ------- Workflow --------- //
    @Action(actionPipeline.AddPipelineStage)
    addStage(ctx: StateContext<PipelinesStateModel>, action: actionPipeline.AddPipelineStage) {
        let stage = action.payload.stage;
        let url = '/project/' + action.payload.projectKey + '/pipeline/' + action.payload.pipelineName + '/stage';
        return this._http.post<Pipeline>(url, stage)
            .pipe(tap((pip) => {
                const state = ctx.getState();
                let pipKey = pip.projectKey + '/' + pip.name;
                let pipUpdated = Object.assign({}, state.pipelines[pipKey], <Pipeline>{ stages: pip.stages });

                ctx.setState({
                    ...state,
                    pipelines: Object.assign({}, state.pipelines, { [pipKey]: pipUpdated }),
                });
            }));
    }

    @Action(actionPipeline.UpdatePipelineStage)
    updateStage(ctx: StateContext<PipelinesStateModel>, action: actionPipeline.UpdatePipelineStage) {
        let stage = action.payload.changes;
        let url = '/project/' + action.payload.projectKey +
            '/pipeline/' + action.payload.pipelineName +
            '/stage/' + stage.id;

        return this._http.put<Pipeline>(url, stage)
            .pipe(tap((pip) => {
                const state = ctx.getState();
                let pipKey = pip.projectKey + '/' + pip.name;
                let pipUpdated = Object.assign({}, state.pipelines[pipKey], <Pipeline>{ stages: pip.stages });

                ctx.setState({
                    ...state,
                    pipelines: Object.assign({}, state.pipelines, { [pipKey]: pipUpdated }),
                });
            }));
    }

    @Action(actionPipeline.MovePipelineStage)
    moveStage(ctx: StateContext<PipelinesStateModel>, action: actionPipeline.MovePipelineStage) {
        let stage = action.payload.stage;
        let url = '/project/' + action.payload.projectKey +
            '/pipeline/' + action.payload.pipelineName +
            '/stage/move';

        return this._http.post<Pipeline>(url, stage)
            .pipe(tap((pip) => {
                const state = ctx.getState();
                let pipKey = pip.projectKey + '/' + pip.name;
                let pipUpdated = Object.assign({}, state.pipelines[pipKey], <Pipeline>{ stages: pip.stages });

                ctx.setState({
                    ...state,
                    pipelines: Object.assign({}, state.pipelines, { [pipKey]: pipUpdated }),
                });
            }));
    }

    @Action(actionPipeline.DeletePipelineStage)
    deleteStage(ctx: StateContext<PipelinesStateModel>, action: actionPipeline.DeletePipelineStage) {
        let stage = action.payload.stage;
        let url = `/project/${action.payload.projectKey}/pipeline/${action.payload.pipelineName}/stage/${stage.id}`;
        return this._http.delete<Pipeline>(url)
            .pipe(tap((pip) => {
                const state = ctx.getState();
                let pipKey = action.payload.projectKey + '/' + action.payload.pipelineName;
                let pipUpdated = Object.assign({}, state.pipelines[pipKey], { stages: pip.stages });

                ctx.setState({
                    ...state,
                    pipelines: Object.assign({}, state.pipelines, { [pipKey]: pipUpdated }),
                });
            }));
    }

    @Action(actionPipeline.AddPipelineJob)
    addJob(ctx: StateContext<PipelinesStateModel>, action: actionPipeline.AddPipelineJob) {
        let job = action.payload.job;
        let url = '/project/' + action.payload.projectKey + '/pipeline/' + action.payload.pipelineName +
            '/stage/' + action.payload.stageId + '/job';
        return this._http.post<Pipeline>(url, job)
            .pipe(tap((pip) => {
                const state = ctx.getState();
                let pipKey = pip.projectKey + '/' + pip.name;
                let pipUpdated = Object.assign({}, state.pipelines[pipKey], <Pipeline>{ stages: pip.stages });

                ctx.setState({
                    ...state,
                    pipelines: Object.assign({}, state.pipelines, { [pipKey]: pipUpdated }),
                });
            }));
    }

    @Action(actionPipeline.UpdatePipelineJob)
    updateJob(ctx: StateContext<PipelinesStateModel>, action: actionPipeline.UpdatePipelineJob) {
        let job = action.payload.changes;
        let url = '/project/' + action.payload.projectKey +
            '/pipeline/' + action.payload.pipelineName +
            '/stage/' + action.payload.stageId + '/job/' + job.pipeline_action_id;

        return this._http.put<Pipeline>(url, job)
            .pipe(tap((pip) => {
                const state = ctx.getState();
                let pipKey = pip.projectKey + '/' + pip.name;
                let pipUpdated = Object.assign({}, state.pipelines[pipKey], <Pipeline>{ stages: pip.stages });

                ctx.setState({
                    ...state,
                    pipelines: Object.assign({}, state.pipelines, { [pipKey]: pipUpdated }),
                });
            }));
    }

    @Action(actionPipeline.DeletePipelineJob)
    deleteJob(ctx: StateContext<PipelinesStateModel>, action: actionPipeline.DeletePipelineJob) {
        let job = action.payload.job;
        let url = '/project/' + action.payload.projectKey +
            '/pipeline/' + action.payload.pipelineName +
            '/stage/' + action.payload.stageId + '/job/' + job.pipeline_action_id;
        return this._http.delete<Pipeline>(url)
            .pipe(tap((pip) => {
                const state = ctx.getState();
                let pipKey = action.payload.projectKey + '/' + action.payload.pipelineName;
                let pipUpdated = Object.assign({}, state.pipelines[pipKey], { stages: pip.stages });

                ctx.setState({
                    ...state,
                    pipelines: Object.assign({}, state.pipelines, { [pipKey]: pipUpdated }),
                });
            }));
    }

    //  ------- Misc --------- //
    @Action(actionPipeline.FetchAsCodePipeline)
    fetchAsCode(ctx: StateContext<PipelinesStateModel>, action: actionPipeline.FetchAsCodePipeline) {
        let params = new HttpParams();
        params = params.append('format', 'yaml');

        return this._http.get<string>(
            `/project/${action.payload.projectKey}/export/pipeline/${action.payload.pipelineName}`,
            { params, responseType: <any>'text' }
        ).pipe(tap((asCode: string) => {
            const state = ctx.getState();
            let pipKey = action.payload.projectKey + '/' + action.payload.pipelineName;

            ctx.dispatch(new actionPipeline.LoadPipeline({
                projectKey: action.payload.projectKey,
                pipeline: Object.assign({}, state.pipelines[pipKey], <Pipeline>{ asCode })
            }));
        }));
    }

    @Action(actionPipeline.PreviewPipeline)
    preview(ctx: StateContext<PipelinesStateModel>, action: actionPipeline.PreviewPipeline) {
        let headers = new HttpHeaders();
        headers = headers.append('Content-Type', 'application/x-yaml');
        let params = new HttpParams();
        params = params.append('format', 'yaml');

        return this._http.post<Pipeline>(
            `/project/${action.payload.projectKey}/preview/pipeline`,
            action.payload.pipCode,
            { params, headers }
        ).pipe(tap((pip: Pipeline) => {
            const state = ctx.getState();
            const pipKey = action.payload.projectKey + '/' + action.payload.pipelineName;

            ctx.dispatch(new actionPipeline.LoadPipeline({
                projectKey: action.payload.projectKey,
                pipeline: Object.assign({}, state.pipelines[pipKey], { preview: pip })
            }));
        }));
    }

    @Action(actionPipeline.ExternalChangePipeline)
    externalChange(ctx: StateContext<PipelinesStateModel>, action: actionPipeline.ExternalChangePipeline) {
        const state = ctx.getState();
        const pipKey = action.payload.projectKey + '/' + action.payload.pipelineName;
        const pipUpdated = Object.assign({}, state.pipelines[pipKey], { externalChange: true });

        ctx.setState({
            ...state,
            pipelines: Object.assign({}, state.pipelines, { [pipKey]: pipUpdated }),
        });
    }

    @Action(actionPipeline.DeleteFromCachePipeline)
    deleteFromCache(ctx: StateContext<PipelinesStateModel>, action: actionPipeline.DeleteFromCachePipeline) {
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
        params = params.append('withApplications', 'true');
        params = params.append('withWorkflows', 'true');
        params = params.append('withEnvironments', 'true');

        return this._http.get<Pipeline>(
            `/project/${action.payload.projectKey}/pipeline/${action.payload.pipelineName}`,
            { params }
        ).pipe(tap((pip) => {
            ctx.dispatch(new actionPipeline.LoadPipeline({
                projectKey: action.payload.projectKey,
                pipeline: pip
            }));
        }));
    }

    @Action(actionPipeline.ClearCachePipeline)
    clearCache(ctx: StateContext<PipelinesStateModel>, _: actionPipeline.ClearCachePipeline) {
        ctx.setState(getInitialPipelinesState());
    }
}
