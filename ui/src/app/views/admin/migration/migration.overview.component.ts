import {Component, OnInit} from '@angular/core';
import {TranslateService} from '@ngx-translate/core';
import {Project} from '../../../model/project.model';
import {ProjectStore} from '../../../service/project/project.store';
import {ToastService} from '../../../shared/toast/ToastService';

@Component({
    selector: 'app-migration-overview',
    templateUrl: './migration-overview.html',
    styleUrls: ['./migration-overview.scss']
})
export class MigrationOverviewComponent implements OnInit {

    projects: Array<Project>;
    mapNotBegun: Map<string, Project>;
    mapStarted: Map<string, ProgressData>;
    keysDone: Array<string>;
    keysStarted: Array<string>;
    keysNotBegun: Array<string>;

    selectedTab = 'NOT_BEGUN';

    constructor(private _projectStore: ProjectStore, private _translate: TranslateService, private _toast: ToastService) {
    }

    ngOnInit() {
        this._projectStore.getProjectsList().subscribe(ps => {
            this.projects = ps.toArray();
            this.mapNotBegun = new Map<string, Project>();
            this.mapStarted = new Map<string, ProgressData>();
            this.keysDone = new Array<string>();
            this.keysStarted = new Array<string>();
            this.keysNotBegun = new Array<string>();
            this.projects.forEach(proj => {
                let data = new ProgressData();
                if (proj.applications && proj.applications.length > 0) {
                    data.total = proj.applications.length;
                    proj.applications.forEach( app => {
                        if (app.workflow_migration === 'DONE') {
                            data.progress++;
                        } else if (app.workflow_migration === 'STARTED' || app.workflow_migration === 'CLEANING') {
                            data.progress += 0.5;
                        }
                    });
                    if (data.progress === data.total) {
                        this.keysDone.push(proj.key);
                    } else if (proj.workflow_migration === 'NOT_BEGUN') {
                        this.mapNotBegun.set(proj.key, proj);
                        this.keysNotBegun.push(proj.key);
                    } else {
                        this.keysStarted.push(proj.key);
                        this.mapStarted.set(proj.key, data);
                    }
                } else {
                    data.total = 1;
                    data.progress = 1;
                    this.keysDone.push(proj.key);
                }
            });
        });
    }

    showTab(tab: string): void {
        this.selectedTab = tab;
    }

    enableProject(key: string): void {
        let p = this.mapNotBegun.get(key);
        p.workflow_migration = 'STARTED';
        this._projectStore.updateProject(p).subscribe(() => {
            this._toast.success('', this._translate.instant('project_updated'));
        });
    }
}

export class ProgressData {
    total: number;
    progress: number;

    constructor() {
        this.progress = 0;
    }
}
