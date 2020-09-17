import { Injectable } from '@angular/core';
import { Action, createSelector, State, StateContext } from '@ngxs/store';
import * as actionFeature from './feature.action';

export class FeatureResults {
    key: string;
    results: Array<FeatureResult>;
}

export class FeatureResult {
    paramString: string;
    enabled: boolean;
}

export class FeatureStateModel {
    public features: Array<FeatureResults>;
}
@State<FeatureStateModel>({
    name: 'feature',
    defaults: {
        features: [],
    }
})
@Injectable()
export class FeatureState {
    constructor() { }

    static feature(key: string) {
        return createSelector([FeatureState], (state: FeatureStateModel) => {
            return state.features.filter(f => f.key === key);
        });
    }

    @Action(actionFeature.AddFeatureResult)
    addFeatureResult(ctx: StateContext<FeatureStateModel>, action: actionFeature.AddFeatureResult) {
        const state = ctx.getState();

        let feature = state.features.find(f => f.key === action.payload.key);
        if (!feature) {
            feature = <FeatureResults>{ key: action.payload.key, results: [] };
        }
        feature.results = feature.results.filter(r => r.paramString !== action.payload.result.paramString).concat(action.payload.result);

        ctx.setState({
            ...state,
            features: state.features.filter(f => f.key !== feature.key).concat(feature)
        });
    }
}
