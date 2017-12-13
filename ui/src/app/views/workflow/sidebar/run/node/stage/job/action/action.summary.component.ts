import {Component, Input, OnInit} from '@angular/core';
import {PipelineStatus} from '../../../../../../../../model/pipeline.model';
import {StepStatus} from '../../../../../../../../model/job.model';
import {Action} from '../../../../../../../../model/action.model';

@Component({
    selector: 'app-action-step-summary',
    templateUrl: './action.summary.component.html',
    styleUrls: ['./action.summary.component.scss']
})
export class ActionStepSummaryComponent implements OnInit {

    @Input() action: Action;
    @Input() actionStatus: StepStatus;

    open = false;
    constructor() {

    }

    ngOnInit() {
      this.open = this.actionStatus.status === PipelineStatus.FAIL;
    }
}
