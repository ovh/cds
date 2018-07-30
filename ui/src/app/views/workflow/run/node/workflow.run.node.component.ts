import {Component} from '@angular/core';
import {ActivatedRoute, NavigationExtras, Router} from '@angular/router';
import {Subscription} from 'rxjs';
import {first} from 'rxjs/operators';
import {PipelineStatus} from '../../../../model/pipeline.model';
import {Project} from '../../../../model/project.model';
import {Workflow} from '../../../../model/workflow.model';
import {WorkflowNodeRun, WorkflowRun} from '../../../../model/workflow.run.model';
import {AuthentificationStore} from '../../../../service/auth/authentification.store';
import {RouterService} from '../../../../service/router/router.service';
import {WorkflowRunService} from '../../../../service/workflow/run/workflow.run.service';
import {WorkflowEventStore} from '../../../../service/workflow/workflow.event.store';
import {AutoUnsubscribe} from '../../../../shared/decorator/autoUnsubscribe';
import {DurationService} from '../../../../shared/duration/duration.service';

@Component({
    selector: 'app-workflow-run-node',
    templateUrl: './node.html',
    styleUrls: ['./node.scss']
})
@AutoUnsubscribe()
export class WorkflowNodeRunComponent {

    nodeRun: WorkflowNodeRun;
    subNodeRun: Subscription;

    // Context info
    project: Project;
    workflowName: string;

    duration: string;

    workflowRun: WorkflowRun;

    // History
    nodeRunsHistory = new Array<WorkflowNodeRun>();
    selectedTab: string;

    isAdmin: boolean;

    constructor(private _activatedRoute: ActivatedRoute,
                private _router: Router, private _routerService: RouterService, private _workflowRunService: WorkflowRunService,
                private _durationService: DurationService, private _authStore: AuthentificationStore,
                private _workflowEventStore: WorkflowEventStore) {

        this._activatedRoute.data.subscribe(datas => {
            this.project = datas['project'];
        });

        this.isAdmin = this._authStore.getUser().admin;

        // Tab selection
        this._activatedRoute.queryParams.subscribe(q => {
            if (q['tab']) {
                this.selectedTab = q['tab'];
            } else {
                this.selectedTab = 'pipeline';
            }
        });

        // Get workflow name
        this.workflowName = this._routerService.getRouteSnapshotParams({}, this._router.routerState.snapshot.root)['workflowName'];

        this._activatedRoute.params.subscribe(params => {
            this.nodeRun = null;
            let number = params['number'];
            let nodeRunId = params['nodeId'];

            if (this.project && this.project.key && this.workflowName && number && nodeRunId) {
                // Get workflow Run
                this.initWorkflowNodeRun(number, nodeRunId);
            }
        });
    }

    initWorkflowNodeRun(number, nodeRunId): void {
        this._workflowRunService.getWorkflowRun(this.project.key, this.workflowName, number).pipe(first()).subscribe(wr => {
            this.workflowRun = wr;
            this._workflowEventStore.setSelectedRun(this.workflowRun);

            let historyChecked = false;
            this.subNodeRun = this._workflowRunService.getWorkflowNodeRun(this.project.key, this.workflowName, number, nodeRunId)
                .pipe(first()).subscribe(nodeRun => {
                this.nodeRun = nodeRun;
                if (!historyChecked) {
                    historyChecked = true;
                    this._workflowRunService.nodeRunHistory(
                        this.project.key, this.workflowName,
                        number, this.nodeRun.workflow_node_id)
                        .pipe(first())
                        .subscribe(nrs => this.nodeRunsHistory = nrs);
                }

                this._workflowEventStore.setSelectedNodeRun(this.nodeRun);
                this.subNodeRun = this._workflowEventStore.selectedNodeRun().subscribe(wnr => {
                    this.nodeRun = wnr;
                    if (this.nodeRun) {
                        this._workflowEventStore.setSelectedNode(
                            Workflow.getNodeByID(this.nodeRun.workflow_node_id, this.workflowRun.workflow),
                            false);
                    }

                    if (this.nodeRun && !PipelineStatus.isActive(this.nodeRun.status)) {
                        this.duration = this._durationService.duration(new Date(this.nodeRun.start), new Date(this.nodeRun.done));
                    }
                });
            });
        });
    }

    showTab(tab: string): void {
        let queryParams = Object.assign({}, this._activatedRoute.snapshot.queryParams, { tab })
        let navExtras: NavigationExtras = { queryParams };
        this._router.navigate(['project', this.project.key,
            'workflow', this.workflowName,
            'run', this.nodeRun.num,
            'node', this.nodeRun.id], navExtras);
    }
}
