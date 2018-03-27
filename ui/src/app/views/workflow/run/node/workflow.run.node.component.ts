import {Component, NgZone, OnDestroy} from '@angular/core';
import {ActivatedRoute, Router} from '@angular/router';
import {CDSWorker} from '../../../../shared/worker/worker';
import {AuthentificationStore} from '../../../../service/auth/authentification.store';
import {environment} from '../../../../../environments/environment';
import {WorkflowNodeRun, WorkflowRun} from '../../../../model/workflow.run.model';
import {Subscription} from 'rxjs/Subscription';
import {AutoUnsubscribe} from '../../../../shared/decorator/autoUnsubscribe';
import {PipelineStatus} from '../../../../model/pipeline.model';
import {Project} from '../../../../model/project.model';
import {RouterService} from '../../../../service/router/router.service';
import {WorkflowRunService} from '../../../../service/workflow/run/workflow.run.service';
import {DurationService} from '../../../../shared/duration/duration.service';
import {first} from 'rxjs/operators';
import {WorkflowCoreService} from '../../../../service/workflow/workflow.core.service';

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
    project: Project;
    workflowName: string;
    duration: string;

    workflowRun: WorkflowRun;
    // History
    nodeRunsHistory = new Array<WorkflowNodeRun>();

    selectedTab: string;

    constructor(private _activatedRoute: ActivatedRoute, private _authStore: AuthentificationStore,
                private _router: Router, private _routerService: RouterService, private _workflowRunService: WorkflowRunService,
                private _durationService: DurationService, private _workflowCoreSerivce: WorkflowCoreService) {
        this.zone = new NgZone({enableLongStackTrace: false});

        this._activatedRoute.data.subscribe(datas => {
            this.project = datas['project'];
        });


        this._activatedRoute.queryParams.subscribe(q => {
            if (q['tab']) {
                this.selectedTab = q['tab'];
            } else {
                this.selectedTab = 'pipeline';
            }
        });

        this.workflowName = this._routerService.getRouteSnapshotParams({}, this._router.routerState.snapshot.root)['workflowName'];

        this._activatedRoute.params.subscribe(params => {
            this.nodeRun = null;
            let number = params['number'];
            let nodeRunId = params['nodeId'];

            if (this.project && this.project.key && this.workflowName && number && nodeRunId) {
                // Get workflow Run
                this._workflowRunService.getWorkflowRun(this.project.key, this.workflowName, number)
                    .pipe(first())
                    .subscribe(wr => {
                        this.workflowRun = wr;
                        this._workflowCoreSerivce.setCurrentWorkflowRun(this.workflowRun);
                    });

                this.startWorker(number, nodeRunId);
                this.runSubscription = this.nodeRunWorker.response().subscribe(wrString => {
                    if (!wrString) {
                        return;
                    }
                    let historyChecked = false;
                    this.zone.run(() => {
                        this.nodeRun = <WorkflowNodeRun>JSON.parse(wrString);
                        if (!historyChecked) {
                            historyChecked = true;
                            this._workflowRunService.nodeRunHistory(
                                this.project.key, this.workflowName,
                                number, this.nodeRun.workflow_node_id)
                            .pipe(first())
                            .subscribe(nrs => this.nodeRunsHistory = nrs);
                        }

                        if (this.nodeRun && !PipelineStatus.isActive(this.nodeRun.status)) {
                            this.nodeRunWorker.stop();
                            this.nodeRunWorker = undefined;
                            this.duration = this._durationService.duration(new Date(this.nodeRun.start), new Date(this.nodeRun.done));
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

    startWorker(number: number, nodeRunId: number) {
      if (this.nodeRunWorker) {
        this.nodeRunWorker.stop();
      }
      // Start web worker
      this.nodeRunWorker = new CDSWorker('./assets/worker/web/noderun.js');
      this.nodeRunWorker.start({
          'user': this._authStore.getUser(),
          'session': this._authStore.getSessionToken(),
          'api': environment.apiURL,
          key: this.project.key,
          workflowName: this.workflowName,
          number: number,
          nodeRunId: nodeRunId
      });
    }

    showTab(tab: string): void {
        this._router.navigateByUrl('/project/' + this.project.key +
            '/workflow/' + this.workflowName +
            '/run/' + this.nodeRun.num +
            '/node/' + this.nodeRun.id +
            '?&tab=' + tab);
    }
}
