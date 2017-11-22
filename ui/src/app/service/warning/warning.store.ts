import {Injectable} from '@angular/core';
import {Observable} from 'rxjs/Observable';
import {BehaviorSubject} from 'rxjs/BehaviorSubject'
import {
    WarningAPI, WarningUI, WarningPipeline, WarningApplication,
    WarningEnvironment
} from '../../model/warning.model';


@Injectable()
export class WarningStore {

    // List of all project. Use by Navbar
    private _warningCache: BehaviorSubject<Map<string, WarningUI>> = new BehaviorSubject(new Map<string, WarningUI>());

    /**
     * Get a WarningsUI Observable
     * @returns {Observable<WarningsUI>}
     */
    getWarnings(): Observable<Map<string, WarningUI>> {
        return new Observable<Map<string, WarningUI>>(fn => this._warningCache.subscribe(fn));
    }

    /**
     * Update warning Store.
     */
    updateWarnings(warnings: WarningAPI[]): void {
        let updatedWarnings: Map<string, WarningUI> = new Map<string, WarningUI>();
        warnings.forEach(function (w) {
            if (w.project && w.project.key) {

                if (!updatedWarnings.get(w.project.key)) {
                    updatedWarnings.set(w.project.key, new WarningUI());
                }
                let warningUI = updatedWarnings.get(w.project.key);

                // If warning on pipeline
                if (w.pipeline && w.pipeline.name) {
                    if (!warningUI.pipelines.get(w.pipeline.name)) {
                        warningUI.pipelines.set(w.pipeline.name, new WarningPipeline());
                    }
                }

                // If warning on application
                if (w.application && w.application.name) {
                    if (!warningUI.applications.get(w.application.name)) {
                        warningUI.applications.set(w.application.name, new WarningApplication());
                    }
                }

                // If warning on an action
                if (w.action && w.action.name) {
                    warningUI.pipelines.get(w.pipeline.name).jobs.push(w);

                    // If action link to an application
                    if (w.application && w.application.name) {
                        warningUI.applications.get(w.application.name).actions.push(w);
                    }
                }

                // If Warning on environment
                if (w.environment && w.environment.name) {
                    if (!warningUI.environments.get(w.environment.name)) {
                        warningUI.environments.set(w.environment.name, new WarningEnvironment());
                    }
                }

            }
        });
        this._warningCache.next(updatedWarnings);
    }
}
