import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    EventEmitter,
    Input, NgZone,
    OnDestroy, OnInit,
    Output,
    ViewChild
} from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Select, Store } from '@ngxs/store';
import * as AU from 'ansi_up';
import { Parameter } from 'app/model/parameter.model';
import { PipelineStatus, SpawnInfo } from 'app/model/pipeline.model';
import { WorkflowNodeJobRun } from 'app/model/workflow.run.model';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { CDSWebWorker } from 'app/shared/worker/web.worker';
import { ProjectState } from 'app/store/project.state';
import { WorkflowState, WorkflowStateModel } from 'app/store/workflow.state';
import { Observable, Subscription } from 'rxjs';
import { WorkflowRunJobVariableComponent } from '../variables/job.variables.component';

@Component({
    selector: 'app-workflow-run-job-spawn-info',
    templateUrl: './spawninfo.html',
    styleUrls: ['./spawninfo.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowRunJobSpawnInfoComponent implements OnDestroy, OnInit {

    @Select(WorkflowState.getSelectedWorkflowNodeJobRun()) nodeJobRun$: Observable<WorkflowNodeJobRun>;
    nodeJobRunSubs: Subscription;

    currentNodeJobRunID: number;
    currentJobID: number;
    jobStatus: string;
    spawnInfos: String;
    variables: Array<Parameter>;

    @Input('displayServicesLogs')
    set displayServicesLogs(data: boolean) {
        this._displayServiceLogs = data;
        this.displayServicesLogsChange.emit(data);
    }
    get displayServicesLogs(): boolean {
        return this._displayServiceLogs;
    }
    @Output() displayServicesLogsChange = new EventEmitter<boolean>();

    @ViewChild('jobVariable', { static: false })
    jobVariable: WorkflowRunJobVariableComponent;

    worker: CDSWebWorker;
    workerSubscription: Subscription;
    zone: NgZone;

    serviceSpawnInfos: Array<SpawnInfo>;
    loading = true;

    show = true;
    displayServiceLogsLink = false;
    _displayServiceLogs: boolean;
    ansi_up = new AU.default;

    ngOnDestroy(): void {
        this.stopWorker();
    }

    constructor(
        private _translate: TranslateService,
        private _cd: ChangeDetectorRef,
        private _store: Store
    ) {
        this.zone = new NgZone({ enableLongStackTrace: false });
    }

    ngOnInit(): void {
        this.nodeJobRunSubs = this.nodeJobRun$.subscribe(njr => {
            if (!njr) {
                return;
            }
            let refresh = false;

            // Just update data if we are on the job
            if (njr.id === this.currentJobID) {
                if (this.jobStatus !== njr.status) {
                    this.jobStatus = njr.status;
                    if (PipelineStatus.isDone(njr.status)) {
                        this.stopWorker();
                    }
                    refresh = true;
                }
            } else {
                refresh = true;
                this.jobStatus = njr.status;
                this.currentJobID = njr.id;
                this.variables = njr.parameters;
                if (!njr.spawninfos) {
                    this.initWorker();
                } else {
                    this.spawnInfos = this.getSpawnInfos(njr.spawninfos);
                }
            }

            if (refresh) {
                this._cd.markForCheck();
            }
        });
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
        if (!this.worker) {
            this.worker = new CDSWebWorker('./assets/worker/web/workflow-spawninfos.js');
            this.worker.start({
                key: this._store.selectSnapshot(ProjectState.projectSnapshot).key,
                workflowName: this._store.selectSnapshot(WorkflowState.workflowSnapshot).name,
                number: (<WorkflowStateModel>this._store.selectSnapshot(WorkflowState)).workflowNodeRun.num,
                nodeRunId: (<WorkflowStateModel>this._store.selectSnapshot(WorkflowState)).workflowNodeRun.id,
                runJobId: this.currentNodeJobRunID,
            });

            this.workerSubscription = this.worker.response().subscribe(msg => {
                if (msg) {
                    let serviceSpawnInfos: Array<SpawnInfo> = JSON.parse(msg);
                    this.zone.run(() => {
                        if (serviceSpawnInfos && serviceSpawnInfos.length > 0) {
                            this.spawnInfos = this.getSpawnInfos(serviceSpawnInfos);
                            this._cd.detectChanges();
                        }

                    });
                }
            });
        }
    }

    stopWorker() {
        if (this.workerSubscription) {
            this.workerSubscription.unsubscribe();
        }
        if (this.worker) {
            this.worker.stop();
            this.worker = null;
        }
    }

    openVariableModal(event: Event): void {
        event.stopPropagation();
        if (this.jobVariable) {
            this.jobVariable.show();
        }
    }
}
