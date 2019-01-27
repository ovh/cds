import { Component, EventEmitter, Input, OnDestroy, Output, ViewChild } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import * as AU from 'ansi_up';
import { CDSWebWorker } from 'app/shared/worker/web.worker';
import { Subscription } from 'rxjs';
import { environment } from '../../../../../../../environments/environment';
import { Parameter } from '../../../../../../model/parameter.model';
import { PipelineStatus, SpawnInfo } from '../../../../../../model/pipeline.model';
import { Project } from '../../../../../../model/project.model';
import { WorkflowNodeJobRun, WorkflowNodeRun } from '../../../../../../model/workflow.run.model';
import { AuthentificationStore } from '../../../../../../service/auth/authentification.store';
import { WorkflowRunJobVariableComponent } from '../variables/job.variables.component';

@Component({
    selector: 'app-workflow-run-job-spawn-info',
    templateUrl: './spawninfo.html',
    styleUrls: ['./spawninfo.scss']
})
export class WorkflowRunJobSpawnInfoComponent implements OnDestroy {

    @Input() project: Project;
    @Input() workflowName: string;
    @Input() jobStatus: string;
    @Input() nodeRun: WorkflowNodeRun;
    @Input('nodeJobRun')
    set nodeJobRun(data: WorkflowNodeJobRun) {
        this.stopWorker();
        if (data) {
            this._nodeJobRun = data;
            this.refreshDisplayServiceLogsLink();
            if (PipelineStatus.isDone(data.status)) {
                this.stopWorker();
            }
            this.initWorker();
        }
    }
    get nodeJobRun(): WorkflowNodeJobRun {
        return this._nodeJobRun;
    }

    spawnInfos: String;
    @Input() variables: Array<Parameter>;
    @Input('displayServiceLogs')
    set displayServiceLogs(data: boolean) {
        this._displayServiceLogs = data;
        this.displayServicesLogsChange.emit(data);
    }
    get displayServiceLogs(): boolean {
        return this._displayServiceLogs;
    }

    @Output() displayServicesLogsChange = new EventEmitter<boolean>();

    @ViewChild('jobVariable')
    jobVariable: WorkflowRunJobVariableComponent;

    _nodeJobRun: WorkflowNodeJobRun;

    worker: CDSWebWorker;
    workerSubscription: Subscription;

    serviceSpawnInfos: Array<SpawnInfo>;
    loading = true;

    show = true;
    displayServiceLogsLink = false;
    _displayServiceLogs: boolean;
    ansi_up = new AU.default;

    ngOnDestroy(): void {
        this.stopWorker();
    }

    constructor(private _authStore: AuthentificationStore, private _translate: TranslateService) { }

    refreshDisplayServiceLogsLink() {
        if (this.nodeJobRun.job && this.nodeJobRun.job.action && Array.isArray(this.nodeJobRun.job.action.requirements)) {
            this.displayServiceLogsLink = this.nodeJobRun.job.action.requirements.some((req) => req.type === 'service');
        }
    }

    toggle() {
        this.show = !this.show;
    }

    getSpawnInfos(spawnInfosIn: Array<SpawnInfo>) {
        this.loading = false;
        let msg = '';
        if (spawnInfosIn) {
            spawnInfosIn.forEach(s => {
                msg += '[' + s.api_time.toString().substr(0, 19) + '] ' + s.user_message + '\n';
            });
        }
        if (msg !== '') {
            return this.ansi_up.ansi_to_html(msg);
        }
        return this._translate.instant('job_spawn_no_information');
    }

    initWorker(): void {
        if (!this.serviceSpawnInfos) {
            this.loading = true;
        }

        if (this.jobStatus !== PipelineStatus.WAITING && this.jobStatus !== PipelineStatus.BUILDING) {
            if (this.nodeJobRun.spawninfos && this.nodeJobRun.spawninfos.length > 0) {
                this.spawnInfos = this.getSpawnInfos(this.nodeJobRun.spawninfos);
            }
            return;
        }

        if (!this.worker) {
            this.worker = new CDSWebWorker('./assets/worker/web/workflow-spawninfos.js');
            this.worker.start({
                user: this._authStore.getUser(),
                session: this._authStore.getSessionToken(),
                api: environment.apiURL,
                key: this.project.key,
                workflowName: this.workflowName,
                number: this.nodeRun.num,
                nodeRunId: this.nodeRun.id,
                runJobId: this.nodeJobRun.id,
            });

            this.workerSubscription = this.worker.response().subscribe(msg => {
                if (msg) {
                    let serviceSpawnInfos: Array<SpawnInfo> = JSON.parse(msg);
                    if (serviceSpawnInfos && serviceSpawnInfos.length > 0) {
                        this.spawnInfos = this.getSpawnInfos(serviceSpawnInfos);
                    }
                    if (this.jobStatus === PipelineStatus.SUCCESS || this.jobStatus === PipelineStatus.FAIL ||
                        this.jobStatus === PipelineStatus.STOPPED) {
                        this.stopWorker();
                        if (this.nodeJobRun.spawninfos && this.nodeJobRun.spawninfos.length > 0) {
                            this.spawnInfos = this.getSpawnInfos(this.nodeJobRun.spawninfos);
                        }
                    }
                }
            });
        }
    }

    stopWorker() {
        if (this.worker) {
            this.worker.stop();
            this.worker = null;
        }
    }

    openVariableModal(event: Event): void {
        event.stopPropagation();
        if (this.jobVariable) {
            this.jobVariable.show({ autofocus: false, closable: false, observeChanges: true });
        }
    }
}
