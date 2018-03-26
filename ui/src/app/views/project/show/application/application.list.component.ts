import { Component, Input } from '@angular/core';
import {Project, IdName} from '../../../../model/project.model';

@Component({
    selector: 'app-project-applications',
    templateUrl: './application.list.html',
    styleUrls: ['./application.list.scss']
})
export class ProjectApplicationListComponent {

    @Input()
    set project(project: Project) {
      this._project = project;
      if (project.application_names) {
        let filter = this.filter.toLowerCase();
        this.filteredApplications = project.application_names.filter((app) => app.name.toLowerCase().indexOf(filter) !== -1);
      }
    }
    get project(): Project {
      return this._project;
    }

    set filter(filter: string) {
      this._filter = filter;
      if (this.project.application_names) {
        let filterLower = filter.toLowerCase();
        this.filteredApplications = this.project.application_names.filter((app) => app.name.toLowerCase().indexOf(filterLower) !== -1);
      }
    }
    get filter(): string {
      return this._filter;
    }

    _project: Project;
    _filter = '';

    filteredApplications: Array<IdName> = [];

    constructor() { }
}
