import { Component, Input } from '@angular/core';
import { IdName, Project } from 'app/model/project.model';
import { Warning } from 'app/model/warning.model';

@Component({
    selector: 'app-project-environments',
    templateUrl: './environment.list.html',
    styleUrls: ['./environment.list.scss']
})
export class ProjectEnvironmentListComponent {

    warnMap: Map<string, Array<Warning>>;
    @Input('warnings')
    set warnings(data: Array<Warning>) {
        if (data) {
            this.warnMap = new Map<string, Array<Warning>>();
            data.forEach(w => {
                let arr = this.warnMap.get(w.environment_name);
                if (!arr) {
                    arr = new Array<Warning>();
                }
                arr.push(w);
                this.warnMap.set(w.environment_name, arr);
            });
        }
    }

    @Input()
    set project(project: Project) {
        this._project = project;
        if (project.environment_names) {
            let filter = this.filter.toLowerCase();
            this.filteredEnvironments = project.environment_names.filter((env) => env.name.toLowerCase().indexOf(filter) !== -1);
        }
    }
    get project(): Project {
        return this._project;
    }

    set filter(filter: string) {
        this._filter = filter;
        if (this.project.environment_names) {
            let filterLower = filter.toLowerCase();
            this.filteredEnvironments = this.project.environment_names.filter((env) => env.name.toLowerCase().indexOf(filterLower) !== -1);
        }
    }
    get filter(): string {
        return this._filter;
    }

    _project: Project;
    _filter = '';

    filteredEnvironments: Array<IdName> = [];

    constructor() {

    }
}
