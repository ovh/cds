import {Component, Input} from '@angular/core';
import {TranslateService} from '@ngx-translate/core';
import {cloneDeep} from 'lodash';
import {finalize, first} from 'rxjs/operators';
import {IdName, LoadOpts, Project} from '../../../../model/project.model';
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
        this.loadProjectWorkflow(projFilter, true);

        this.selectedProjectKey = '';
    }

    removeProject(pf: ProjectFilter): void {
        this.filter.projects = this.filter.projects.filter(p => p.key !== pf.key);
    }

    // load workflow when opening dropdown
    loadProjectWorkflow(projFilter: ProjectFilter, mute: boolean) {
        if (projFilter.project) {
            return
        }
        projFilter.loading = true;

        let opts = new Array<LoadOpts>();
        opts.push(new LoadOpts('withWorkflowNames', 'workflow_names'));
        this._projectStore.resync(projFilter.key, opts).pipe(first(), finalize(() => projFilter.loading = false)).subscribe(proj => {
            projFilter.project = proj;
            if (projFilter.project && projFilter.project.workflow_names) {
                projFilter.project.workflow_names.forEach(wn => {
                    if (mute) {
                        wn.mute = true;
                    } else {
                        let index = projFilter.workflow_names.findIndex(wwnn => wwnn === wn.name);
                        if (index >= 0) {
                            wn.mute = true;
                        } else {
                            wn.mute = false;
                        }
                    }
                });
                if (mute) {
                    projFilter.workflow_names = projFilter.project.workflow_names.map(idname => idname.name);
                }
            }

        });
    }

    updateWorkflowInFilter(w: IdName, projFilter: ProjectFilter): void {
        if (!projFilter.workflow_names) {
            projFilter.workflow_names = new Array<string>();
        }
        w.mute = !w.mute;
        if (!w.mute) {
            let index = projFilter.workflow_names.findIndex(wn => wn === w.name);
            if (index >= 0) {
                projFilter.workflow_names.splice(index, 1);
            }
        } else {
            let index = projFilter.workflow_names.findIndex(wn => wn === w.name);
            if (index === -1) {
                projFilter.workflow_names.push(w.name);
            }
        }
    }

    saveFilter(): void {
        this.loading = true;
        this._timelineStore.saveFilter(this.filter).pipe(finalize(() => this.loading = false)).subscribe(() => {
            this._toast.success('', this._translate.instant('timeline_filter_updated'));
        });
    }

}
