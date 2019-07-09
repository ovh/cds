import { Component, OnInit, ViewChild } from '@angular/core';
import { Title } from '@angular/platform-browser';
import { ActivatedRoute } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { PipelineStatus } from 'app/model/pipeline.model';
import { Project } from 'app/model/project.model';
import { WNode, Workflow } from 'app/model/workflow.model';
import { WorkflowRun } from 'app/model/workflow.run.model';
import { NotificationService } from 'app/service/notification/notification.service';
import { WorkflowStore } from 'app/service/workflow/workflow.store';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { WorkflowNodeRunParamComponent } from 'app/shared/workflow/node/run/node.run.param.component';
import { ProjectState, ProjectStateModel } from 'app/store/project.state';
import { ChangeToRunView, GetWorkflowRun } from 'app/store/workflow.action';
import { WorkflowState, WorkflowStateModel } from 'app/store/workflow.state';
import cloneDeep from 'lodash-es/cloneDeep';
import { Subscription } from 'rxjs';
import { filter } from 'rxjs/operators';
import { ErrorMessageMap } from './errors';

@Component({
    selector: 'app-workflow-run',
    templateUrl: './workflow.run.html',
    styleUrls: ['./workflow.run.scss']
})
@AutoUnsubscribe()
export class WorkflowRunComponent implements OnInit {
    @ViewChild('workflowRunParam', { static: false })
    runWithParamComponent: WorkflowNodeRunParamComponent;

    workflow: Workflow;
    subWorkflow: Subscription;

    project: Project;
    workflowRun: WorkflowRun;
    project$: Subscription;
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
    loadingRun = true;
    errorsMap = ErrorMessageMap;

    // copy of root node to send it into run modal
    nodeToRun: WNode;

    constructor(
        private _store: Store,
        private _activatedRoute: ActivatedRoute,
        private _workflowStore: WorkflowStore,
        private _notification: NotificationService,
        private _translate: TranslateService,
        private _titleService: Title
    ) {
        // Get project
        this.dataSubs = this._activatedRoute.data.subscribe(datas => {
            if (!this.project || (<Project>datas['project']).key !== this.project.key) {
                this.project = datas['project'];
                this.workflowRun = null;
                this.workflowName = '';
            }
        });

        this.project$ = this._store.select(ProjectState)
            .pipe(filter((prj) => prj != null))
            .subscribe((projState: ProjectStateModel) => this.project = projState.project);

        this.workflowName = this._activatedRoute.snapshot.parent.params['workflowName'];
        this._store.dispatch(new ChangeToRunView({}));

        // Subscribe to route event
        this.paramsSubs = this._activatedRoute.params.subscribe(ps => {
            // if there is no current workflow run
            if (!this.workflowRun) {
                this._store.dispatch(
                    new GetWorkflowRun({
                        projectKey: this.project.key,
                        workflowName: this.workflowName,
                        num: ps['number']
                    }));
            } else {
                if (this.workflowRun.workflow.name !== this.workflowName || this.workflowRun.num !== ps['number']) {
                    this._store.dispatch(
                        new GetWorkflowRun({
                            projectKey: this.project.key,
                            workflowName: this.workflowName,
                            num: ps['number']
                        }));
                }
            }
        });

        this.subWorkflow = this._store.select(WorkflowState.getCurrent()).subscribe((s: WorkflowStateModel) => {
            this.loadingRun = s.loadingWorkflowRun;
            if (s.workflowRun) {
                this.workflow = s.workflow;
                this.workflowName = this.workflow.name;

                let previousWR: WorkflowRun;
                if (this.workflowRun) {
                    previousWR = this.workflowRun;
                }
                this.workflowRun = s.workflowRun;
                if (previousWR && this.workflowRun && previousWR.id === s.workflowRun.id && previousWR.status !== this.workflowRun.status &&
                    PipelineStatus.isDone(this.workflowRun.status)) {
                    this.handleNotification();
                }
                this.updateTitle();
            } else {
                delete this.workflowRun;
            }
        });
    }

    ngOnInit(): void {
        this.direction = this._workflowStore.getDirection(this.project.key, this.workflowName);
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
