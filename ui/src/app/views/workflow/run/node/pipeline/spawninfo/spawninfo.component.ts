import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    EventEmitter,
    Input, NgZone,
    OnDestroy,
    OnInit,
    Output,
    ViewChild
} from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Select, Store } from '@ngxs/store';
import * as AU from 'ansi_up';
import { Parameter } from 'app/model/parameter.model';
import { PipelineStatus, SpawnInfo } from 'app/model/pipeline.model';
import { WorkflowNodeJobRun } from 'app/model/workflow.run.model';
import { WorkflowService } from 'app/service/workflow/workflow.service';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { ProjectState } from 'app/store/project.state';
import { WorkflowState, WorkflowStateModel } from 'app/store/workflow.state';
import { interval, Observable, Subscription } from 'rxjs';
import { mergeMap } from 'rxjs/operators';
import { WorkflowRunJobVariableComponent } from '../variables/job.variables.component';

@Component({
    selector: 'app-workflow-run-job-spawn-info',
    templateUrl: './spawninfo.html',
    styleUrls: ['./spawninfo.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowRunJobSpawnInfoComponent implements OnInit, OnDestroy {

    @Input()
    set displayServicesLogs(data: boolean) {
        this._displayServiceLogs = data;
        this.displayServicesLogsChange.emit(data);
    }
    get displayServicesLogs(): boolean {
        return this._displayServiceLogs;
    }

    @Select(WorkflowState.getSelectedWorkflowNodeJobRun()) nodeJobRun$: Observable<WorkflowNodeJobRun>;
    nodeJobRunSubs: Subscription;

    currentJobID: number;
    jobStatus: string;
    spawnInfos: string;
    variables: Array<Parameter>;
    @Output() displayServicesLogsChange = new EventEmitter<boolean>();

    @ViewChild('jobVariable')
    jobVariable: WorkflowRunJobVariableComponent;

    pollingSubscription: Subscription;
    zone: NgZone;

    loading = true;

    show = true;
    displayServiceLogsLink = false;
    _displayServiceLogs: boolean;
    ansi_up = new AU.default();

    constructor(
        private _translate: TranslateService,
        private _cd: ChangeDetectorRef,
        private _store: Store,
        private _workflowService: WorkflowService,
        private _ngZone: NgZone
    ) {
        this.zone = new NgZone({ enableLongStackTrace: false });
    }

    ngOnDestroy(): void {} // Should be set to use @AutoUnsubscribe with AOT

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
                if ((!this.variables || this.variables.length) && njr.parameters) {
                    this.variables = njr.parameters;
                    refresh = true;
                }
            } else {
                refresh = true;
                this.jobStatus = njr.status;
                this.currentJobID = njr.id;
                this.variables = njr.parameters;
                if (!PipelineStatus.isDone(njr.status)) {
                    this.initWorker();
                } else {
                    this.spawnInfos = this.getSpawnInfos(njr.spawninfos);
                }
                if (njr.job.action.requirements) {
                    this.displayServiceLogsLink = njr.job.action.requirements.find(r => r.type === 'service') != null;
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
        let projectKey = this._store.selectSnapshot(ProjectState.projectSnapshot).key;
        let workflowName = this._store.selectSnapshot(WorkflowState.workflowSnapshot).name;
        let runNumber = (<WorkflowStateModel>this._store.selectSnapshot(WorkflowState)).workflowNodeRun.num;
        let nodeRunID = (<WorkflowStateModel>this._store.selectSnapshot(WorkflowState)).workflowNodeRun.id;
        let runJobId = this.currentJobID;

        let callback = (is: Array<SpawnInfo>) => {
            if (is.length > 0) {
                this.spawnInfos = this.getSpawnInfos(is);
                this._cd.markForCheck();
            }
        }

        this._workflowService.getNodeJobRunInfo(projectKey, workflowName, runNumber, nodeRunID, runJobId).subscribe(spawnInfos => {
            callback(spawnInfos);
        });

        this.stopWorker();
        this._ngZone.runOutsideAngular(() => {
            this.pollingSubscription = interval(4000)
                .pipe(
                    mergeMap(_ =>
                        this._workflowService.getNodeJobRunInfo(projectKey, workflowName, runNumber, nodeRunID, runJobId))
                )
                .subscribe(spawnInfos => {
                    this._ngZone.run(() => {
                        callback(spawnInfos);
                    });
                });
        });
    }

    stopWorker() {
        if (this.pollingSubscription) {
            this.pollingSubscription.unsubscribe();
        }
    }

    openVariableModal(event: Event): void {
        event.stopPropagation();
        if (this.jobVariable) {
            this.jobVariable.show();
        }
    }
}
