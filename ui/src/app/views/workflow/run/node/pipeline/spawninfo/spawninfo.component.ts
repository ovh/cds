import { Component, EventEmitter, Input, OnDestroy, OnInit, Output, ViewChild } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import * as AU from 'ansi_up';
import { CDSWebWorker } from 'app/shared/worker/web.worker';
import { Subscription } from 'rxjs';
import { environment } from '../../../../../../../environments/environment';
import { Job } from '../../../../../../model/job.model';
import { Parameter } from '../../../../../../model/parameter.model';
import { SpawnInfo } from '../../../../../../model/pipeline.model';
import { PipelineStatus } from '../../../../../../model/pipeline.model';
import { Project } from '../../../../../../model/project.model';
import { WorkflowNodeJobRun, WorkflowNodeRun } from '../../../../../../model/workflow.run.model';
import { AuthentificationStore } from '../../../../../../service/auth/authentification.store';
import { JobVariableComponent } from '../../../../../run/workflow/variables/job.variables.component';

@Component({
    selector: 'app-workflow-run-job-spawn-info',
    templateUrl: './spawninfo.html',
    styleUrls: ['./spawninfo.scss']
})
export class WorkflowRunJobSpawnInfoComponent implements OnInit, OnDestroy {

    @Input() project: Project;
    @Input() workflowName: string;
    @Input() nodeRun: WorkflowNodeRun;
    @Input('nodeJobRun')
    set nodeJobRun(data: WorkflowNodeJobRun) {
        if (data) {
            this._nodeJobRun = data;
            if (data.status === PipelineStatus.SUCCESS || data.status === PipelineStatus.FAIL || data.status === PipelineStatus.STOPPED) {
                this.stopWorker();
            }
        }
    }
    get nodeJobRun(): WorkflowNodeJobRun {
        return this._nodeJobRun;
    }

    spawnInfos: String;
    @Input() variables: Array<Parameter>;
    @Input('job')
    set job(data: Job) {
        this._job = data;
        this.refreshDisplayServiceLogsLink();
    }
    get job(): Job {
        return this._job
    }
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
    jobVariable: JobVariableComponent;

    _nodeJobRun: WorkflowNodeJobRun;

    worker: CDSWebWorker;
    workerSubscription: Subscription;

    serviceSpawnInfos: Array<SpawnInfo>;
    loading = true;

    show = true;
    displayServiceLogsLink = false;
    _job: Job;
    _displayServiceLogs: boolean;
    ansi_up = new AU.default;

    ngOnDestroy(): void {
        this.stopWorker();
    }

    ngOnInit(): void {
        this.initWorker();
    }

    constructor(private _authStore: AuthentificationStore, private _translate: TranslateService) { }

    refreshDisplayServiceLogsLink() {
        if (this.job && this.job.action && Array.isArray(this.job.action.requirements)) {
            this.displayServiceLogsLink = this.job.action.requirements.some((req) => req.type === 'service');
        }
    }

    toggle() {
        this.show = !this.show;
    }

    getSpawnInfos() {
        let msg = '';
        if (this.nodeJobRun.spawninfos) {
            this.nodeJobRun.spawninfos.forEach(s => {
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

        if (this.nodeJobRun.status !== PipelineStatus.WAITING && this.nodeJobRun.status !== PipelineStatus.BUILDING) {
            this.spawnInfos = this.getSpawnInfos();
            this.loading = false;
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
                    if (this.loading) {
                        this.loading = false;
                    }
                    let infos = '';
                    serviceSpawnInfos.forEach(s => {
                        infos += '[' + s.api_time.toString().substr(0, 19) + '] ' + s.user_message + '\n';
                    });
                    this.spawnInfos = this.ansi_up.ansi_to_html(infos);
                    if (this.nodeJobRun.status === PipelineStatus.SUCCESS || this.nodeJobRun.status === PipelineStatus.FAIL ||
                        this.nodeJobRun.status === PipelineStatus.STOPPED) {
                        this.stopWorker();
                        this.spawnInfos = this.getSpawnInfos();
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
