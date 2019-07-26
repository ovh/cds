import { ChangeDetectionStrategy, ChangeDetectorRef, Component, NgZone, OnDestroy } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Subscription } from 'rxjs';
import { finalize } from 'rxjs/operators';
import { environment } from '../../../../environments/environment';
import { PipelineStatus } from '../../../model/pipeline.model';
import { User } from '../../../model/user.model';
import { WorkflowNodeJobRun } from '../../../model/workflow.run.model';
import { AuthentificationStore } from '../../../service/auth/authentification.store';
import { WorkflowRunService } from '../../../service/workflow/run/workflow.run.service';
import { PathItem } from '../../../shared/breadcrumb/breadcrumb.component';
import { AutoUnsubscribe } from '../../../shared/decorator/autoUnsubscribe';
import { ToastService } from '../../../shared/toast/ToastService';
import { CDSWebWorker } from '../../../shared/worker/web.worker';

@Component({
    selector: 'app-queue',
    templateUrl: './queue.component.html',
    styleUrls: ['./queue.component.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class QueueComponent implements OnDestroy {
    queueWorker: CDSWebWorker;
    zone: NgZone;
    queueSubscription: Subscription;
    user: User;
    nodeJobRuns: Array<WorkflowNodeJobRun> = [];
    parametersMaps: Array<{}> = [];
    requirementsList: Array<string> = [];
    bookedOrBuildingByList: Array<string> = [];
    loading = true;
    statusOptions: Array<string> = [PipelineStatus.WAITING, PipelineStatus.BUILDING];
    status: Array<string>;
    path: Array<PathItem>;

    constructor(
        private _authStore: AuthentificationStore,
        private _wfRunService: WorkflowRunService,
        private _toast: ToastService,
        private _translate: TranslateService,
        private _cd: ChangeDetectorRef
    ) {
        this.zone = new NgZone({ enableLongStackTrace: false });
        this.loading = true;
        this.status = [this.statusOptions[0]];
        this.user = this._authStore.getUser();
        this.startWorker();

        this.path = [<PathItem>{
            translate: 'common_settings'
        }, <PathItem>{
            translate: 'admin_queue_title'
        }];
    }

    ngOnDestroy(): void {
        if (this.queueWorker) {
            this.queueWorker.stop();
        }
    }

    statusFilterChange(): void {
        this.queueWorker.stop();
        this.queueWorker.start(this.getQueueWorkerParams());
    }

    getQueueWorkerParams(): any {
        return {
            'user': this._authStore.getUser(),
            'session': this._authStore.getSessionToken(),
            'api': environment.apiURL,
            'status': this.status.length > 0 ? this.status : this.statusOptions
        };
    }

    startWorker() {
        if (this.queueWorker) {
            this.queueWorker.stop();
        }
        if (this.queueSubscription) {
            this.queueSubscription.unsubscribe();
        }
        // Start web worker
        this.queueWorker = new CDSWebWorker('./assets/worker/web/queue.js');
        this.queueWorker.start(this.getQueueWorkerParams());

        this.queueSubscription = this.queueWorker.response().subscribe(wrString => {
            if (!wrString) {
                return;
            }
            this.loading = false;
            this.zone.run(() => {
                this.nodeJobRuns = <Array<WorkflowNodeJobRun>>JSON.parse(wrString);

                if (Array.isArray(this.nodeJobRuns) && this.nodeJobRuns.length > 0) {
                    this.requirementsList = [];
                    this.bookedOrBuildingByList = [];
                    this.parametersMaps = this.nodeJobRuns.map((nj) => {
                        if (this.user.admin && nj.job && nj.job.action && nj.job.action.requirements) {
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
        });
    }

    stopNode(index: number) {
        let parameters = this.parametersMaps[index];
        this.nodeJobRuns[index].updating = true;
        this._wfRunService.stopNodeRun(
            parameters['cds.project'],
            parameters['cds.workflow'],
            parseInt(parameters['cds.run.number'], 10),
            parseInt(parameters['cds.node.id'], 10)
        ).pipe(finalize(() => {
            this.nodeJobRuns[index].updating = false;
            this._cd.markForCheck();
        }))
            .subscribe(() => this._toast.success('', this._translate.instant('pipeline_stop')))
    }
}
