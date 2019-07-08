import { ChangeDetectionStrategy, Component, Input, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { PipelineStatus } from 'app/model/pipeline.model';
import { Stage } from 'app/model/stage.model';
import { WNode } from 'app/model/workflow.model';
import { WorkflowNodeJobRun, WorkflowRun } from 'app/model/workflow.run.model';

@Component({
    selector: 'app-job-step-summary',
    templateUrl: './job.summary.component.html',
    styleUrls: ['./job.summary.component.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class JobStepSummaryComponent implements OnInit {

    @Input() job: WorkflowNodeJobRun;
    @Input() workflowRun: WorkflowRun;
    @Input() workflowNode: WNode;
    @Input() stage: Stage;

    open = false;
    warning = false;

    constructor(private _router: Router, private _route: ActivatedRoute) {

    }

    ngOnInit() {
        this.open = this.job.status === PipelineStatus.FAIL || PipelineStatus.isActive(this.job.status);

        if (this.job.status === PipelineStatus.SUCCESS && this.job.job && Array.isArray(this.job.job.step_status)) {
            this.warning = this.job.job.step_status.reduce((fail, step) => fail || step.status === PipelineStatus.FAIL, false);
        }
    }

    goToJobLogs() {
        this._router.navigate([
            'project',
            this._route.snapshot.params['key'],
            'workflow',
            this._route.snapshot.params['workflowName'],
            'run',
            this.workflowRun.num,
            'node',
            this.job.workflow_node_run_id
        ], {
            queryParams: {
                stageId: this.stage.id,
                actionId: this.job.job.pipeline_action_id,
                selectedNodeRunId: this.job.workflow_node_run_id,
                selectedNodeRunNum: this.workflowRun.num,
                selectedNodeId: this._route.snapshot.queryParams['selectedNodeId'],
                name: this.workflowNode.name
            }
        });
    }
}
