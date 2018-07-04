import { Component } from '@angular/core';
import {Subscription} from 'rxjs/Subscription';
import {Project} from '../../../model/project.model';
import {ProjectStore} from '../../../service/project/project.store';
import {AutoUnsubscribe} from '../../../shared/decorator/autoUnsubscribe';

@Component({
    selector: 'app-project-list',
    templateUrl: './project.list.component.html',
    styleUrls: ['./project.list.component.scss']
})
@AutoUnsubscribe()
export class ProjectListComponent {
    projects: Array<Project> = [];
    filteredProjects: Array<Project> = [];
    loading = true;

    projectSub: Subscription;

    set filter(filter: string) {
        let filterLower = filter.toLowerCase();
        this.filteredProjects = this.projects.filter((proj) => {
          return proj.name.toLowerCase().indexOf(filterLower) !== -1 || proj.key === filter;
        });
    }

    constructor(private _projectStore: ProjectStore) {
        this.projectSub = this._projectStore.getProjects()
            .subscribe((projects) => {
                if (projects) {
                    this.loading = false;
                    this.projects = projects.toArray();
                    this.filteredProjects = projects.toArray();
                }
            });
    }
}
