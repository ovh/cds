import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Action, createSelector, Selector, State, StateContext } from '@ngxs/store';
import { Environment } from 'app/model/environment.model';
import { GroupPermission } from 'app/model/group.model';
import { IdName, Label, LoadOpts, Project } from 'app/model/project.model';
import { Variable } from 'app/model/variable.model';
import { ProjectService } from 'app/service/project/project.service';
import { cloneDeep } from 'lodash-es';
import { catchError, tap } from 'rxjs/operators';
import * as ProjectAction from './project.action';
import { of } from 'rxjs';
import { Bookmark, BookmarkType } from 'app/model/bookmark.model';

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
@Injectable()
export class ProjectState {

    constructor(
        private _http: HttpClient,
        private _projectService: ProjectService
    ) { }

    @Selector()
    static projectSnapshot(state: ProjectStateModel) {
        return state.project;
    }

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

        return this._projectService.getProject(action.payload.projectKey, [
            ...action.payload.opts,
            new LoadOpts('withLabels', 'labels')
        ]).pipe(
            catchError(() => of(null)),
            tap((res: Project) => {
                if (!res) {
                    ctx.dispatch(new ProjectAction.DeleteProjectFromCache());
                    return;
                }
                ctx.dispatch(new ProjectAction.LoadProject(res));
            })
        );
    }

    @Action(ProjectAction.EnrichProject)
    resync(ctx: StateContext<ProjectStateModel>, action: ProjectAction.EnrichProject) {
        let params = new HttpParams();
        let opts = action.payload.opts;
        opts.push(new LoadOpts('withFeatures', 'features'));
        opts.forEach(opt => params = params.append(opt.queryParam, 'true'));
        const state = ctx.getState();
        ctx.setState({
            ...state,
            loading: true,
        });
        return this._http
            .get<Project>('/project/' + action.payload.projectKey, { params })
            .pipe(tap((res: Project) => {
                let projectUpdated = { ...state.project };
                (action.payload.opts ?? []).forEach(o => {
                    switch (o.fieldName) {
                        case 'workflow_names':
                            projectUpdated.workflow_names = res.workflow_names;
                            break;
                        case 'pipeline_names':
                            projectUpdated.pipeline_names = res.pipeline_names;
                            break;
                        case 'application_names':
                            projectUpdated.application_names = res.application_names;
                            break;
                        case 'environments':
                            projectUpdated.environments = res.environments;
                            break;
                        case 'environment_names':
                            projectUpdated.environment_names = res.environment_names;
                            break;
                        case 'integrations':
                            projectUpdated.integrations = res.integrations;
                            break;
                        case 'keys':
                            projectUpdated.keys = res.keys;
                            break;
                        case 'labels':
                            projectUpdated.labels = res.labels;
                            break;
                    }
                });
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
            currentProjectKey: null
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
        ).pipe(tap((project: Project) => {
            ctx.setState({
                ...state,
                project,
                loading: false,
            });
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
        return this._http.post<Variable>(`/project/${state.project.key}/variable/${action.payload.name}`, action.payload)
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
        let resyncLabels = !action.payload.label.id;
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

            if (resyncLabels) {
                ctx.dispatch(new ProjectAction.EnrichProject({
                    projectKey: state.project.key,
                    opts: [new LoadOpts('withLabels', 'labels')]
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

    //  ------- Environment --------- //
    @Action(ProjectAction.AddEnvironmentInProject)
    addEnvironment(ctx: StateContext<ProjectStateModel>, action: ProjectAction.AddEnvironmentInProject) {
        const state = ctx.getState();
        let environments = state.project.environments ? state.project.environments.concat([action.payload]) : [action.payload];
        let environment_names = state.project.environment_names ? state.project.environment_names.concat([]) : [];

        let idName = new IdName();
        idName.id = action.payload.id;
        idName.name = action.payload.name;
        environment_names.push(idName);

        return ctx.setState({
            ...state,
            project: Object.assign({}, state.project, { environments, environment_names }),
        });
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
        let application_names = cloneDeep(state.project.application_names ? state.project.application_names.concat([]) : []);

        if (!application_names.length) {
            let idName = new IdName();
            idName.name = action.payload.changes.name;
            idName.description = action.payload.changes.description;
            idName.icon = action.payload.changes.icon;
            application_names = [idName];
        } else {
            application_names = application_names.map((app) => {
                if (app.name === action.payload.previousAppName) {
                    app.name = action.payload.changes.name;
                    app.description = action.payload.changes.description;
                    app.icon = action.payload.changes.icon;
                }
                return app;
            });
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
            workflow_names = [idName];
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

        return this._http.post<GroupPermission>('/project/' + action.payload.projectKey + '/group', action.payload.group,
            { params }
        ).pipe(tap((perm: GroupPermission) => {
            let perms = state.project.groups ? state.project.groups : [];
            ctx.setState({
                ...state,
                project: Object.assign({}, state.project, <Project>{ groups: perms.concat(perm) }),
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
        ).pipe(tap((perm: GroupPermission) => {
            let perms = state.project.groups ? state.project.groups.filter((g: GroupPermission) => g.group.id !== perm.group.id) : [];
            ctx.setState({
                ...state,
                project: Object.assign({}, state.project, <Project>{ groups: perms.concat(perm) }),
            });
        }));
    }
}