import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnChanges, OnInit } from '@angular/core';
import { Store } from '@ngxs/store';
import { IdName, LoadOpts, Project } from 'app/model/project.model';
import { EnrichProject } from 'app/store/project.action';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-project-pipelines',
    templateUrl: './pipeline.list.html',
    styleUrls: ['./pipeline.list.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ProjectPipelinesComponent implements OnInit, OnChanges {
    @Input() project: Project;

    filter = '';
    loading = true;
    filteredPipelines: Array<IdName> = [];

    constructor(
        private store: Store,
        private _cd: ChangeDetectorRef
    ) { }

    ngOnInit(): void {
        let opts: Array<LoadOpts> = [new LoadOpts('withPipelineNames', 'pipeline_names')];
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
            this.filteredPipelines = this.project.pipeline_names;
        } else {
            this.filteredPipelines = (this.project.pipeline_names ?? []).filter(pip =>
                pip.name.toLowerCase().indexOf(this.filter.toLowerCase()) !== -1);
        }
        this._cd.markForCheck();
    }
}
