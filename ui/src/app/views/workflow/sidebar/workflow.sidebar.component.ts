import {Component, Input, NgZone, OnDestroy, OnInit} from '@angular/core';
import {Project} from '../../../model/project.model';
import {Workflow} from '../../../model/workflow.model';
import {AutoUnsubscribe} from '../../../shared/decorator/autoUnsubscribe';
import {CDSWorker} from '../../../shared/worker/worker';
import {AuthentificationStore} from '../../../service/auth/authentification.store';
import {environment} from '../../../../environments/environment';
import {Subscription} from 'rxjs/Subscription';
import {WorkflowRun} from '../../../model/workflow.run.model';
import {WorkflowRunService} from '../../../service/workflow/run/workflow.run.service';
import {cloneDeep} from 'lodash';

@Component({
    selector: 'app-workflow-sidebar',
    templateUrl: './workflow.sidebar.component.html',
    styleUrls: ['./workflow.sidebar.component.scss']
})
@AutoUnsubscribe()
export class WorkflowSidebarComponent implements OnInit, OnDestroy {

    // Project that contains the workflow
    @Input() project: Project;
    // Workflow
    @Input() workflow: Workflow;
    // Flag indicate if sidebar is open
    @Input() open: boolean;

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

    ready = false;

    constructor(private _authStore: AuthentificationStore, private _workflowRunService: WorkflowRunService) {
        this.zone = new NgZone({enableLongStackTrace: false});
    }

    ngOnInit(): void {
        // Start webworker
        this.runWorker = new CDSWorker('./assets/worker/web/workflow-run.js');
        this.runWorker.start({
            'user': this._authStore.getUser(),
            'session': this._authStore.getSessionToken(),
            'api': environment.apiURL,
            key: this.project.key,
            workflowName: this.workflow.name
        });

        // Listening to web worker responses
        this.runWorkerSubscription = this.runWorker.response().subscribe(msg => {
            this.zone.run(() => {
                this.ready = true;
                if (!msg) {
                    return;
                }
                this.workflowRuns = <Array<WorkflowRun>>JSON.parse(msg);
                this.refreshRun();
            });

        });

        this._workflowRunService.getTags(this.project.key, this.workflow.name).subscribe(tags => {
            this.tagsSelectable = new Array<string>();
            Object.keys(tags).forEach(k => {
                if (tags.hasOwnProperty(k)) {
                    tags[k].forEach(v => {
                        this.tagsSelectable.push(k + ':' + v);
                    });
                }
            });
        });
    }

    refreshRun(): void {
        this.filteredWorkflowRuns = cloneDeep(this.workflowRuns);
        if (!this.selectedTags) {
            return;
        }
        this.selectedTags.forEach(t => {
            let splitted = t.split(':');
            let key = splitted.shift();
            let value = splitted.join(':');
            this.filteredWorkflowRuns = this.filteredWorkflowRuns.filter(r => {
                return r.tags.find(tag => {
                    return tag.tag === key && tag.value === value;
                });
            });
        });
    }

    ngOnDestroy(): void {
        if (this.runWorker) {
            this.runWorker.stop();
        }
    }
}
