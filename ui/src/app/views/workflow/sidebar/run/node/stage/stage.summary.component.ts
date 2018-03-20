import {Component, Input, OnInit} from '@angular/core';
import {Router, ActivatedRoute} from '@angular/router';
import {PipelineStatus} from '../../../../../../model/pipeline.model';
import {Stage} from '../../../../../../model/stage.model';
import {WorkflowRun, WorkflowNodeRun} from '../../../../../../model/workflow.run.model';

@Component({
    selector: 'app-stage-step-summary',
    templateUrl: './stage.summary.component.html',
    styleUrls: ['./stage.summary.component.scss']
})
export class StageStepSummaryComponent implements OnInit {

    @Input() stage: Stage;
    @Input() workflowRun: WorkflowRun;
    @Input() workflowNodeRun: WorkflowNodeRun;

    open = false;
    warning = false;
    constructor(private _router: Router, private _route: ActivatedRoute) {

    }

    ngOnInit() {
      this.open = this.stage.status === PipelineStatus.FAIL || PipelineStatus.isActive(this.stage.status);
      if (Array.isArray(this.stage.run_jobs)) {
        this.warning = this.stage.run_jobs.reduce((fail, job) => {
            if (!job.job || !Array.isArray(job.job.step_status)) {
                return fail;
            }
            return fail || job.job.step_status.reduce((failStep, step) => failStep || step.status === PipelineStatus.FAIL, false);
        }, false);
      }

    }

    goToStageLogs() {
      // /project/TEST/workflow/coucou/run/38/node/805?name=deploy&stageId=2
      this._router.navigate([
          'project',
          this._route.snapshot.params['key'],
          'workflow',
          this._route.snapshot.params['workflowName'],
          'run',
          this.workflowRun.num,
          'node',
          this.workflowNodeRun.id
      ], {queryParams: {stageId: this.stage.id}});
    }
}
