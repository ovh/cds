import {Component, Input, OnInit} from '@angular/core';
import {ActivatedRoute, Router} from '@angular/router';
import {Action} from '../../../../../../../model/action.model';
import {StepStatus} from '../../../../../../../model/job.model';
import {PipelineStatus} from '../../../../../../../model/pipeline.model';
import {Stage} from '../../../../../../../model/stage.model';
import {WNode} from '../../../../../../../model/workflow.model';
import {WorkflowNodeJobRun, WorkflowRun} from '../../../../../../../model/workflow.run.model';

@Component({
    selector: 'app-action-step-summary',
    templateUrl: './action.summary.component.html',
    styleUrls: ['./action.summary.component.scss']
})
export class ActionStepSummaryComponent implements OnInit {

    @Input() action: Action;
    @Input() actionStatus: StepStatus;
    @Input() workflowRun: WorkflowRun;
    @Input() workflowNode: WNode;
    @Input() stage: Stage;
    @Input() job: WorkflowNodeJobRun;

    open = false;

    constructor(private _router: Router, private _route: ActivatedRoute) {

    }

    ngOnInit() {
      this.open = this.actionStatus.status === PipelineStatus.FAIL;
    }

    goToActionLogs() {
      this._router.navigate([
          'project',
          this._route.snapshot.params['key'],
          'workflow',
          this._route.snapshot.params['workflowName'],
          'run',
          this.workflowRun.num,
          'node',
          this.job.workflow_node_run_id
      ], {queryParams: {
          stageId: this.stage.id,
          actionId: this.job.job.pipeline_action_id,
          stepOrder: this.actionStatus.step_order,
          name: this.workflowNode.name,
      }});
    }
}
