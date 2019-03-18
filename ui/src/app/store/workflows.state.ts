import { HttpClient, HttpHeaders, HttpParams } from '@angular/common/http';
import { Action, createSelector, State, StateContext } from '@ngxs/store';
import { GroupPermission } from 'app/model/group.model';
import { Label } from 'app/model/project.model';
import { WNode, WNodeTrigger, Workflow } from 'app/model/workflow.model';
import { NavbarService } from 'app/service/navbar/navbar.service';
import { cloneDeep } from 'lodash';
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
                const newWfKey = action.payload.projectKey + '/' + wf.name;
                let workflows = Object.assign({}, state.workflows, {
                    [newWfKey]: wf,
                });
                workflows[newWfKey].audits = workflows[wfKey].audits;
                workflows[newWfKey].from_template = workflows[wfKey].from_template;
                workflows[newWfKey].template_up_to_date = workflows[wfKey].template_up_to_date;
                workflows[newWfKey].as_code_events = workflows[wfKey].as_code_events;
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

    //  ------- Nodes --------- //
    @Action(actionWorkflow.AddNodeTriggerWorkflow)
    addNodeTrigger(ctx: StateContext<WorkflowsStateModel>, action: actionWorkflow.AddNodeTriggerWorkflow) {
        const state = ctx.getState();
        const wfKey = action.payload.projectKey + '/' + action.payload.workflowName;
        let currentWorkflow = cloneDeep(state.workflows[wfKey]);
        let node = Workflow.getNodeByID(action.payload.parentId, currentWorkflow);
        if (!node.triggers) {
            node.triggers = new Array<WNodeTrigger>();
        }
        node.triggers.push(action.payload.trigger);

        const workflow: Workflow = {
            ...state.workflows[wfKey],
            workflow_data: {
                ...state.workflows[wfKey].workflow_data,
                node: currentWorkflow.workflow_data.node
            }
        };

        return ctx.dispatch(new actionWorkflow.UpdateWorkflow({
            projectKey: action.payload.projectKey,
            workflowName: action.payload.workflowName,
            changes: workflow
        }));
    }

    //  ------- Joins --------- //
    @Action(actionWorkflow.AddJoinWorkflow)
    addJoinTrigger(ctx: StateContext<WorkflowsStateModel>, action: actionWorkflow.AddJoinWorkflow) {
        const state = ctx.getState();
        const wfKey = action.payload.projectKey + '/' + action.payload.workflowName;
        let joins = state.workflows[wfKey].workflow_data.joins ? state.workflows[wfKey].workflow_data.joins : [];
        joins.push(action.payload.join);

        const workflow: Workflow = {
            ...state.workflows[wfKey],
            workflow_data: {
                ...state.workflows[wfKey].workflow_data,
                joins
            }
        };

        return ctx.dispatch(new actionWorkflow.UpdateWorkflow({
            projectKey: action.payload.projectKey,
            workflowName: action.payload.workflowName,
            changes: workflow
        }));
    }

    //  ------- Hooks --------- //
    @Action(actionWorkflow.AddHookWorkflow)
    addHook(ctx: StateContext<WorkflowsStateModel>, action: actionWorkflow.AddHookWorkflow) {
        const state = ctx.getState();
        const wfKey = action.payload.projectKey + '/' + action.payload.workflowName;
        const hooks = state.workflows[wfKey].workflow_data.node.hooks || [];
        const root = Object.assign({}, state.workflows[wfKey].workflow_data.node, <WNode>{
            hooks: hooks.concat([action.payload.hook])
        });
        const workflow: Workflow = {
            ...state.workflows[wfKey],
            workflow_data: {
                ...state.workflows[wfKey].workflow_data,
                node: root
            }
        };

        return ctx.dispatch(new actionWorkflow.UpdateWorkflow({
            projectKey: action.payload.projectKey,
            workflowName: action.payload.workflowName,
            changes: workflow
        }));
    }

    @Action(actionWorkflow.UpdateHookWorkflow)
    updateHook(ctx: StateContext<WorkflowsStateModel>, action: actionWorkflow.UpdateHookWorkflow) {
        const state = ctx.getState();
        const wfKey = action.payload.projectKey + '/' + action.payload.workflowName;
        const root = Object.assign({}, state.workflows[wfKey].workflow_data.node, <WNode>{
            hooks: state.workflows[wfKey].workflow_data.node.hooks.map((hook) => {
                if (hook.uuid === action.payload.hook.uuid) {
                    return action.payload.hook;
                }
                return hook;
            })
        });
        const workflow: Workflow = {
            ...state.workflows[wfKey],
            workflow_data: {
                ...state.workflows[wfKey].workflow_data,
                node: root
            }
        };

        return ctx.dispatch(new actionWorkflow.UpdateWorkflow({
            projectKey: action.payload.projectKey,
            workflowName: action.payload.workflowName,
            changes: workflow
        }));
    }

    @Action(actionWorkflow.DeleteHookWorkflow)
    deleteHook(ctx: StateContext<WorkflowsStateModel>, action: actionWorkflow.DeleteHookWorkflow) {
        const state = ctx.getState();
        const wfKey = action.payload.projectKey + '/' + action.payload.workflowName;
        const root = Object.assign({}, state.workflows[wfKey].workflow_data.node, <WNode>{
            hooks: state.workflows[wfKey].workflow_data.node.hooks.filter((hook) => hook.uuid !== action.payload.hook.uuid)
        });
        const workflow: Workflow = {
            ...state.workflows[wfKey],
            workflow_data: {
                ...state.workflows[wfKey].workflow_data,
                node: root
            }
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
        const state = ctx.getState();
        const wfKey = action.payload.projectKey + '/' + action.payload.workflowName;

        if (!state.workflows[wfKey] || !state.workflows[wfKey].audits) {
            return ctx.dispatch(new actionWorkflow.ResyncWorkflow({
                projectKey: action.payload.projectKey,
                workflowName: action.payload.workflowName,
            }));
        }
    }

    @Action(actionWorkflow.RollbackWorkflow)
    rollback(ctx: StateContext<WorkflowsStateModel>, action: actionWorkflow.RollbackWorkflow) {
        let auditId = action.payload.auditId;
        return this._http.post<Workflow>(
            `/project/${action.payload.projectKey}/workflows/${action.payload.workflowName}/rollback/${auditId}`,
            {}
        ).pipe(tap((workflow: Workflow) => {
            ctx.dispatch(new actionWorkflow.LoadWorkflow({
                projectKey: action.payload.projectKey,
                workflow
            }));
        }));
    }

    //  ------- Labels --------- //
    @Action(actionWorkflow.LinkLabelOnWorkflow)
    linkLabel(ctx: StateContext<WorkflowsStateModel>, action: actionWorkflow.LinkLabelOnWorkflow) {
        return this._http.post<Label>(
            `/project/${action.payload.projectKey}/workflows/${action.payload.workflowName}/label`,
            action.payload.label
        ).pipe(tap((label: Label) => {
            const state = ctx.getState();
            const wfKey = action.payload.projectKey + '/' + action.payload.workflowName;

            ctx.dispatch(new ActionProject.AddLabelWorkflowInProject({
                workflowName: action.payload.workflowName,
                label: action.payload.label
            }));
            if (state.workflows[wfKey]) {
                const labels = state.workflows[wfKey].labels ? state.workflows[wfKey].labels.concat(label) : [label];

                ctx.dispatch(new actionWorkflow.LoadWorkflow({
                    projectKey: action.payload.projectKey,
                    workflow: Object.assign({}, state.workflows[wfKey], <Workflow>{
                        labels
                    })
                }));
            }
        }));
    }

    @Action(actionWorkflow.UnlinkLabelOnWorkflow)
    unlinkLabel(ctx: StateContext<WorkflowsStateModel>, action: actionWorkflow.UnlinkLabelOnWorkflow) {
        return this._http.delete<null>(
            `/project/${action.payload.projectKey}/workflows/${action.payload.workflowName}/label/${action.payload.label.id}`
        ).pipe(tap(() => {
            const state = ctx.getState();
            const wfKey = action.payload.projectKey + '/' + action.payload.workflowName;

            ctx.dispatch(new ActionProject.DeleteLabelWorkflowInProject({
                workflowName: action.payload.workflowName,
                labelId: action.payload.label.id
            }));
            if (state.workflows[wfKey]) {
                let labels = state.workflows[wfKey].labels ? state.workflows[wfKey].labels.concat([]) : [];
                labels = labels.filter((lbl) => lbl.id !== action.payload.label.id);

                ctx.dispatch(new actionWorkflow.LoadWorkflow({
                    projectKey: action.payload.projectKey,
                    workflow: Object.assign({}, state.workflows[wfKey], <Workflow>{
                        labels
                    })
                }));
            }
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
            if (state.workflows[wfKey]) {
                const wfUpdated = Object.assign({}, state.workflows[wfKey], <Workflow>{
                    favorite: !state.workflows[wfKey].favorite
                });

                ctx.setState({
                    ...state,
                    workflows: Object.assign({}, state.workflows, { [wfKey]: wfUpdated }),
                });
            }
            // TODO: dispatch action on global state to update project in list and user state
            // TODO: move this one on user state and just update state here, not XHR
        }));
    }

    @Action(actionWorkflow.ClearCacheWorkflow)
    clearCache(ctx: StateContext<WorkflowsStateModel>, action: actionWorkflow.ClearCacheWorkflow) {
        ctx.setState(getInitialWorkflowsState());
    }
}
