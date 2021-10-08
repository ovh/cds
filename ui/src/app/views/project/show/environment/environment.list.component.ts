import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnChanges, OnInit } from '@angular/core';
import { Store } from '@ngxs/store';
import { IdName, LoadOpts, Project } from 'app/model/project.model';
import { EnrichProject } from 'app/store/project.action';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-project-environments',
    templateUrl: './environment.list.html',
    styleUrls: ['./environment.list.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ProjectEnvironmentListComponent implements OnInit, OnChanges {
    @Input() project: Project;

    filter = '';
    loading = true;
    filteredEnvironments: Array<IdName> = [];

    constructor(
        private store: Store,
        private _cd: ChangeDetectorRef
    ) { }

    ngOnInit(): void {
        let opts: Array<LoadOpts> = [new LoadOpts('withEnvironmentNames', 'environment_names')];
        this.store.dispatch(new EnrichProject({ projectKey: this.project.key, opts }))
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
            this.filteredEnvironments = this.project.environment_names;
        } else {
            this.filteredEnvironments = (this.project.environment_names ?? []).filter(env =>
                env.name.toLowerCase().indexOf(this.filter.toLowerCase()) !== -1);
        }
        this._cd.markForCheck();
    }
}
