import { HttpClient, HttpHeaders, HttpParams } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Action, createSelector, State, StateContext } from '@ngxs/store';
import { Job } from 'app/model/job.model';
import { Parameter } from 'app/model/parameter.model';
import { Pipeline, PipelineAudit } from 'app/model/pipeline.model';
import { PipelineService } from 'app/service/pipeline/pipeline.service';
import * as actionAsCode from 'app/store/ascode.action';
import { cloneDeep } from 'lodash-es';
import { tap } from 'rxjs/operators';
import * as actionPipeline from './pipelines.action';
import * as ActionProject from './project.action';

export class PipelinesStateModel {
    public pipeline: Pipeline;
    public editPipeline: Pipeline;
    public currentProjectKey: string;
    public loading: boolean;
    public editMode: boolean;
}

export function getInitialPipelinesState(): PipelinesStateModel {
    return {
        pipeline: null,
        editPipeline: null,
        currentProjectKey: null,
        loading: true,
        editMode: false
    };
}

@State<PipelinesStateModel>({
    name: 'pipelines',
    defaults: getInitialPipelinesState()
})
@Injectable()
export class PipelinesState {

    constructor(private _http: HttpClient, private _pipelineService: PipelineService) { }

    static getCurrent() {
        return createSelector(
            [PipelinesState],
            (state: PipelinesStateModel): PipelinesStateModel => state
        );
    }

    @Action(actionPipeline.AddPipeline)
    add(ctx: StateContext<PipelinesStateModel>, action: actionPipeline.AddPipeline) {
        const state = ctx.getState();

        return this._http.post<Pipeline>(
            `/project/${action.payload.projectKey}/pipeline`,
            action.payload.pipeline
        ).pipe(tap((pip) => {
            ctx.setState({
                ...state,
                currentProjectKey: action.payload.projectKey,
                editPipeline: null,
                editMode: false,
                pipeline: pip,
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
                pipeline: null,
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

    @Action(actionPipeline.FetchPipeline)
    fetch(ctx: StateContext<PipelinesStateModel>, action: actionPipeline.FetchPipeline) {
        const state = ctx.getState();
        if (state.pipeline && state.pipeline.name === action.payload.pipelineName &&
            state.currentProjectKey === action.payload.projectKey) {
            return;
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
            ctx.setState({
                ...state,
                pipeline: pip,
            });
            if (pip.name !== action.payload.pipelineName) {
                ctx.dispatch(new ActionProject.UpdatePipelineInProject({
                    previousPipName: action.payload.pipelineName,
                    changes: pip
                }));
            }
        }));
    }

    @Action(actionPipeline.DeletePipeline)
    delete(ctx: StateContext<PipelinesStateModel>, action: actionPipeline.DeletePipeline) {
        return this._http.delete(
            `/project/${action.payload.projectKey}/pipeline/${action.payload.pipelineName}`
        ).pipe(tap(() => {
            const state = ctx.getState();
            ctx.setState({
                ...state,
                pipeline: null,
            });
            ctx.dispatch(new ActionProject.DeletePipelineInProject({ pipelineName: action.payload.pipelineName }));
        }));
    }

    //  ------- Parameter --------- //
    @Action(actionPipeline.AddPipelineParameter)
    addParameter(ctx: StateContext<PipelinesStateModel>, action: actionPipeline.AddPipelineParameter) {
        let parameter = action.payload.parameter;
        const stateEditMode = ctx.getState();
        if (stateEditMode.editMode) {
            let pipToUpdate = cloneDeep(stateEditMode.editPipeline);
            if (!pipToUpdate.parameters) {
                pipToUpdate.parameters = new Array<Parameter>();
            }
            pipToUpdate.parameters.push(action.payload.parameter);
            pipToUpdate.editModeChanged = true;
            ctx.setState({
                ...stateEditMode,
                editPipeline: pipToUpdate,
            });
            return;
        }

        let url = '/project/' + action.payload.projectKey + '/pipeline/' + action.payload.pipelineName + '/parameter/' + parameter.name;
        return this._http.post<Parameter>(url, parameter)
            .pipe(tap((param) => {
                const state = ctx.getState();
                let pipToUpdate = cloneDeep(state.pipeline);
                if (!pipToUpdate.parameters) {
                    pipToUpdate.parameters = new Array<Parameter>();
                }
                pipToUpdate.parameters.push(param);
                ctx.setState({
                    ...state,
                    pipeline: pipToUpdate,
                });
            }));
    }

    @Action(actionPipeline.UpdatePipelineParameter)
    updateParameter(ctx: StateContext<PipelinesStateModel>, action: actionPipeline.UpdatePipelineParameter) {
        let parameter = action.payload.parameter;

        const stateEditMode = ctx.getState();
        if (stateEditMode.editMode) {
            let pipToUpdate = cloneDeep(stateEditMode.editPipeline);
            let indexParam = pipToUpdate.parameters.findIndex(p => p.name === action.payload.parameterName);
            action.payload.parameter.hasChanged = false;
            action.payload.parameter.updating = false;
            pipToUpdate.parameters[indexParam] = action.payload.parameter
            pipToUpdate.editModeChanged = true;
            ctx.setState({
                ...stateEditMode,
                editPipeline: pipToUpdate,
            });
            return;
        }

        let url = '/project/' + action.payload.projectKey +
            '/pipeline/' + action.payload.pipelineName +
            '/parameter/' + action.payload.parameterName;

        return this._http.put<Parameter>(url, parameter)
            .pipe(tap((param) => {
                const state = ctx.getState();
                let pipToUpdate = cloneDeep(state.pipeline);
                pipToUpdate.parameters = pipToUpdate.parameters.map(p => {
                   if (p.id === param.id) {
                       return param;
                   }
                   return p;
                });
                ctx.setState({
                    ...state,
                    pipeline: pipToUpdate,
                });
            }));
    }

    @Action(actionPipeline.DeletePipelineParameter)
    deleteParameter(ctx: StateContext<PipelinesStateModel>, action: actionPipeline.DeletePipelineParameter) {
        let parameter = action.payload.parameter;

        const stateEditMode = ctx.getState();
        if (stateEditMode.editMode) {
            let pipToUpdate = cloneDeep(stateEditMode.editPipeline);
            pipToUpdate.parameters = pipToUpdate.parameters.filter(p => p.name !== action.payload.parameter.name);
            pipToUpdate.editModeChanged = true;
            ctx.setState({
                ...stateEditMode,
                editPipeline: pipToUpdate,
            });
            return;
        }


        let url = `/project/${action.payload.projectKey}/pipeline/${action.payload.pipelineName}/parameter/${parameter.name}`;
        return this._http.delete<Parameter>(url)
            .pipe(tap(() => {
                const state = ctx.getState();
                let pipToUpdate = cloneDeep(state.pipeline);
                pipToUpdate.parameters = pipToUpdate.parameters.filter(p => p.id !== action.payload.parameter.id);
                ctx.setState({
                    ...state,
                    pipeline: pipToUpdate,
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
            ctx.setState({
                ...state,
                currentProjectKey: action.payload.projectKey,
                pipeline: Object.assign({}, state.pipeline, { audits }),
                loading: false,
            });
        }));
    }

    @Action(actionPipeline.RollbackPipeline)
    rollback(ctx: StateContext<PipelinesStateModel>, action: actionPipeline.RollbackPipeline) {
        let auditId = action.payload.auditId;
        return this._http.post<Pipeline>(
            `/project/${action.payload.projectKey}/pipeline/${action.payload.pipelineName}/rollback/${auditId}`,
            {}
        ).pipe(tap((pip: Pipeline) => {
            const state = ctx.getState();
            ctx.setState({
                ...state,
                currentProjectKey: action.payload.projectKey,
                pipeline: pip,
                loading: false,
            });
        }));
    }

    //  ------- Workflow --------- //
    @Action(actionPipeline.AddPipelineStage)
    addStage(ctx: StateContext<PipelinesStateModel>, action: actionPipeline.AddPipelineStage) {
        const stateEditMode = ctx.getState();
        let stage = action.payload.stage;
        if (stateEditMode.editMode) {
            let pipUpdated = cloneDeep(stateEditMode.editPipeline);
            pipUpdated.stages.push(action.payload.stage);
            pipUpdated.editModeChanged = true;
            ctx.setState({
                ...stateEditMode,
                editPipeline: pipUpdated,
            });
            return;
        }

        let url = '/project/' + action.payload.projectKey + '/pipeline/' + action.payload.pipelineName + '/stage';
        return this._http.post<Pipeline>(url, stage)
            .pipe(tap((pip) => {
                const state = ctx.getState();
                let pipUpdated = Object.assign({}, state.pipeline, <Pipeline>{ stages: pip.stages });
                ctx.setState({
                    ...state,
                    pipeline: pipUpdated,
                });
            }));
    }

    @Action(actionPipeline.UpdatePipelineStage)
    updateStage(ctx: StateContext<PipelinesStateModel>, action: actionPipeline.UpdatePipelineStage) {
        const stateEditMode = ctx.getState();
        let stage = action.payload.changes;

        if (stateEditMode.editMode) {
            let pipToUpdate = cloneDeep(stateEditMode.editPipeline);
            pipToUpdate.stages = pipToUpdate.stages.map(s => {
                if (s.ref === action.payload.changes.ref) {
                    return action.payload.changes;
                }
                return s;
            });
            pipToUpdate.editModeChanged = true;
            ctx.setState({
                ...stateEditMode,
                editPipeline: pipToUpdate,
            });
            return;
        }

        let url = '/project/' + action.payload.projectKey +
            '/pipeline/' + action.payload.pipelineName +
            '/stage/' + stage.id;

        return this._http.put<Pipeline>(url, stage)
            .pipe(tap((pip) => {
                const state = ctx.getState();
                let pipUpdated = Object.assign({}, state.pipeline, <Pipeline>{ stages: pip.stages });

                ctx.setState({
                    ...state,
                    pipeline: pipUpdated,
                });
            }));
    }

    @Action(actionPipeline.MovePipelineStage)
    moveStage(ctx: StateContext<PipelinesStateModel>, action: actionPipeline.MovePipelineStage) {
        const stateEditMode = ctx.getState();
        if (stateEditMode.editMode) {
            let pipToUpdate = cloneDeep(action.payload.pipeline);
            for (let i = 0; i < pipToUpdate.stages.length; i++) {
                pipToUpdate.stages[i].build_order = i + 1;
            }
            pipToUpdate.editModeChanged = true;
            ctx.setState({
                ...stateEditMode,
                pipeline: pipToUpdate,
            });
            return
        }

        let stage = action.payload.stage;
        let url = '/project/' + action.payload.projectKey +
            '/pipeline/' + action.payload.pipeline.name +
            '/stage/move';

        return this._http.post<Pipeline>(url, stage)
            .pipe(tap((pip) => {
                const state = ctx.getState();
                let pipUpdated = Object.assign({}, state.pipeline, <Pipeline>{ stages: pip.stages });
                ctx.setState({
                    ...state,
                    pipeline: pipUpdated,
                });
            }));
    }

    @Action(actionPipeline.DeletePipelineStage)
    deleteStage(ctx: StateContext<PipelinesStateModel>, action: actionPipeline.DeletePipelineStage) {
        const stateEditMode = ctx.getState();
        let stage = action.payload.stage;

        if (stateEditMode.editMode) {
            let pipToUpdate = cloneDeep(stateEditMode.editPipeline);
            pipToUpdate.stages = pipToUpdate.stages.filter( s => s.ref !== action.payload.stage.ref);
            pipToUpdate.editModeChanged = true;
            ctx.setState({
                ...stateEditMode,
                editPipeline: pipToUpdate,
            });
            return;
        }

        let url = `/project/${action.payload.projectKey}/pipeline/${action.payload.pipelineName}/stage/${stage.id}`;
        return this._http.delete<Pipeline>(url)
            .pipe(tap((pip) => {
                const state = ctx.getState();
                let pipUpdated = Object.assign({}, state.pipeline, { stages: pip.stages });

                ctx.setState({
                    ...state,
                    pipeline: pipUpdated,
                });
            }));
    }

    @Action(actionPipeline.AddPipelineJob)
    addJob(ctx: StateContext<PipelinesStateModel>, action: actionPipeline.AddPipelineJob) {
        let job = action.payload.job;
        const stateEditMode = ctx.getState();
        if (stateEditMode.editMode) {
            let pipUpdated = cloneDeep(stateEditMode.editPipeline);
            pipUpdated.editModeChanged = true;
            let stage = pipUpdated.stages.find(s => s.ref === action.payload.stage.ref);
            if (!stage.jobs) {
                stage.jobs = new Array<Job>();
            }
            stage.jobs.push(job);
            ctx.setState({
                ...stateEditMode,
                editPipeline: pipUpdated,
            });
            return;
        }

        let url = '/project/' + action.payload.projectKey + '/pipeline/' + action.payload.pipelineName +
            '/stage/' + action.payload.stage.id + '/job';
        return this._http.post<Pipeline>(url, job)
            .pipe(tap((pip) => {
                const state = ctx.getState();
                let pipUpdated = Object.assign({}, state.pipeline, <Pipeline>{ stages: pip.stages });
                ctx.setState({
                    ...state,
                    pipeline: pipUpdated,
                });
            }));
    }

    @Action(actionPipeline.UpdatePipelineJob)
    updateJob(ctx: StateContext<PipelinesStateModel>, action: actionPipeline.UpdatePipelineJob) {
        const stateEditMode = ctx.getState();
        let job = action.payload.changes;
        if (stateEditMode.editMode) {
            job.action.hasChanged = false;
            job.action.loading = false;
            let pipToUpdate = cloneDeep(stateEditMode.editPipeline);
            pipToUpdate.editModeChanged = true;
            let stage = pipToUpdate.stages.find(s => s.ref === action.payload.stage.ref);
            stage.jobs = stage.jobs.map(j => {
                if (j.ref === job.ref) {
                    return cloneDeep(job);
                }
                return j;
            });
            ctx.setState({
                ...stateEditMode,
                editPipeline: pipToUpdate,
            });
            return;
        }
        let url = '/project/' + action.payload.projectKey +
            '/pipeline/' + action.payload.pipelineName +
            '/stage/' + action.payload.stage.id + '/job/' + job.pipeline_action_id;

        return this._http.put<Pipeline>(url, job)
            .pipe(tap((pip) => {
                const state = ctx.getState();
                let pipUpdated = Object.assign({}, state.pipeline, <Pipeline>{ stages: pip.stages });

                ctx.setState({
                    ...state,
                    pipeline: pipUpdated,
                });
            }));
    }

    @Action(actionPipeline.DeletePipelineJob)
    deleteJob(ctx: StateContext<PipelinesStateModel>, action: actionPipeline.DeletePipelineJob) {
        const stateEditMode = ctx.getState();
        let job = action.payload.job;

        if (stateEditMode.editMode) {
            let pipToUpdate = cloneDeep(stateEditMode.editPipeline);
            pipToUpdate.editModeChanged = true;
            let stage = pipToUpdate.stages.find(s => s.ref === action.payload.stage.ref);
            stage.jobs = stage.jobs.filter(j => j.ref !== job.ref);
            ctx.setState({
                ...stateEditMode,
                editPipeline: pipToUpdate,
            });
            return;
        }

        let url = '/project/' + action.payload.projectKey +
            '/pipeline/' + action.payload.pipelineName +
            '/stage/' + action.payload.stage.id + '/job/' + job.pipeline_action_id;
        return this._http.delete<Pipeline>(url)
            .pipe(tap((pip) => {
                const state = ctx.getState();
                let pipUpdated = Object.assign({}, state.pipeline, { stages: pip.stages });
                ctx.setState({
                    ...state,
                    pipeline: pipUpdated,
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
            ctx.setState({
                ...state,
                currentProjectKey: action.payload.projectKey,
                pipeline: Object.assign({}, state.pipeline, <Pipeline>{ asCode }),
                loading: false,
            });
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
            ctx.setState({
                ...state,
                currentProjectKey: action.payload.projectKey,
                pipeline: Object.assign({}, state.pipeline, { preview: pip }),
                loading: false,
            });
        }));
    }

    @Action(actionPipeline.ExternalChangePipeline)
    externalChange(ctx: StateContext<PipelinesStateModel>, action: actionPipeline.ExternalChangePipeline) {
        const state = ctx.getState();
        const pipUpdated = Object.assign({}, state.pipeline, { externalChange: true });

        ctx.setState({
            ...state,
            pipeline: pipUpdated,
        });
    }

    @Action(actionPipeline.ResyncPipeline)
    resync(ctx: StateContext<PipelinesStateModel>, action: actionPipeline.ResyncPipeline) {
        return this._pipelineService.getPipeline(action.payload.projectKey, action.payload.pipelineName)
            .pipe(tap((pip) => {
            const state = ctx.getState();
            let editMode = false;
            let editPipeline: Pipeline;
            if (pip.from_repository) {
                editMode = true;
                editPipeline = cloneDeep(pip);
                Pipeline.InitRef(editPipeline);
            }
            ctx.setState({
                ...state,
                pipeline: pip,
                editPipeline,
                currentProjectKey: action.payload.projectKey,
                editMode,
            });
        }));
    }

    @Action(actionPipeline.ClearCachePipeline)
    clearCache(ctx: StateContext<PipelinesStateModel>, _: actionPipeline.ClearCachePipeline) {
        ctx.setState(getInitialPipelinesState());
    }

    @Action(actionPipeline.CancelPipelineEdition)
    cancelPipelineEdition(ctx: StateContext<PipelinesStateModel>, _: actionPipeline.CancelPipelineEdition) {
        const state = ctx.getState();
        let editMode = state.editMode;
        if (state.pipeline.from_repository) {
            editMode = true;
        }
        let editPipeline = cloneDeep(state.pipeline);
        Pipeline.InitRef(editPipeline);
        ctx.setState({
            ...state,
            editPipeline,
            editMode,
        });
    }

    @Action(actionAsCode.ResyncEvents)
    refreshAsCodeEvents(ctx: StateContext<PipelinesStateModel>, _) {
        const state = ctx.getState();
        if (state.pipeline) {
            ctx.dispatch(new actionPipeline.ResyncPipeline({projectKey: state.currentProjectKey, pipelineName: state.pipeline.name}));
        }
    }
}
