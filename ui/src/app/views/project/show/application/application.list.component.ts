import { Component, Input } from '@angular/core';
import {IdName, Project} from '../../../../model/project.model';
import {Warning} from '../../../../model/warning.model';

@Component({
    selector: 'app-project-applications',
    templateUrl: './application.list.html',
    styleUrls: ['./application.list.scss']
})
export class ProjectApplicationListComponent {

    warnMap: Map<string, Array<Warning>>;
    @Input('warnings')
    set warnings(data: Array<Warning>) {
        if (data) {
            this.warnMap = new Map<string, Array<Warning>>();
            data.forEach(w => {
                let arr = this.warnMap.get(w.application_name);
                if (!arr) {
                    arr = new Array<Warning>();
                }
                arr.push(w);
                this.warnMap.set(w.application_name, arr);
            });
        }
    }

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
