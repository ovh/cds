import {Component, Input, OnInit} from '@angular/core';
import {PipelineStatus} from '../../../../../../model/pipeline.model';
import {Stage} from '../../../../../../model/stage.model';

@Component({
    selector: 'app-stage-step-summary',
    templateUrl: './stage.summary.component.html',
    styleUrls: ['./stage.summary.component.scss']
})
export class StageStepSummaryComponent implements OnInit {

    @Input() stage: Stage;

    open = false;
    constructor() {

    }

    ngOnInit() {
      this.open = this.stage.status === PipelineStatus.FAIL || PipelineStatus.isActive(this.stage.status);
    }
}
