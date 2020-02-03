import { ChangeDetectionStrategy, Component, Input } from '@angular/core';
import { IdName, Project } from 'app/model/project.model';

@Component({
    selector: 'app-project-environments',
    templateUrl: './environment.list.html',
    styleUrls: ['./environment.list.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ProjectEnvironmentListComponent {

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
