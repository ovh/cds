import { Injectable } from '@angular/core';
import { Action, Selector, State, StateContext } from '@ngxs/store';
import { AuthConsumer, AuthCurrentConsumerResponse, AuthSession } from 'app/model/authentication.model';
import { AuthentifiedUser, AuthSummary } from 'app/model/user.model';
import { AuthenticationService } from 'app/service/authentication/authentication.service';
import { UserService } from 'app/service/user/user.service';
import { throwError } from 'rxjs';
import { catchError, finalize, tap } from 'rxjs/operators';
import * as ActionAuthentication from './authentication.action';

export class AuthenticationStateModel {
    public error: any;
    public summary: AuthSummary;
    public user: AuthentifiedUser;
    public loading: boolean;
}

export function getInitialApplicationsState(): AuthenticationStateModel {
    return <AuthenticationStateModel>{};
}

@State<AuthenticationStateModel>({
    name: 'authentication',
    defaults: getInitialApplicationsState()
})
@Injectable()
export class AuthenticationState {

    constructor(
        private _userService: UserService,
        private _authenticationService: AuthenticationService
    ) { }

    @Selector()
    static user(state: AuthenticationStateModel) {
        return state.user;
    }

    @Selector()
    static error(state: AuthenticationStateModel) {
        return state.error;
    }

    @Selector()
    static summary(state: AuthenticationStateModel) {
        return state.summary;
    }

    @Action(ActionAuthentication.FetchCurrentUser)
    fetchCurrentUser(ctx: StateContext<AuthenticationStateModel>, action: ActionAuthentication.FetchCurrentUser) {
        ctx.patchState({ loading: true });

        return this._userService.getMe().pipe(
            finalize(() => {
                ctx.patchState({ loading: false });
            }),
            tap((me: AuthentifiedUser) => {
                ctx.patchState({
                    user: me,
                    error: null
                });
            }),
            catchError(err => {
                ctx.patchState({
                    user: null,
                    error: err
                })
                return throwError(err);
            })
        );
    }

    @Action(ActionAuthentication.FetchCurrentAuth)
    fetchCurrentAuth(ctx: StateContext<AuthenticationStateModel>, action: ActionAuthentication.FetchCurrentAuth) {
        ctx.patchState({ loading: true });

        return this._authenticationService.getMe().pipe(
            finalize(() => {
                ctx.patchState({ loading: false });
            }),
            tap((res: AuthCurrentConsumerResponse) => {
                let s = new AuthSummary();
                s.user = res.user;
                s.consumer = res.consumer;
                s.session = res.session;
                ctx.patchState({
                    summary: s,
                    error: null
                });
            }),
            catchError(err => {
                ctx.patchState({
                    summary: null,
                    error: err
                })
                return throwError(err);
            })
        );
    }

    @Action(ActionAuthentication.SignoutCurrentUser)
    signoutCurrentUser(ctx: StateContext<AuthenticationStateModel>, action: ActionAuthentication.FetchCurrentUser) {
        ctx.patchState({ loading: true });

        return this._authenticationService.signout().pipe(
            finalize(() => {
                ctx.patchState({ loading: false })
            }),
            tap(_ => {
                ctx.patchState({
                    user: null,
                    error: null
                });
            }),
            catchError(err => {
                ctx.patchState({
                    user: null,
                    error: err
                })
                return throwError(err);
            })
        );
    }
}
