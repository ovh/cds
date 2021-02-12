import { FeatureNames } from 'app/service/feature/feature.service';
export class FeatureEnabledResponse {
    name: FeatureNames;
    enabled: boolean;
    exists: boolean;
}
