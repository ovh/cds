import { Injectable } from '@angular/core';
import { Action, createSelector, State, StateContext } from '@ngxs/store';
import { MonitoringService } from 'app/service/monitoring/monitoring.service';
import { GetCDSStatus, UpdateMaintenance } from 'app/store/cds.action';
import { tap } from 'rxjs/operators';

export class CDSStateModel {
    public maintenance: boolean;
}

export function getInitialCDSState(): CDSStateModel {
    return {
        maintenance: false
    };
}

@State<CDSStateModel>({
    name: 'cds',
    defaults: getInitialCDSState()
})
@Injectable()
export class CDSState {

    constructor(private _monitoringService: MonitoringService) { }

    static getCurrentState() {
        return createSelector(
            [CDSState],
            (state: CDSStateModel): CDSStateModel => state
        );
    }

    @Action(UpdateMaintenance)
    updateMaintenance(ctx: StateContext<CDSStateModel>, action: UpdateMaintenance) {
        const state = ctx.getState();
        ctx.setState({
            ...state,
            maintenance: action.enable
        });
    }

    @Action(GetCDSStatus)
    getCDSStatus(ctx: StateContext<CDSStateModel>, _: GetCDSStatus) {
        return this._monitoringService.getStatus().pipe(
            tap(s => {
                let maintenance = 'false';
                let line = s.lines.find(m => {
                    if (m.component === 'Global/Maintenance') {
                        return m;
                    }
                });
                if (line) {
                    maintenance = line.value;
                }
                ctx.dispatch(new UpdateMaintenance(maintenance === 'true'));
            })
        );
    }
}
