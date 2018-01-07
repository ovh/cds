import {Component, ViewChild, OnInit} from '@angular/core';
import {SemanticSidebarComponent} from 'ng-semantic/ng-semantic';
import {ActivatedRoute, ResolveEnd, Router} from '@angular/router';
import {Project} from '../../model/project.model';
import {Subscription} from 'rxjs/Subscription';
import {AutoUnsubscribe} from '../../shared/decorator/autoUnsubscribe';
import {Workflow, WorkflowNode, WorkflowNodeJoin} from '../../model/workflow.model';
import {WorkflowStore} from '../../service/workflow/workflow.store';
import {ProjectStore} from '../../service/project/project.store';
import {RouterService} from '../../service/router/router.service';
import {WorkflowCoreService} from '../../service/workflow/workflow.core.service';
import {cloneDeep} from 'lodash';

@Component({
    selector: 'app-workflow',
    templateUrl: './workflow.html',
    styleUrls: ['./workflow.scss']
})
@AutoUnsubscribe()
export class WorkflowComponent implements OnInit {

    project: Project;
    workflow: Workflow;
    loading = true;
    number: number;
    workflowSubscription: Subscription;
    sideBarSubscription: Subscription;
    projectSubscription: Subscription;
    sidebarOpen: boolean;
    currentNodeName: string;
    selectedNodeId: number;
    selectedNode: WorkflowNode;
    selectedJoinId: number;
    selectedJoin: WorkflowNodeJoin;
    selectedNodeRunId: number;
    selectedNodeRunNum: number;

    @ViewChild('invertedSidebar')
    sidebar: SemanticSidebarComponent;

    constructor(private _activatedRoute: ActivatedRoute,
                private _workflowStore: WorkflowStore,
                private _router: Router,
                private _routerService: RouterService,
                private _projectStore: ProjectStore,
                private _workflowCore: WorkflowCoreService) {
        this._activatedRoute.data.subscribe(datas => {
            console.log(datas);
            this.project = datas['project'];
        });

        this.sideBarSubscription = this._workflowCore.getSidebarStatus().subscribe(b => {
            this.sidebarOpen = b;
        });

        this._activatedRoute.params.subscribe(p => {
            let workflowName = p['workflowName'];
            let key = p['key'];
            let snapshotparams = this._routerService.getRouteSnapshotParams({}, this._activatedRoute.snapshot);
            if (snapshotparams) {
                this.number = snapshotparams['number'] ? parseInt(snapshotparams['number'], 10) : null;
            }

            if (this.project.key && workflowName) {
                if (this.workflowSubscription) {
                    this.workflowSubscription.unsubscribe();
                }
                this.loading = true;
                this.workflowSubscription = this._workflowStore.getWorkflows(key, workflowName)
                    .subscribe(ws => {
                        if (ws) {
                            let updatedWorkflow = ws.get(key + '-' + workflowName);
                            if (updatedWorkflow && !updatedWorkflow.externalChange) {
                                this.workflow = updatedWorkflow;
                            }

                            if (this.selectedNodeId) {
                                this.selectedNode = Workflow.getNodeByID(this.selectedNodeId, this.workflow);
                            } else if (this.selectedJoinId) {
                                this.selectedJoin = Workflow.getJoinById(this.selectedJoinId, this.workflow);
                            }
                        }
                        this.loading = false;
                    }, () => {
                        this.loading = false;
                        this._router.navigate(['/project', key]);
                    });
            }
        });

        let qp = this._routerService.getRouteSnapshotQueryParams({}, this._activatedRoute.snapshot);
        if (qp) {
            this.currentNodeName = qp['name'];
        }

        this.listenQueryParams();

        this._router.events.subscribe(p => {
            if (p instanceof ResolveEnd) {
                let params = this._routerService.getRouteSnapshotParams({}, p.state.root);
                let queryParams = this._routerService.getRouteSnapshotQueryParams({}, p.state.root);
                this.currentNodeName = queryParams['name'];
                this.number = params['number'] ? parseInt(params['number'], 10) : null;
                if (queryParams['selectedNodeId']) {
                    this.selectedNodeId = Number.isNaN(queryParams['selectedNodeId']) ? null : parseInt(queryParams['selectedNodeId'], 10);
                }
                if (queryParams['selectedJoinId']) {
                    this.selectedJoinId = Number.isNaN(queryParams['selectedJoinId']) ? null : parseInt(queryParams['selectedJoinId'], 10);
                }

                if (this.selectedNodeId && !this.loading) {
                    this.selectedNode = Workflow.getNodeByID(this.selectedNodeId, this.workflow);
                } else if (this.selectedJoinId && !this.loading) {
                    this.selectedJoin = Workflow.getJoinById(this.selectedJoinId, this.workflow);
                }
            }
        });
    }

    ngOnInit() {
        this.projectSubscription = this._projectStore.getProjects(this.project.key)
          .subscribe((proj) => {
            if (!this.project || !proj || !proj.get(this.project.key)) {
              return;
            }
            this.project = proj.get(this.project.key);
          });
    }

    listenQueryParams() {
      this._activatedRoute.queryParams.subscribe((queryp) => {
          if (queryp['selectedNodeId']) {
              this.selectedJoinId = null;
              this.selectedJoin = null;
              this.selectedNodeId = Number.isNaN(queryp['selectedNodeId']) ? null : parseInt(queryp['selectedNodeId'], 10);
          } else {
              this.selectedNodeId = null;
              this.selectedNode = null;
          }

          if (queryp['selectedJoinId']) {
              this.selectedNodeId = null;
              this.selectedNode = null;
              this.selectedJoinId = Number.isNaN(queryp['selectedJoinId']) ? null : parseInt(queryp['selectedJoinId'], 10);
          } else {
              this.selectedJoinId = null;
              this.selectedJoin = null;
          }

          if (queryp['selectedNodeRunId'] || queryp['selectedNodeRunNum']) {
              this.selectedJoinId = null;
              this.selectedJoin = null;
              this.selectedNodeRunId = Number.isNaN(queryp['selectedNodeRunId']) ? null : parseInt(queryp['selectedNodeRunId'], 10);
              this.selectedNodeRunNum = Number.isNaN(queryp['selectedNodeRunNum']) ? null : parseInt(queryp['selectedNodeRunNum'], 10);
          } else {
              this.selectedNodeRunId = null;
              this.selectedNodeRunNum = null;
          }

          if (this.selectedNodeId && !this.loading && this.workflow) {
              this.selectedNode = Workflow.getNodeByID(this.selectedNodeId, this.workflow);
          } else if (this.selectedJoinId && !this.loading && this.workflow) {
              this.selectedJoin = Workflow.getJoinById(this.selectedJoinId, this.workflow);
          }
      });
    }

    toggleSidebar(): void {
        this._workflowCore.moveSideBar(!this.sidebarOpen);
    }

    closeEditSidebar(): void {
        let qps = cloneDeep(this._activatedRoute.snapshot.queryParams);
        let snapshotparams = this._routerService.getRouteSnapshotParams({}, this._activatedRoute.snapshot);
        qps['selectedNodeId'] = null;
        qps['selectedJoinId'] = null;
        qps['selectedNodeRunNum'] = null;
        qps['selectedNodeRunId'] = null;
        this.selectedNode = null;
        this.selectedNodeId = null;
        this.selectedJoin = null;
        this.selectedJoinId = null;
        this.selectedNodeRunNum = null;
        this.selectedNodeRunId = null;

        this._workflowCore.linkJoinEvent(null);

        if (snapshotparams['number']) {
          this._router.navigate([
            '/project',
            this.project.key,
            'workflow',
            this.workflow.name,
            'run',
            snapshotparams['number']
          ], {queryParams: qps});
        } else {
          this._router.navigate(['/project', this.project.key, 'workflow', this.workflow.name], {queryParams: qps});
        }
    }

    displayToggleButton(): boolean {
        return this.selectedNode == null && this.selectedJoin == null &&
            this.selectedNodeRunId == null && this.selectedNodeRunNum == null;
    }
}
