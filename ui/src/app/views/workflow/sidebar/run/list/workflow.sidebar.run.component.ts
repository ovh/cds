import {Component, Input, ElementRef, ViewChild} from '@angular/core';
import {Project} from '../../../../../model/project.model';
import {PipelineStatus} from '../../../../../model/pipeline.model';
import {Workflow} from '../../../../../model/workflow.model';
import {AutoUnsubscribe} from '../../../../../shared/decorator/autoUnsubscribe';
import {Subscription} from 'rxjs/Subscription';
import {WorkflowRun, WorkflowRunTags} from '../../../../../model/workflow.run.model';
import {cloneDeep} from 'lodash';
import {WorkflowRunService} from '../../../../../service/workflow/run/workflow.run.service';
import {DurationService} from '../../../../../shared/duration/duration.service';
import {Router} from '@angular/router';
import {WorkflowEventStore} from '../../../../../service/workflow/workflow.event.store';

@Component({
    selector: 'app-workflow-sidebar-run-list',
    templateUrl: './workflow.sidebar.run.component.html',
    styleUrls: ['./workflow.sidebar.run.component.scss']
})
@AutoUnsubscribe()
export class WorkflowSidebarRunListComponent {

    // Project that contains the workflow
    @Input() project: Project;

    // Workflow
    _workflow: Workflow;
    @Input('workflow')
    set workflow(data: Workflow) {
        if (data) {
            let haveToStart = false;
            if (!this._workflow || (this._workflow && data.name !== this._workflow.name)) {
                haveToStart = true;
            }

            if (this._workflow && this._workflow.id !== data.id) {
                this.ready = false;
            }
            this._workflow = data;
            this.initSelectableTags();
            if (haveToStart) {
                this.eventSubscription = this._eventStore.workflowRuns().subscribe(m => {
                    this.workflowRuns = Array.from(m.valueSeq().toArray()).sort((a, b) => {
                        return b.num - a.num;
                    });
                    this.refreshRun();
                    this.ready = true;
                });
            }
        }
    }
    get workflow() { return this._workflow; }

    @ViewChild('tagsList') tagsList: ElementRef;

    eventSubscription: Subscription;
    // List of workflow run
    workflowRuns: Array<WorkflowRun>;
    filteredWorkflowRuns: Array<WorkflowRun>;

    // search part
    selectedTags: Array<string>;
    tagsSelectable: Array<string>;
    tagToDisplay: Array<string>;
    pipelineStatusEnum = PipelineStatus;
    ready = false;
    filteredTags: {[key: number]: WorkflowRunTags[]} = {};

    durationIntervalID: number;

    selectedWorkfowRun: WorkflowRun;
    subWorkflowRun: Subscription;

    constructor(private _workflowRunService: WorkflowRunService,
      private _duration: DurationService, private _router: Router, private _eventStore: WorkflowEventStore) {

        this.subWorkflowRun = this._eventStore.selectedRun().subscribe(wr => {
            this.selectedWorkfowRun = wr;
        })
    }

    startWorker(): void {

    }

    initSelectableTags(): void {
        this.tagToDisplay = new Array<string>();
        if (this.workflow.metadata && this.workflow.metadata['default_tags']) {
            this.tagToDisplay = this.workflow.metadata['default_tags'].split(',');
        }
        this._workflowRunService.getTags(this.project.key, this.workflow.name).subscribe(tags => {
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

    refreshRun(): void {
        if (this.workflowRuns) {
            this.filteredTags = {};
            this.filteredWorkflowRuns = cloneDeep(this.workflowRuns);

            if (this.selectedTags) {
              this.selectedTags.forEach(t => {
                  let splitted = t.split(':');
                  let key = splitted.shift();
                  let value = splitted.join(':');
                  this.filteredWorkflowRuns = this.filteredWorkflowRuns.filter(r => {
                      return r.tags.find(tag => {
                          return tag.tag === key && tag.value.indexOf(value) !== -1;
                      });
                  });
              });
            }
            if (this.durationIntervalID) {
                clearInterval(this.durationIntervalID);
                this.durationIntervalID = 0;
            }
            this.refreshDuration();
            this.durationIntervalID = setInterval(() => {
                this.refreshDuration();
            }, 5000);
        }
    }

    refreshDuration(): void {
        if (this.filteredWorkflowRuns) {
            let stillWorking = false;
            this.filteredWorkflowRuns.forEach((r) => {
                if (PipelineStatus.isActive(r.status)) {
                    stillWorking = true;
                }
                this.filteredTags[r.id] = this.getFilteredTags(r.tags);
                r.duration = this.getDuration(r.status, r.start, r.last_execution);
            });
            if (!stillWorking) {
                clearInterval(this.durationIntervalID);
            }
        }

    }

    changeRun(num: number) {
        this._router.navigate(['/project', this.project.key, 'workflow', this.workflow.name, 'run', num]);
    }
}
