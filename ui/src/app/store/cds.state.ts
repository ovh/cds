import { Action, createSelector, State, StateContext } from '@ngxs/store';
import { MonitoringService } from 'app/service/monitoring/monitoring.service';
import { GetCDSStatus, UpdateMaintenance } from 'app/store/cds.action';


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
export class CDSState {

    static getCurrentState() {
        return createSelector(
            [CDSState],
            (state: CDSStateModel): CDSStateModel => state
        );
    }

    constructor(private _monitoringService: MonitoringService) { }


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
        this._monitoringService.getStatus().subscribe(s => {
            let maintenance = s.lines.find(m => {
                if (m.component === 'Global/Maintenance') {
                    return m
                }
            }).value;
            ctx.dispatch(new UpdateMaintenance(maintenance === 'true'));
        });
    }
}
