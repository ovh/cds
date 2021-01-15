import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    ElementRef,
    Input,
    OnDestroy, OnInit,
    ViewChild
} from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { Select, Store } from '@ngxs/store';
import { PipelineStatus } from 'app/model/pipeline.model';
import { Project } from 'app/model/project.model';
import { Workflow } from 'app/model/workflow.model';
import { WorkflowRun, WorkflowRunSummary, WorkflowRunTags } from 'app/model/workflow.run.model';
import { WorkflowRunService } from 'app/service/workflow/run/workflow.run.service';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { DurationService } from 'app/shared/duration/duration.service';
import { ToastService } from 'app/shared/toast/ToastService';
import { ProjectState } from 'app/store/project.state';
import { CleanWorkflowRun, ClearListRuns, SetWorkflowRuns } from 'app/store/workflow.action';
import { WorkflowState } from 'app/store/workflow.state';
import { Observable, Subscription } from 'rxjs';
import { finalize, first } from 'rxjs/operators';

const limitWorkflowRun = 30;

@Component({
    selector: 'app-workflow-sidebar-run-list',
    templateUrl: './workflow.sidebar.run.component.html',
    styleUrls: ['./workflow.sidebar.run.component.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowSidebarRunListComponent implements OnDestroy {
    @ViewChild('tagsList') tagsList: ElementRef;

    _workflow: Workflow;
    @Input()
    set workflow(data: Workflow) {
        if (data) {
            if (!this._workflow || this._workflow.id !== data.id) {
                this.filteredTags = {};
                this._workflow = data;
                this.deleteInterval();
                this.initSelectableTags();
                this.offset = 0;
                this._store.dispatch(new ClearListRuns()).subscribe(() => {
                    this.getRuns();
                })
            }
            this._workflow = data;
        }
    }
    get workflow() {
        return this._workflow;
    }

    @Select(WorkflowState.getSelectedWorkflowRun()) wrun$: Observable<WorkflowRun>
    wrunSub: Subscription;
    @Select(WorkflowState.getListRuns()) listRuns$: Observable<Array<WorkflowRunSummary>>;
    listRunSubs: Subscription;
    @Select(WorkflowState.getRunSidebarFilters()) filters$: Observable<{}>;
    filtersSubs: Subscription;

    project: Project;
    workflowRuns = new Array<WorkflowRunSummary>();

    // search part
    selectedTags: Array<string>;
    tagsSelectable: Array<string>;
    tagToDisplay: Array<string>;
    pipelineStatusEnum = PipelineStatus;
    ready = false;
    filteredTags: { [key: number]: WorkflowRunTags[] } = {};
    durationMap: { [key: number]: string } = {};

    durationIntervalID: number;
    currentWorkflowRunNumber: number;
    offset = 0;

    constructor(
        private _workflowRunService: WorkflowRunService,
        private _router: Router,
        private _toast: ToastService,
        private _store: Store,
        private _cd: ChangeDetectorRef
    ) {
        this.project = this._store.selectSnapshot(ProjectState.projectSnapshot);

        this.wrunSub = this.wrun$.subscribe(wr => {
            if (!wr && !this.currentWorkflowRunNumber) {
                return;
            }
            if (wr?.num === this.currentWorkflowRunNumber) {
                return;
            }
            this.currentWorkflowRunNumber = wr?.num;
            this._cd.markForCheck();
        });

        this.listRunSubs = this.listRuns$.subscribe(runs => {
            if (runs.length === 0 && this.workflowRuns.length === 0) {
                return;
            }
            this.workflowRuns = runs;
            if (this.workflowRuns && this.workflow && this.workflowRuns.length > 0) {
                this.refreshRun();
            }
            this._cd.markForCheck();
        });
        this.filtersSubs = this.filters$.subscribe(fs => {
            this.selectedTags = new Array<string>();
            for (const key in fs) {
                if (fs.hasOwnProperty(key)) {
                    this.selectedTags.push(key + ':' + fs[key]);
                }
            }
            this._cd.markForCheck();
            return;
        });
    }

    getRuns(filter?: any): void {
        this._workflowRunService.runs(this.project.key, this.workflow.name, limitWorkflowRun.toString(), this.offset.toString(), filter)
            .pipe(finalize(() => {
                this.ready = true;
                this._cd.markForCheck();
            }))
            .subscribe((runs) => {
                this._store.dispatch(new SetWorkflowRuns(
                    { projectKey: this.project.key, workflowName: this.workflow.name, runs, filters: filter }))
            });
    }

    scroll() {
        if (!Array.isArray(this.selectedTags) || !this.selectedTags.length) {
            this.offset = this.workflowRuns.length;
            this.getRuns();
        }
    }

    initSelectableTags(): void {
        this.tagToDisplay = new Array<string>();
        if (this.workflow.metadata && this.workflow.metadata['default_tags']) {
            this.tagToDisplay = this.workflow.metadata['default_tags'].split(',');
        }
        this._workflowRunService.getTags(this.project.key, this.workflow.name)
            .pipe(first(), finalize(() => this._cd.markForCheck()))
            .subscribe(tags => {
                this.tagsSelectable = new Array<string>();
                Object.keys(tags).forEach(k => {
                    if (tags.hasOwnProperty(k)) {
                        tags[k].forEach(v => {
                            if (v !== '') {
                                let newEntry = k + ':' + v;
                                if (this.tagsSelectable.indexOf(newEntry) === -1) {
                                    this.tagsSelectable.push(newEntry);
                                }
                            }
                        });
                    }
                });
            });
        if (!this.durationIntervalID && this.workflowRuns && this.workflow && this.workflowRuns.length > 0) {
            this.refreshRun();
        }
        this._cd.detectChanges();
    }

    getFilteredTags(tags: WorkflowRunTags[]): WorkflowRunTags[] {
        if (!Array.isArray(tags) || !this.tagToDisplay) {
            return [];
        }
        return tags
            .filter((tg) => this.tagToDisplay.indexOf(tg.tag) !== -1)
            .sort((tga, tgb) => this.tagToDisplay.indexOf(tga.tag) - this.tagToDisplay.indexOf(tgb.tag));
    }

    getDuration(status: string, start: string, done: string): string {
        if (status === PipelineStatus.BUILDING || status === PipelineStatus.WAITING) {
            return DurationService.duration(new Date(start), new Date());
        }
        if (!done) {
            done = new Date().toString();
        }
        return DurationService.duration(new Date(start), new Date(done));
    }

    filterRuns(): void {
        this.offset = 0;
        let filters;

        if (Array.isArray(this.selectedTags) && this.selectedTags.length) {
            filters = this.selectedTags.reduce((prev, cur) => {
                let splitted = cur.split(':');
                if (splitted.length === 2) {
                    prev[splitted[0]] = splitted[1];
                }

                return prev;
            }, {});
        }
        this.offset = 0;
        this.getRuns(filters)
    }

    refreshRun(): void {
        if (this.durationIntervalID) {
            this.deleteInterval();
        }
        this.refreshDuration();
        this.durationIntervalID = window.setInterval(() => {
            this.refreshDuration();
            this._cd.markForCheck();
        }, 5000);
    }

    refreshDuration(): void {
        let stillWorking = false;
        if (this.workflow && this.workflow.metadata && this.workflow.metadata['default_tags']) {
            this.tagToDisplay = this.workflow.metadata['default_tags'].split(',');
        }
        this.workflowRuns.forEach((r) => {
            if (PipelineStatus.isActive(r.status)) {
                stillWorking = true;
            }
            this.filteredTags[r.id] = this.getFilteredTags(r.tags);
            this.durationMap[r.id] = this.getDuration(r.status, r.start, r.last_execution);
        });
        if (!stillWorking) {
            this.deleteInterval();
        }
    }

    changeRun(num: number) {
        if (this.currentWorkflowRunNumber === num) {
            return
        }
        this._store.dispatch(new CleanWorkflowRun({}));
        this._router.navigate(['/project', this.project.key, 'workflow', this.workflow.name, 'run', num]);
    }

    ngOnDestroy(): void {
        this.deleteInterval();
    }

    deleteInterval(): void {
        if (this.durationIntervalID) {
            clearInterval(this.durationIntervalID);
            this.durationIntervalID = 0;
        }
    }

    filterTags(options: Array<string>, query: string): Array<string> | false {
        if (!options) {
            return false;
        }
        if (!query || query.length < 3) {
            return options.slice(0, 100);
        }
        let queryLowerCase = query.toLowerCase();
        return options.filter(o => o.toLowerCase().indexOf(queryLowerCase) !== -1);
    }

    confirmCopy() {
        this._toast.success('', 'Workflow run version copied!');
    }
}
