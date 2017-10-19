import {Component, Input, NgZone, OnDestroy, OnInit} from '@angular/core';
import {Project} from '../../../model/project.model';
import {Workflow} from '../../../model/workflow.model';
import {AutoUnsubscribe} from '../../../shared/decorator/autoUnsubscribe';
import {CDSWorker} from '../../../shared/worker/worker';
import {AuthentificationStore} from '../../../service/auth/authentification.store';
import {environment} from '../../../../environments/environment';
import {Subscription} from 'rxjs/Subscription';
import {WorkflowRun, WorkflowRunTags} from '../../../model/workflow.run.model';
import {cloneDeep} from 'lodash';

@Component({
    selector: 'app-workflow-sidebar',
    templateUrl: './workflow.sidebar.component.html',
    styleUrls: ['./workflow.sidebar.component.scss']
})
@AutoUnsubscribe()
export class WorkflowSidebarComponent implements OnDestroy {

    // Project that contains the workflow
    @Input() project: Project;

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

    constructor(private _authStore: AuthentificationStore) {
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
            workflowName: this.workflow.name
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
        this.tagsSelectable = new Array<string>();
        if (this.workflow.metadata && this.workflow.metadata['default_tags']) {
            this.tagsSelectable = this.workflow.metadata['default_tags'].split(',');
        }
        this.refreshRun();
    }

    refreshRun(): void {
        if (this.workflowRuns) {
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
    }

    canDisplayTag(tg: WorkflowRunTags): boolean {
        if (this.tagsSelectable) {
            return this.tagsSelectable.indexOf(tg.tag) !== -1;
        }
        return false;
    }

    ngOnDestroy(): void {
        if (this.runWorker) {
            this.runWorker.stop();
        }
    }
}
