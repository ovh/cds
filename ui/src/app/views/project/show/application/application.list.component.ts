import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnInit } from '@angular/core';
import { Store } from '@ngxs/store';
import { IdName, LoadOpts, Project } from 'app/model/project.model';
import { ResyncProject } from 'app/store/project.action';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-project-applications',
    templateUrl: './application.list.html',
    styleUrls: ['./application.list.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ProjectApplicationListComponent implements OnInit {

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
    loading = true;

    filteredApplications: Array<IdName> = [];

    constructor(private store: Store, private _cd: ChangeDetectorRef) {}

    ngOnInit(): void {
        let opts: Array<LoadOpts> = [new LoadOpts('withApplicationNames', 'application_names')];
        this.store.dispatch(new ResyncProject({ projectKey: this.project.key, opts }))
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe();
    }
}
