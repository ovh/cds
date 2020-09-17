import { FeatureResult } from './feature.state';

export class FeaturePayload {
    key: string;
    result: FeatureResult;
}

export class AddFeatureResult {
    static readonly type = '[Feature] Add feature result';
    constructor(public payload: FeaturePayload) { }
}
