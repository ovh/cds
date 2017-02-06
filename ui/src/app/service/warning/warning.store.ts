import {Injectable} from '@angular/core';
import {BehaviorSubject, Observable} from 'rxjs/Rx';
import {
    WarningsUI, WarningAPI, WarningUI, WarningPipeline, WarningApplication,
    WarningEnvironment
} from '../../model/warning.model';


@Injectable()
export class WarningStore {

    // List of all project. Use by Navbar
    private _warningCache: BehaviorSubject<WarningsUI> = new BehaviorSubject(null);

    /**
     * Get a WarningsUI Observable
     * @returns {Observable<WarningsUI>}
     */
    getWarnings(): Observable<WarningsUI> {
        return new Observable<WarningsUI>(fn => this._warningCache.subscribe(fn));
    }

    /**
     * Update warning Store.
     */
    updateWarnings(warnings: WarningAPI[]): void {
        let updatedWarnings: WarningsUI = new WarningsUI();
        warnings.forEach(function (w) {
            if (w.project && w.project.key) {

                // if not in list, create new warning on the current project
                if (!updatedWarnings[w.project.key]) {
                    updatedWarnings[w.project.key] = new WarningUI();
                }

                let warningUI = updatedWarnings[w.project.key];

                // If warning on pipeline
                if (w.pipeline && w.pipeline.name) {
                    if (!warningUI.pipelines[w.pipeline.name]) {
                        warningUI.pipelines[w.pipeline.name] = new WarningPipeline();
                    }
                }

                // If warning on application
                if (w.application && w.application.name) {
                    if (!warningUI.applications[w.application.name]) {
                        warningUI.applications[w.application.name] = new WarningApplication();
                    }
                }

                // If warning on an action
                if (w.action && w.action.name) {
                    warningUI.pipelines[w.pipeline.name].jobs.push(w);

                    // If action link to an application
                    if (w.application && w.application.name) {
                        warningUI.applications[w.application.name].actions.push(w);
                    }
                }

                // If Warning on environment
                if (w.environment && w.environment.name) {
                    if (!warningUI.environments[w.environment.name]) {
                        warningUI.environments[w.environment.name] = new WarningEnvironment();
                    }
                }

            }
        });
        this._warningCache.next(updatedWarnings);
    }
}
