import { HttpClient, HttpHeaders, HttpParams } from '@angular/common/http';
import { Action, createSelector, State, StateContext } from '@ngxs/store';
import { AuditWorkflow } from 'app/model/audit.model';
import { Workflow } from 'app/model/workflow.model';
import { tap } from 'rxjs/operators';
import * as ActionProject from './project.action';
import * as actionWorkflow from './workflows.action';

export class WorkflowsStateModel {
    public workflows: { [key: string]: Workflow };
    public currentProjectKey: string;
    public loading: boolean;
}

export function getInitialWorkflowsState(): WorkflowsStateModel {
    return {
        workflows: {},
        currentProjectKey: null,
        loading: true,
    };
}

@State<WorkflowsStateModel>({
    name: 'workflows',
    defaults: getInitialWorkflowsState()
})
export class WorkflowsState {

    static selectWorkflow(projectKey: string, workflowName: string) {
        return createSelector(
            [WorkflowsState],
            (state: WorkflowsStateModel): Workflow => state.workflows[projectKey + '/' + workflowName]
        );
    }

    constructor(private _http: HttpClient) { }

    @Action(actionWorkflow.AddWorkflow)
    add(ctx: StateContext<WorkflowsStateModel>, action: actionWorkflow.AddWorkflow) {
        const state = ctx.getState();
        let wfKey = `${action.payload.projectKey}/${action.payload.workflow.name}`;
        let workflows = state.workflows;

        // Refresh when change project
        if (state.currentProjectKey !== action.payload.projectKey) {
            workflows = {};
        }

        return this._http.post<Workflow>(
            `/project/${action.payload.projectKey}/workflow`,
            action.payload.workflow
        ).pipe(tap((wf) => {
            ctx.setState({
                ...state,
                currentProjectKey: action.payload.projectKey,
                workflows: Object.assign({}, workflows, { [wfKey]: wf }),
                loading: false,
            });
            ctx.dispatch(new ActionProject.AddWorkflowInProject(wf));
        }));
    }

    @Action(actionWorkflow.ImportWorkflow)
    import(ctx: StateContext<WorkflowsStateModel>, action: actionWorkflow.ImportWorkflow) {
        const state = ctx.getState();

        // Refresh when change project
        if (state.currentProjectKey !== action.payload.projectKey) {
            ctx.setState({
                ...state,
                currentProjectKey: action.payload.projectKey,
                workflows: {},
                loading: false,
            });
        }

        let headers = new HttpHeaders();
        headers = headers.append('Content-Type', 'application/x-yaml');
        let params = new HttpParams();
        params = params.append('format', 'yaml');

        let request = this._http.post<Array<string>>(
            `/project/${action.payload.projectKey}/import/workflow`,
            action.payload.workflowCode,
            { headers, params }
        );
        if (action.payload.wfName) {
            request = this._http.put<Array<string>>(
                `/project/${action.payload.projectKey}/import/workflow/${action.payload.wfName}`,
                action.payload.workflowCode,
                { headers, params }
            );
        }

        return request.pipe(tap(() => {
            if (action.payload.wfName) {
                return ctx.dispatch(new actionWorkflow.ResyncWorkflow({
                    projectKey: action.payload.projectKey,
                    workflowName: action.payload.wfName
                }));
            }
        }));

    }

    @Action(actionWorkflow.LoadWorkflow)
    load(ctx: StateContext<WorkflowsStateModel>, action: actionWorkflow.LoadWorkflow) {
        const state = ctx.getState();
        let wfKey = `${action.payload.projectKey}/${action.payload.workflow.name}`;
        let workflows = state.workflows;

        // Refresh when change project
        if (state.currentProjectKey !== action.payload.projectKey) {
            workflows = {};
        }

        ctx.setState({
            ...state,
            currentProjectKey: action.payload.projectKey,
            workflows: Object.assign({}, workflows, { [wfKey]: action.payload.workflow }),
            loading: false,
        });
    }

    @Action(actionWorkflow.FetchWorkflow)
    fetch(ctx: StateContext<WorkflowsStateModel>, action: actionWorkflow.FetchWorkflow) {
        const state = ctx.getState();
        const wfKey = action.payload.projectKey + '/' + action.payload.workflowName;

        if (state.workflows[wfKey]) {
            return ctx.dispatch(new actionWorkflow.LoadWorkflow({
                projectKey: action.payload.projectKey,
                workflow: state.workflows[wfKey]
            }));
        }

        return ctx.dispatch(new actionWorkflow.ResyncWorkflow({ ...action.payload }));
    }

    @Action(actionWorkflow.UpdateWorkflow)
    update(ctx: StateContext<WorkflowsStateModel>, action: actionWorkflow.UpdateWorkflow) {
        return this._http.put<Workflow>(
            `/project/${action.payload.projectKey}/workflow/${action.payload.workflowName}`,
            action.payload.changes
        ).pipe(tap((wf) => {
            const state = ctx.getState();

            let wfKey = action.payload.projectKey + '/' + action.payload.workflowName;
            if (wf.name !== action.payload.workflowName) {
                let workflows = Object.assign({}, state.workflows, {
                    [action.payload.projectKey + '/' + wf.name]: wf,
                });
                delete workflows[wfKey];

                ctx.setState({
                    ...state,
                    workflows,
                });
                ctx.dispatch(new ActionProject.UpdateWorkflowInProject({
                    previousWorkflowName: action.payload.workflowName,
                    changes: wf
                }));
            } else {
                let wfUpdated: Workflow = {
                    ...state.workflows[wfKey],
                    ...wf,
                    preview: null
                };

                ctx.setState({
                    ...state,
                    workflows: Object.assign({}, state.workflows, { [wfKey]: wfUpdated }),
                });
            }
        }));
    }

    @Action(actionWorkflow.DeleteWorkflow)
    delete(ctx: StateContext<WorkflowsStateModel>, action: actionWorkflow.DeleteWorkflow) {
        return this._http.delete(
            `/project/${action.payload.projectKey}/workflow/${action.payload.workflowName}`
        ).pipe(tap(() => {
            const state = ctx.getState();
            let wfKey = action.payload.projectKey + '/' + action.payload.workflowName;
            let workflows = Object.assign({}, state.workflows);
            delete workflows[wfKey];

            ctx.setState({
                ...state,
                workflows
            });

            ctx.dispatch(new ActionProject.DeleteWorkflowInProject({ workflowName: action.payload.workflowName }));
        }));
    }

    //  ------- Audit --------- //
    @Action(actionWorkflow.FetchWorkflowAudits)
    fetchAudits(ctx: StateContext<WorkflowsStateModel>, action: actionWorkflow.FetchWorkflowAudits) {
        return this._http.get<AuditWorkflow[]>(
            `/project/${action.payload.projectKey}/workflow/${action.payload.workflowName}/audits`
        ).pipe(tap((audits: AuditWorkflow[]) => {
            const state = ctx.getState();
            let wfKey = action.payload.projectKey + '/' + action.payload.workflowName;

            ctx.dispatch(new actionWorkflow.LoadWorkflow({
                projectKey: action.payload.projectKey,
                workflow: Object.assign({}, state.workflows[wfKey], { audits })
            }));
        }));
    }

    @Action(actionWorkflow.RollbackWorkflow)
    rollback(ctx: StateContext<WorkflowsStateModel>, action: actionWorkflow.RollbackWorkflow) {
        let auditId = action.payload.auditId;
        return this._http.post<Workflow>(
            `/project/${action.payload.projectKey}/workflow/${action.payload.workflowName}/rollback/${auditId}`,
            {}
        ).pipe(tap((pip: Workflow) => {
            ctx.dispatch(new actionWorkflow.LoadWorkflow({
                projectKey: action.payload.projectKey,
                workflow: pip
            }));
        }));
    }

    //  ------- Misc --------- //
    @Action(actionWorkflow.FetchAsCodeWorkflow)
    fetchAsCode(ctx: StateContext<WorkflowsStateModel>, action: actionWorkflow.FetchAsCodeWorkflow) {
        let params = new HttpParams();
        params = params.append('format', 'yaml');
        params = params.append('withPermissions', 'true');

        return this._http.get<string>(
            `/project/${action.payload.projectKey}/export/workflow/${action.payload.workflowName}`,
            { params, responseType: <any>'text' }
        ).pipe(tap((asCode: string) => {
            const state = ctx.getState();
            let wfKey = action.payload.projectKey + '/' + action.payload.workflowName;

            ctx.dispatch(new actionWorkflow.LoadWorkflow({
                projectKey: action.payload.projectKey,
                workflow: Object.assign({}, state.workflows[wfKey], <Workflow>{ asCode })
            }));
        }));
    }

    @Action(actionWorkflow.PreviewWorkflow)
    preview(ctx: StateContext<WorkflowsStateModel>, action: actionWorkflow.PreviewWorkflow) {
        let headers = new HttpHeaders();
        headers = headers.append('Content-Type', 'application/x-yaml');
        let params = new HttpParams();
        params = params.append('format', 'yaml');

        return this._http.post<Workflow>(
            `/project/${action.payload.projectKey}/preview/workflow`,
            action.payload.pipCode,
            { params, headers }
        ).pipe(tap((pip: Workflow) => {
            const state = ctx.getState();
            const wfKey = action.payload.projectKey + '/' + action.payload.workflowName;

            ctx.dispatch(new actionWorkflow.LoadWorkflow({
                projectKey: action.payload.projectKey,
                workflow: Object.assign({}, state.workflows[wfKey], { preview: pip })
            }));
        }));
    }

    @Action(actionWorkflow.ExternalChangeWorkflow)
    externalChange(ctx: StateContext<WorkflowsStateModel>, action: actionWorkflow.ExternalChangeWorkflow) {
        const state = ctx.getState();
        const wfKey = action.payload.projectKey + '/' + action.payload.workflowName;
        const pipUpdated = Object.assign({}, state.workflows[wfKey], { externalChange: true });

        ctx.setState({
            ...state,
            workflows: Object.assign({}, state.workflows, { [wfKey]: pipUpdated }),
        });
    }

    @Action(actionWorkflow.DeleteFromCacheWorkflow)
    deleteFromCache(ctx: StateContext<WorkflowsStateModel>, action: actionWorkflow.DeleteFromCacheWorkflow) {
        const state = ctx.getState();
        const wfKey = action.payload.projectKey + '/' + action.payload.workflowName;
        let workflows = Object.assign({}, state.workflows);
        delete workflows[wfKey];

        ctx.setState({
            ...state,
            workflows,
        });
    }

    @Action(actionWorkflow.ResyncWorkflow)
    resync(ctx: StateContext<WorkflowsStateModel>, action: actionWorkflow.ResyncWorkflow) {
        let params = new HttpParams();
        params = params.append('withUsage', 'true');
        params = params.append('withAudits', 'true');
        params = params.append('withTemplate', 'true');
        params = params.append('withAsCodeEvents', 'true');

        return this._http.get<Workflow>(
            `/project/${action.payload.projectKey}/workflows/${action.payload.workflowName}`,
            { params }
        ).pipe(tap((wf) => {
            ctx.dispatch(new actionWorkflow.LoadWorkflow({
                projectKey: action.payload.projectKey,
                workflow: wf
            }));
        }));
    }
}
