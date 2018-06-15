import { Component } from '@angular/core';
import {finalize} from 'rxjs/operators';
import {Project} from '../../../model/project.model';
import {ProjectService} from '../../../service/project/project.service';

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

    constructor(private _projectService: ProjectService) {
        this._projectService.getProjects(false)
            .pipe(finalize(() => this.loading = false))
            .subscribe((projects) => {
                this.projects = projects;
                this.filteredProjects = projects;
            });
    }
}
