import {Component, Input, NgZone, OnDestroy, ElementRef, ViewChild} from '@angular/core';
import {Project} from '../../../../../model/project.model';
import {PipelineStatus} from '../../../../../model/pipeline.model';
import {Workflow} from '../../../../../model/workflow.model';
import {AutoUnsubscribe} from '../../../../../shared/decorator/autoUnsubscribe';
import {CDSWorker} from '../../../../../shared/worker/worker';
import {AuthentificationStore} from '../../../../../service/auth/authentification.store';
import {environment} from '../../../../../../environments/environment';
import {Subscription} from 'rxjs/Subscription';
import {WorkflowRun, WorkflowRunTags} from '../../../../../model/workflow.run.model';
import {cloneDeep} from 'lodash';
import {WorkflowRunService} from '../../../../../service/workflow/run/workflow.run.service';
import {DurationService} from '../../../../../shared/duration/duration.service';

@Component({
    selector: 'app-workflow-sidebar-run-list',
    templateUrl: './workflow.sidebar.run.component.html',
    styleUrls: ['./workflow.sidebar.run.component.scss']
})
@AutoUnsubscribe()
export class WorkflowSidebarRunListComponent implements OnDestroy {

    // Project that contains the workflow
    @Input() project: Project;
    @Input() runNumber: number;

    // Workflow
    _workflow: Workflow;
    @Input('workflow')
    set workflow(data: Workflow) {
        if (data) {
            let haveToStar = false;
            if (!this._workflow || (this._workflow && data.name !== this._workflow.name)) {
                haveToStar = true;
            }
            this._workflow = data;
            this.initSelectableTags();
            if (haveToStar) {
                this.startWorker();
            }
        }
    }
    get workflow() { return this._workflow; }
    // Flag indicate if sidebar is open
    @Input() open: boolean;

    @ViewChild('tagsList') tagsList: ElementRef;

    // List of workflow run, updated by  webworker
    workflowRuns: Array<WorkflowRun>;
    filteredWorkflowRuns: Array<WorkflowRun>;

    // Webworker
    runWorker: CDSWorker;
    runWorkerSubscription: Subscription;

    // Angular zone to update model with webworker data
    zone: NgZone;

    // search part
    selectedTags: Array<string>;
    tagsSelectable: Array<string>;
    tagToDisplay: Array<string>;
    pipelineStatusEnum = PipelineStatus;
    ready = false;
    filteredTags: {[key: number]: WorkflowRunTags[]} = {};

    constructor(private _authStore: AuthentificationStore, private _workflowRunService: WorkflowRunService,
      private _duration: DurationService) {
        this.zone = new NgZone({enableLongStackTrace: false});
    }

    startWorker(): void {
        // Start webworker
        if (this.runWorkerSubscription) {
            this.runWorkerSubscription.unsubscribe();
        }
        if (this.runWorker) {
            this.runWorker.stop();
        }

        this.runWorker = new CDSWorker('./assets/worker/web/workflow-run.js');
        this.runWorker.start({
            'user': this._authStore.getUser(),
            'session': this._authStore.getSessionToken(),
            'api': environment.apiURL,
            key: this.project.key,
            workflowName: this.workflow.name,
            limit: 50
        });

        // Listening to web worker responses
        this.runWorkerSubscription = this.runWorker.response().subscribe(msg => {
            this.zone.run(() => {
                if (!msg) {
                    return;
                }
                this.workflowRuns = <Array<WorkflowRun>>JSON.parse(msg);
                this.refreshRun();
                this.ready = true;
            });

        });
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

    getFilteredTagsString(tags: WorkflowRunTags[]): string {
        if (!Array.isArray(tags) || !this.tagToDisplay) {
            return '';
        }
        let tagsFormatted = '';
        for (let i = 0; i < tags.length; i++) {
            if (i === 0) {
                tagsFormatted += tags[i].value;
            } else {
                tagsFormatted += (' , ' + tags[i].value);
            }
        }

        return tagsFormatted;
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
            this.filteredWorkflowRuns.forEach((r) => this.filteredTags[r.id] = this.getFilteredTags(r.tags));
        }
    }

    ngOnDestroy(): void {
        if (this.runWorker) {
            this.runWorker.stop();
        }
    }
}
