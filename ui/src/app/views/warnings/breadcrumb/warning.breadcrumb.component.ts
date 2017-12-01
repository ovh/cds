import {Component, Input, OnInit} from '@angular/core';
import {Project} from '../../../model/project.model';
import {ProjectStore} from '../../../service/project/project.store';
import {Subscription} from 'rxjs/Subscription';
import {AutoUnsubscribe} from '../../../shared/decorator/autoUnsubscribe';
import {List} from 'immutable';
import {WarningStore} from '../../../service/warning/warning.store';
import {WarningUI} from '../../../model/warning.model';
import {WarningService} from '../../../service/warning/warning.service';
import {Router} from '@angular/router';

@Component({
    selector: 'app-warning-breadcrumb',
    templateUrl: './warning.breadcrumb.html',
    styleUrls: ['./warning.breadcrumb.scss']
})
@AutoUnsubscribe()
export class WarningBreadCrumbComponent implements OnInit {

    @Input() project: Project;

    projects: List<Project>;
    projectSub: Subscription;
    warnings: Map<string, WarningUI>;
    warnSub: Subscription;

    warningsCount: Map<string, number>;

    constructor(private _projectStore: ProjectStore, private _warningStore: WarningStore,
                private _warningService: WarningService, private _router: Router) {

    }

    ngOnInit(): void {
        this.warnSub = this._warningStore.getWarnings().subscribe(ws => {
            this.warnings = ws;
            if (this.projects) {
                this.updateCountMap();
            }
        });
        this.projectSub = this._projectStore.getProjectsList().subscribe(ps => {
            this.projects = ps;
            if (this.warnings) {
                this.updateCountMap();
            }
        });
    }

    updateCountMap(): void {
        this.warningsCount = new Map<string, number>();
        this.projects.forEach(p => {
            if (this.warnings.get(p.key)) {
                this.warningsCount.set(p.key, this._warningService.calculateWarningCountForProject(p.key, this.warnings));
            }
        });
        if (!this.project) {
            this.warningsCount.forEach((v, k) => {
                if (v > 0 && !this.project) {
                    this.project = this.projects.find(prj => prj.key === k );
                    this._router.navigate(['/warnings', 'show'], { queryParams: {key: k}});
                }
            });

        }
    }
}
