import { Component } from '@angular/core';
import {Project} from '../../../model/project.model';
import {ProjectStore} from '../../../service/project/project.store';

@Component({
    selector: 'app-project-list',
    templateUrl: './project.list.component.html',
    styleUrls: ['./project.list.component.scss']
})
export class ProjectListComponent {
    projects: Array<Project> = [];
    filteredProjects: Array<Project> = [];
    loading = true;

    set filter(filter: string) {
        let filterLower = filter.toLowerCase();
        this.filteredProjects = this.projects.filter((proj) => {
          return proj.name.toLowerCase().indexOf(filterLower) !== -1 || proj.key === filter;
        });
    }

    constructor(private _projectStore: ProjectStore) {
        // TODO NEVER CALL ProjectSERVICE
        this._projectStore.getProjects()
            .subscribe((projects) => {
                if (projects) {
                    this.loading = false;
                    this.projects = projects.toArray();
                    this.filteredProjects = projects.toArray();
                }
            });
    }
}
