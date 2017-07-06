import {Component, Input, NgZone, OnDestroy, OnInit} from '@angular/core';
import {Project} from '../../../model/project.model';
import {Workflow} from '../../../model/workflow.model';
import {AutoUnsubscribe} from '../../../shared/decorator/autoUnsubscribe';
import {CDSWorker} from '../../../shared/worker/worker';
import {AuthentificationStore} from '../../../service/auth/authentification.store';
import {environment} from '../../../../environments/environment';
import {Subscription} from 'rxjs/Subscription';
import {WorkflowRun} from '../../../model/workflow.run.model';

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

    // Webworker
    runWorker: CDSWorker;
    runWorkerSubscription: Subscription;

    // Angular zone to update model with webworker data
    zone: NgZone;

    constructor(private _authStore: AuthentificationStore) {
        this.zone = new NgZone({enableLongStackTrace: false});
    }

    ngOnInit(): void {
        // Star  webworker
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
                this.workflowRuns = <Array<WorkflowRun>>JSON.parse(msg);
            });

        });
    }

    ngOnDestroy(): void {
        if (this.runWorker) {
            this.runWorker.stop();
        }
    }
}