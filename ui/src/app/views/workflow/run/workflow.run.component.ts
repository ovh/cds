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

        this._activatedRoute.params.first().subscribe(params => {
            let key = params['key'];
            this.workflowName = params['workflowName'];
            let number = params['number'];
            if (key && this.workflowName && number) {
                // Start web worker
                this.runWorkflowWorker = new CDSWorker('./assets/worker/web/workflow2.js');
                this.runWorkflowWorker.start({
                    'user': this._authStore.getUser(),
                    'session': this._authStore.getSessionToken(),
                    'api': environment.apiURL,
                    key: key,
                    workflowName: this.workflowName,
                    number: number
                });
                this.runSubsription = this.runWorkflowWorker.response().subscribe(wrString => {
                    this.zone.run(() => {
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
