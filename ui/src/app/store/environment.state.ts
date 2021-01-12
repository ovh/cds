import { HttpClient } from '@angular/common/http';
import { Injectable } from '@angular/core';
import { Action, createSelector, State, StateContext } from '@ngxs/store';
import { Environment } from 'app/model/environment.model';
import { Key } from 'app/model/keys.model';
import { Project } from 'app/model/project.model';
import { Variable } from 'app/model/variable.model';
import { EnvironmentService } from 'app/service/environment/environment.service';
import { cloneDeep } from 'lodash-es';
import { tap } from 'rxjs/operators';
import * as ActionEnvironment from './environment.action';

export class EnvironmentStateModel {
    public environment: Environment;
    public editEnvironment: Environment;
    public currentProjectKey: string;
    public loading: boolean;
    public editMode: boolean;
}

export function getInitialEnvironmentState(): EnvironmentStateModel {
    return {
        environment: null,
        editEnvironment: null,
        currentProjectKey: null,
        loading: true,
        editMode: false
    };
}

@State<EnvironmentStateModel>({
    name: 'environment',
    defaults: getInitialEnvironmentState()
})
@Injectable()
export class EnvironmentState {

    constructor(private _http: HttpClient, private _envService: EnvironmentService) { }

    static currentState() {
        return createSelector(
            [EnvironmentState],
            (state: EnvironmentStateModel) => state
        );
    }

    @Action(ActionEnvironment.AddEnvironment)
    addEnvironment(ctx: StateContext<EnvironmentStateModel>, action: ActionEnvironment.AddEnvironment) {
        return this._http.post<Project>('/project/' + action.payload.projectKey + '/environment', action.payload.environment)
            .pipe(tap((project: Project) => {
                const state = ctx.getState();
                ctx.setState({
                    ...state,
                    currentProjectKey: action.payload.projectKey,
                    environment: project.environments.find(e => e.name === action.payload.environment.name),
                    loading: false,
                    editEnvironment: null,
                    editMode: false
                });
            }));
    }

    @Action(ActionEnvironment.CloneEnvironment)
    cloneEnvironment(ctx: StateContext<EnvironmentStateModel>, action: ActionEnvironment.CloneEnvironment) {
        return this._http.post<Project>(
            '/project/' + action.payload.projectKey + '/environment/' +
            action.payload.environment.name + '/clone/' + action.payload.cloneName,
            null
        ).pipe(tap((project: Project) => {
            const state = ctx.getState();
            ctx.setState({
                ...state,
                currentProjectKey: action.payload.projectKey,
                environment: project.environments.find(e => e.name === action.payload.environment.name),
                loading: false,
                editEnvironment: null,
                editMode: false
            });
        }));
    }

    @Action(ActionEnvironment.UpdateEnvironment)
    update(ctx: StateContext<EnvironmentStateModel>, action: ActionEnvironment.UpdateEnvironment) {
        const stateEditMode = ctx.getState();
        if (stateEditMode.editMode) {
            let envToUpdate = cloneDeep(stateEditMode.editEnvironment);
            envToUpdate.name = action.payload.changes.name;
            envToUpdate.editModeChanged = true;
            return ctx.setState({
                ...stateEditMode,
                editEnvironment: envToUpdate,
            });
        }


        return this._http.put<Project>(
            `/project/${action.payload.projectKey}/environment/${action.payload.environmentName}`,
            action.payload.changes
        ).pipe(tap((p: Project) => {
            const state = ctx.getState();
            ctx.setState({
                ...state,
                environment: p.environments.find(e => e.name === action.payload.changes.name),
                editMode: false,
                editEnvironment: null,
                loading: false
            });
        }));
    }

    @Action(ActionEnvironment.DeleteEnvironment)
    deleteEnvironment(ctx: StateContext<EnvironmentStateModel>, action: ActionEnvironment.DeleteEnvironment) {
        return this._http.delete<Project>('/project/' + action.payload.projectKey + '/environment/' + action.payload.environment.name);
    }

    @Action(ActionEnvironment.FetchEnvironment)
    fetchEnvironment(ctx: StateContext<EnvironmentStateModel>, action: ActionEnvironment.FetchEnvironment) {
        const state = ctx.getState();
        if (state.environment && state.environment.name === action.payload.envName
            && state.currentProjectKey === action.payload.projectKey) {
            return ctx.dispatch(new ActionEnvironment.LoadEnvironment({projectKey: action.payload.projectKey, env: state.environment}));
        }
        return ctx.dispatch(new ActionEnvironment.ResyncEnvironment({ ...action.payload }));
    }

    @Action(ActionEnvironment.LoadEnvironment)
    load(ctx: StateContext<EnvironmentStateModel>, action: ActionEnvironment.LoadEnvironment) {
        const state = ctx.getState();
        let editMode = false;
        if (action.payload.env.from_repository) {
            editMode = true;
        }
        ctx.setState({
            ...state,
            currentProjectKey: action.payload.projectKey,
            environment: action.payload.env,
            editEnvironment: cloneDeep(action.payload.env),
            editMode,
            loading: false,
        });
    }

    @Action(ActionEnvironment.ResyncEnvironment)
    resync(ctx: StateContext<EnvironmentStateModel>, action: ActionEnvironment.ResyncEnvironment) {
        return this._envService.getEnvironment(action.payload.projectKey, action.payload.envName)
            .pipe(tap((environment: Environment) => ctx.dispatch(new ActionEnvironment.LoadEnvironment({projectKey: action.payload.projectKey, env: environment}))));
    }

    // VARIABLES
    @Action(ActionEnvironment.AddEnvironmentVariable)
    addEnvironmentVariable(ctx: StateContext<EnvironmentStateModel>, action: ActionEnvironment.AddEnvironmentVariable) {
        const stateEditMode = ctx.getState();
        if (stateEditMode.editMode) {
            let envToUpdate = cloneDeep(stateEditMode.editEnvironment);
            if (!envToUpdate.variables) {
                envToUpdate.variables = new Array<Variable>();
            }
            delete action.payload.variable.updating;
            delete action.payload.variable.hasChanged;
            envToUpdate.variables.push(action.payload.variable);
            envToUpdate.editModeChanged = true;
            return ctx.setState({
                ...stateEditMode,
                editEnvironment: envToUpdate,
            });
        }
        return this._http.post<Variable>(
            '/project/' + action.payload.projectKey + '/environment/' +
            action.payload.environmentName + '/variable/' + action.payload.variable.name,
            action.payload.variable
        ).pipe(tap((v: Variable) => {
            const state = ctx.getState();
            let env = cloneDeep(state.environment)
            if (!env.variables) {
                env.variables = new Array<Variable>();
            }
            env.variables.push(v);
            ctx.setState({
                ...state,
                environment: env,
            });
        }));
    }

    @Action(ActionEnvironment.DeleteEnvironmentVariable)
    deleteEnvironmentVariable(ctx: StateContext<EnvironmentStateModel>, action: ActionEnvironment.DeleteEnvironmentVariable) {
        const stateEditMode = ctx.getState();
        if (stateEditMode.editMode) {
            let envToUpdate = cloneDeep(stateEditMode.editEnvironment);
            envToUpdate.variables = envToUpdate.variables.filter(e => e.name !== action.payload.variable.name);
            envToUpdate.editModeChanged = true;
            return ctx.setState({
                ...stateEditMode,
                editEnvironment: envToUpdate,
            });
        }
        return this._http.delete<Variable>(
            '/project/' + action.payload.projectKey + '/environment/' +
            action.payload.environmentName + '/variable/' + action.payload.variable.name
        ).pipe(tap(() => {
            const state = ctx.getState();
            let env = cloneDeep(state.environment)
            env.variables = env.variables.filter(va => va.name !== action.payload.variable.name);
            ctx.setState({
                ...state,
                environment: env,
            });
        }));
    }

    @Action(ActionEnvironment.UpdateEnvironmentVariable)
    updateEnvironmentVariable(ctx: StateContext<EnvironmentStateModel>, action: ActionEnvironment.UpdateEnvironmentVariable) {
        const stateEditMode = ctx.getState();
        if (stateEditMode.editMode) {
            delete action.payload.changes.updating;
            delete action.payload.changes.hasChanged;
            let envToUpdate = cloneDeep(stateEditMode.editEnvironment);
            envToUpdate.variables = envToUpdate.variables.map( v => {
                if (v.name === action.payload.variableName) {
                    return action.payload.changes;
                }
                return v;
            });
            envToUpdate.editModeChanged = true;
            return ctx.setState({
                ...stateEditMode,
                editEnvironment: envToUpdate,
            });
        }
        return this._http.put<Variable>(
            '/project/' + action.payload.projectKey + '/environment/' +
            action.payload.environmentName + '/variable/' + action.payload.variableName,
            action.payload.changes
        ).pipe(tap((v: Variable) => {
            const state = ctx.getState();
            let env = cloneDeep(state.environment)
            env.variables = env.variables.map(va => {
                if (va.name !== action.payload.variableName) {
                    return va;
                }
                return v;
            });
            ctx.setState({
                ...state,
                environment: env,
            })
        }));
    }

    @Action(ActionEnvironment.AddEnvironmentKey)
    addEnvironmentKey(ctx: StateContext<EnvironmentStateModel>, action: ActionEnvironment.AddEnvironmentKey) {
        const stateEditMode = ctx.getState();
        if (stateEditMode.editMode) {
            let envToUpdate = cloneDeep(stateEditMode.editEnvironment);
            if (!envToUpdate.keys) {
                envToUpdate.keys = new Array<Key>();
            }
            envToUpdate.keys.push(action.payload.key);
            envToUpdate.editModeChanged = true;
            return ctx.setState({
                ...stateEditMode,
                editEnvironment: envToUpdate,
            });
        }
        return this._http.post<Key>(`/project/${action.payload.projectKey}/environment/${action.payload.envName}/keys`, action.payload.key)
            .pipe(tap((key: Key) => {
                const state = ctx.getState();
                let env = cloneDeep(state.environment)
                if (!env.keys) {
                    env.keys = new Array<Key>();
                }
                env.keys.push(key);
                ctx.setState({
                    ...state,
                    environment: env
                });
            }));
    }

    @Action(ActionEnvironment.DeleteEnvironmentKey)
    deleteEnvironmentKey(ctx: StateContext<EnvironmentStateModel>, action: ActionEnvironment.DeleteEnvironmentKey) {
        const stateEditMode = ctx.getState();
        if (stateEditMode.editMode) {
            let envToUpdate = cloneDeep(stateEditMode.editEnvironment);
            envToUpdate.keys = envToUpdate.keys.filter(e => e.name !== action.payload.key.name);
            envToUpdate.editModeChanged = true;
            return ctx.setState({
                ...stateEditMode,
                editEnvironment: envToUpdate,
            });
        }
        return this._http.delete<null>('/project/' + action.payload.projectKey +
            '/environment/' + action.payload.envName + '/keys/' + action.payload.key.name)
            .pipe(tap(() => {
                const state = ctx.getState();
                let env = cloneDeep(state.environment)
                env.keys = env.keys.filter(k => k.name !== action.payload.key.name);
                ctx.setState({
                    ...state,
                    environment: env
                });
            }));
    }

    @Action(ActionEnvironment.CleanEnvironmentState)
    cleanEnvironmentState(ctx: StateContext<EnvironmentStateModel>, _: ActionEnvironment.DeleteEnvironmentKey) {
        ctx.setState(getInitialEnvironmentState())  ;
    }

}
