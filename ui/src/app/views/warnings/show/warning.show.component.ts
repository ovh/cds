import {Component} from '@angular/core';
import {ActivatedRoute} from '@angular/router';
import {WarningStore} from '../../../service/warning/warning.store';
import {WarningUI} from '../../../model/warning.model';
import {AutoUnsubscribe} from '../../../shared/decorator/autoUnsubscribe';
import {Subscription} from 'rxjs/Subscription';
import {ProjectStore} from '../../../service/project/project.store';
import {Project} from '../../../model/project.model';
import {first} from 'rxjs/operators';

@Component({
    selector: 'app-warning-show',
    templateUrl: './warning.show.html',
    styleUrls: ['./warning.show.scss']
})
@AutoUnsubscribe()
export class WarningShowComponent {

    warnings: WarningUI;
    allwarnings: Map<string, WarningUI>;
    warningsSubscription: Subscription;
    key: string;
    appName: string;
    pipName: string;
    project: Project;

    constructor(private _activatedRoute: ActivatedRoute, private _warningStore: WarningStore, private _projectStore: ProjectStore) {
        this._activatedRoute.queryParams.subscribe(q => {
            if (q['key'] && q['key'] !== this.key) {
                this.loadProject(q['key']);
            }
            this.key = q['key'];
            this.appName = q['appName'];
            this.pipName = q['pipName'];
            if (this.allwarnings) {
                this.warnings = this.allwarnings.get(this.key);
            }
        });
        this.warningsSubscription = this._warningStore.getWarnings().subscribe(ws => {
            this.allwarnings = ws;
            this.warnings = ws.get(this.key);
        });
    }

    loadProject(key: string): void {
        this._projectStore.getProjectResolver(key, []).pipe(first()).subscribe(p => {
            this.project = p;
        });
    }
}
