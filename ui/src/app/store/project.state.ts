import { HttpClient, HttpParams } from '@angular/common/http';
import { Action, createSelector, State, StateContext } from '@ngxs/store';
import { Environment } from 'app/model/environment.model';
import { GroupPermission } from 'app/model/group.model';
import { ProjectIntegration } from 'app/model/integration.model';
import { Key } from 'app/model/keys.model';
import { IdName, Label, LoadOpts, Project } from 'app/model/project.model';
import { Usage } from 'app/model/usage.model';
import { Variable } from 'app/model/variable.model';
import { NavbarService } from 'app/service/navbar/navbar.service';
import { cloneDeep } from 'lodash-es';
import { tap } from 'rxjs/operators';
import * as ProjectAction from './project.action';
import { AddGroupInAllWorkflows } from './workflow.action';

export class ProjectStateModel {
    public project: Project;
    public loading: boolean;
    public currentProjectKey: string;
    public repoManager: { request_token?: string, url?: string, auth_type?: string };
}

@State<ProjectStateModel>({
    name: 'project',
    defaults: {
        project: null,
        loading: true,
        repoManager: {},
        currentProjectKey: null
    }
})
export class ProjectState {

    static selectEnvironment(name: string) {
        return createSelector(
            [ProjectState],
            (state: ProjectStateModel): Environment => {
                if (!state.project || !state.project.environments) {
                    return null;
                }
                return state.project.environments.find((env) => env.name === name);
            }
        );
    }

    constructor(private _http: HttpClient, private _navbarService: NavbarService) { }


    @Action(ProjectAction.LoadProject)
    load(ctx: StateContext<ProjectStateModel>, action: ProjectAction.LoadProject) {
        const state = ctx.getState();
        ctx.setState({
            ...state,
            project: {
                ...state.project,
                ...action.payload
            },
            currentProjectKey: action.payload.key,
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
        opts.push(new LoadOpts('withLabels', 'labels'));
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
                            case 'environment_names':
                                if (!res.environment_names) {
                                    projectUpdated.environment_names = [];
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
        });
    }

    @Action(ProjectAction.DeleteProjectFromCache)
    deleteFromCache(ctx: StateContext<ProjectStateModel>, action: ProjectAction.DeleteProjectFromCache) {
        const state = ctx.getState();
        return ctx.setState({
            ...state,
            project: null,
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

    //  ------- Variable --------- //
    @Action(ProjectAction.FetchVariablesInProject)
    fetchVariable(ctx: StateContext<ProjectStateModel>, action: ProjectAction.FetchVariablesInProject) {
        const state = ctx.getState();

        if (state.currentProjectKey && state.currentProjectKey === action.payload.projectKey &&
            state.project && state.project.key && state.project.variables) {
            return ctx.dispatch(new ProjectAction.LoadProject(state.project));
        }
        if (state.currentProjectKey && state.currentProjectKey !== action.payload.projectKey) {
            ctx.dispatch(new ProjectAction.FetchProject({ projectKey: action.payload.projectKey, opts: [] }));
        }

        return ctx.dispatch(new ProjectAction.ResyncVariablesInProject(action.payload));
    }

    @Action(ProjectAction.LoadVariablesInProject)
    loadVariable(ctx: StateContext<ProjectStateModel>, action: ProjectAction.LoadVariablesInProject) {
        const state = ctx.getState();
        ctx.setState({
            ...state,
            project: Object.assign({}, state.project, <Project>{ variables: action.payload }),
        });
    }

    @Action(ProjectAction.ResyncVariablesInProject)
    resyncVariable(ctx: StateContext<ProjectStateModel>, action: ProjectAction.ResyncVariablesInProject) {
        return this._http
            .get<Variable[]>(`/project/${action.payload.projectKey}/variable`)
            .pipe(tap((variables: Variable[]) => {
                ctx.dispatch(new ProjectAction.LoadVariablesInProject(variables));
            }));
    }

    @Action(ProjectAction.AddVariableInProject)
    addVariable(ctx: StateContext<ProjectStateModel>, action: ProjectAction.AddVariableInProject) {
        const state = ctx.getState();
        return this._http.post<Variable>('/project/' + state.project.key + '/variable/' + action.payload.name, action.payload)
            .pipe(tap((v: Variable) => {
                let p = cloneDeep(state.project);
                if (!p.variables) {
                    p.variables = new Array<Variable>();
                }
                p.variables.push(v);
                ctx.setState({
                    ...state,
                    project: p,
                });
            }));
    }

    @Action(ProjectAction.UpdateVariableInProject)
    updateVariable(ctx: StateContext<ProjectStateModel>, action: ProjectAction.UpdateVariableInProject) {
        const state = ctx.getState();

        return this._http
            .put<Variable>('/project/' + state.project.key + '/variable/' + action.payload.variableName, action.payload.changes)
            .pipe(tap((variableRes: Variable) => {
                let variables = state.project.variables ? state.project.variables.concat([]) : [];
                variables = variables.map((variable) => {
                    if (variable.id === action.payload.changes.id) {
                        variable = variableRes;
                    }
                    return variable;
                });

                ctx.setState({
                    ...state,
                    project: Object.assign({}, state.project, <Project>{ variables }),
                });
            }));
    }

    @Action(ProjectAction.DeleteVariableInProject)
    deleteVariable(ctx: StateContext<ProjectStateModel>, action: ProjectAction.DeleteVariableInProject) {
        const state = ctx.getState();
        return this._http
            .delete('/project/' + state.project.key + '/variable/' + action.payload.name)
            .pipe(tap(() => {
                let variables = state.project.variables ? state.project.variables.concat([]) : [];
                variables = variables.filter((variable) => variable.name !== action.payload.name);

                ctx.setState({
                    ...state,
                    project: Object.assign({}, state.project, <Project>{ variables }),
                });
            }));
    }


    //  ------- Label --------- //
    @Action(ProjectAction.SaveLabelsInProject)
    saveLabels(ctx: StateContext<ProjectStateModel>, action: ProjectAction.SaveLabelsInProject) {
        const state = ctx.getState();
        return this._http.put<Project>(
            '/project/' + action.payload.projectKey + '/labels',
            action.payload.labels
        ).pipe(tap((proj: Project) => {
            ctx.setState({
                ...state,
                project: Object.assign({}, state.project, <Project>{
                    workflow_names: proj.workflow_names,
                    labels: proj.labels
                }),
            });
        }));
    }

    @Action(ProjectAction.AddLabelWorkflowInProject)
    addLabelWorkflow(ctx: StateContext<ProjectStateModel>, action: ProjectAction.AddLabelWorkflowInProject) {
        // check if we will a to resync project to get new labels
        let resyncProject = !action.payload.label.id;
        const state = ctx.getState();
        return this._http.post<Label>(
            `/project/${state.project.key}/workflows/${action.payload.workflowName}/label`,
            action.payload.label
        ).pipe(tap((label: Label) => {
            let workflow_names = state.project.workflow_names ? state.project.workflow_names.concat([]) : [];
            workflow_names = workflow_names.map((wf) => {
                if (action.payload.workflowName === wf.name) {
                    let workflow: IdName;
                    if (wf.labels) {
                        workflow = Object.assign({}, wf, { labels: wf.labels.concat(label) });
                    } else {
                        workflow = Object.assign({}, wf, {
                            labels: [label]
                        });
                    }
                    return workflow;
                }
                return wf;
            });

            ctx.setState({
                ...state,
                project: Object.assign({}, state.project, <Project>{ workflow_names }),
            });

            if (resyncProject) {
                ctx.dispatch(new  ProjectAction.ResyncProject({
                    projectKey: state.project.key,
                    opts: []
                }));
            }
        }));
    }

    @Action(ProjectAction.DeleteLabelWorkflowInProject)
    deleteLabelWorkflow(ctx: StateContext<ProjectStateModel>, action: ProjectAction.DeleteLabelWorkflowInProject) {
        const state = ctx.getState();

        return this._http.delete<null>(
            `/project/${state.project.key}/workflows/${action.payload.workflowName}/label/${action.payload.labelId}`
        ).pipe(tap(() => {

            let workflow_names = state.project.workflow_names ? state.project.workflow_names.concat([]) : [];
            workflow_names = workflow_names.map((wf) => {
                if (action.payload.workflowName === wf.name) {
                    let workflow: IdName;
                    if (wf.labels) {
                        workflow = Object.assign({}, wf, { labels: wf.labels.filter((label) => label.id !== action.payload.labelId) });
                    } else {
                        workflow = Object.assign({}, wf, { labels: [] });
                    }
                    return workflow;
                }
                return wf;
            });

            ctx.setState({
                ...state,
                project: Object.assign({}, state.project, <Project>{ workflow_names }),
            });
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
            this._navbarService.getData(); // TODO: to delete
            if (state.project && state.project.key) {
                ctx.setState({
                    ...state,
                    project: Object.assign({}, state.project, <Project>{ favorite: !state.project.favorite }),
                });
            }
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
        });
    }

    @Action(ProjectAction.UpdateApplicationInProject)
    updateApplication(ctx: StateContext<ProjectStateModel>, action: ProjectAction.UpdateApplicationInProject) {
        const state = ctx.getState();
        let application_names = state.project.application_names ? state.project.application_names.concat([]) : [];

        if (!application_names.length) {
            let idName = new IdName();
            idName.name = action.payload.changes.name;
            idName.description = action.payload.changes.description;
            idName.icon = action.payload.changes.icon;
            application_names = [idName]
        } else {
            application_names = application_names.map((app) => {
                if (app.name === action.payload.previousAppName) {
                    app.name = action.payload.changes.name;
                    app.description = action.payload.changes.description;
                    app.icon = action.payload.changes.icon;
                }
                return app;
            })
        }

        return ctx.setState({
            ...state,
            project: Object.assign({}, state.project, { application_names }),
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
        idName.labels = action.payload.labels;
        if (!workflow_names) {
            workflow_names = [idName]
        } else {
            workflow_names.push(idName);
        }

        return ctx.setState({
            ...state,
            project: Object.assign({}, state.project, { workflows, workflow_names }),
        });
    }

    @Action(ProjectAction.UpdateWorkflowInProject)
    updateWorkflow(ctx: StateContext<ProjectStateModel>, action: ProjectAction.UpdateWorkflowInProject) {
        const state = ctx.getState();
        let workflows = state.project.workflows ? state.project.workflows.concat([]) : [];
        let workflow_names = state.project.workflow_names ? state.project.workflow_names.concat([]) : [];

        workflows = workflows.map((workflow) => {
            if (workflow.name === action.payload.previousWorkflowName) {
                return action.payload.changes;
            }
            return workflow;
        });
        workflow_names = workflow_names.map((workflow) => {
            if (workflow.name === action.payload.previousWorkflowName) {
                return Object.assign({}, workflow, <IdName>{
                    name: action.payload.changes.name,
                    description: action.payload.changes.description,
                    icon: action.payload.changes.icon
                });
            }
            return workflow;
        });

        return ctx.setState({
            ...state,
            project: Object.assign({}, state.project, <Project>{ workflows, workflow_names }),
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
        });
    }

    @Action(ProjectAction.UpdatePipelineInProject)
    updatePipeline(ctx: StateContext<ProjectStateModel>, action: ProjectAction.UpdatePipelineInProject) {
        const state = ctx.getState();
        let pipelines = state.project.pipelines ? state.project.pipelines.concat([]) : [];
        let pipeline_names = state.project.pipeline_names ? state.project.pipeline_names.concat([]) : [];

        pipelines = pipelines.map((pip) => {
            if (pip.name === action.payload.previousPipName) {
                return Object.assign({}, pip, action.payload.changes);
            }
            return pip;
        });
        pipeline_names = pipeline_names.map((pip) => {
            if (pip.name === action.payload.previousPipName) {
                return Object.assign({}, pip, <IdName>{
                    name: action.payload.changes.name,
                    description: action.payload.changes.description
                });
            }
            return pip;
        });

        return ctx.setState({
            ...state,
            project: Object.assign({}, state.project, <Project>{ pipelines, pipeline_names }),
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
        });
    }

    //  ------- Group Permission --------- //
    @Action(ProjectAction.AddGroupInProject)
    addGroup(ctx: StateContext<ProjectStateModel>, action: ProjectAction.AddGroupInProject) {
        const state = ctx.getState();
        let params = new HttpParams();
        if (action.payload.onlyProject) {
            params = params.append('onlyProject', 'true');
        }

        return this._http.post<GroupPermission[]>('/project/' + action.payload.projectKey + '/group', action.payload.group,
            { params }
        ).pipe(tap((groups: GroupPermission[]) => {
            if (!action.payload.onlyProject) {
                ctx.dispatch(new AddGroupInAllWorkflows({
                    projectKey: action.payload.projectKey,
                    group: action.payload.group
                }));
            }

            ctx.setState({
                ...state,
                project: Object.assign({}, state.project, <Project>{ groups }),
            });
        }));
    }

    @Action(ProjectAction.DeleteGroupInProject)
    deleteGroup(ctx: StateContext<ProjectStateModel>, action: ProjectAction.DeleteGroupInProject) {
        const state = ctx.getState();
        return this._http.delete('/project/' + action.payload.projectKey + '/group/' + action.payload.group.group.name)
            .pipe(tap(() => {
                let groups = state.project.groups ? state.project.groups.concat([]) : [];
                groups = groups.filter((group) => group.group.name !== action.payload.group.group.name);

                ctx.setState({
                    ...state,
                    project: Object.assign({}, state.project, <Project>{ groups }),
                });
            }));
    }

    @Action(ProjectAction.UpdateGroupInProject)
    updateGroup(ctx: StateContext<ProjectStateModel>, action: ProjectAction.UpdateGroupInProject) {
        const state = ctx.getState();
        return this._http.put<GroupPermission>(
            '/project/' + action.payload.projectKey + '/group/' + action.payload.group.group.name,
            action.payload.group
        ).pipe(tap((group) => {
            let groups = state.project.groups ? state.project.groups.concat([]) : [];
            groups = groups.map((gr) => {
                if (gr.group.name === group.group.name) {
                    return group;
                }
                return gr;
            });

            ctx.setState({
                ...state,
                project: Object.assign({}, state.project, <Project>{ groups }),
            });
        }));
    }

    //  ------- Key --------- //
    @Action(ProjectAction.FetchKeysInProject)
    fetchKeys(ctx: StateContext<ProjectStateModel>, action: ProjectAction.FetchKeysInProject) {
        const state = ctx.getState();

        if (state.currentProjectKey && state.currentProjectKey === action.payload.projectKey &&
            state.project && state.project.key && state.project.keys) {
            return ctx.dispatch(new ProjectAction.LoadProject(state.project));
        }
        if (state.currentProjectKey && state.currentProjectKey !== action.payload.projectKey) {
            ctx.dispatch(new ProjectAction.FetchProject({ projectKey: action.payload.projectKey, opts: [] }));
        }

        return ctx.dispatch(new ProjectAction.ResyncKeysInProject(action.payload));
    }

    @Action(ProjectAction.LoadKeysInProject)
    loadKeys(ctx: StateContext<ProjectStateModel>, action: ProjectAction.LoadKeysInProject) {
        const state = ctx.getState();
        ctx.setState({
            ...state,
            project: Object.assign({}, state.project, <Project>{ keys: action.payload }),
        });
    }

    @Action(ProjectAction.ResyncKeysInProject)
    resyncKeys(ctx: StateContext<ProjectStateModel>, action: ProjectAction.ResyncKeysInProject) {
        return this._http
            .get<Key[]>(`/project/${action.payload.projectKey}/keys`)
            .pipe(tap((keys: Key[]) => {
                ctx.dispatch(new ProjectAction.LoadKeysInProject(keys));
            }));
    }

    @Action(ProjectAction.AddKeyInProject)
    addKey(ctx: StateContext<ProjectStateModel>, action: ProjectAction.AddKeyInProject) {
        const state = ctx.getState();
        return this._http.post<Key>('/project/' + action.payload.projectKey + '/keys', action.payload.key)
            .pipe(tap((key: Key) => {
                let keys = state.project.keys ? state.project.keys.concat([key]) : [key];
                ctx.setState({
                    ...state,
                    project: Object.assign({}, state.project, <Project>{ keys }),
                });
            }));
    }

    @Action(ProjectAction.DeleteKeyInProject)
    deleteKey(ctx: StateContext<ProjectStateModel>, action: ProjectAction.DeleteKeyInProject) {
        const state = ctx.getState();
        return this._http.delete('/project/' + action.payload.projectKey + '/keys/' + action.payload.key.name)
            .pipe(tap(() => {
                let keys = state.project.keys ? state.project.keys.concat([]) : [];
                keys = keys.filter((key) => key.name !== action.payload.key.name);

                ctx.setState({
                    ...state,
                    project: Object.assign({}, state.project, <Project>{ keys }),
                });
            }));
    }

    //  ------- Integration --------- //
    @Action(ProjectAction.FetchIntegrationsInProject)
    fetchIntegrations(ctx: StateContext<ProjectStateModel>, action: ProjectAction.FetchIntegrationsInProject) {
        const state = ctx.getState();

        if (state.currentProjectKey && state.currentProjectKey === action.payload.projectKey &&
            state.project && state.project.key && state.project.keys) {
            return ctx.dispatch(new ProjectAction.LoadProject(state.project));
        }
        if (state.currentProjectKey && state.currentProjectKey !== action.payload.projectKey) {
            ctx.dispatch(new ProjectAction.FetchProject({ projectKey: action.payload.projectKey, opts: [] }));
        }

        return ctx.dispatch(new ProjectAction.ResyncIntegrationsInProject(action.payload));
    }

    @Action(ProjectAction.LoadIntegrationsInProject)
    loadIntegrations(ctx: StateContext<ProjectStateModel>, action: ProjectAction.LoadIntegrationsInProject) {
        const state = ctx.getState();
        ctx.setState({
            ...state,
            project: Object.assign({}, state.project, <Project>{ integrations: action.payload }),
        });
    }

    @Action(ProjectAction.ResyncIntegrationsInProject)
    resyncIntegrations(ctx: StateContext<ProjectStateModel>, action: ProjectAction.ResyncIntegrationsInProject) {
        return this._http
            .get<ProjectIntegration[]>(`/project/${action.payload.projectKey}/integrations`)
            .pipe(tap((integrations: ProjectIntegration[]) => {
                ctx.dispatch(new ProjectAction.LoadIntegrationsInProject(integrations));
            }));
    }

    @Action(ProjectAction.AddIntegrationInProject)
    addIntegration(ctx: StateContext<ProjectStateModel>, action: ProjectAction.AddIntegrationInProject) {
        const state = ctx.getState();
        return this._http.post<ProjectIntegration>('/project/' + action.payload.projectKey + '/integrations', action.payload.integration)
            .pipe(tap((integration: ProjectIntegration) => {
                let integrations = state.project.integrations ? state.project.integrations.concat([integration]) : [integration];
                ctx.setState({
                    ...state,
                    project: Object.assign({}, state.project, <Project>{ integrations }),
                });
            }));
    }

    @Action(ProjectAction.DeleteIntegrationInProject)
    deleteIntegration(ctx: StateContext<ProjectStateModel>, action: ProjectAction.DeleteIntegrationInProject) {
        const state = ctx.getState();
        return this._http.delete('/project/' + action.payload.projectKey + '/integrations/' + action.payload.integration.name)
            .pipe(tap(() => {
                let integrations = state.project.integrations ? state.project.integrations.concat([]) : [];
                integrations = integrations.filter((integration) => integration.name !== action.payload.integration.name);

                ctx.setState({
                    ...state,
                    project: Object.assign({}, state.project, <Project>{ integrations }),
                });
            }));
    }

    @Action(ProjectAction.UpdateIntegrationInProject)
    updateIntegration(ctx: StateContext<ProjectStateModel>, action: ProjectAction.UpdateIntegrationInProject) {
        const state = ctx.getState();
        return this._http.put<ProjectIntegration>(
            '/project/' + action.payload.projectKey + '/integrations/' + action.payload.integrationName,
            action.payload.changes
        ).pipe(tap((integration) => {
            let integrations = state.project.integrations ? state.project.integrations.concat([]) : [];
            integrations = integrations.map((integ) => {
                if (integ.name === integration.name) {
                    return integration;
                }
                return integ;
            });

            ctx.setState({
                ...state,
                project: Object.assign({}, state.project, <Project>{ integrations }),
            });
        }));
    }

    //  ------- Environment --------- //
    @Action(ProjectAction.FetchEnvironmentInProject)
    fetchEnvironment(ctx: StateContext<ProjectStateModel>, action: ProjectAction.FetchEnvironmentInProject) {
        const state = ctx.getState();

        if (state.currentProjectKey && state.currentProjectKey !== action.payload.projectKey) {
            ctx.dispatch(new ProjectAction.FetchProject({ projectKey: action.payload.projectKey, opts: [] }));
        }
        let params = new HttpParams();
        params = params.append('withUsage', 'true');

        return this._http
            .get<Environment>(`/project/${action.payload.projectKey}/environment/${action.payload.envName}`, { params })
            .pipe(tap((environment: Environment) => {
                let envs = state.project.environments;
                if (Array.isArray(envs)) {
                    envs = envs.map((env) => {
                        if (env.name === action.payload.envName) {
                            return environment;
                        }
                        return env;
                    })
                } else {
                    envs = [environment];
                }
                ctx.setState(<ProjectStateModel>{
                    ...state,
                    project: {
                        ...state.project,
                        environments: envs,
                    }
                });
            }));
    }

    @Action(ProjectAction.AddEnvironmentKey)
    addEnvironmentKey(ctx: StateContext<ProjectStateModel>, action: ProjectAction.AddEnvironmentKey) {
        const state = ctx.getState();
        return this._http.post<Key>(`/project/${action.payload.projectKey}/environment/${action.payload.envName}/keys`, action.payload.key)
            .pipe(tap((key: Key) => {
                let envs = state.project.environments;
                if (Array.isArray(envs)) {
                    envs = envs.map((env) => {
                        if (env.name === action.payload.envName) {
                            return { ...env, keys: [key].concat(env.keys) };
                        }
                        return env;
                    })
                }
                ctx.setState(<ProjectStateModel>{
                    ...state,
                    project: {
                        ...state.project,
                        environments: envs,
                    }
                });
            }));
    }

    @Action(ProjectAction.DeleteEnvironmentKey)
    deleteEnvironmentKey(ctx: StateContext<ProjectStateModel>, action: ProjectAction.DeleteEnvironmentKey) {
        const state = ctx.getState();
        return this._http.delete<null>('/project/' + action.payload.projectKey +
            '/environment/' + action.payload.envName + '/keys/' + action.payload.key.name)
            .pipe(tap(() => {
                let envs = state.project.environments;
                if (Array.isArray(envs)) {
                    envs = envs.map((env) => {
                        if (env.name === action.payload.envName) {
                            return { ...env, keys: env.keys.filter((key) => key.name === action.payload.key.name) };
                        }
                        return env;
                    })
                }
                ctx.setState(<ProjectStateModel>{
                    ...state,
                    project: {
                        ...state.project,
                        environments: envs,
                    }
                });
            }));
    }

    @Action(ProjectAction.FetchEnvironmentsInProject)
    fetchEnvironments(ctx: StateContext<ProjectStateModel>, action: ProjectAction.FetchEnvironmentsInProject) {
        const state = ctx.getState();

        if (state.currentProjectKey && state.currentProjectKey === action.payload.projectKey &&
            state.project && state.project.key && state.project.environments) {
            return ctx.dispatch(new ProjectAction.LoadProject(state.project));
        }
        if (state.currentProjectKey && state.currentProjectKey !== action.payload.projectKey) {
            ctx.dispatch(new ProjectAction.FetchProject({ projectKey: action.payload.projectKey, opts: [] }));
        }

        return ctx.dispatch(new ProjectAction.ResyncEnvironmentsInProject(action.payload));
    }

    @Action(ProjectAction.LoadEnvironmentsInProject)
    loadEnvironments(ctx: StateContext<ProjectStateModel>, action: ProjectAction.LoadEnvironmentsInProject) {
        const state = ctx.getState();
        ctx.setState({
            ...state,
            project: Object.assign({}, state.project, <Project>{ environments: action.payload }),
        });
    }

    @Action(ProjectAction.ResyncEnvironmentsInProject)
    resyncEnvironments(ctx: StateContext<ProjectStateModel>, action: ProjectAction.ResyncEnvironmentsInProject) {
        let params = new HttpParams();
        params = params.append('withUsage', 'true');
        return this._http
            .get<Environment[]>(`/project/${action.payload.projectKey}/environment`, { params })
            .pipe(tap((environments: Environment[]) => {
                ctx.dispatch(new ProjectAction.LoadEnvironmentsInProject(environments));
            }));
    }

    @Action(ProjectAction.AddEnvironmentInProject)
    addEnvironment(ctx: StateContext<ProjectStateModel>, action: ProjectAction.AddEnvironmentInProject) {
        return this._http.post<Project>('/project/' + action.payload.projectKey + '/environment', action.payload.environment)
            .pipe(tap((project: Project) => ctx.dispatch(new ProjectAction.LoadEnvironmentsInProject(project.environments))));
    }

    @Action(ProjectAction.CloneEnvironmentInProject)
    cloneEnvironment(ctx: StateContext<ProjectStateModel>, action: ProjectAction.CloneEnvironmentInProject) {
        return this._http.post<Project>(
            '/project/' + action.payload.projectKey + '/environment/' +
            action.payload.environment.name + '/clone/' + action.payload.cloneName,
            null
        ).pipe(tap((project: Project) => ctx.dispatch(new ProjectAction.LoadEnvironmentsInProject(project.environments))));
    }

    @Action(ProjectAction.DeleteEnvironmentInProject)
    deleteEnvironment(ctx: StateContext<ProjectStateModel>, action: ProjectAction.DeleteEnvironmentInProject) {
        return this._http.delete<Project>('/project/' + action.payload.projectKey + '/environment/' + action.payload.environment.name)
            .pipe(tap((project: Project) => ctx.dispatch(new ProjectAction.LoadEnvironmentsInProject(project.environments))));
    }

    @Action(ProjectAction.UpdateEnvironmentInProject)
    updateEnvironment(ctx: StateContext<ProjectStateModel>, action: ProjectAction.UpdateEnvironmentInProject) {
        return this._http.put<Project>(
            '/project/' + action.payload.projectKey + '/environment/' + action.payload.environmentName,
            action.payload.changes
        ).pipe(tap((project: Project) => {
            ctx.dispatch(new ProjectAction.LoadEnvironmentsInProject(project.environments));
        }));
    }

    @Action(ProjectAction.AddEnvironmentVariableInProject)
    addEnvironmentVariable(ctx: StateContext<ProjectStateModel>, action: ProjectAction.AddEnvironmentVariableInProject) {
        return this._http.post<Variable>(
            '/project/' + action.payload.projectKey + '/environment/' +
            action.payload.environmentName + '/variable/' + action.payload.variable.name,
            action.payload.variable
        ).pipe(tap((v: Variable) => {
            const state = ctx.getState();
            let proj = cloneDeep(state.project);
            let env = proj.environments.find(e => e.name === action.payload.environmentName);
            if (!env) {
                return;
            }
            if (!env.variables) {
                env.variables = new Array<Variable>();
            }
            env.variables.push(v);
            ctx.setState({
                ...state,
                project: proj,
            });
        }));
    }

    @Action(ProjectAction.DeleteEnvironmentVariableInProject)
    deleteEnvironmentVariable(ctx: StateContext<ProjectStateModel>, action: ProjectAction.DeleteEnvironmentVariableInProject) {
        return this._http.delete<Variable>(
            '/project/' + action.payload.projectKey + '/environment/' +
            action.payload.environmentName + '/variable/' + action.payload.variable.name
        ).pipe(tap((v: Variable) => {
            const state = ctx.getState();
            let proj = cloneDeep(state.project);
            let env = proj.environments.find(e => e.name === action.payload.environmentName);
            if (!env) {
                return;
            }
            env.variables = env.variables.filter(va => va.name !== action.payload.variable.name);
            ctx.setState({
                ...state,
                project: proj,
            });
        }));
    }

    @Action(ProjectAction.UpdateEnvironmentVariableInProject)
    updateEnvironmentVariable(ctx: StateContext<ProjectStateModel>, action: ProjectAction.UpdateEnvironmentVariableInProject) {
        return this._http.put<Variable>(
            '/project/' + action.payload.projectKey + '/environment/' +
            action.payload.environmentName + '/variable/' + action.payload.variableName,
            action.payload.changes
        ).pipe(tap((v: Variable) => {
            const state = ctx.getState();
            let proj = cloneDeep(state.project);
            let env = proj.environments.find(e => e.name === action.payload.environmentName);
            if (!env) {
                return;
            }
            env.variables = env.variables.map(va => {
                if (va.name !== action.payload.variableName) {
                    return va;
                }
                return v;
            });
            ctx.setState({
                ...state,
                project: proj,
            })
        }));
    }

    @Action(ProjectAction.FetchEnvironmentUsageInProject)
    fetchEnvironmentUsage(ctx: StateContext<ProjectStateModel>, action: ProjectAction.FetchEnvironmentUsageInProject) {
        return this._http
            .get<Usage>(`/project/${action.payload.projectKey}/environment/${action.payload.environmentName}/usage`)
            .pipe(tap((usage: Usage) => {
                const state = ctx.getState();
                const environments = state.project.environments.map((env) => {
                    if (env.name === action.payload.environmentName) {
                        return { ...env, usage };
                    }
                    return env;
                });
                return ctx.dispatch(new ProjectAction.LoadEnvironmentsInProject(environments));
            }));
    }

    //  ------- Repository Manager --------- //
    @Action(ProjectAction.ConnectRepositoryManagerInProject)
    connectRepositoryManager(ctx: StateContext<ProjectStateModel>, action: ProjectAction.ConnectRepositoryManagerInProject) {
        const state = ctx.getState();
        return this._http.post<{ request_token: string, url: string, auth_type: string }>(
            '/project/' + action.payload.projectKey + '/repositories_manager/' +
            action.payload.repoManager + '/authorize',
            null
        ).pipe(tap(({ request_token, url, auth_type }) => {
            ctx.setState({
                ...state,
                repoManager: {
                    request_token,
                    url,
                    auth_type
                },
            });
        }));
    }

    @Action(ProjectAction.CallbackRepositoryManagerBasicAuthInProject)
    callbackRepositoryManagerBasicAuth(ctx: StateContext<ProjectStateModel>,
        action: ProjectAction.CallbackRepositoryManagerBasicAuthInProject) {
        const state = ctx.getState();
        let data = {
            'username': action.payload.basicUser,
            'secret': action.payload.basicPassword
        };
        return this._http.post<Project>(
            '/project/' + action.payload.projectKey + '/repositories_manager/' +
            action.payload.repoManager + '/authorize/basicauth',
            data
        ).pipe(tap((project: Project) => {
            ctx.setState({
                ...state,
                project: Object.assign({}, state.project, <Project>{ vcs_servers: project.vcs_servers }),
            });
        }));
    }

    @Action(ProjectAction.CallbackRepositoryManagerInProject)
    callbackRepositoryManager(ctx: StateContext<ProjectStateModel>, action: ProjectAction.CallbackRepositoryManagerInProject) {
        const state = ctx.getState();
        let data = {
            'request_token': action.payload.requestToken,
            'verifier': action.payload.code
        };
        return this._http.post<Project>(
            '/project/' + action.payload.projectKey + '/repositories_manager/' +
            action.payload.repoManager + '/authorize/callback',
            data
        ).pipe(tap((project: Project) => {
            ctx.setState({
                ...state,
                project: Object.assign({}, state.project, <Project>{ vcs_servers: project.vcs_servers }),
            });
        }));
    }

    @Action(ProjectAction.DisconnectRepositoryManagerInProject)
    disconnectRepositoryManager(ctx: StateContext<ProjectStateModel>, action: ProjectAction.DisconnectRepositoryManagerInProject) {
        const state = ctx.getState();
        return this._http.delete<Project>(
            '/project/' + action.payload.projectKey + '/repositories_manager/' +
            action.payload.repoManager
        ).pipe(tap((project: Project) => {
            ctx.setState({
                ...state,
                project: Object.assign({}, state.project, <Project>{ vcs_servers: project.vcs_servers }),
            });
        }));
    }
}
