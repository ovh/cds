import { Action, Selector, State, StateContext } from '@ngxs/store';
import { AuthentifiedUser } from 'app/model/user.model';
import { UserService } from 'app/service/services.module';
import { tap } from 'rxjs/operators';
import * as ActionAuthentication from './authentication.action';

export class AuthenticationStateModel {
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
export class AuthenticationState {
  @Selector()
  static user(state: AuthenticationStateModel) {
    return state.user;
  }

  constructor(
    private _userService: UserService
  ) { }

  @Action(ActionAuthentication.FetchCurrentUser)
  fetchCurrentUser(ctx: StateContext<AuthenticationStateModel>, action: ActionAuthentication.FetchCurrentUser) {
    const state = ctx.getState();

    ctx.setState({
      ...state,
      loading: true
    });

    return this._userService.getMe().pipe(tap((me: AuthentifiedUser) => {
      ctx.setState({
        ...state,
        user: me,
        loading: false
      });
    }));
  }
}
