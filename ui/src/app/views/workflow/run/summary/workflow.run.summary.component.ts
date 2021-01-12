import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    EventEmitter,
    Input,
    OnDestroy,
    OnInit,
    Output
} from '@angular/core';
import { Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Select, Store } from '@ngxs/store';
import * as AU from 'ansi_up';
import { PipelineStatus } from 'app/model/pipeline.model';
import { Project } from 'app/model/project.model';
import { Workflow } from 'app/model/workflow.model';
import { WorkflowRun } from 'app/model/workflow.run.model';
import { WorkflowRunService } from 'app/service/workflow/run/workflow.run.service';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { ToastService } from 'app/shared/toast/ToastService';
import { DeleteWorkflowRun } from 'app/store/workflow.action';
import { WorkflowState } from 'app/store/workflow.state';
import { Observable } from 'rxjs';
import { finalize } from 'rxjs/operators';
import { Subscription } from 'rxjs/Subscription';

@Component({
    selector: 'app-workflow-run-summary',
    templateUrl: './workflow.run.summary.html',
    styleUrls: ['./workflow.run.summary.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowRunSummaryComponent implements OnInit, OnDestroy {
    @Input()
    set direction(val) {
        this._direction = val;
        this.directionChange.emit(val);
    }

    get direction() {
        return this._direction;
    }

    @Input() project: Project;
    @Output() directionChange = new EventEmitter();


    workflowName: string;

    @Select(WorkflowState.getSelectedWorkflowRun()) workflowRun$: Observable<WorkflowRun>;
    workflowRun: WorkflowRun;
    subWorkflowRun: Subscription;

    @Select(WorkflowState.getWorkflow()) workflow$: Observable<Workflow>;
    subWorkflow: Subscription;
    canExecute: boolean;

    _direction: string;
    author: string;
    loadingAction = false;
    loadingDelete = false;
    showInfos = false;
    ansi_up = new AU.default();

    pipelineStatusEnum = PipelineStatus;

    constructor(
        private _workflowRunService: WorkflowRunService,
        private _toast: ToastService,
        private _translate: TranslateService,
        private _store: Store,
        private router: Router,
        private _cd: ChangeDetectorRef
    ) {}

    ngOnDestroy(): void {} // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit(): void {
        this.subWorkflowRun = this.workflowRun$.subscribe(wr => {
            if (!wr) {
                return;
            }
            // If same run and status doesn't change, lets check spawninfos && tags
            if (this.workflowRun && this.workflowRun.id === wr.id && this.workflowRun.status === wr.status) {
                let refreshView = false;
                if (!this.workflowRun.tags && wr.tags) {
                    refreshView = true;
                }
                if (this.workflowRun.tags && wr.tags && this.workflowRun.tags.length !== wr.tags.length) {
                    refreshView = true;
                }
                if (!this.workflowRun.infos && wr.infos) {
                    refreshView = true;
                }
                if (this.workflowRun.infos && wr.infos && this.workflowRun.infos.length !== wr.infos.length) {
                    refreshView = true;
                }
                if (!refreshView) {
                    return;
                }
            }
            this.workflowRun = wr;
            if (this.workflowRun.tags) {
                let tagTriggeredBy = this.workflowRun.tags.find((tag) => tag.tag === 'triggered_by');
                if (tagTriggeredBy) {
                    this.author = tagTriggeredBy.value;
                }
            }
            this._cd.markForCheck();
        });

        this.subWorkflow = this.workflow$.subscribe(w => {
            if (!w) {
                return;
            }
            if (w.permissions.executable === this.canExecute && this.workflowName) {
                return;
            }
            this.workflowName = w.name;
            this.canExecute = w.permissions.executable;
            this._cd.detectChanges();
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
            .pipe(finalize(() => {
                this.loadingAction = false;
                this._cd.markForCheck();
            }))
            .subscribe(() => this._toast.success('', this._translate.instant('workflow_stopped')));
    }

    resyncVCSStatus() {
        this.loadingAction = true;
        this._workflowRunService.resyncVCSStatus(this.project.key, this.workflowName, this.workflowRun.num)
            .pipe(finalize(() => {
                this.loadingAction = false;
                this._cd.markForCheck();
            }))
            .subscribe(() => this._toast.success('', this._translate.instant('workflow_vcs_resynced')));
    }

    delete() {
        this.loadingDelete = true;
        this._store.dispatch(new DeleteWorkflowRun({
            projectKey: this.project.key,
            workflowName: this.workflowName,
            num: this.workflowRun.num
        })).pipe(finalize(() => {
            this.loadingDelete = false;
            this._cd.markForCheck();
        }))
            .subscribe(() => {
                this._toast.success('', this._translate.instant('common_deleted'));
                this.router.navigate(['/project', this.project.key, 'workflow', this.workflowName]);
            });
    }
}
