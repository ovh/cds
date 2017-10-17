import {Component, OnInit} from '@angular/core';
import {ProjectStore} from '../../../service/project/project.store';
import {Project} from '../../../model/project.model';

@Component({
    selector: 'app-migration-overview',
    templateUrl: './migration-overview.html',
    styleUrls: ['./migration-overview.scss']
})
export class MigrationOverviewComponent implements OnInit {

    projects: Array<Project>;
    mapStarted: Map<string, ProgressData>;
    keysDone: Array<string>;
    keysStarted: Array<string>;
    keysNotBegun: Array<string>;

    selectedTab = 'NOT_BEGUN';

    constructor(private _projectStore: ProjectStore) {
    }

    ngOnInit() {
        this._projectStore.getProjectsList().subscribe(p => {
            this.projects = p.toArray();
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
                        } else if (app.workflow_migration === 'STARTED') {
                            data.progress += 0.5;
                        }
                    });
                    if (data.progress === 0) {
                        this.keysNotBegun.push(proj.key);
                    } else if (data.progress === data.total) {
                        this.keysDone.push(proj.key);
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
}

export class ProgressData {
    total: number;
    progress: number;

    constructor() {
        this.progress = 0;
    }
}
