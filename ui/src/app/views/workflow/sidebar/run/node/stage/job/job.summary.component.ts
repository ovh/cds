import {Component, Input, OnInit} from '@angular/core';
import {PipelineStatus} from '../../../../../../../model/pipeline.model';
import {WorkflowNodeJobRun} from '../../../../../../../model/workflow.run.model';

@Component({
    selector: 'app-job-step-summary',
    templateUrl: './job.summary.component.html',
    styleUrls: ['./job.summary.component.scss']
})
export class JobStepSummaryComponent implements OnInit {

    @Input() job: WorkflowNodeJobRun;

    open = false;
    constructor() {

    }

    ngOnInit() {
      this.open = this.job.status === PipelineStatus.FAIL || PipelineStatus.isActive(this.job.status);
    }
}
