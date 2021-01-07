import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnInit } from '@angular/core';
import { Store } from '@ngxs/store';
import { IdName, LoadOpts, Project } from 'app/model/project.model';
import { ResyncProject } from 'app/store/project.action';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-project-pipelines',
    templateUrl: './pipeline.list.html',
    styleUrls: ['./pipeline.list.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ProjectPipelinesComponent implements OnInit {

    @Input()
    set project(project: Project) {
        this._project = project;
        if (project.pipeline_names) {
            let filter = this.filter.toLowerCase();
            this.filteredPipelines = project.pipeline_names.filter((pip) => pip.name.toLowerCase().indexOf(filter) !== -1);
        }
    }

    get project(): Project {
        return this._project;
    }

    set filter(filter: string) {
        this._filter = filter;
        if (this.project.pipeline_names) {
            let filterLower = filter.toLowerCase();
            this.filteredPipelines = this.project.pipeline_names.filter((pip) => pip.name.toLowerCase().indexOf(filterLower) !== -1);
        }
    }

    get filter(): string {
        return this._filter;
    }

    _project: Project;
    _filter = '';

    filteredPipelines: Array<IdName> = [];
    loading = true;

    constructor(private store: Store, private _cd: ChangeDetectorRef) {
    }

    ngOnInit(): void {
        let opts: Array<LoadOpts> = [new LoadOpts('withPipelineNames', 'pipeline_names')];
        this.store.dispatch(new ResyncProject({projectKey: this.project.key, opts}))
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe();
    }
}
