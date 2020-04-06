import { HttpClient, HttpHeaders, HttpParams } from '@angular/common/http';
import { Action, createSelector, Selector, State, StateContext } from '@ngxs/store';
import { WNode, WNodeHook, WNodeTrigger, Workflow } from 'app/model/workflow.model';
import { WorkflowNodeRun, WorkflowRun } from 'app/model/workflow.run.model';
import { NavbarService } from 'app/service/navbar/navbar.service';
import { WorkflowRunService } from 'app/service/workflow/run/workflow.run.service';
import { WorkflowService } from 'app/service/workflow/workflow.service';
import { WorkflowSidebarMode } from 'app/service/workflow/workflow.sidebar.store';
import * as actionAsCode from 'app/store/ascode.action';
import cloneDeep from 'lodash-es/cloneDeep';
import { finalize, first, tap } from 'rxjs/operators';
import * as ActionProject from './project.action';
import * as actionWorkflow from './workflow.action';
import { UpdateModal, UpdateWorkflowRunList } from './workflow.action';

export class WorkflowStateModel {
    workflow: Workflow; // selected workflow
    projectKey: string; // current project key
    node: WNode; // selected node
    hook: WNodeHook; // selected hook
    editModal: boolean; // is edit modal is opened
    loadingWorkflow: boolean;
    loadingWorkflowRuns: boolean;
    loadingWorkflowRun: boolean;
    loadingWorkflowNodeRun: boolean;
    canEdit: boolean; // user permission

    sidebar: string;
    workflowRun: WorkflowRun;
    workflowNodeRun: WorkflowNodeRun;
    listRuns: Array<WorkflowRun>;
    filters?: {};
    editWorkflow: Workflow;
    editMode: boolean;
    editModeWorkflowChanged: boolean;
}

export function getInitialWorkflowState(): WorkflowStateModel {
    return {
        projectKey: null,
        workflow: null,
        editWorkflow: null,
        node: null,
        hook: null,
        editModal: false,
        loadingWorkflow: false,
        loadingWorkflowRuns: false,
        loadingWorkflowRun: false,
        loadingWorkflowNodeRun: false,
        canEdit: false,
        workflowRun: null,
        workflowNodeRun: null,
        listRuns: new Array<WorkflowRun>(),
        sidebar: WorkflowSidebarMode.RUNS,
        filters: {},
        editMode: false,
        editModeWorkflowChanged: false
    };
}

@State<WorkflowStateModel>({
    name: 'workflow',
    defaults: getInitialWorkflowState()
})
export class WorkflowState {

    static getEditModal() {
        return createSelector(
            [WorkflowState],
            (state: WorkflowStateModel): boolean => state.editModal
        );
    }

    @Selector()
    static workflowSnapshot(state: WorkflowStateModel) {
        return state.workflow;
    }

    /** @deprecated */
    static getCurrent() {
        return createSelector(
            [WorkflowState],
            (state: WorkflowStateModel): WorkflowStateModel => state
        );
    }

    static getWorkflow() {
        return createSelector(
            [WorkflowState],
            (state: WorkflowStateModel): Workflow => state.workflow
        );
    }

    static getSelectedHook() {
        return createSelector(
            [WorkflowState],
            (state: WorkflowStateModel): WNodeHook => state.hook
        );
    }

    @Selector()
    static workflowRunSnapshot(state: WorkflowStateModel) {
        return state.workflowRun;
    }

    static getSelectedWorkflowRun() {
        return createSelector(
            [WorkflowState],
            (state: WorkflowStateModel): WorkflowRun => state.workflowRun
        );
    }

    static getListRuns() {
        return createSelector(
            [WorkflowState],
            (state: WorkflowStateModel): Array<WorkflowRun> => state.listRuns
        );
    }

    static getLoadingRuns() {
        return createSelector(
            [WorkflowState],
            (state: WorkflowStateModel): boolean => state.loadingWorkflowRuns
        );
    }

    static getRunSidebarFilters() {
        return createSelector(
            [WorkflowState],
            (state: WorkflowStateModel): {} => state.filters
        );
    }

    static getSidebarMode() {
        return createSelector(
            [WorkflowState],
            (state: WorkflowStateModel): string => state.sidebar
        );
    }

    @Selector()
    static nodeSnapshot(state: WorkflowStateModel) {
        return state.node;
    }

    static getSelectedNode() {
        return createSelector(
            [WorkflowState],
            (state: WorkflowStateModel): WNode => state.node
        );
    }

    @Selector()
    static nodeRunByNodeID(state: WorkflowStateModel) {
        return (id: number) => {
            if (!state.workflowRun || !state.workflowRun.nodes || !state.workflowRun.nodes[id]
                || state.workflowRun.nodes[id].length === 0) {
                return null;
            }
            return state.workflowRun.nodes[id][0];
        };
    }

    @Selector()
    static nodeRunSnapshot(state: WorkflowStateModel) {
        return state.workflowNodeRun;
    }

    static getSelectedNodeRun() {
        return createSelector(
            [WorkflowState],
            (state: WorkflowStateModel): WorkflowNodeRun => state.workflowNodeRun
        );
    }

    @Selector()
    static nodeRunStage(state: WorkflowStateModel) {
        return (id: number) => {
            if (!state.workflowNodeRun || !state.workflowNodeRun.stages) {
                return null;
            }
            return state.workflowNodeRun.stages.find(s => s.id === id);
        };
    }

    @Selector()
    static nodeRunJob(state: WorkflowStateModel) {
        return (idStage: number, idJob: number) => {
            if (!state.workflowNodeRun || !state.workflowNodeRun.stages) {
                return null;
            }
            let stageJob = state.workflowNodeRun.stages.find(s => s.id === idStage);
            if (!stageJob || !stageJob.run_jobs) {
                return null;
            }
            return stageJob.run_jobs.find(rj => rj.id === idJob);
        };
    }

    @Selector()
    static nodeRunJobStep(state: WorkflowStateModel) {
        return (idStage: number, idJob: number, stepNum: number) => {
            if (!state.workflowNodeRun || !state.workflowNodeRun.stages) {
                return null;
            }
            let stageRunStep = state.workflowNodeRun.stages.find(s => s.id === idStage);
            if (!stageRunStep || !stageRunStep.run_jobs) {
                return null;
            }
            let j = stageRunStep.run_jobs.find(rj => rj.id === idJob);
            if (!j || !j.job || !j.job.step_status) {
                return null;
            }
            return j.job.step_status[stepNum];
        };
    }

    constructor(private _http: HttpClient, private _navbarService: NavbarService,
        private _workflowService: WorkflowService, private _workflowRunService: WorkflowRunService) {
    }

    @Action(actionWorkflow.OpenEditModal)
    openEditModal(ctx: StateContext<WorkflowStateModel>, action: actionWorkflow.OpenEditModal) {
        const state = ctx.getState();

        ctx.setState({
            ...state,
            node: action.payload.node,
            hook: action.payload.hook,
            editModal: true,
        });
    }

    @Action(actionWorkflow.CloseEditModal)
    closeEditModal(ctx: StateContext<WorkflowStateModel>) {
        const state = ctx.getState();
        if (!state.workflowNodeRun) {
            ctx.setState({
                ...state,
                node: null,
                hook: null,
                editModal: false
            });
            return
        }
        ctx.setState({
            ...state,
            editModal: false
        });
    }

    @Action(actionWorkflow.CreateWorkflow)
    create(ctx: StateContext<WorkflowStateModel>, action: actionWorkflow.CreateWorkflow) {
        const state = ctx.getState();
        return this._http.post<Workflow>(
            `/project/${action.payload.projectKey}/workflows`,
            action.payload.workflow
        ).pipe(tap((wf) => {
            ctx.setState({
                ...state,
                projectKey: action.payload.projectKey,
                workflow: wf,
                canEdit: true,
                loadingWorkflow: false
            });
            ctx.dispatch(new ActionProject.AddWorkflowInProject(wf));
        }));
    }

    @Action(actionWorkflow.ImportWorkflow)
    import(ctx: StateContext<WorkflowStateModel>, action: actionWorkflow.ImportWorkflow) {
        const state = ctx.getState();

        // Refresh when change project
        if (state.projectKey !== action.payload.projectKey) {
            ctx.setState({
                ...state,
                projectKey: action.payload.projectKey,
                workflow: null,
                loadingWorkflow: false,
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
                return ctx.dispatch(new actionWorkflow.GetWorkflow({
                    projectKey: action.payload.projectKey,
                    workflowName: action.payload.wfName
                }));
            }
        }));

    }

    @Action(actionWorkflow.UpdateWorkflow)
    update(ctx: StateContext<WorkflowStateModel>, action: actionWorkflow.UpdateWorkflow) {
        const stateEdit = ctx.getState();
        // As code Update Cache
        if (stateEdit.editMode) {
            let n: WNode;
            let h: WNodeHook;
            if (stateEdit.node) {
                n = Workflow.getNodeByRef(stateEdit.node.ref, action.payload.changes);
            }
            if (stateEdit.hook) {
                h = Workflow.getHookByRef(stateEdit.hook.ref, action.payload.changes);
            }
            ctx.setState({
                ...stateEdit,
                editWorkflow: action.payload.changes,
                node: n,
                hook: h,
                editModeWorkflowChanged: true,
            });
            return;
        }

        // Update Non as code workflow
        return this._http.put<Workflow>(
            `/project/${action.payload.projectKey}/workflows/${action.payload.workflowName}`,
            action.payload.changes
        ).pipe(tap((wf) => {
            const state = ctx.getState();
            let oldWorkflow = cloneDeep(state.workflow);
            if (action.payload.workflowName !== wf.name) {
                wf.audits = cloneDeep(oldWorkflow.audits);
                wf.from_template = cloneDeep(oldWorkflow.from_template);
                wf.template_up_to_date = cloneDeep(oldWorkflow.template_up_to_date);
                wf.as_code_events = cloneDeep(oldWorkflow.as_code_events);

                // Generate hook ref for UI edition
                if (wf && wf.workflow_data && wf.workflow_data.node && wf.workflow_data.node.hooks) {
                    wf.workflow_data.node.hooks.forEach(h => {
                        if (!h.ref) {
                            h.ref = h.uuid;
                        }
                    });
                }

                ctx.setState({
                    ...state,
                    workflow: wf
                });
                ctx.dispatch(new ActionProject.UpdateWorkflowInProject({
                    previousWorkflowName: action.payload.workflowName,
                    changes: wf
                }));
                ctx.dispatch(new UpdateModal({ workflow: wf }));
            } else {
                let wfUpdated: Workflow = {
                    ...state.workflow,
                    ...wf,
                    preview: null
                };
                if (!wf.notifications) {
                    wfUpdated.notifications = [];
                }
                wfUpdated.audits = state.workflow.audits;
                wfUpdated.from_template = state.workflow.from_template;
                wfUpdated.template_up_to_date = state.workflow.template_up_to_date;
                wfUpdated.as_code_events = state.workflow.as_code_events;

                // Generate hook ref for UI edition
                if (wfUpdated.workflow_data && wfUpdated.workflow_data.node && wfUpdated.workflow_data.node.hooks) {
                    wfUpdated.workflow_data.node.hooks.forEach(h => {
                        if (!h.ref) {
                            h.ref = h.uuid;
                        }
                    });
                }

                ctx.setState({
                    ...state,
                    workflow: wfUpdated
                });
                ctx.dispatch(new UpdateModal({ workflow: wfUpdated }));
            }
        }));
    }

    @Action(actionWorkflow.DeleteWorkflow)
    delete(ctx: StateContext<WorkflowStateModel>, action: actionWorkflow.DeleteWorkflow) {
        return this._http.delete(
            `/project/${action.payload.projectKey}/workflows/${action.payload.workflowName}`
        ).pipe(tap(() => {
            const state = ctx.getState();
            ctx.setState({
                ...state,
                workflow: null,
                editModal: false,
                hook: null,
                node: null,
                loadingWorkflow: false,
                canEdit: false
            });

            ctx.dispatch(new ActionProject.DeleteWorkflowInProject({ workflowName: action.payload.workflowName }));
        }));
    }

    @Action(actionWorkflow.UpdateWorkflowIcon)
    updateIcon(ctx: StateContext<WorkflowStateModel>, action: actionWorkflow.UpdateWorkflowIcon) {
        return this._http.put<null>(
            `/project/${action.payload.projectKey}/workflows/${action.payload.workflowName}/icon`,
            action.payload.icon
        ).pipe(tap(() => {
            const state = ctx.getState();
            let wfUpdated = {
                ...state.workflow,
                icon: action.payload.icon
            };

            ctx.dispatch(new ActionProject.UpdateWorkflowInProject({
                previousWorkflowName: action.payload.workflowName,
                changes: wfUpdated
            }));

            return ctx.setState({
                ...state,
                workflow: wfUpdated,
            });
        }));
    }

    @Action(actionWorkflow.DeleteWorkflowIcon)
    deleteIcon(ctx: StateContext<WorkflowStateModel>, action: actionWorkflow.DeleteWorkflowIcon) {
        return this._http.delete<null>(
            `/project/${action.payload.projectKey}/workflows/${action.payload.workflowName}/icon`
        ).pipe(tap(() => {
            const state = ctx.getState();
            let wfUpdated = {
                ...state.workflow,
                icon: ''
            };

            ctx.dispatch(new ActionProject.UpdateWorkflowInProject({
                previousWorkflowName: action.payload.workflowName,
                changes: wfUpdated
            }));

            return ctx.setState({
                ...state,
                workflow: wfUpdated,
            });
        }));
    }

    @Action(actionWorkflow.AddGroupInWorkflow)
    addGroupPermission(ctx: StateContext<WorkflowStateModel>, action: actionWorkflow.AddGroupInWorkflow) {
        return this._http.post<Workflow>(
            `/project/${action.payload.projectKey}/workflows/${action.payload.workflowName}/groups`,
            action.payload.group
        ).pipe(tap((wf: Workflow) => {
            const state = ctx.getState();
            ctx.setState({
                ...state,
                workflow: Object.assign({}, state.workflow, <Workflow>{ groups: wf.groups }),
            });
        }));
    }

    @Action(actionWorkflow.UpdateGroupInWorkflow)
    updateGroupPermission(ctx: StateContext<WorkflowStateModel>, action: actionWorkflow.UpdateGroupInWorkflow) {
        return this._http.put<Workflow>(
            `/project/${action.payload.projectKey}/workflows/${action.payload.workflowName}/groups/${action.payload.group.group.name}`,
            action.payload.group
        ).pipe(tap((wf: Workflow) => {
            const state = ctx.getState();
            ctx.setState({
                ...state,
                workflow: Object.assign({}, state.workflow, <Workflow>{ groups: wf.groups }),
            });
        }));
    }

    @Action(actionWorkflow.DeleteGroupInWorkflow)
    deleteGroupPermission(ctx: StateContext<WorkflowStateModel>, action: actionWorkflow.DeleteGroupInWorkflow) {
        return this._http.delete<Workflow>(
            `/project/${action.payload.projectKey}/workflows/${action.payload.workflowName}/groups/${action.payload.group.group.name}`
        ).pipe(tap((wf: Workflow) => {
            const state = ctx.getState();
            ctx.setState({
                ...state,
                workflow: Object.assign({}, state.workflow, <Workflow>{ groups: wf.groups }),
            });
        }));
    }

    //  ------- Notification --------- //
    @Action(actionWorkflow.AddNotificationWorkflow)
    addNotification(ctx: StateContext<WorkflowStateModel>, action: actionWorkflow.AddNotificationWorkflow) {
        const state = ctx.getState();

        // As code Update Cache
        if (state.workflow && state.editMode) {
            const notificationsEdit = state.editWorkflow.notifications || [];
            const editWorkflow: Workflow = {
                ...state.editWorkflow,
                notifications: notificationsEdit.concat([action.payload.notification])
            };
            ctx.setState({
                ...state,
                editWorkflow: editWorkflow,
                editModeWorkflowChanged: true
            });
            return;
        }


        const notifications = state.workflow.notifications || [];
        const workflow: Workflow = {
            ...state.workflow,
            notifications: notifications.concat([action.payload.notification])
        };

        return ctx.dispatch(new actionWorkflow.UpdateWorkflow({
            projectKey: action.payload.projectKey,
            workflowName: action.payload.workflowName,
            changes: workflow
        }));
    }

    @Action(actionWorkflow.UpdateNotificationWorkflow)
    updateNotification(ctx: StateContext<WorkflowStateModel>, action: actionWorkflow.UpdateNotificationWorkflow) {
        const state = ctx.getState();

        // As code Update Cache
        if (state.workflow && state.editMode) {
            const editWorkflow: Workflow = {
                ...state.editWorkflow,
                notifications: state.editWorkflow.notifications.map((no) => {
                    if (no.id === action.payload.notification.id) {
                        return action.payload.notification;
                    }
                    return no;
                })
            };
            ctx.setState({
                ...state,
                editWorkflow: editWorkflow,
                editModeWorkflowChanged: true
            });
            return;
        }

        const workflow: Workflow = {
            ...state.workflow,
            notifications: state.workflow.notifications.map((no) => {
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
    deleteNotification(ctx: StateContext<WorkflowStateModel>, action: actionWorkflow.DeleteNotificationWorkflow) {
        const state = ctx.getState();

        // As code Update Cache
        if (state.workflow && state.editMode) {
            const editWorkflow: Workflow = {
                ...state.editWorkflow,
                notifications: state.editWorkflow.notifications.filter(no => {
                    return action.payload.notification.id !== no.id;
                })
            };
            ctx.setState({
                ...state,
                editWorkflow: editWorkflow,
                editModeWorkflowChanged: true
            });
            return;
        }

        const workflow: Workflow = {
            ...state.workflow,
            notifications: state.workflow.notifications.filter(no => {
                return action.payload.notification.id !== no.id;
            })
        };

        return ctx.dispatch(new actionWorkflow.UpdateWorkflow({
            projectKey: action.payload.projectKey,
            workflowName: action.payload.workflowName,
            changes: workflow
        }));
    }

    //  ------- Event integration --------- //
    @Action(actionWorkflow.UpdateEventIntegrationsWorkflow)
    addEventIntegration(ctx: StateContext<WorkflowStateModel>, action: actionWorkflow.UpdateEventIntegrationsWorkflow) {
        const state = ctx.getState();
        // As code Update Cache
        if (state.workflow && state.editMode) {
            const editWorkflow: Workflow = {
                ...state.editWorkflow,
                event_integrations: action.payload.eventIntegrations
            };
            ctx.setState({
                ...state,
                editWorkflow: editWorkflow,
                editModeWorkflowChanged: true
            });
            return;
        }

        const workflow: Workflow = {
            ...state.workflow,
            event_integrations: action.payload.eventIntegrations
        };

        return ctx.dispatch(new actionWorkflow.UpdateWorkflow({
            projectKey: action.payload.projectKey,
            workflowName: action.payload.workflowName,
            changes: workflow
        }));
    }

    @Action(actionWorkflow.DeleteEventIntegrationWorkflow)
    deleteEventIntegration(ctx: StateContext<WorkflowStateModel>, action: actionWorkflow.DeleteEventIntegrationWorkflow) {
        const state = ctx.getState();
        // As code Update Cache
        if (state.workflow && state.editMode) {
            const editWorkflow: Workflow = {
                ...state.editWorkflow,
                event_integrations: state.editWorkflow.event_integrations.filter((integ) => integ.id !== action.payload.integrationId)
            };
            ctx.setState({
                ...state,
                editWorkflow: editWorkflow,
                editModeWorkflowChanged: true
            });
            return;
        }


        return this._http.delete<null>(
            `/project/${action.payload.projectKey}/workflows/` +
            `${action.payload.workflowName}/eventsintegration/${action.payload.integrationId}`
        ).pipe(tap(() => {
            const workflow: Workflow = {
                ...state.workflow,
                event_integrations: state.workflow.event_integrations.filter((integ) => integ.id !== action.payload.integrationId)
            };

            return ctx.dispatch(new actionWorkflow.UpdateWorkflow({
                projectKey: action.payload.projectKey,
                workflowName: action.payload.workflowName,
                changes: workflow
            }));
        }));
    }

    //  ------- Nodes --------- //
    @Action(actionWorkflow.AddNodeTriggerWorkflow)
    addNodeTrigger(ctx: StateContext<WorkflowStateModel>, action: actionWorkflow.AddNodeTriggerWorkflow) {
        const state = ctx.getState();

        let currentWorkflow: Workflow;

        // As code Update Cache
        if (state.workflow && state.editMode) {
            currentWorkflow = cloneDeep(state.editWorkflow);
        } else {
            currentWorkflow = cloneDeep(state.workflow);
        }
        let node = Workflow.getNodeByID(action.payload.parentId, currentWorkflow);
        if (!node.triggers) {
            node.triggers = new Array<WNodeTrigger>();
        }
        node.triggers.push(action.payload.trigger);

        // As code Update Cache
        if (state.workflow && state.editMode) {
            ctx.setState({
                ...state,
                editWorkflow: currentWorkflow,
                editModeWorkflowChanged: true
            });
            return;
        }

        const workflow: Workflow = {
            ...state.workflow,
            workflow_data: {
                ...state.workflow.workflow_data,
                node: currentWorkflow.workflow_data.node,
                joins: currentWorkflow.workflow_data.joins
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
    addJoinTrigger(ctx: StateContext<WorkflowStateModel>, action: actionWorkflow.AddJoinWorkflow) {
        const state = ctx.getState();

        // As code Update Cache
        if (state.workflow && state.editMode) {
            let joinsAsCode = state.editWorkflow.workflow_data.joins ? [...state.editWorkflow.workflow_data.joins] : [];
            joinsAsCode.push(action.payload.join);
            const editWorkflow: Workflow = {
                ...state.editWorkflow,
                workflow_data: {
                    ...state.editWorkflow.workflow_data,
                    joins: joinsAsCode
                }
            };
            ctx.setState({
                ...state,
                editWorkflow: editWorkflow,
                editModeWorkflowChanged: true
            });
            return;
        }

        let joins = state.workflow.workflow_data.joins ? [...state.workflow.workflow_data.joins] : [];
        joins.push(action.payload.join);

        const workflow: Workflow = {
            ...state.workflow,
            workflow_data: {
                ...state.workflow.workflow_data,
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
    @Action(actionWorkflow.SelectHook)
    selectHook(ctx: StateContext<WorkflowStateModel>, action: actionWorkflow.SelectHook) {
        const state = ctx.getState();
        let sidebar = WorkflowSidebarMode.EDIT_HOOK;
        if (state.workflowRun) {
            sidebar = WorkflowSidebarMode.RUN_HOOK;
        }
        ctx.setState({
            ...state,
            node: action.payload.node,
            hook: action.payload.hook,
            sidebar: sidebar
        });
    }

    @Action(actionWorkflow.AddHookWorkflow)
    addHook(ctx: StateContext<WorkflowStateModel>, action: actionWorkflow.AddHookWorkflow) {
        const state = ctx.getState();

        if (state.workflow && state.editMode) {
            const hooksAsCode = state.editWorkflow.workflow_data.node.hooks || [];
            if (!action.payload.hook.ref) {
                action.payload.hook.ref = new Date().getTime().toString();
            }
            const rootAsCode = Object.assign({}, state.editWorkflow.workflow_data.node, <WNode>{
                hooks: hooksAsCode.concat([action.payload.hook])
            });
            const editWorkflow: Workflow = {
                ...state.editWorkflow,
                workflow_data: {
                    ...state.editWorkflow.workflow_data,
                    node: rootAsCode
                }
            };
            ctx.setState({
                ...state,
                editWorkflow: editWorkflow,
                editModeWorkflowChanged: true
            });
            return;
        }
        const hooks = state.workflow.workflow_data.node.hooks || [];
        const root = Object.assign({}, state.workflow.workflow_data.node, <WNode>{
            hooks: hooks.concat([action.payload.hook])
        });
        const workflow: Workflow = {
            ...state.workflow,
            workflow_data: {
                ...state.workflow.workflow_data,
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
    updateHook(ctx: StateContext<WorkflowStateModel>, action: actionWorkflow.UpdateHookWorkflow) {
        const state = ctx.getState();

        if (state.workflow && state.editMode) {
            const rootAsCode = Object.assign({}, state.editWorkflow.workflow_data.node, <WNode>{
                hooks: state.editWorkflow.workflow_data.node.hooks.map((hook) => {
                    if (hook.uuid === action.payload.hook.uuid) {
                        return action.payload.hook;
                    }
                    return hook;
                })
            });
            const editWorkflow: Workflow = {
                ...state.editWorkflow,
                workflow_data: {
                    ...state.editWorkflow.workflow_data,
                    node: rootAsCode
                }
            };
            ctx.setState({
                ...state,
                editWorkflow: editWorkflow,
                editModeWorkflowChanged: true
            });
            return;
        }

        const root = Object.assign({}, state.workflow.workflow_data.node, <WNode>{
            hooks: state.workflow.workflow_data.node.hooks.map((hook) => {
                if (hook.uuid === action.payload.hook.uuid) {
                    return action.payload.hook;
                }
                return hook;
            })
        });
        const workflow: Workflow = {
            ...state.workflow,
            workflow_data: {
                ...state.workflow.workflow_data,
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
    deleteHook(ctx: StateContext<WorkflowStateModel>, action: actionWorkflow.DeleteHookWorkflow) {
        const state = ctx.getState();

        if (state.workflow && state.editMode) {
            const rootAsCode = Object.assign({}, state.editWorkflow.workflow_data.node, <WNode>{
                hooks: state.editWorkflow.workflow_data.node.hooks.filter((hook) => hook.uuid !== action.payload.hook.uuid)
            });
            const editWorkflow: Workflow = {
                ...state.editWorkflow,
                workflow_data: {
                    ...state.editWorkflow.workflow_data,
                    node: rootAsCode,
                }
            };
            ctx.setState({
                ...state,
                editWorkflow: editWorkflow,
                editModeWorkflowChanged: true
            });
            return;
        }

        const root = Object.assign({}, state.workflow.workflow_data.node, <WNode>{
            hooks: state.workflow.workflow_data.node.hooks.filter((hook) => hook.uuid !== action.payload.hook.uuid)
        });
        const workflow: Workflow = {
            ...state.workflow,
            workflow_data: {
                ...state.workflow.workflow_data,
                node: root,
            }
        };

        return ctx.dispatch(new actionWorkflow.UpdateWorkflow({
            projectKey: action.payload.projectKey,
            workflowName: action.payload.workflowName,
            changes: workflow
        })).pipe(tap(() => {
            const stateD = ctx.getState();
            ctx.setState({
                ...stateD,
                hook: null,
            });
        }));
    }

    //  ------- Audit --------- //
    @Action(actionWorkflow.FetchWorkflowAudits)
    fetchAudits(ctx: StateContext<WorkflowStateModel>, action: actionWorkflow.FetchWorkflowAudits) {
        const state = ctx.getState();

        if (!state.workflow || !state.workflow.audits) {
            return ctx.dispatch(new actionWorkflow.GetWorkflow({
                projectKey: action.payload.projectKey,
                workflowName: action.payload.workflowName,
            }));
        }
    }

    @Action(actionWorkflow.RollbackWorkflow)
    rollback(ctx: StateContext<WorkflowStateModel>, action: actionWorkflow.RollbackWorkflow) {
        let auditId = action.payload.auditId;
        return this._http.post<Workflow>(
            `/project/${action.payload.projectKey}/workflows/${action.payload.workflowName}/rollback/${auditId}`,
            {}
        ).pipe(tap((workflow: Workflow) => {
            const state = ctx.getState();
            ctx.setState({
                ...state,
                workflow: workflow,
            });
            return ctx.dispatch(new actionWorkflow.FetchWorkflowAudits({
                projectKey: action.payload.projectKey,
                workflowName: workflow.name
            }));
        }));
    }

    //  ------- Misc --------- //
    @Action(actionWorkflow.FetchAsCodeWorkflow)
    fetchAsCode(ctx: StateContext<WorkflowStateModel>, action: actionWorkflow.FetchAsCodeWorkflow) {
        let params = new HttpParams();
        params = params.append('format', 'yaml');

        return this._http.get<string>(
            `/project/${action.payload.projectKey}/export/workflows/${action.payload.workflowName}`,
            { params, responseType: <any>'text' }
        ).pipe(tap((asCode: string) => {
            const state = ctx.getState();

            ctx.setState({
                ...state,
                workflow: Object.assign({}, state.workflow, <Workflow>{ asCode }),
            });
        }));
    }

    @Action(actionWorkflow.PreviewWorkflow)
    preview(ctx: StateContext<WorkflowStateModel>, action: actionWorkflow.PreviewWorkflow) {
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
            ctx.setState({
                ...state,
                workflow: Object.assign({}, state.workflow, { preview: wf })
            });
        }));
    }

    @Action(actionWorkflow.ExternalChangeWorkflow)
    externalChange(ctx: StateContext<WorkflowStateModel>, action: actionWorkflow.ExternalChangeWorkflow) {
        const state = ctx.getState();
        ctx.setState({
            ...state,
            workflow: Object.assign({}, state.workflow, { externalChange: true }),
        });
    }

    @Action(actionWorkflow.GetWorkflow)
    resync(ctx: StateContext<WorkflowStateModel>, action: actionWorkflow.GetWorkflow) {
        return this._workflowService.getWorkflow(action.payload.projectKey, action.payload.workflowName).pipe(first(),
            tap(wf => {
                const state = ctx.getState();
                let canEdit = wf.permissions.writable;
                let editWorkflow: Workflow;
                let editMode: boolean;

                // Generate hook ref for UI edition
                if (wf && wf.workflow_data && wf.workflow_data.node && wf.workflow_data.node.hooks) {
                    wf.workflow_data.node.hooks.forEach(h => {
                        if (!h.ref) {
                            h.ref = h.uuid;
                        }
                    });
                }

                if (wf.from_repository) {
                    editWorkflow = cloneDeep(wf);
                    editMode = true;
                    // compute ref on node
                    Workflow.getAllNodes(editWorkflow).forEach(n => {
                        if (!n.ref) {
                            n.ref = new Date().getTime().toString();
                        }
                    });
                }
                ctx.setState({
                    ...state,
                    projectKey: action.payload.projectKey,
                    workflow: wf,
                    editWorkflow: editWorkflow,
                    workflowRun: null,
                    workflowNodeRun: null,
                    canEdit: state.workflowRun ? false : canEdit,
                    sidebar: WorkflowSidebarMode.RUNS,
                    editMode: editMode
                });
            }));
    }

    @Action(actionWorkflow.UpdateFavoriteWorkflow)
    updateFavorite(ctx: StateContext<WorkflowStateModel>, action: actionWorkflow.UpdateFavoriteWorkflow) {
        const state = ctx.getState();

        return this._http.post(
            '/user/favorite', {
                type: 'workflow',
                project_key: action.payload.projectKey,
                workflow_name: action.payload.workflowName,
            }
        ).pipe(tap(() => {
            this._navbarService.refreshData();
            if (state.workflow) {
                ctx.setState({
                    ...state,
                    workflow: Object.assign({}, state.workflow, <Workflow>{
                        favorite: !state.workflow.favorite
                    })
                });
            }
            // TODO: dispatch action on global state to update project in list and user state
            // TODO: move this one on user state and just update state here, not XHR
        }));
    }

    @Action(actionWorkflow.UpdateModal)
    updateModal(ctx: StateContext<WorkflowStateModel>, action: actionWorkflow.UpdateModal) {
        const state = ctx.getState();
        let node;
        let hook;
        if (state.node) {
            node = Workflow.getNodeByRef(state.node.ref, action.payload.workflow);
        }
        if (state.hook) {
            hook = Workflow.getHookByRef(state.hook.ref, action.payload.workflow);
        }
        ctx.setState({
            ...state,
            node: node,
            hook: hook
        });
    }

    @Action(actionWorkflow.CleanWorkflowState)
    clearCache(ctx: StateContext<WorkflowStateModel>, action: actionWorkflow.CleanWorkflowState) {
        ctx.setState(getInitialWorkflowState());
    }

    @Action(actionWorkflow.ChangeToRunView)
    changeToRunView(ctx: StateContext<WorkflowStateModel>) {
        const state = ctx.getState();
        ctx.setState({
            ...state,
            workflowNodeRun: null,
            canEdit: false,
            sidebar: WorkflowSidebarMode.RUNS
        });
    }


    @Action(actionWorkflow.GetWorkflowRun)
    getWorkflowRun(ctx: StateContext<WorkflowStateModel>, action: actionWorkflow.GetWorkflowRun) {
        const state = ctx.getState();
        ctx.setState({
            ...state,
            loadingWorkflowRun: true,
        });
        return this._workflowRunService
            .getWorkflowRun(action.payload.projectKey, action.payload.workflowName, action.payload.num).pipe(first(),
                finalize(() => {
                    const stateFin = ctx.getState();
                    ctx.setState({
                        ...stateFin,
                        loadingWorkflowRun: false
                    });
                }),
                tap((wr: WorkflowRun) => {
                    const stateRun = ctx.getState();
                    ctx.setState({
                        ...stateRun,
                        projectKey: action.payload.projectKey,
                        workflowRun: wr
                    });
                    ctx.dispatch(new UpdateWorkflowRunList({ workflowRun: wr }));
                    return wr;
                }));
    }

    @Action(actionWorkflow.DeleteWorkflowRun)
    deleteWorkflowRun(ctx: StateContext<WorkflowStateModel>, action: actionWorkflow.DeleteWorkflowRun) {
        return this._http.delete<null>(
            `/project/${action.payload.projectKey}/workflows/${action.payload.workflowName}/runs/${action.payload.num}`
        ).pipe(tap(() => {
            const state = ctx.getState();

            if (state.listRuns) {
                ctx.setState({
                    ...state,
                    listRuns: state.listRuns.filter((run) => run.num !== action.payload.num),
                });
            }
        }));
    }

    @Action(actionWorkflow.GetWorkflowRuns)
    getWorkflowRuns(ctx: StateContext<WorkflowStateModel>, action: actionWorkflow.GetWorkflowRuns) {
        const state = ctx.getState();
        ctx.setState({
            ...state,
            loadingWorkflowRuns: true,
            filters: action.payload.filters
        });
        return this._workflowRunService
            .runs(action.payload.projectKey,
                action.payload.workflowName,
                action.payload.limit,
                action.payload.offset,
                action.payload.filters).pipe(first(),
                finalize(() => {
                    const stateFin = ctx.getState();
                    ctx.setState({
                        ...stateFin,
                        loadingWorkflowRuns: false
                    });
                }),
                tap((wrs: Array<WorkflowRun>) => {
                    const stateRun = ctx.getState();
                    ctx.setState({
                        ...stateRun,
                        projectKey: action.payload.projectKey,
                        listRuns: wrs,
                    });
                }));

    }

    @Action(actionWorkflow.GetWorkflowNodeRun)
    getWorkflowNodeRun(ctx: StateContext<WorkflowStateModel>, action: actionWorkflow.GetWorkflowNodeRun) {
        const state = ctx.getState();
        ctx.setState({
            ...state,
            loadingWorkflowNodeRun: true,
        });
        return this._workflowRunService
            .getWorkflowNodeRun(action.payload.projectKey, action.payload.workflowName,
                action.payload.num, action.payload.nodeRunID)
            .pipe(first(), finalize(() => {
                const stateFin = ctx.getState();
                ctx.setState({
                    ...stateFin,
                    loadingWorkflowNodeRun: false
                });
            }), tap((wnr: WorkflowNodeRun) => {
                const stateNR = ctx.getState();
                let node = Workflow.getNodeByID(wnr.workflow_node_id, stateNR.workflowRun.workflow);
                ctx.setState({
                    ...stateNR,
                    projectKey: action.payload.projectKey,
                    workflowNodeRun: wnr,
                    node: node,
                    sidebar: WorkflowSidebarMode.RUN_NODE
                });
            }));
    }

    @Action(actionWorkflow.CleanWorkflowRun)
    cleanWorkflowRun(ctx: StateContext<WorkflowStateModel>, action: actionWorkflow.CleanWorkflowRun) {
        const state = ctx.getState();
        ctx.setState({
            ...state,
            workflowRun: null,
            workflowNodeRun: null,
            node: null,
            canEdit: state.workflow.permissions.writable,
            sidebar: WorkflowSidebarMode.RUNS
        });

    }

    @Action(actionWorkflow.UpdateWorkflowRunList)
    updateWorkflowRunList(ctx: StateContext<WorkflowStateModel>, action: actionWorkflow.UpdateWorkflowRunList) {
        const state = ctx.getState();
        let runs = cloneDeep(state.listRuns);
        let index = runs.findIndex(wklwRun => wklwRun.id === action.payload.workflowRun.id);
        if (index === -1) {
            runs.push(action.payload.workflowRun);
            ctx.setState({
                ...state,
                listRuns: runs.sort( (a, b) => b.num - a.num)
            });
            return

        }
        if (runs[index].status === action.payload.workflowRun.status
            && runs[index].tags.length === action.payload.workflowRun.tags.length) {
            return;
        }
        runs[index] = action.payload.workflowRun;
        ctx.setState({
            ...state,
            listRuns: runs
        });
    }

    @Action(actionWorkflow.SelectWorkflowNode)
    selectWorkflowNode(ctx: StateContext<WorkflowStateModel>, action: actionWorkflow.SelectWorkflowNode) {
        const state = ctx.getState();
        if (state.node && state.node.id === action.payload.node.id) {
            return;
        }
        ctx.setState({
            ...state,
            workflowNodeRun: null,
            node: action.payload.node,
            sidebar: WorkflowSidebarMode.RUNS
        });
    }


    @Action(actionWorkflow.SelectWorkflowNodeRun)
    selectWorkflowNodeRun(ctx: StateContext<WorkflowStateModel>, action: actionWorkflow.SelectWorkflowNodeRun) {
        const state = ctx.getState();
        ctx.setState({
            ...state,
            workflowNodeRun: action.payload.workflowNodeRun,
            node: action.payload.node,
            sidebar: WorkflowSidebarMode.RUN_NODE
        });
    }

    @Action(actionWorkflow.SidebarRunsMode)
    sidebarRunsMode(ctx: StateContext<WorkflowStateModel>, action: actionWorkflow.SidebarRunsMode) {
        const state = ctx.getState();
        ctx.setState({
            ...state,
            sidebar: WorkflowSidebarMode.RUNS
        });
    }

    @Action(actionWorkflow.CancelWorkflowEditMode)
    cancelWorkflowEditMode(ctx: StateContext<WorkflowStateModel>, action: actionWorkflow.CancelWorkflowEditMode) {
        const state = ctx.getState();
        let editMode = false;
        if (state.workflow.from_repository) {
            editMode = true;
        }
        let editWorkflow = cloneDeep(state.workflow);
        // compute ref on node
        Workflow.getAllNodes(editWorkflow).forEach(n => {
            if (!n.ref) {
                n.ref = new Date().getTime().toString();
            }
        });
        ctx.setState({
            ...state,
            editModeWorkflowChanged: false,
            editMode: editMode,
            editWorkflow: editWorkflow
        });
    }

    @Action(actionAsCode.ResyncEvents)
    refreshAsCodeEvents(ctx: StateContext<WorkflowStateModel>, _) {
        const state = ctx.getState();
        if (state.workflow) {
            ctx.dispatch(new actionWorkflow
                .GetWorkflow({projectKey: state.projectKey, workflowName: state.workflow.name}));
        }
    }

    @Action(actionAsCode.AsCodeEvent)
    receivedAsCodeEvent(ctx: StateContext<WorkflowStateModel>, action: actionAsCode.AsCodeEvent) {
        if (!action.payload.data || !action.payload.data.workflows) {
            // Event not on a workflow
            return;
        }
        const state = ctx.getState();
        if (!state.workflow) {
            // No workflow in the state
            return;
        }
        if (!action.payload.data.workflows[state.workflow.id]) {
            // Not the same workflow
            return
        }
        ctx.dispatch(new actionWorkflow.GetWorkflow({projectKey: state.projectKey, workflowName: state.workflow.name}));

    }
}
