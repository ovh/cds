import {Component, NgZone, OnDestroy} from '@angular/core';
import {ActivatedRoute, Router} from '@angular/router';
import {CDSWorker} from '../../../../shared/worker/worker';
import {AuthentificationStore} from '../../../../service/auth/authentification.store';
import {environment} from '../../../../../environments/environment';
import {WorkflowNodeRun} from '../../../../model/workflow.run.model';
import {Subscription} from 'rxjs/Subscription';
import {AutoUnsubscribe} from '../../../../shared/decorator/autoUnsubscribe';
import {PipelineStatus} from '../../../../model/pipeline.model';

@Component({
    selector: 'app-workflow-run-node',
    templateUrl: './node.html',
    styleUrls: ['./node.scss']
})
@AutoUnsubscribe()
export class WorkflowNodeRunComponent implements OnDestroy {

    nodeRunWorker: CDSWorker;
    nodeRun: WorkflowNodeRun;
    zone: NgZone;
    runSubscription: Subscription;

    // Context info
    projectKey: string;
    workflowName: string;

    selectedTab: string;

    constructor(private _activatedRoute: ActivatedRoute, private _authStore: AuthentificationStore,
        private _router: Router) {
        this.zone = new NgZone({enableLongStackTrace: false});

        this._activatedRoute.queryParams.subscribe(q => {
            if (q['tab']) {
                this.selectedTab = q['tab'];
            } else {
                this.selectedTab = 'workflow';
            }
        });

        this._activatedRoute.params.first().subscribe(params => {
            this.projectKey = params['key'];
            this.workflowName = params['workflowName'];
            let number = params['number'];
            let nodeRunId = params['nodeId'];

            if (this.projectKey && this.workflowName && number && nodeRunId) {
                // Start web worker
                this.nodeRunWorker = new CDSWorker('./assets/worker/web/noderun.js');
                this.nodeRunWorker.start({
                    'user': this._authStore.getUser(),
                    'session': this._authStore.getSessionToken(),
                    'api': environment.apiURL,
                    key: this.projectKey,
                    workflowName: this.workflowName,
                    number: number,
                    nodeRunId: nodeRunId
                });
                this.runSubscription = this.nodeRunWorker.response().subscribe(wrString => {
                    if (!wrString) {
                        return;
                    }
                    this.zone.run(() => {
                        this.nodeRun = <WorkflowNodeRun>JSON.parse(wrString);

                        if (this.nodeRun && this.nodeRun.status === PipelineStatus.SUCCESS || this.nodeRun.status === PipelineStatus.FAIL) {
                            this.nodeRunWorker.stop();
                            this.nodeRunWorker = undefined;
                        }
                    });
                });
            }
        });
    }

    ngOnDestroy(): void {
        if (this.nodeRunWorker) {
            this.nodeRunWorker.stop();
        }
    }


    showTab(tab: string): void {
        this._router.navigateByUrl('/project/' + this.projectKey +
            '/workflow/' + this.workflowName +
            '/run/' + this.nodeRun.num +
            '/node/' + this.nodeRun.id,
            '?&tab=' + tab);
    }
}
