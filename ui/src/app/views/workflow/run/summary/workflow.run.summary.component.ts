import {Component, Input, Output, EventEmitter, OnInit} from '@angular/core';
import {Project} from '../../../../model/project.model';
import {WorkflowRun} from '../../../../model/workflow.run.model';
import {PipelineStatus} from '../../../../model/pipeline.model';
import {Subscription} from 'rxjs/Subscription';
import {AutoUnsubscribe} from '../../../../shared/decorator/autoUnsubscribe';
import {WorkflowStore} from '../../../../service/workflow/workflow.store';
import {WorkflowRunService} from '../../../../service/workflow/run/workflow.run.service';
import {ToastService} from '../../../../shared/toast/ToastService';
import {TranslateService} from 'ng2-translate';

@Component({
    selector: 'app-workflow-run-summary',
    templateUrl: './workflow.run.summary.html',
    styleUrls: ['./workflow.run.summary.scss']
})
@AutoUnsubscribe()
export class WorkflowRunSummaryComponent implements OnInit {
    @Input('direction')
    set direction(val) {
      this._direction = val;
      this.directionChange.emit(val);
    }
    get direction() {
        return this._direction;
    }
    @Input() project: Project;
    @Input() workflowRun: WorkflowRun;
    @Input() workflowName: string;
    @Output() directionChange = new EventEmitter();

    stopSubsription: Subscription;
    version: string;
    _direction: string;

    pipelineStatusEnum = PipelineStatus;

    constructor(private _workflowStore: WorkflowStore, private _workflowRunService: WorkflowRunService,
        private _toast: ToastService, private _translate: TranslateService) {

    }

    ngOnInit() {
        this.getVersion();
    }

    getVersion() {
      let maxNum = 0;
      let maxSubV = 0;

      Object.keys(this.workflowRun.nodes).forEach((keyWr) => {
        this.workflowRun.nodes[keyWr].forEach((wrnv) => {
          if (maxNum < wrnv.num) {
            maxNum = wrnv.num
          }
          if (maxSubV < wrnv.subnumber) {
            maxSubV = wrnv.subnumber
          }
        });
      });

      this.version = maxNum + '.' + maxSubV;
    }

    changeDirection() {
      this.direction = this.direction === 'LR' ? 'TB' : 'LR';
    }

    relaunchWorkflow() {
      if (this.workflowRun && this.workflowRun.nodes && Object.keys(this.workflowRun.nodes).length) {

      }
    }

    stopWorkflow() {
        this._workflowRunService.stopWorkflowRun(this.project.key, this.workflowName, this.workflowRun.num)
            .subscribe(() => this._toast.success('', this._translate.instant('workflow_stopped')));
    }
}
