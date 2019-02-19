import { HttpClient, HttpParams } from '@angular/common/http';
import { Action, State, StateContext } from '@ngxs/store';
import { IdName, LoadOpts, Project } from 'app/model/project.model';
import { tap } from 'rxjs/operators';
import * as ProjectAction from './project.action';

export class ProjectStateModel {
    public project: Project;
    public loading: boolean;
    public currentProjectKey: string;
}

@State<ProjectStateModel>({
    name: 'project',
    defaults: {
        project: null,
        loading: true,
        currentProjectKey: null
    }
})
export class ProjectState {

    constructor(private _http: HttpClient) { }


    @Action(ProjectAction.LoadProject)
    load(ctx: StateContext<ProjectStateModel>, action: ProjectAction.LoadProject) {
        const state = ctx.getState();
        ctx.setState({
            ...state,
            project: {
                ...state.project,
                ...action.payload
            },
            loading: false,
        });
    }


    @Action(ProjectAction.FetchProject)
    fetch(ctx: StateContext<ProjectStateModel>, action: ProjectAction.FetchProject) {
        const state = ctx.getState();

        if (state.currentProjectKey && state.currentProjectKey === action.payload.projectKey && state.project && state.project.key) {
            let funcs = action.payload.opts.filter((opt) => state.project[opt.fieldName] == null);

            if (!funcs.length) {
                return ctx.dispatch(new ProjectAction.LoadProject(state.project));
            }
        }
        if (state.currentProjectKey && state.currentProjectKey !== action.payload.projectKey) {
            ctx.setState(<ProjectStateModel>{
                project: null,
                loading: true,
                currentProjectKey: action.payload.projectKey
            });
        }

        return ctx.dispatch(new ProjectAction.ResyncProject(action.payload));
    }

    @Action(ProjectAction.ResyncProject)
    resync(ctx: StateContext<ProjectStateModel>, action: ProjectAction.ResyncProject) {
        let params = new HttpParams();
        let opts = action.payload.opts;

        if (Array.isArray(opts) && opts.length) {
            opts = opts.concat([
                new LoadOpts('withGroups', 'groups'),
                new LoadOpts('withPermission', 'permission')
            ]);
        } else {
            opts = [
                new LoadOpts('withGroups', 'groups'),
                new LoadOpts('withPermission', 'permission')
            ];
        }
        opts.push(new LoadOpts('withFeatures', 'features'));
        opts.push(new LoadOpts('withIntegrations', 'integrations'));
        opts.forEach((opt) => params = params.append(opt.queryParam, 'true'));

        const state = ctx.getState();
        ctx.setState({
            ...state,
            loading: true,
        });
        return this._http
            .get<Project>('/project/' + action.payload.projectKey, { params })
            .pipe(tap((res: Project) => {
                const proj = state.project;
                let projectUpdated: Project;
                if (action.payload.opts) {
                    projectUpdated = Object.assign({}, proj, res);
                    action.payload.opts.forEach(o => {
                        switch (o.fieldName) {
                            case 'workflow_names':
                                if (!res.workflow_names) {
                                    projectUpdated.workflow_names = [];
                                }
                                break;
                            case 'pipeline_names':
                                if (!res.pipeline_names) {
                                    projectUpdated.pipeline_names = [];
                                }
                                break;
                            case 'application_names':
                                if (!res.application_names) {
                                    projectUpdated.application_names = [];
                                }
                                break;
                            case 'environments':
                                if (!res.environments) {
                                    projectUpdated.environments = [];
                                }
                                break;
                            case 'integrations':
                                if (!res.integrations) {
                                    projectUpdated.integrations = [];
                                }
                                break;
                            case 'keys':
                                if (!res.keys) {
                                    projectUpdated.keys = [];
                                }
                                break;
                            case 'labels':
                                if (!res.labels) {
                                    projectUpdated.labels = [];
                                }
                                break;
                        }
                    });
                } else {
                    projectUpdated = res;
                }

                ctx.dispatch(new ProjectAction.LoadProject(projectUpdated));
            }));
    }

    @Action(ProjectAction.ExternalChangeProject)
    externalChange(ctx: StateContext<ProjectStateModel>, action: ProjectAction.ExternalChangeProject) {
        const state = ctx.getState();
        return ctx.setState({
            ...state,
            project: Object.assign({}, state.project, <Project>{ externalChange: true }),
            loading: false,
        });
    }

    @Action(ProjectAction.DeleteProjectFromCache)
    deleteFromCache(ctx: StateContext<ProjectStateModel>, action: ProjectAction.DeleteProjectFromCache) {
        const state = ctx.getState();
        return ctx.setState({
            ...state,
            project: null,
            loading: false,
        });
    }

    @Action(ProjectAction.AddProject)
    add(ctx: StateContext<ProjectStateModel>, action: ProjectAction.AddProject) {
        const state = ctx.getState();

        ctx.setState({
            ...state,
            loading: true,
        });
        return this._http.post<Project>(
            '/project',
            action.payload
        ).pipe(tap((project) => {
            ctx.setState({
                ...state,
                project,
                loading: false,
            });
            // TODO: dispatch action on global state to add project in list
        }));
    }

    @Action(ProjectAction.UpdateProject)
    update(ctx: StateContext<ProjectStateModel>, action: ProjectAction.UpdateProject) {
        const state = ctx.getState();

        ctx.setState({
            ...state,
            loading: true,
        });
        return this._http.put<Project>(
            '/project/' + action.payload.key,
            action.payload
        ).pipe(tap((project: Project) => {
            ctx.setState({
                ...state,
                project: Object.assign({}, state.project, project),
                loading: false,
            });
            // TODO: dispatch action on global state to update project in list
        }));
    }

    @Action(ProjectAction.DeleteProject)
    delete(ctx: StateContext<ProjectStateModel>, action: ProjectAction.DeleteProject) {
        const state = ctx.getState();

        ctx.setState({
            ...state,
            loading: true,
        });
        return this._http.delete(
            '/project/' + action.payload.projectKey
        ).pipe(tap(() => {
            ctx.setState({
                ...state,
                project: null,
                loading: false,
            });
            // TODO: dispatch action on global state to delete project in list
        }));
    }


    //  ------- Misc --------- //
    @Action(ProjectAction.UpdateFavoriteProject)
    updateFavorite(ctx: StateContext<ProjectStateModel>, action: ProjectAction.UpdateFavoriteProject) {
        const state = ctx.getState();

        return this._http.post(
            '/user/favorite', {
                type: 'project',
                project_key: action.payload.projectKey
            }
        ).pipe(tap(() => {
            ctx.setState({
                ...state,
                project: Object.assign({}, state.project, <Project>{ favorite: !state.project.favorite }),
                loading: false,
            });
            // TODO: dispatch action on global state to update project in list and user state
            // TODO: move this one on user state and just update state here, not XHR
        }));
    }

    //  ------- Application --------- //
    @Action(ProjectAction.AddApplicationInProject)
    addApplication(ctx: StateContext<ProjectStateModel>, action: ProjectAction.AddApplicationInProject) {
        const state = ctx.getState();
        let applications = state.project.applications ? state.project.applications.concat([action.payload]) : [action.payload];
        let application_names = state.project.application_names ? state.project.application_names.concat([]) : [];

        let idName = new IdName();
        idName.id = action.payload.id;
        idName.name = action.payload.name;
        idName.description = action.payload.description;
        idName.icon = action.payload.icon;
        application_names.push(idName);

        return ctx.setState({
            ...state,
            project: Object.assign({}, state.project, { applications, application_names }),
            loading: false,
        });
    }

    @Action(ProjectAction.RenameApplicationInProject)
    renameApplication(ctx: StateContext<ProjectStateModel>, action: ProjectAction.RenameApplicationInProject) {
        const state = ctx.getState();
        let application_names = state.project.application_names ? state.project.application_names.concat([]) : [];

        if (!application_names.length) {
            let idName = new IdName();
            idName.name = action.payload.newAppName;
            application_names = [idName]
        } else {
            application_names = application_names.map((app) => {
                if (app.name === action.payload.previousAppName) {
                    app.name = action.payload.newAppName;
                }
                return app;
            })
        }

        return ctx.setState({
            ...state,
            project: Object.assign({}, state.project, { application_names }),
            loading: false,
        });
    }

    @Action(ProjectAction.DeleteApplicationInProject)
    deleteApplication(ctx: StateContext<ProjectStateModel>, action: ProjectAction.DeleteApplicationInProject) {
        const state = ctx.getState();
        let applications = state.project.applications ? state.project.applications.concat([]) : [];
        let application_names = state.project.application_names ? state.project.application_names.concat([]) : [];

        applications = applications.filter((app) => app.name !== action.payload.applicationName);
        application_names = application_names.filter((app) => app.name !== action.payload.applicationName);

        return ctx.setState({
            ...state,
            project: Object.assign({}, state.project, { applications, application_names }),
            loading: false,
        });
    }

    //  ------- Workflow --------- //
    @Action(ProjectAction.AddWorkflowInProject)
    addWorkflow(ctx: StateContext<ProjectStateModel>, action: ProjectAction.AddWorkflowInProject) {
        const state = ctx.getState();
        let workflows = state.project.workflows ? state.project.workflows.concat([action.payload]) : [action.payload];
        let workflow_names = state.project.workflow_names ? state.project.workflow_names.concat([]) : [];

        let idName = new IdName();
        idName.id = action.payload.id;
        idName.name = action.payload.name;
        idName.description = action.payload.description;
        idName.icon = action.payload.icon;
        if (!workflow_names) {
            workflow_names = [idName]
        } else {
            workflow_names.push(idName);
        }

        return ctx.setState({
            ...state,
            project: Object.assign({}, state.project, { workflows, workflow_names }),
            loading: false,
        });
    }

    @Action(ProjectAction.DeleteWorkflowInProject)
    deleteWorkflow(ctx: StateContext<ProjectStateModel>, action: ProjectAction.DeleteWorkflowInProject) {
        const state = ctx.getState();
        let workflows = state.project.workflows ? state.project.workflows.concat([]) : [];
        let workflow_names = state.project.workflow_names ? state.project.workflow_names.concat([]) : [];

        workflows = workflows.filter((workflow) => workflow.name !== action.payload.workflowName);
        workflow_names = workflow_names.filter((workflow) => workflow.name !== action.payload.workflowName);

        return ctx.setState({
            ...state,
            project: Object.assign({}, state.project, <Project>{ workflows, workflow_names }),
            loading: false,
        });
    }

    //  ------- Pipeline --------- //
    @Action(ProjectAction.AddPipelineInProject)
    addPipeline(ctx: StateContext<ProjectStateModel>, action: ProjectAction.AddPipelineInProject) {
        const state = ctx.getState();
        let pipelines = state.project.pipelines ? state.project.pipelines.concat([action.payload]) : [action.payload];
        let pipeline_names = state.project.pipeline_names ? state.project.pipeline_names.concat([]) : [];

        let idName = new IdName();
        idName.id = action.payload.id;
        idName.name = action.payload.name;
        idName.description = action.payload.description;
        idName.icon = action.payload.icon;
        pipeline_names.push(idName);

        return ctx.setState({
            ...state,
            project: Object.assign({}, state.project, <Project>{ pipelines, pipeline_names }),
            loading: false,
        });
    }

    @Action(ProjectAction.DeletePipelineInProject)
    deletePipeline(ctx: StateContext<ProjectStateModel>, action: ProjectAction.DeletePipelineInProject) {
        const state = ctx.getState();
        let pipelines = state.project.pipelines ? state.project.pipelines.concat([]) : [];
        let pipeline_names = state.project.pipeline_names ? state.project.pipeline_names.concat([]) : [];

        pipelines = pipelines.filter((workflow) => workflow.name !== action.payload.pipelineName);
        pipeline_names = pipeline_names.filter((workflow) => workflow.name !== action.payload.pipelineName);

        return ctx.setState({
            ...state,
            project: Object.assign({}, state.project, <Project>{ pipelines, pipeline_names }),
            loading: false,
        });
    }

}
