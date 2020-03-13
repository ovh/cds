import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { Store } from '@ngxs/store';
import { PipelineStatus } from 'app/model/pipeline.model';
import { WorkflowState } from 'app/store/workflow.state';
import { map } from 'rxjs/operators';

@Component({
    selector: 'app-action-step-summary',
    templateUrl: './action.summary.component.html',
    styleUrls: ['./action.summary.component.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ActionStepSummaryComponent implements OnInit {

    // Step identifiers
    @Input() stageId: number;
    @Input() jobId: number;
    @Input() stepOrder: number;
    @Input() runNumber: number;

    // Static step information
    @Input() stepName: string;
    @Input() stepOptionnal: boolean;

    // Data for router
    @Input() pipelineActionId: number;
    @Input() workflowNodeRunId: number;
    @Input() nodeName: string;

    // Dynamic values
    stepStatus: string;
    open = false;

    constructor(private _router: Router, private _route: ActivatedRoute, private _store: Store, private _cd: ChangeDetectorRef) {}

    ngOnInit() {
        this._store.select(WorkflowState.nodeRunJobStep)
            .pipe(map(filterFn => filterFn(this.stageId, this.jobId, this.stepOrder))).subscribe( ss => {
                if (!ss && !this.stepStatus) {
                    return;
                }

                if (ss && this.stepStatus && ss.status === this.stepStatus) {
                    return;
                }
                if (ss) {
                    this.stepStatus = ss.status;
                    this.open = this.stepStatus === PipelineStatus.FAIL;
                } else {
                    delete this.stepStatus;
                    this.open = false;
                }

                this._cd.detectChanges();
        });
    }

    goToActionLogs() {
      this._router.navigate([
          'project',
          this._route.snapshot.params['key'],
          'workflow',
          this._route.snapshot.params['workflowName'],
          'run',
          this.runNumber,
          'node',
          this.workflowNodeRunId
      ], {queryParams: {
          stageId: this.stageId,
          actionId: this.pipelineActionId,
          stepOrder: this.stepOrder,
          name: this.nodeName,
      }});
    }
}
