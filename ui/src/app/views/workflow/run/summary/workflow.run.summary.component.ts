import { Component, EventEmitter, Input, Output } from '@angular/core';
import { Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import * as AU from 'ansi_up';
import { PermissionValue } from 'app/model/permission.model';
import { PipelineStatus } from 'app/model/pipeline.model';
import { Project } from 'app/model/project.model';
import { Workflow } from 'app/model/workflow.model';
import { WorkflowRun } from 'app/model/workflow.run.model';
import { WorkflowRunService } from 'app/service/workflow/run/workflow.run.service';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { ToastService } from 'app/shared/toast/ToastService';
import { DeleteWorkflowRun } from 'app/store/workflow.action';
import { WorkflowState, WorkflowStateModel } from 'app/store/workflow.state';
import { finalize } from 'rxjs/operators';
import { Subscription } from 'rxjs/Subscription';

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
    workflow: Workflow;
    workflowRun: WorkflowRun;
    subWR: Subscription;
    @Input() workflowName: string;
    @Output() directionChange = new EventEmitter();
    @Output() relaunch = new EventEmitter();

    _direction: string;
    author: string;
    loadingAction = false;
    loadingDelete = false;
    showInfos = false;
    ansi_up = new AU.default;

    pipelineStatusEnum = PipelineStatus;
    permissionEnum = PermissionValue;

    constructor(
        private _workflowRunService: WorkflowRunService,
        private _toast: ToastService,
        private _translate: TranslateService,
        private _store: Store,
        private router: Router
    ) {
        this.subWR = this._store.select(WorkflowState.getCurrent()).subscribe((state: WorkflowStateModel) => {
            this.workflow = state.workflow;
            this.workflowRun = state.workflowRun;
            if (this.workflowRun) {
                if (this.workflowRun.tags) {
                    let tagTriggeredBy = this.workflowRun.tags.find((tag) => tag.tag === 'triggered_by');
                    if (tagTriggeredBy) {
                        this.author = tagTriggeredBy.value;
                    }
                }
            }
        });
    }

    getSpawnInfos() {
        let msg = '';
        if (this.workflowRun.infos) {
            this.workflowRun.infos.forEach(s => {
                msg += '[' + s.api_time.toString().substr(0, 19) + '] ' + s.user_message + '\n';
            });
        }
        if (msg !== '') {
            return this.ansi_up.ansi_to_html(msg);
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

    delete() {
        this.loadingDelete = true;
        this._store.dispatch(new DeleteWorkflowRun({
            projectKey: this.project.key,
            workflowName: this.workflowName,
            num: this.workflowRun.num
        })).pipe(finalize(() => this.loadingDelete = false))
            .subscribe(() => {
                this._toast.success('', this._translate.instant('common_deleted'));
                this.router.navigate(['/project', this.project.key, 'workflow', this.workflowName]);
            });
    }
}
