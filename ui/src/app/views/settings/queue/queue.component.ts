import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { EventType } from 'app/model/event.model';
import { PipelineStatus } from 'app/model/pipeline.model';
import { AuthSummary } from 'app/model/user.model';
import { WorkflowNodeJobRun } from 'app/model/workflow.run.model';
import { QueueService } from 'app/service/queue/queue.service';
import { WorkflowRunService } from 'app/service/workflow/run/workflow.run.service';
import { PathItem } from 'app/shared/breadcrumb/breadcrumb.component';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { ToastService } from 'app/shared/toast/ToastService';
import { AuthenticationState } from 'app/store/authentication.state';
import { EventState } from 'app/store/event.state';
import { AddOrUpdateJob, RemoveJob, SetJobs, SetJobUpdating } from 'app/store/queue.action';
import { QueueState } from 'app/store/queue.state';
import * as moment from 'moment';
import { Subscription } from 'rxjs';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-queue',
    templateUrl: './queue.component.html',
    styleUrls: ['./queue.component.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class QueueComponent implements OnDestroy {
    queueSubscription: Subscription;
    currentAuthSummary: AuthSummary;
    nodeJobRuns: Array<WorkflowNodeJobRun> = [];
    parametersMaps: Array<{}> = [];
    requirementsList: Array<string> = [];
    bookedOrBuildingByList: Array<string> = [];
    loading = true;
    statusOptions: Array<string> = [PipelineStatus.WAITING, PipelineStatus.BUILDING];
    status: Array<string>;
    path: Array<PathItem>;

    constructor(
        private _store: Store,
        private _wfRunService: WorkflowRunService,
        private _queueService: QueueService,
        private _toast: ToastService,
        private _translate: TranslateService,
        private _cd: ChangeDetectorRef
    ) {
        this.currentAuthSummary = this._store.selectSnapshot(AuthenticationState.summary);
        this.status = [this.statusOptions[0]];

        this.path = [<PathItem>{
            translate: 'common_settings'
        }, <PathItem>{
            translate: 'admin_queue_title'
        }];

        this._store.select(EventState.last).subscribe(e => {
            if (!e || e.type_event !== EventType.RUN_WORKFLOW_JOB) {
                return
            }
            let jobID = e.payload['id'];
            if (e.status === PipelineStatus.WAITING || e.status === PipelineStatus.BUILDING) {
                this._queueService.getJobInfos(jobID).subscribe(wnr => {
                    this._store.dispatch(new AddOrUpdateJob(wnr));
                });
            } else {
                this._store.dispatch(new RemoveJob(jobID));
            }
        });

        this.queueSubscription = this._store.select(QueueState.jobs).subscribe(js => {
            let fitlers = this.status.length > 0 ? this.status : this.statusOptions;
            this.nodeJobRuns = js.filter(j => !!fitlers.find(f => f === j.status)).sort((a: WorkflowNodeJobRun, b: WorkflowNodeJobRun) => moment(a.queued).isBefore(moment(b.queued)) ? -1 : 1);
            if (this.nodeJobRuns.length > 0) {
                this.requirementsList = [];
                this.bookedOrBuildingByList = [];
                this.parametersMaps = this.nodeJobRuns.map((nj) => {
                    if (this.currentAuthSummary.user.ring === 'ADMIN' && nj.job && nj.job.action && nj.job.action.requirements) {
                        let requirements = nj.job.action.requirements
                            .reduce((reqs, req) => `type: ${req.type}, value: ${req.value}; ${reqs}`, '');
                        this.requirementsList.push(requirements);
                    }
                    this.bookedOrBuildingByList.push(((): string => {
                        if (nj.status === PipelineStatus.BUILDING) {
                            return nj.job.worker_name;
                        }
                        if (nj.bookedby !== null) {
                            return nj.bookedby.name;
                        }
                        return '';
                    })());
                    if (!nj.parameters) {
                        return null;
                    }
                    return nj.parameters.reduce((params, param) => {
                        params[param.name] = param.value;
                        return params;
                    }, {});
                });
            }
            this._cd.markForCheck();
        });

        this.loadAll();
    }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    statusFilterChange() {
        this.loadAll();
    }

    loadAll() {
        this.loading = true;
        let status = this.status.length > 0 ? this.status : this.statusOptions;
        this._queueService.getWorkflows(status).pipe(finalize(() => {
            this.loading = false;
            this._cd.markForCheck();
        })).subscribe(js => {
            this._store.dispatch(new SetJobs(js))
        })
    }

    stopNode(index: number) {
        let parameters = this.parametersMaps[index];
        this._store.dispatch(new SetJobUpdating(this.nodeJobRuns[index].id));
        this._wfRunService.stopNodeRun(
            parameters['cds.project'],
            parameters['cds.workflow'],
            parseInt(parameters['cds.run.number'], 10),
            parseInt(parameters['cds.node.id'], 10)
        )
            .pipe(finalize(() => {
                this._cd.markForCheck();
            }))
            .subscribe(() => this._toast.success('', this._translate.instant('pipeline_stop')))
    }
}
