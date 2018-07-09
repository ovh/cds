import {Component, Input} from '@angular/core';
import {TranslateService} from '@ngx-translate/core';
import {cloneDeep} from 'lodash';
import {finalize, first} from 'rxjs/operators';
import {LoadOpts, Project} from '../../../../model/project.model';
import {ProjectFilter, TimelineFilter} from '../../../../model/timeline.model';
import {ProjectStore} from '../../../../service/project/project.store';
import {TimelineStore} from '../../../../service/timeline/timeline.store';
import {AutoUnsubscribe} from '../../../../shared/decorator/autoUnsubscribe';
import {ToastService} from '../../../../shared/toast/ToastService';

@Component({
    selector: 'app-home-timeline-filter',
    templateUrl: './home.timeline.filter.html',
    styleUrls: ['./home.timeline.filter.scss']
})
@AutoUnsubscribe()
export class HomeTimelineFilterComponent {

    _filter: TimelineFilter;
    filterToEdit: TimelineFilter;
    @Input('filter')
    set filter(data: TimelineFilter) {
        if (data) {
            this._filter = data;
            this.filterToEdit = cloneDeep(data);
        }
    }
    get filter() {
        return this.filterToEdit;
    }

    projects: Array<Project>;

    selectedProjectKey: string;
    selectedWorkflow: string;

    loading = false;

    constructor(private _projectStore: ProjectStore, private _timelineStore: TimelineStore, private _translate: TranslateService,
                private _toast: ToastService) {
        this._projectStore.getProjectsList().subscribe(ps => {
            if (ps) {
                this.projects = ps.toArray();
            }
        });
    }

    addProject(): void {
        if (!this.selectedProjectKey || this.selectedProjectKey === '') {
            return;
        }
        if (this.filter.projects) {
            let exist = this.filter.projects.find(p => p.key === this.selectedProjectKey);
            if (exist) {
                return;
            }
        } else {
            this.filter.projects = new Array<ProjectFilter>();
        }


        let projFilter = new ProjectFilter();
        projFilter.key = this.selectedProjectKey;
        this.filter.projects.push(projFilter);

        this.loadProjectWorkflow(projFilter);

        this.selectedProjectKey = '';
    }

    removeProject(pf: ProjectFilter): void {
        this.filter.projects = this.filter.projects.filter(p => p.key !== pf.key);
    }

    removeWorkflow(pf: ProjectFilter, wname: string): void {
        pf.workflow_names = pf.workflow_names.filter(w => w !== wname);
    }

    addWorkflow(projFilter: ProjectFilter): void {
        if (!this.selectedWorkflow || this.selectedWorkflow === '') {
            return;
        }

        if (projFilter.workflow_names) {
            let exist = projFilter.workflow_names.find(w => w === this.selectedWorkflow);
            if (exist) {
                return;
            }
        } else {
            projFilter.workflow_names = new Array<string>();
        }
        projFilter.workflow_names.push(this.selectedWorkflow);
        this.selectedWorkflow = '';
    }

    openProject(projFilter: ProjectFilter): void {
        projFilter.display = !projFilter.display;
        this.loadProjectWorkflow(projFilter);
    }

    loadProjectWorkflow(projFilter: ProjectFilter) {
        if (projFilter.project) {
            return
        }
        projFilter.loading = true;

        let opts = new Array<LoadOpts>();
        opts.push(new LoadOpts('withWorkflowNames', 'workflow_names'));
        this._projectStore.resync(projFilter.key, opts).pipe(first(), finalize(() => projFilter.loading = false)).subscribe(proj => {
            projFilter.project = proj;
        });
    }

    saveFilter(): void {
        this.loading = true;
        this._timelineStore.saveFilter(this.filter).pipe(finalize(() => this.loading = false)).subscribe(() => {
           this._toast.success('', this._translate.instant('timeline_filter_updated'));
        });
    }

}
