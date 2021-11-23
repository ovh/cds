import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnChanges, OnInit } from '@angular/core';
import { Store } from '@ngxs/store';
import { IdName, LoadOpts, Project } from 'app/model/project.model';
import { EnrichProject } from 'app/store/project.action';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-project-applications',
    templateUrl: './application.list.html',
    styleUrls: ['./application.list.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ProjectApplicationListComponent implements OnInit, OnChanges {
    @Input() project: Project;

    filter = '';
    loading = true;
    filteredApplications: Array<IdName> = [];

    constructor(
        private _store: Store,
        private _cd: ChangeDetectorRef
    ) { }

    ngOnInit(): void {
        let opts: Array<LoadOpts> = [new LoadOpts('withApplicationNames', 'application_names')];
        this._store.dispatch(new EnrichProject({ projectKey: this.project.key, opts }))
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe();
    }

    ngOnChanges(): void {
        this.filterChanged(this.filter);
    }

    filterChanged(filter: string): void {
        this.filter = filter;
        if (!this.filter) {
            this.filteredApplications = this.project.application_names;
        } else {
            this.filteredApplications = (this.project.application_names ?? []).filter(app =>
                app.name.toLowerCase().indexOf(this.filter.toLowerCase()) !== -1);
        }
        this._cd.markForCheck();
    }
}
