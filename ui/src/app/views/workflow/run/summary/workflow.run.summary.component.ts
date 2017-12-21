import {Component, Input, Output, EventEmitter, OnInit} from '@angular/core';
import {Project} from '../../../../model/project.model';
import {WorkflowRun} from '../../../../model/workflow.run.model';
import {PipelineStatus} from '../../../../model/pipeline.model';
import {AutoUnsubscribe} from '../../../../shared/decorator/autoUnsubscribe';
import {WorkflowRunService} from '../../../../service/workflow/run/workflow.run.service';
import {ToastService} from '../../../../shared/toast/ToastService';
import {TranslateService} from '@ngx-translate/core';
import {finalize} from 'rxjs/operators';

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
    @Output() relaunch = new EventEmitter();

    _direction: string;
    author: string;
    loadingAction = false;

    pipelineStatusEnum = PipelineStatus;

    constructor(private _workflowRunService: WorkflowRunService,
        private _toast: ToastService, private _translate: TranslateService) {

    }

    ngOnInit() {
        let tagTriggeredBy = this.workflowRun.tags.find((tag) => tag.tag === 'triggered_by');

        if (tagTriggeredBy) {
            this.author = tagTriggeredBy.value;
        }
    }

    changeDirection() {
      this.direction = this.direction === 'LR' ? 'TB' : 'LR';
    }

    stopWorkflow() {
        this.loadingAction = true;
        this._workflowRunService.stopWorkflowRun(this.project.key, this.workflowName, this.workflowRun.num)
            .pipe(finalize(() => this.loadingAction = false))
            .subscribe(() => this._toast.success('', this._translate.instant('workflow_stopped')));
    }
}
