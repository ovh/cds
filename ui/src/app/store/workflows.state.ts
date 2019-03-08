import { HttpClient, HttpHeaders, HttpParams } from '@angular/common/http';
import { Action, createSelector, State, StateContext } from '@ngxs/store';
import { AuditWorkflow } from 'app/model/audit.model';
import { GroupPermission } from 'app/model/group.model';
import { Workflow } from 'app/model/workflow.model';
import { NavbarService } from 'app/service/navbar/navbar.service';
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

    constructor(private _http: HttpClient, private _navbarService: NavbarService) { }

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
            `/project/${action.payload.projectKey}/workflows`,
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
            `/project/${action.payload.projectKey}/import/workflows`,
            action.payload.workflowCode,
            { headers, params }
        );
        if (action.payload.wfName) {
            request = this._http.put<Array<string>>(
                `/project/${action.payload.projectKey}/import/workflows/${action.payload.wfName}`,
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
            `/project/${action.payload.projectKey}/workflows/${action.payload.workflowName}`,
            action.payload.changes
        ).pipe(tap((wf) => {
            const state = ctx.getState();

            let wfKey = action.payload.projectKey + '/' + action.payload.workflowName;
            if (wf.name !== action.payload.workflowName) {
                let workflows = Object.assign({}, state.workflows, {
                    [action.payload.projectKey + '/' + wf.name]: wf,
                });
                workflows[action.payload.projectKey + '/' + wf.name].audits = workflows[wfKey].audits;
                workflows[action.payload.projectKey + '/' + wf.name].from_template = workflows[wfKey].from_template;
                workflows[action.payload.projectKey + '/' + wf.name].template_up_to_date = workflows[wfKey].template_up_to_date;
                workflows[action.payload.projectKey + '/' + wf.name].as_code_events = workflows[wfKey].as_code_events;
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
                if (!wf.notifications) {
                    wfUpdated.notifications = [];
                }
                wfUpdated.audits = state.workflows[wfKey].audits;
                wfUpdated.from_template = state.workflows[wfKey].from_template;
                wfUpdated.template_up_to_date = state.workflows[wfKey].template_up_to_date;
                wfUpdated.as_code_events = state.workflows[wfKey].as_code_events;

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
            `/project/${action.payload.projectKey}/workflows/${action.payload.workflowName}`
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

    //  ------- Group Permission --------- //
    @Action(actionWorkflow.AddGroupInAllWorkflows)
    propagateProjectPermission(ctx: StateContext<WorkflowsStateModel>, action: actionWorkflow.AddGroupInAllWorkflows) {
        const state = ctx.getState();
        let group: GroupPermission = { ...action.payload.group, hasChanged: false, updating: false };
        let workflows = Object.keys(state.workflows).reduce((workflowsObj, key) => {
            let wf = Object.assign({}, state.workflows[key], <Workflow>{
                groups: [group].concat(state.workflows[key].groups)
            });
            return Object.assign({}, workflowsObj, { [key]: wf });
        }, {});

        ctx.setState({
            ...state,
            workflows
        });
    }

    @Action(actionWorkflow.AddGroupInWorkflow)
    addGroupPermission(ctx: StateContext<WorkflowsStateModel>, action: actionWorkflow.AddGroupInWorkflow) {
        return this._http.post<Workflow>(
            `/project/${action.payload.projectKey}/workflows/${action.payload.workflowName}/groups`,
            action.payload.group
        ).pipe(tap((wf: Workflow) => {
            const state = ctx.getState();
            let wfKey = action.payload.projectKey + '/' + action.payload.workflowName;

            ctx.dispatch(new actionWorkflow.LoadWorkflow({
                projectKey: action.payload.projectKey,
                workflow: Object.assign({}, state.workflows[wfKey], <Workflow>{ groups: wf.groups })
            }));
        }));
    }

    @Action(actionWorkflow.UpdateGroupInWorkflow)
    updateGroupPermission(ctx: StateContext<WorkflowsStateModel>, action: actionWorkflow.UpdateGroupInWorkflow) {
        return this._http.put<Workflow>(
            `/project/${action.payload.projectKey}/workflows/${action.payload.workflowName}/groups/${action.payload.group.group.name}`,
            action.payload.group
        ).pipe(tap((wf: Workflow) => {
            const state = ctx.getState();
            let wfKey = action.payload.projectKey + '/' + action.payload.workflowName;

            ctx.dispatch(new actionWorkflow.LoadWorkflow({
                projectKey: action.payload.projectKey,
                workflow: Object.assign({}, state.workflows[wfKey], <Workflow>{ groups: wf.groups })
            }));
        }));
    }

    @Action(actionWorkflow.DeleteGroupInWorkflow)
    deleteGroupPermission(ctx: StateContext<WorkflowsStateModel>, action: actionWorkflow.DeleteGroupInWorkflow) {
        return this._http.delete<Workflow>(
            `/project/${action.payload.projectKey}/workflows/${action.payload.workflowName}/groups/${action.payload.group.group.name}`
        ).pipe(tap((wf: Workflow) => {
            const state = ctx.getState();
            let wfKey = action.payload.projectKey + '/' + action.payload.workflowName;

            ctx.dispatch(new actionWorkflow.LoadWorkflow({
                projectKey: action.payload.projectKey,
                workflow: Object.assign({}, state.workflows[wfKey], <Workflow>{ groups: wf.groups })
            }));
        }));
    }


    //  ------- Notification --------- //
    @Action(actionWorkflow.AddNotificationWorkflow)
    addNotification(ctx: StateContext<WorkflowsStateModel>, action: actionWorkflow.AddNotificationWorkflow) {
        const state = ctx.getState();
        const wfKey = action.payload.projectKey + '/' + action.payload.workflowName;
        const notifications = state.workflows[wfKey].notifications || [];
        const workflow: Workflow = {
            ...state.workflows[wfKey],
            notifications: notifications.concat([action.payload.notification])
        };

        return ctx.dispatch(new actionWorkflow.UpdateWorkflow({
            projectKey: action.payload.projectKey,
            workflowName: action.payload.workflowName,
            changes: workflow
        }));
    }

    @Action(actionWorkflow.UpdateNotificationWorkflow)
    updateNotification(ctx: StateContext<WorkflowsStateModel>, action: actionWorkflow.UpdateNotificationWorkflow) {
        const state = ctx.getState();
        const wfKey = action.payload.projectKey + '/' + action.payload.workflowName;
        const workflow: Workflow = {
            ...state.workflows[wfKey],
            notifications: state.workflows[wfKey].notifications.map((no) => {
                if (no.id === action.payload.notification.id) {
                    return action.payload.notification;
                }
                return no;
            })
        };

        return ctx.dispatch(new actionWorkflow.UpdateWorkflow({
            projectKey: action.payload.projectKey,
            workflowName: action.payload.workflowName,
            changes: workflow
        }));
    }

    @Action(actionWorkflow.DeleteNotificationWorkflow)
    deleteNotification(ctx: StateContext<WorkflowsStateModel>, action: actionWorkflow.DeleteNotificationWorkflow) {
        const state = ctx.getState();
        const wfKey = action.payload.projectKey + '/' + action.payload.workflowName;
        const workflow: Workflow = {
            ...state.workflows[wfKey],
            notifications: state.workflows[wfKey].notifications.filter(no => {
                return action.payload.notification.id !== no.id;
            })
        };

        return ctx.dispatch(new actionWorkflow.UpdateWorkflow({
            projectKey: action.payload.projectKey,
            workflowName: action.payload.workflowName,
            changes: workflow
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
            `/project/${action.payload.projectKey}/export/workflows/${action.payload.workflowName}`,
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
            `/project/${action.payload.projectKey}/preview/workflows`,
            action.payload.wfCode,
            { params, headers }
        ).pipe(tap((wf: Workflow) => {
            const state = ctx.getState();
            const wfKey = action.payload.projectKey + '/' + action.payload.workflowName;

            ctx.dispatch(new actionWorkflow.LoadWorkflow({
                projectKey: action.payload.projectKey,
                workflow: Object.assign({}, state.workflows[wfKey], { preview: wf })
            }));
        }));
    }

    @Action(actionWorkflow.ExternalChangeWorkflow)
    externalChange(ctx: StateContext<WorkflowsStateModel>, action: actionWorkflow.ExternalChangeWorkflow) {
        const state = ctx.getState();
        const wfKey = action.payload.projectKey + '/' + action.payload.workflowName;
        const wfUpdated = Object.assign({}, state.workflows[wfKey], { externalChange: true });

        ctx.setState({
            ...state,
            workflows: Object.assign({}, state.workflows, { [wfKey]: wfUpdated }),
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

    @Action(actionWorkflow.UpdateFavoriteWorkflow)
    updateFavorite(ctx: StateContext<WorkflowsStateModel>, action: actionWorkflow.UpdateFavoriteWorkflow) {
        const state = ctx.getState();

        return this._http.post(
            '/user/favorite', {
                type: 'workflow',
                project_key: action.payload.projectKey,
                workflow_name: action.payload.workflowName,
            }
        ).pipe(tap(() => {
            this._navbarService.getData(); // TODO: to delete
            const wfKey = action.payload.projectKey + '/' + action.payload.workflowName;
            const wfUpdated = Object.assign({}, state.workflows[wfKey], <Workflow>{ favorite: !state.workflows[wfKey].favorite });

            ctx.setState({
                ...state,
                workflows: Object.assign({}, state.workflows, { [wfKey]: wfUpdated }),
            });
            // TODO: dispatch action on global state to update project in list and user state
            // TODO: move this one on user state and just update state here, not XHR
        }));
    }
}
