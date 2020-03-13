import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { Store } from '@ngxs/store';
import { Job } from 'app/model/job.model';
import { PipelineStatus } from 'app/model/pipeline.model';
import { WorkflowState } from 'app/store/workflow.state';
import { map } from 'rxjs/operators';

@Component({
    selector: 'app-job-step-summary',
    templateUrl: './job.summary.component.html',
    styleUrls: ['./job.summary.component.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class JobStepSummaryComponent implements OnInit {

    // Job Identifier - never change
    @Input() stageId: number;
    @Input() jobId: number;
    @Input() runNumber: number;

    // Data for router
    @Input() nodeName: string;
    @Input() workflowRunNodeId: number;

    // Set only once
    job: Job;

    // Dynamic
    open = false;
    warning = false;
    jobStatus: string;
    stepIds: Array<number>;


    constructor(private _router: Router, private _route: ActivatedRoute, private _store: Store, private _cd: ChangeDetectorRef) {
    }

    ngOnInit() {
        this._store.select(WorkflowState.nodeRunJob).pipe(map(filterFn => filterFn(this.stageId, this.jobId))).subscribe( rj => {
            if (!rj && !this.jobStatus) {
                return;
            }
            let warn = this.warning;
            if (rj && rj.job && rj.job.step_status && rj.status === PipelineStatus.SUCCESS && rj.job && Array.isArray(rj.job.step_status)) {
                warn = rj.job.step_status.reduce((fail, step) => fail || step.status === PipelineStatus.FAIL, false);
            }

            // If no modification, we leave
            if (rj && rj.id === this.jobId && this.jobStatus === rj.status && this.warning === warn) {
                return;
            }

            if (rj) {
                if (!this.job || rj.job.pipeline_action_id !== this.job.pipeline_action_id) {
                    this.job = rj.job;
                }
                this.jobStatus = rj.status;
                if (rj.job && rj.job.step_status) {
                    this.stepIds = new Array<number>();
                    this.stepIds.push(...rj.job.step_status.map(ss => ss.step_order));
                }
                this.open = rj.status === PipelineStatus.FAIL || PipelineStatus.isActive(rj.status);
                this.warning = warn;
            } else {
                delete this.job;
                delete this.jobStatus;
                delete this.stepIds;
            }
            this._cd.detectChanges();
        });
    }

    goToJobLogs() {
        this._router.navigate([
            'project',
            this._route.snapshot.params['key'],
            'workflow',
            this._route.snapshot.params['workflowName'],
            'run',
            this.runNumber,
            'node',
            this.workflowRunNodeId
        ], {
            queryParams: {
                stageId: this.stageId,
                actionId: this.job.pipeline_action_id,
                selectedNodeId: this._route.snapshot.queryParams['selectedNodeId'],
                name: this.nodeName
            }
        });
    }
}
