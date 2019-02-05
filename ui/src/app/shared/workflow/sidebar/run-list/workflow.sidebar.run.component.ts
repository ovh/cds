import { Component, ElementRef, Input, OnDestroy, ViewChild } from '@angular/core';
import { Router } from '@angular/router';
import { Subscription } from 'rxjs';
import { PipelineStatus } from '../../../../model/pipeline.model';
import { Project } from '../../../../model/project.model';
import { Workflow } from '../../../../model/workflow.model';
import { WorkflowRun, WorkflowRunTags } from '../../../../model/workflow.run.model';
import { WorkflowRunService } from '../../../../service/workflow/run/workflow.run.service';
import { WorkflowEventStore } from '../../../../service/workflow/workflow.event.store';
import { AutoUnsubscribe } from '../../../decorator/autoUnsubscribe';
import { DurationService } from '../../../duration/duration.service';

@Component({
    selector: 'app-workflow-sidebar-run-list',
    templateUrl: './workflow.sidebar.run.component.html',
    styleUrls: ['./workflow.sidebar.run.component.scss']
})
@AutoUnsubscribe()
export class WorkflowSidebarRunListComponent implements OnDestroy {
    @ViewChild('tagsList') tagsList: ElementRef;

    @Input() project: Project;

    _workflow: Workflow;
    @Input('workflow')
    set workflow(data: Workflow) {
        if (data) {
            let haveToStart = false;
            if (!this._workflow || (this._workflow && data.name !== this._workflow.name)) {
                haveToStart = true;
            }

            if (!this._workflow || this._workflow.id !== data.id) {
                this._workflow = data;
                this.initSelectableTags();
            }
            this._workflow = data;
            if (haveToStart) {
                this.eventSubscription = this._eventStore.workflowRuns()
                    .subscribe((m) => {
                        this.workflowRuns = Array.from(m.valueSeq().toArray()).sort((a, b) => {
                            return b.num - a.num;
                        });
                        this.refreshRun();
                    });
            }
        }
    }
    get workflow() { return this._workflow; }

    eventSubscription: Subscription;
    workflowRuns: Array<WorkflowRun>;

    // search part
    selectedTags: Array<string>;
    tagsSelectable: Array<string>;
    tagToDisplay: Array<string>;
    pipelineStatusEnum = PipelineStatus;
    ready = false;
    listingSub: Subscription;
    tagsSubs: Subscription;
    filteredTags: { [key: number]: WorkflowRunTags[] } = {};

    durationIntervalID: number;

    selectedWorkfowRun: WorkflowRun;
    subWorkflowRun: Subscription;
    offset = 0;

    constructor(
        private _workflowRunService: WorkflowRunService,
        private _duration: DurationService,
        private _router: Router,
        private _eventStore: WorkflowEventStore
    ) {
        this.subWorkflowRun = this._eventStore.selectedRun().subscribe(wr => {
            this.selectedWorkfowRun = wr;
        });
        this.listingSub = this._eventStore.isListingRuns().subscribe(b => {
            this.ready = !b;
        });
    }

    scroll() {
        if (!Array.isArray(this.selectedTags) || !this.selectedTags.length) {
            this.offset += 50;
            this._workflowRunService.runs(this.project.key, this.workflow.name, '50', this.offset.toString())
                .subscribe((runs) => {
                    this.workflowRuns = this.workflowRuns.concat(runs);
                    this.refreshRun();
                });
        }
    }

    initSelectableTags(): void {
        this.tagToDisplay = new Array<string>();
        if (this.workflow.metadata && this.workflow.metadata['default_tags']) {
            this.tagToDisplay = this.workflow.metadata['default_tags'].split(',');
        }
        this.tagsSubs = this._workflowRunService.getTags(this.project.key, this.workflow.name)
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
        this.refreshRun();
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
            return this._duration.duration(new Date(start), new Date());
        }
        if (!done) {
            done = new Date().toString();
        }
        return this._duration.duration(new Date(start), new Date(done));
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

        this._workflowRunService.runs(this.project.key, this.workflow.name, '50', null, filters).subscribe((runs) => {
            this.workflowRuns = runs;
            this.refreshRun();
        });
    }

    refreshRun(): void {
        if (this.workflowRuns) {
            if (this.durationIntervalID) {
                this.deleteInterval();
            }
            this.refreshDuration();
            this.durationIntervalID = window.setInterval(() => {
                this.refreshDuration();
            }, 5000);
        }
    }

    refreshDuration(): void {
        if (this.workflowRuns) {
            let stillWorking = false;
            this.workflowRuns.forEach((r) => {
                if (PipelineStatus.isActive(r.status)) {
                    stillWorking = true;
                }
                this.filteredTags[r.id] = this.getFilteredTags(r.tags);
                r.duration = this.getDuration(r.status, r.start, r.last_execution);
            });
            if (!stillWorking) {
                this.deleteInterval();
            }
        }

    }

    changeRun(num: number) {
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
}
