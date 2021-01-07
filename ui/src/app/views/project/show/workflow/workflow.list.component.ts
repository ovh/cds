import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnInit, ViewChild } from '@angular/core';
import { Store } from '@ngxs/store';
import { IdName, Label, LoadOpts, Project } from 'app/model/project.model';
import { ProjectStore } from 'app/service/project/project.store';
import { LabelsEditComponent } from 'app/shared/labels/edit/labels.edit.component';
import { ResyncProject } from 'app/store/project.action';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-project-workflows',
    templateUrl: './workflow.list.html',
    styleUrls: ['./workflow.list.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ProjectWorkflowListComponent implements OnInit {

    @Input()
    set project(project: Project) {
        this._project = project;
        if (project.workflow_names) {
            let filter = this.filter.toLowerCase();
            this.filteredWorkflows = project.workflow_names.filter((wf) => wf.name.toLowerCase().indexOf(filter) !== -1);
        }
        if (project.labels) {
            let labelFilter = this.labelFilter.toLowerCase();
            this.filteredLabels = project.labels.filter((lbl) => lbl.name.toLowerCase().indexOf(labelFilter) !== -1);
        }
    }

    get project(): Project {
        return this._project;
    }

    set filter(filter: string) {
        this._filter = filter;
        if (this.project.workflow_names) {
            let filterLower = filter.toLowerCase();
            this.filteredWorkflows = this.project.workflow_names.filter((wf) => wf.name.toLowerCase().indexOf(filterLower) !== -1);
        }
    }

    get filter(): string {
        return this._filter;
    }

    set labelFilter(filter: string) {
        this._filterLabel = filter;
        if (this.project.labels) {
            let filterLower = filter.toLowerCase();
            this.filteredLabels = this.project.labels.filter((lbl) => lbl.name.toLowerCase().indexOf(filterLower) !== -1);
        }
    }

    get labelFilter(): string {
        return this._filterLabel;
    }

    // Modal
    @ViewChild('projectLabels')
    projectLabels: LabelsEditComponent;

    _project: Project;
    _filter = '';
    _filterLabel = '';

    viewMode: 'blocs' | 'labels' | 'lines' = 'blocs';
    filteredWorkflows: Array<IdName> = [];
    filteredLabels: Array<Label> = [];
    loading = true;


    constructor(private store: Store, private _cd: ChangeDetectorRef, private _projectStore: ProjectStore) {
    }

    ngOnInit(): void {
        this.viewMode = this._projectStore.getWorkflowViewMode(this.project.key);
        let opts: Array<LoadOpts> = [new LoadOpts('withWorkflowNames', 'workflow_names')];
        this.store.dispatch(new ResyncProject({projectKey: this.project.key, opts}))
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe();
    }

    editLabels() {
        if (this.projectLabels && this.projectLabels.show) {
            this.projectLabels.show();
        }
    }

    setViewMode(mode: 'blocs' | 'labels' | 'lines') {
        this.viewMode = mode;
        this._projectStore.setWorkflowViewMode(this.project.key, mode);
    }
}
