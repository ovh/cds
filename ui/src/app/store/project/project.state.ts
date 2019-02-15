import { HttpClient, HttpParams } from '@angular/common/http';
import { Action, State, StateContext } from '@ngxs/store';
import { LoadOpts, Project } from 'app/model/project.model';
import { tap } from 'rxjs/operators';
import { FetchProject, LoadProject } from './project.action';

export class ProjectStateModel {
    public project: Project;
    public loading: boolean;
}

@State<ProjectStateModel>({
    name: 'project',
    defaults: {
        project: null,
        loading: true,
    }
})
export class ProjectState {

    constructor(private _http: HttpClient) { }


    @Action(LoadProject)
    load(ctx: StateContext<ProjectStateModel>, action: LoadProject) {
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


    @Action(FetchProject)
    fetch(ctx: StateContext<ProjectStateModel>, action: FetchProject) {
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

        return this._http
            .get<Project>('/project/' + action.payload.projectKey, { params })
            .pipe(tap((project) => ctx.dispatch(new LoadProject(project))));
    }

    // @Action(SetFilter)
    // setFilter(ctx: StateContext<TodoItemsStateModel>, action: SetFilter) {
    //     const state = ctx.getState();
    //     ctx.setState({
    //         ...state,
    //         filter: action.payload,
    //     });
    // }

    // @Action(SetSort)
    // setSortField(ctx: StateContext<TodoItemsStateModel>, action: SetSort) {
    //     const state = ctx.getState();
    //     ctx.setState({
    //         ...state,
    //         sort: {
    //             field: action.payload,
    //             ascending: state.sort.field === action.payload ? !state.sort.ascending : true,
    //         }
    //     });
    // }
}
