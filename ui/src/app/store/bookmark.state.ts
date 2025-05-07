import { Injectable } from '@angular/core';
import { Action, Selector, State, StateContext } from '@ngxs/store';
import { Bookmark } from 'app/model/bookmark.model';
import { tap } from 'rxjs';
import { HttpClient } from '@angular/common/http';

export class BookmarkLoad {
    static readonly type = '[Event] Bookmark load';
    constructor() { }
}

export class BookmarkCreate {
    static readonly type = '[Event] Bookmark create';
    constructor(public payload: Bookmark) { }
}

export class BookmarkDelete {
    static readonly type = '[Event] Bookmark delete';
    constructor(public payload: Bookmark) { }
}

export class BookmarkStateModel {
    all: Array<Bookmark>;
    loading: string;
}

@State<BookmarkStateModel>({
    name: 'bookmark',
    defaults: {
        all: [],
        loading: null
    }
})
@Injectable()
export class BookmarkState {

    constructor(
        private _http: HttpClient
    ) { }

    @Selector()
    static state(state: BookmarkStateModel) {
        return state;
    }

    @Action(BookmarkLoad)
    load(ctx: StateContext<BookmarkStateModel>, action: BookmarkLoad) {
        const state = ctx.getState();

        ctx.setState({
            ...state,
            loading: 'all',
        });

        return this._http.get<Array<Bookmark>>('/bookmark').pipe(
            tap((res) => {
                ctx.setState({
                    ...state,
                    loading: null,
                    all: res
                });
            })
        )
    }

    @Action(BookmarkCreate)
    create(ctx: StateContext<BookmarkStateModel>, action: BookmarkCreate) {
        const state = ctx.getState();

        ctx.setState({
            ...state,
            loading: action.payload.type + action.payload.id,
        });

        return this._http.post<Bookmark>('/bookmark', action.payload).pipe(
            tap((b) => {
                ctx.setState({
                    ...state,
                    loading: null,
                    all: state.all.concat(b).sort((a, b) => a.type < b.type || a.label < b.label ? -1 : 1)
                });
            })
        )
    }

    @Action(BookmarkDelete)
    delete(ctx: StateContext<BookmarkStateModel>, action: BookmarkDelete) {
        const state = ctx.getState();

        ctx.setState({
            ...state,
            loading: action.payload.type + action.payload.id,
        });

        return this._http.delete(`/bookmark/${action.payload.type}/${encodeURIComponent(action.payload.id)}`).pipe(
            tap(() => {
                ctx.setState({
                    ...state,
                    loading: null,
                    all: state.all.filter((b) => (!(b.type === action.payload.type && b.id === action.payload.id)))
                });
            })
        )
    }

}
