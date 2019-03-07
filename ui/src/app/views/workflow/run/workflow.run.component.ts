import { Component, OnInit, ViewChild } from '@angular/core';
import { Title } from '@angular/platform-browser';
import { ActivatedRoute } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { FetchWorkflow } from 'app/store/workflows.action';
import { WorkflowsState } from 'app/store/workflows.state';
import { cloneDeep } from 'lodash';
import { Subscription } from 'rxjs';
import { filter } from 'rxjs/operators';
import { PipelineStatus } from '../../../model/pipeline.model';
import { Project } from '../../../model/project.model';
import { WNode, Workflow } from '../../../model/workflow.model';
import { WorkflowNodeRun, WorkflowRun } from '../../../model/workflow.run.model';
import { NotificationService } from '../../../service/notification/notification.service';
import { WorkflowRunService } from '../../../service/workflow/run/workflow.run.service';
import { WorkflowEventStore } from '../../../service/workflow/workflow.event.store';
import { WorkflowStore } from '../../../service/workflow/workflow.store';
import { AutoUnsubscribe } from '../../../shared/decorator/autoUnsubscribe';
import { WorkflowNodeRunParamComponent } from '../../../shared/workflow/node/run/node.run.param.component';

@Component({
    selector: 'app-workflow-run',
    templateUrl: './workflow.run.html',
    styleUrls: ['./workflow.run.scss']
})
@AutoUnsubscribe()
export class WorkflowRunComponent implements OnInit {
    @ViewChild('workflowRunParam')
    runWithParamComponent: WorkflowNodeRunParamComponent;

    workflow: Workflow;
    subWorkflow: Subscription;

    project: Project;
    workflowRun: WorkflowRun;
    subRun: Subscription;

    workflowName: string;
    version: string;
    direction: string;

    selectedNodeID: number;
    selectedNodeRef: string;

    pipelineStatusEnum = PipelineStatus;
    notificationSubscription: Subscription;
    dataSubs: Subscription;
    paramsSubs: Subscription;
    parentParamsSubs: Subscription;
    qpsSubs: Subscription;
    loadingRun = false;

    // copy of root node to send it into run modal
    nodeToRun: WNode;

    constructor(
        private store: Store,
        private _activatedRoute: ActivatedRoute,
        private _workflowStore: WorkflowStore,
        private _notification: NotificationService,
        private _translate: TranslateService,
        private _workflowEventStore: WorkflowEventStore,
        private _workflowRunService: WorkflowRunService,
        private _titleService: Title
    ) {
        this._workflowEventStore.setSelectedNodeRun(null, false);

        // Get project
        this.dataSubs = this._activatedRoute.data.subscribe(datas => {
            if (!this.project || (<Project>datas['project']).key !== this.project.key) {
                this.project = datas['project'];
                this.workflowRun = null;
                this.workflowName = '';
            }
        });

        // Get workflow
        this.parentParamsSubs = this._activatedRoute.parent.params.subscribe(params => {
            this.workflowName = params['workflowName'];
            this.store.dispatch(new FetchWorkflow({
                projectKey: params['key'],
                workflowName: this.workflowName
            }));
        });

        // Get workflow run
        this.subRun = this._workflowEventStore.selectedRun().subscribe(wr => {
            let previousWR: WorkflowRun;
            if (this.workflowRun) {
                previousWR = this.workflowRun;
            }
            this.workflowRun = wr;
            if (previousWR && this.workflowRun && previousWR.id === wr.id && previousWR.status !== this.workflowRun.status &&
                PipelineStatus.isDone(this.workflowRun.status)) {
                this.handleNotification();
            }
            this.updateTitle();
            this.selectNode();
        });

        // Subscribe to route event
        this.paramsSubs = this._activatedRoute.params.subscribe(ps => {
            // if there is no current workflow run
            if (!this.workflowRun) {
                this.initWorkflowRun(ps['number']);
            } else {
                if (this.workflowRun.workflow.name !== this.workflowName || this.workflowRun.num !== ps['number']) {
                    this.initWorkflowRun(ps['number']);
                }
            }
        });

        this.qpsSubs = this._activatedRoute.queryParams.subscribe(params => {
            this.selectedNodeID = params['node_id'];
            this.selectedNodeRef = params['node_ref'];
            this.selectNode();
        });
    }

    ngOnInit(): void {
        this.direction = this._workflowStore.getDirection(this.project.key, this.workflowName);
        let projectKey = this._activatedRoute.snapshot.params['key'];
        let wfName = this._activatedRoute.snapshot.params['workflowName'];
        this.store.select(WorkflowsState.selectWorkflow(projectKey, wfName))
            .pipe(filter((wf) => wf != null))
            .subscribe((wf) => this.workflow = wf);
    }

    selectNode() {
        if (!this.workflowRun) {
            return;
        }
        if (this.selectedNodeID) {
            let n = Workflow.getNodeByID(this.selectedNodeID, this.workflowRun.workflow);
            if (n) {
                this._workflowEventStore.setSelectedNode(n, false);
                let nr: WorkflowNodeRun;
                if (this.workflowRun.nodes && this.workflowRun.nodes[n.id]) {
                    nr = this.workflowRun.nodes[n.id][0];
                }
                this._workflowEventStore.setSelectedNodeRun(nr, true);
                return;
            }
        }
        if (this.selectedNodeRef) {
            let n = Workflow.getNodeByRef(this.selectedNodeRef, this.workflowRun.workflow);
            if (n) {
                this._workflowEventStore.setSelectedNode(n, false);
                let nr: WorkflowNodeRun;
                if (this.workflowRun.nodes && this.workflowRun.nodes[n.id]) {
                    nr = this.workflowRun.nodes[n.id][0];
                }
                this._workflowEventStore.setSelectedNodeRun(nr, true);
                return;
            }
        }
        this._workflowEventStore.setSelectedNode(null, true);
    }

    initWorkflowRun(num): void {
        this.loadingRun = true;
        this._workflowRunService.getWorkflowRun(this.project.key, this.workflowName, num).subscribe(wr => {
            this.workflowRun = wr;

            this._workflowEventStore.setSelectedRun(this.workflowRun);
            this.loadingRun = false;
        });
    }

    handleNotification() {
        if (this.workflowRun.num !== parseInt(this._activatedRoute.snapshot.params['number'], 10)) {
            return;
        }

        switch (this.workflowRun.status) {
            case PipelineStatus.SUCCESS:
                this.notificationSubscription = this._notification.create(this._translate.instant('notification_on_workflow_success', {
                    workflowName: this.workflowName,
                }), {
                        icon: 'assets/images/checked.png',
                        tag: `${this.workflowName}-${this.workflowRun.num}.${this.workflowRun.last_subnumber}`
                    }).subscribe();
                break;
            case PipelineStatus.FAIL:
                this.notificationSubscription = this._notification.create(this._translate.instant('notification_on_workflow_failing', {
                    workflowName: this.workflowName
                }), {
                        icon: 'assets/images/close.png',
                        tag: `${this.workflowName}-${this.workflowRun.num}.${this.workflowRun.last_subnumber}`
                    }).subscribe();
                break;
        }
    }

    updateTitle() {
        if (!this.workflowRun || !Array.isArray(this.workflowRun.tags)) {
            return;
        }
        let branch = this.workflowRun.tags.find((tag) => tag.tag === 'git.branch');
        if (branch) {
            this._titleService.setTitle(`#${this.workflowRun.num} [${branch.value}] • ${this.workflowName}`);
        }
    }

    relaunch() {
        if (this.runWithParamComponent && this.runWithParamComponent.show) {
            let rootNodeRun = this.workflowRun.nodes[this.workflowRun.workflow.workflow_data.node.id][0];
            this.nodeToRun = cloneDeep(this.workflowRun.workflow.workflow_data.node);
            if (rootNodeRun.hook_event) {
                this.nodeToRun.context.default_payload = rootNodeRun.hook_event.payload;
                this.nodeToRun.context.default_pipeline_parameters = rootNodeRun.hook_event.pipeline_parameter;
            }
            if (rootNodeRun.manual) {
                this.nodeToRun.context.default_payload = rootNodeRun.manual.payload;
                this.nodeToRun.context.default_pipeline_parameters = rootNodeRun.manual.pipeline_parameter;
            }

            setTimeout(() => this.runWithParamComponent.show(), 100);
        }
    }
}
