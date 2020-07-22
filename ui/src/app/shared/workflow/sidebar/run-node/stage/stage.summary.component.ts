import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnDestroy, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { Store } from '@ngxs/store';
import { PipelineStatus } from 'app/model/pipeline.model';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { WorkflowState } from 'app/store/workflow.state';
import { Subscription } from 'rxjs';
import { map } from 'rxjs/operators';

@Component({
    selector: 'app-stage-step-summary',
    templateUrl: './stage.summary.component.html',
    styleUrls: ['./stage.summary.component.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class StageStepSummaryComponent implements OnInit, OnDestroy {
    @Input() stageId: number;
    @Input() runNumber: number;
    @Input() workflowNodeRunId: number;
    @Input() nodeName: string;

    open = false;
    stageName: string;
    stageStatus = '';
    stageWarning = false;
    jobsIds: Array<number>;

    storeSubs: Subscription;

    constructor(
        private _router: Router,
        private _route: ActivatedRoute,
        private _store: Store,
        private _cd: ChangeDetectorRef
    ) { }

    ngOnDestroy(): void {} // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit() {
        this.storeSubs = this._store.select(WorkflowState.nodeRunStage).pipe(map(filterFn => filterFn(this.stageId))).subscribe( s => {
            if (!s) {
                return;
            }

            // Check if new stage has a warning
            let warn = false;
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
                this.stageStatus = '';
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
