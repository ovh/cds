import {Component, Input, Output, EventEmitter} from '@angular/core';
import {Project} from '../../../../model/project.model';
import {PermissionValue} from '../../../../model/permission.model';
import {WorkflowRun} from '../../../../model/workflow.run.model';
import {PipelineStatus} from '../../../../model/pipeline.model';
import {AutoUnsubscribe} from '../../../../shared/decorator/autoUnsubscribe';
import {WorkflowRunService} from '../../../../service/workflow/run/workflow.run.service';
import {ToastService} from '../../../../shared/toast/ToastService';
import {TranslateService} from '@ngx-translate/core';
import {finalize} from 'rxjs/operators';
import {WorkflowEventStore} from '../../../../service/workflow/workflow.event.store';
import {Subscription} from 'rxjs/Subscription';
import {Workflow} from '../../../../model/workflow.model';

declare var ansi_up: any;

@Component({
    selector: 'app-workflow-run-summary',
    templateUrl: './workflow.run.summary.html',
    styleUrls: ['./workflow.run.summary.scss']
})
@AutoUnsubscribe()
export class WorkflowRunSummaryComponent {
    @Input('direction')
    set direction(val) {
      this._direction = val;
      this.directionChange.emit(val);
    }
    get direction() {
        return this._direction;
    }
    @Input() project: Project;
    @Input() workflow: Workflow;
    workflowRun: WorkflowRun;
    subWR: Subscription;
    @Input() workflowName: string;
    @Output() directionChange = new EventEmitter();
    @Output() relaunch = new EventEmitter();

    _direction: string;
    author: string;
    loadingAction = false;
    showInfos = false;

    pipelineStatusEnum = PipelineStatus;
    permissionEnum = PermissionValue;

    constructor(private _workflowRunService: WorkflowRunService, private _workflowEventStore: WorkflowEventStore,
        private _toast: ToastService, private _translate: TranslateService) {
        this.subWR = this._workflowEventStore.selectedRun().subscribe(wr => {
            this.workflowRun = wr;
            if (this.workflowRun) {
                let tagTriggeredBy = this.workflowRun.tags.find((tag) => tag.tag === 'triggered_by');

                if (tagTriggeredBy) {
                    this.author = tagTriggeredBy.value;
                }
            }
        });
    }

    getSpawnInfos() {
        let msg = '';
        if (this.workflowRun.infos) {
            this.workflowRun.infos.forEach( s => {
                msg += '[' + s.api_time.toString().substr(0, 19) + '] ' + s.user_message + '\n';
            });
        }
        if (msg !== '') {
            return ansi_up.ansi_to_html(msg);
        }
        return '';
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

    resyncVCSStatus() {
      this.loadingAction = true;
      this._workflowRunService.resyncVCSStatus(this.project.key, this.workflowName, this.workflowRun.num)
          .pipe(finalize(() => this.loadingAction = false))
          .subscribe(() => this._toast.success('', this._translate.instant('workflow_vcs_resynced')));
    }
}
