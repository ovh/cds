import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { PipelineStatus } from 'app/model/pipeline.model';
import { Store } from '@ngxs/store';
import { WorkflowState } from 'app/store/workflow.state';
import { map } from 'rxjs/operators';
import { Subscription } from 'rxjs';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';

@Component({
    selector: 'app-stage-step-summary',
    templateUrl: './stage.summary.component.html',
    styleUrls: ['./stage.summary.component.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class StageStepSummaryComponent implements OnInit {
    @Input() stageId: number;
    @Input() runNumber: number;
    @Input() workflowNodeRunId: number;
    @Input() nodeName: string;

    open = false;
    stageName: string;
    stageStatus: string;
    stageWarning = false;
    jobsIds: Array<number>;

    storeSubs: Subscription;

    constructor(
        private _router: Router,
        private _route: ActivatedRoute,
        private _store: Store,
        private _cd: ChangeDetectorRef
    ) { }

    ngOnInit() {
        this.storeSubs = this._store.select(WorkflowState.nodeRunStage).pipe(map(filterFn => filterFn(this.stageId))).subscribe( s => {
            console.log('>>>>>>>>>>', s);
            if (!s && !this.stageStatus) {
                return;
            }

            // Check if new stage has a warning
            let warn :boolean;
            if (s && Array.isArray(s.run_jobs)) {
                warn = s.run_jobs.reduce((fail, job) => {
                    if (!job.job || !Array.isArray(job.job.step_status)) {
                        return fail;
                    }
                    return fail || job.job.step_status.reduce((failStep, step) => failStep || step.status === PipelineStatus.FAIL, false);
                }, false);
            }
            if (s && s.id === this.stageId && this.stageStatus === s.status  && this.stageWarning === warn) {
                return;
            }

            console.log('REFESH SIDEBAR STAGE ' + this.stageId + ' ' + s.status);
            if (s) {
                this.stageName = s.name;
                this.stageStatus = s.status;
                if (s.run_jobs) {
                    this.jobsIds = new Array();
                    this.jobsIds.push(...s.run_jobs.map(j => j.id));
                }
                this.stageWarning = warn;
                this.open = this.stageStatus === PipelineStatus.FAIL || PipelineStatus.isActive(this.stageStatus);
            } else {
                delete this.stageStatus;
                delete this.jobsIds;
                this.stageWarning = false;
            }
            this._cd.detectChanges();
        });


    }

    goToStageLogs() {
        this._router.navigate([
            'project',
            this._route.snapshot.params['key'],
            'workflow',
            this._route.snapshot.params['workflowName'],
            'run',
            this.runNumber,
            'node',
            this.workflowNodeRunId
        ], { queryParams: { stageId: this.stageId, name: this.nodeName } });
    }
}
