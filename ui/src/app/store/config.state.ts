import { Injectable } from '@angular/core';
import { Action, Selector, State, StateContext } from '@ngxs/store';
import { APIConfig } from 'app/model/config.service';
import { ConfigService } from 'app/service/services.module';
import { tap } from 'rxjs/operators';
import * as ActionConfig from './config.action';

export class ConfigStateModel {
    public api: APIConfig
}

export function getInitialApplicationsState(): ConfigStateModel {
    return <ConfigStateModel>{};
}

@State<ConfigStateModel>({
    name: 'config',
    defaults: getInitialApplicationsState()
})
@Injectable()
export class ConfigState {

    constructor(
        private _configService: ConfigService
    ) { }

    @Selector()
    static api(state: ConfigStateModel) {
        return state.api;
    }

    @Action(ActionConfig.FetchAPIConfig)
    fetchAPIConfig(ctx: StateContext<ConfigStateModel>) {
        return this._configService.getAPIConfig().pipe(
            tap((res: APIConfig) => {
                ctx.patchState({ api: res });
            })
        );
    }
}
