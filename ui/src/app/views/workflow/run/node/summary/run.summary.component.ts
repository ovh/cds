import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, OnInit } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { ProjectState } from 'app/store/project.state';
import { WorkflowState, WorkflowStateModel } from 'app/store/workflow.state';
import { Subscription } from 'rxjs';
import { finalize, first } from 'rxjs/operators';
import { PipelineStatus } from 'app/model/pipeline.model';
import { Project } from 'app/model/project.model';
import { WNode, Workflow } from 'app/model/workflow.model';
import { WorkflowRunService } from 'app/service/workflow/run/workflow.run.service';
import { ToastService } from 'app/shared/toast/ToastService';
import { WorkflowNodeRunParamComponent } from 'app/shared/workflow/node/run/node.run.param.component';
import { NzModalService } from 'ng-zorro-antd/modal';
import { DurationService } from '../../../../../../../libs/workflow-graph/src/lib/duration.service';

@Component({
    selector: 'app-workflow-node-run-summary',
    templateUrl: './run.summary.html',
    styleUrls: ['./run.summary.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowNodeRunSummaryComponent implements OnInit, OnDestroy {

    duration: string;

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
    readOnlyRun: boolean;

    constructor(
        private _modalService: NzModalService,
        private _wrService: WorkflowRunService,
        private _toast: ToastService,
        private _translate: TranslateService,
        private _cd: ChangeDetectorRef,
        private _store: Store
    ) {
        this.project = this._store.selectSnapshot(ProjectState.projectSnapshot);
        this.workflow = (<WorkflowStateModel>this._store.selectSnapshot(WorkflowState)).workflowRun.workflow;
    }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit(): void {
        this.nodeRunSubs = this._store.select(WorkflowState.getSelectedNodeRun()).subscribe(nr => {
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
                    this.duration = DurationService.duration(new Date(nr.start), new Date(nr.done));
                }
                this._cd.markForCheck();
            } else if (this.nodeRunStatus !== nr.status) {
                this.nodeRunStatus = nr.status;
                if (!PipelineStatus.isActive(nr.status)) {
                    this.duration = DurationService.duration(new Date(nr.start), new Date(nr.done));
                }
                this._cd.markForCheck();
            }
            this.readOnlyRun = this._store.selectSnapshot(WorkflowState)?.workflowRun?.read_only;
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
        this._modalService.create({
            nzWidth: '900px',
            nzTitle: 'Run worklow',
            nzContent: WorkflowNodeRunParamComponent,
        });
    }
}
