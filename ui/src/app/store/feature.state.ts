import { Injectable } from '@angular/core';
import { Action, Selector, State, StateContext } from '@ngxs/store';
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
    name: 'event',
    defaults: {
        features: [],
    }
})
@Injectable()
export class FeatureState {
    constructor() { }

    @Selector()
    static features(state: FeatureStateModel) {
        return state.features
    }

    @Action(actionFeature.AddFeatureResult)
    addFeatureResult(ctx: StateContext<FeatureStateModel>, action: actionFeature.AddFeatureResult) {
        const state = ctx.getState();

        let feature = state.features.find(f => f.key === action.payload.key);
        if (!feature) {
            feature = <FeatureResults>{ key: action.payload.key, results: [] };
        }
        feature.results.filter(r => r.paramString !== action.payload.result.paramString).push(action.payload.result);

        ctx.setState({
            ...state,
            features: state.features.filter(f => f.key !== feature.key).concat(feature)
        });
    }
}
