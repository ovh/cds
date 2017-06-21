import {Injectable} from '@angular/core';
import {WarningUI} from '../../model/warning.model';

@Injectable()
export class WarningService {

    calculateWarningCountForProject(k: string, warnings: Map<string, WarningUI>): number {
        let warningsCount = 0;
        if (k && warnings.get(k)) {
            warningsCount += warnings.get(k).variables.length;

            warnings.get(k).pipelines.forEach((v) => {
                warningsCount += v.jobs.length + v.parameters.length;
            });
            warnings.get(k).applications.forEach((v) => {
                warningsCount += v.variables.length + v.actions.length;
            });
        }
        return warningsCount;
    }

    calculateWarningCountForCurrentRoute(currentRoute: {}, warnings: Map<string, WarningUI>): number {
        if (!currentRoute || !warnings) {
            return;
        }

        let warningsCount = 0;
        let k = currentRoute['key'];

        if (k && warnings.get(k)) {
            warningsCount += warnings.get(k).variables.length;

            // If on pipeline page
            let pip = currentRoute['pipName'];
            if (pip && warnings.get(k).pipelines.get(pip)) {
                warningsCount += warnings.get(k).pipelines.get(pip).jobs.length
                    + warnings.get(k).pipelines.get(pip).parameters.length;
            }

            // If on application page
            let app = currentRoute['appName'];
            if (app && warnings.get(k).applications.get(app)) {
                warningsCount += warnings.get(k).applications.get(app).variables.length
                    + warnings.get(k).applications.get(app).actions.length;
            }

            // On project page
            if (!currentRoute['appName'] && !currentRoute['pipName']) {
                warnings.get(k).pipelines.forEach((v) => {
                    warningsCount += v.jobs.length + v.parameters.length;
                });
                warnings.get(k).applications.forEach((v) => {
                    warningsCount += v.variables.length + v.actions.length;
                });
            }
        }
        return warningsCount;
    }
}
