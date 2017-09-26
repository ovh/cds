import {Component, NgZone, OnDestroy} from '@angular/core';
import {ActivatedRoute, Router} from '@angular/router';
import {Project} from '../../../model/project.model';
import {CDSWorker} from '../../../shared/worker/worker';
import {WorkflowRun} from '../../../model/workflow.run.model';
import {environment} from '../../../../environments/environment';
import {AuthentificationStore} from '../../../service/auth/authentification.store';
import {Subscription} from 'rxjs/Subscription';
import {AutoUnsubscribe} from '../../../shared/decorator/autoUnsubscribe';

@Component({
    selector: 'app-workflow-run',
    templateUrl: './workflow.run.html',
    styleUrls: ['./workflow.run.scss']
})
@AutoUnsubscribe()
export class WorkflowRunComponent implements OnDestroy {

    project: Project;
    runWorkflowWorker: CDSWorker;
    runSubsription: Subscription;
    workflowRun: WorkflowRun;
    zone: NgZone;
    workflowName: string;

    constructor(private _activatedRoute: ActivatedRoute, private _authStore: AuthentificationStore, private _router: Router) {
        this.zone = new NgZone({enableLongStackTrace: false});
        // Update data if route change
        this._activatedRoute.data.subscribe(datas => {
            this.project = datas['project'];
        });

        this._activatedRoute.parent.params.subscribe(params => {
            this.workflowName = params['workflowName'];
        });
        this._activatedRoute.params.subscribe(params => {
            let number = params['number'];
            if (this.project.key && this.workflowName && number) {
                // Start web worker
                if (this.runWorkflowWorker) {
                    this.runWorkflowWorker.stop();
                }
                this.runWorkflowWorker = new CDSWorker('./assets/worker/web/workflow2.js');
                this.runWorkflowWorker.start({
                    'user': this._authStore.getUser(),
                    'session': this._authStore.getSessionToken(),
                    'api': environment.apiURL,
                    key: this.project.key,
                    workflowName: this.workflowName,
                    number: number
                });
                this.runSubsription = this.runWorkflowWorker.response().subscribe(wrString => {
                    this.zone.run(() => {
                        let wrUpdated = <WorkflowRun>JSON.parse(wrString);
                        if (this.workflowRun && this.workflowRun.last_modified === wrUpdated.last_modified) {
                            return;
                        }
                       this.workflowRun = <WorkflowRun>JSON.parse(wrString);
                    });
                });
            }
        });
    }

    ngOnDestroy(): void {
        if (this.runWorkflowWorker) {
            this.runWorkflowWorker.stop();
        }
    }
}
