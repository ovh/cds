import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnDestroy } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { IdName, LoadOpts, Project } from 'app/model/project.model';
import { ProjectFilter, TimelineFilter } from 'app/model/timeline.model';
import { ProjectStore } from 'app/service/project/project.store';
import { TimelineStore } from 'app/service/timeline/timeline.store';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { ToastService } from 'app/shared/toast/ToastService';
import { FetchProject } from 'app/store/project.action';
import { ProjectState, ProjectStateModel } from 'app/store/project.state';
import cloneDeep from 'lodash-es/cloneDeep';
import { finalize, flatMap } from 'rxjs/operators';

@Component({
    selector: 'app-home-filter',
    templateUrl: './home.filter.html',
    styleUrls: ['./home.filter.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class HomeFilterComponent implements OnDestroy {

    _filter: TimelineFilter;
    filterToEdit: TimelineFilter;

    @Input()
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

    loading = false;

    constructor(
        private store: Store,
        private _projectStore: ProjectStore,
        private _timelineStore: TimelineStore,
        private _translate: TranslateService,
        private _toast: ToastService,
        private _cd: ChangeDetectorRef
    ) {
        this._projectStore.getProjectsList().subscribe(ps => {
            if (ps) {
                this.projects = ps.toArray();
            }
        });
    }

    ngOnDestroy(): void {} // Should be set to use @AutoUnsubscribe with AOT

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

        this.store.dispatch(new FetchProject({
            projectKey: projFilter.key,
            opts
        })).pipe(
            finalize(() => {
                projFilter.loading = false;
                this._cd.markForCheck();
            }),
            flatMap(() => this.store.selectOnce(ProjectState))
        ).subscribe((proj: ProjectStateModel) => {
            projFilter.project = cloneDeep(proj.project);
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
        this._timelineStore.saveFilter(this.filter)
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(() => {
                this._toast.success('', this._translate.instant('timeline_filter_updated'));
            });
    }
}
