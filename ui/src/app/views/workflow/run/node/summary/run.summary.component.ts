import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnInit, ViewChild } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { finalize, first } from 'rxjs/operators';
import { PipelineStatus } from '../../../../../model/pipeline.model';
import { Project } from '../../../../../model/project.model';
import { WNode, Workflow } from '../../../../../model/workflow.model';
import { WorkflowNodeRun } from '../../../../../model/workflow.run.model';
import { WorkflowRunService } from '../../../../../service/workflow/run/workflow.run.service';
import { ToastService } from '../../../../../shared/toast/ToastService';
import { WorkflowNodeRunParamComponent } from '../../../../../shared/workflow/node/run/node.run.param.component';
import { Select, Store } from '@ngxs/store';
import { ProjectState } from 'app/store/project.state';
import { WorkflowState, WorkflowStateModel } from 'app/store/workflow.state';
import { Observable, Subscription } from 'rxjs';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { DurationService } from 'app/shared/duration/duration.service';

@Component({
    selector: 'app-workflow-node-run-summary',
    templateUrl: './run.summary.html',
    styleUrls: ['./run.summary.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowNodeRunSummaryComponent implements OnInit {

    duration: string;

    @ViewChild('workflowNodeRunParam', {static: false})
    runWithParamComponent: WorkflowNodeRunParamComponent;

    @Select(WorkflowState.getSelectedNodeRun()) nodeRun$: Observable<WorkflowNodeRun>;
    nodeRunSubs: Subscription;

    workflow: Workflow;
    project: Project;

    node: WNode;
    pipelineStatusEnum = PipelineStatus;

    nodeRunStatus: string;
    nodeRunID: number;
    nodeRunNum: number;
    nodeRunSubNum: number;
    nodeRunStart: string;

    loading = false;

    constructor(
        private _wrService: WorkflowRunService,
        private _toast: ToastService,
        private _translate: TranslateService,
        private _cd: ChangeDetectorRef,
        private _durationService: DurationService,
        private _store: Store) {
        this.project = this._store.selectSnapshot(ProjectState.projectSnapshot);
        this.workflow = (<WorkflowStateModel>this._store.selectSnapshot(WorkflowState)).workflowRun.workflow;
    }

    ngOnInit(): void {
        this.nodeRunSubs = this.nodeRun$.subscribe(nr => {
            if (!nr) {
                return;
            }
            if (this.nodeRunID !== nr.id) {
                this.node = Workflow.getNodeByID(nr.workflow_node_id, this.workflow);
                this.nodeRunID = nr.id;
                this.nodeRunNum = nr.num;
                this.nodeRunSubNum = nr.subnumber;
                this.nodeRunStart = nr.start;
                this.nodeRunStatus = nr.status;
                if (!PipelineStatus.isActive(nr.status)) {
                    this.duration = this._durationService.duration(new Date(nr.start), new Date(nr.done));
                }
                this._cd.markForCheck();
            } else if (this.nodeRunStatus !== nr.status) {
                this.nodeRunStatus = nr.status;
                if (!PipelineStatus.isActive(nr.status)) {
                    this.duration = this._durationService.duration(new Date(nr.start), new Date(nr.done));
                }
                this._cd.markForCheck();
            }
        });

    }

    stop(): void {
        this.loading = true;
        this._cd.markForCheck();
        this._wrService.stopNodeRun(this.project.key, this.workflow.name, this.nodeRunNum, this.nodeRunID)
            .pipe(
                first(),
                finalize(() => {
                    this.loading = false;
                    this._cd.markForCheck();
                })
            ).subscribe(() => {
            this._toast.success('', this._translate.instant('pipeline_stop'));
        });
    }

    runNewWithParameter(): void {
        if (this.runWithParamComponent) {
            this.runWithParamComponent.show();
        }
    }
}
