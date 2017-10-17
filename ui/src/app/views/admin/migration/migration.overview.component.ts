import { Component } from '@angular/core';
import {ProjectStore} from '../../../service/project/project.store';
import {Project} from '../../../model/project.model';

@Component({
    selector: 'app-migration-overview',
    templateUrl: './migration-overview.html',
    styleUrls: ['./migration-overview.scss']
})
export class MigrationOverviewComponent {

    projects: Array<Project>;
    mapProgress: Map<string, ProgressData>;
    keys: Array<string>;

    constructor(private _projectStore: ProjectStore) {
        this._projectStore.getProjectsList().subscribe(p => {
            this.projects = p.toArray();
            this.mapProgress = new Map<string, ProgressData>();
            this.keys = new Array<string>();
            this.projects.forEach(proj => {
               let data = new ProgressData();
               if (proj.applications) {
                   data.total = proj.applications.length;
                   proj.applications.forEach( app => {
                      if (app.workflow_migration === 'DONE') {
                          data.progress++;
                      } else if (app.workflow_migration === 'STARTED') {
                          data.progress += 0.5;
                      }
                   });
               } else {
                   data.total = 1;
                   data.progress = 1;
               }
               this.keys.push(proj.key);
               this.mapProgress.set(proj.key, data);
            });
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
