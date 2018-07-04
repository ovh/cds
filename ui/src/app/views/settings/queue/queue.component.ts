import {Component, NgZone, OnDestroy} from '@angular/core';
import {TranslateService} from '@ngx-translate/core';
import {Subscription} from 'rxjs';
import {finalize} from 'rxjs/operators';
import {environment} from '../../../../environments/environment';
import {User} from '../../../model/user.model';
import {WorkflowNodeJobRun} from '../../../model/workflow.run.model';
import {AuthentificationStore} from '../../../service/auth/authentification.store';
import {WorkflowRunService} from '../../../service/workflow/run/workflow.run.service';
import {AutoUnsubscribe} from '../../../shared/decorator/autoUnsubscribe';
import {ToastService} from '../../../shared/toast/ToastService';
import {CDSWebWorker} from '../../../shared/worker/web.worker';

@Component({
    selector: 'app-queue',
    templateUrl: './queue.component.html',
    styleUrls: ['./queue.component.scss']
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
    loading = true;

    constructor(
        private _authStore: AuthentificationStore,
        private _wfRunService: WorkflowRunService,
        private _toast: ToastService,
        private _translate: TranslateService
    ) {
        this.zone = new NgZone({enableLongStackTrace: false});
        this.loading = true;
        this.startWorker();
        this.user = this._authStore.getUser();
    }

    ngOnDestroy(): void {
        if (this.queueWorker) {
            this.queueWorker.stop();
        }
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
        this.queueWorker.start({
            'user': this._authStore.getUser(),
            'session': this._authStore.getSessionToken(),
            'api': environment.apiURL,
        });

        this.queueSubscription = this.queueWorker.response().subscribe(wrString => {
            if (!wrString) {
                return;
            }
            this.loading = false;
            this.zone.run(() => {
                this.nodeJobRuns = <Array<WorkflowNodeJobRun>>JSON.parse(wrString);

                if (Array.isArray(this.nodeJobRuns) && this.nodeJobRuns.length > 0) {
                    this.requirementsList = [];
                    this.parametersMaps = this.nodeJobRuns.map((nj) => {
                        if (this.user.admin && nj.job && nj.job.action && nj.job.action.requirements) {
                            let requirements = nj.job.action.requirements
                                .reduce((reqs, req) => `type: ${req.type}, value: ${req.value}; ${reqs}`, '');
                            this.requirementsList.push(requirements);
                        }
                        if (!nj.parameters) {
                            return null;
                        }
                        return nj.parameters.reduce((params, param) => {
                            params[param.name] = param.value;
                            return params;
                        }, {});
                    });
                }
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
        ).pipe(finalize(() => this.nodeJobRuns[index].updating = false))
        .subscribe(() => this._toast.success('', this._translate.instant('pipeline_stop')))
    }
}
