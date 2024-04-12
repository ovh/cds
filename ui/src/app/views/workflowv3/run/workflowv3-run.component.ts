import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, OnInit, ViewChild } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { Store } from '@ngxs/store';
import { EventType } from 'app/model/event.model';
import { CDNLine, CDNStreamFilter } from 'app/model/pipeline.model';
import { UIArtifact } from 'app/model/workflow.run.model';
import { WorkflowRunService } from 'app/service/services.module';
import { WorkflowHelper } from 'app/service/workflow/workflow.helper';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { Tab } from 'app/shared/tabs/tabs.component';
import { ToastService } from 'app/shared/toast/ToastService';
import { EventState } from 'app/store/event.state';
import { WorkflowV3RunJobComponent } from 'app/views/workflowv3/run/workflowv3-run-job.component';
import { Subscription, timer } from 'rxjs';
import { debounce, delay, filter, retryWhen } from 'rxjs/operators';
import { webSocket, WebSocketSubject } from 'rxjs/webSocket';
import { GraphDirection } from '../graph/workflowv3-graph.lib';
import { WorkflowV3StagesGraphComponent } from '../graph/workflowv3-stages-graph.component';
import { JobRun, WorkflowRunV3 } from '../workflowv3.model';
import { WorkflowV3RunService } from '../workflowv3.run.service';
import * as actionPreferences from 'app/store/preferences.action';
import { PreferencesState } from 'app/store/preferences.state';

@Component({
    selector: 'app-workflowv3-run',
    templateUrl: './workflowv3-run.html',
    styleUrls: ['./workflowv3-run.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowV3RunComponent implements OnInit, OnDestroy {
    static INFO_PANEL_KEY = 'workflow-v3-run-info'
    static JOB_PANEL_KEY = 'workflow-v3-run-job'

    @ViewChild('graph') graph: WorkflowV3StagesGraphComponent;
    @ViewChild('v3RunJob') v3JobComponent: WorkflowV3RunJobComponent;

    data: WorkflowRunV3;
    direction: GraphDirection = GraphDirection.VERTICAL;
    resizing = false;
    loading = false;
    errors: Array<{ jobName: string, stepNumber: number }> = [];
    problems: Array<string> = [];
    infos: Array<string> = [];
    projectKey: string;
    tabs: Array<Tab>;
    selectedTab: Tab;
    selectJobRun: JobRun;
    eventSubscription: Subscription;
    results: Array<UIArtifact> = [];
    infoPanelSize: string;
    jobPanelSize: string;
    websocket: WebSocketSubject<any>;
    websocketSubscription: Subscription;
    cdnFilter: CDNStreamFilter;
    resizingSubscription: Subscription;

    constructor(
        private _cd: ChangeDetectorRef,
        private _activatedRoute: ActivatedRoute,
        private _store: Store,
        private _router: Router,
        private _workflowRunService: WorkflowRunService,
        private _workflowV3RunService: WorkflowV3RunService,
        private _toast: ToastService
    ) {
        this.tabs = [<Tab>{
            title: 'Problems',
            icon: 'warning',
            iconTheme: 'fill',
            key: 'problems',
            default: true
        }, <Tab>{
            title: 'Infos',
            icon: 'info-circle',
            iconTheme: 'outline',
            key: 'infos'
        }, <Tab>{
            title: 'Results',
            icon: 'unordered-list',
            iconTheme: 'outline',
            key: 'results'
        }];
    }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit(): void {
        const parentParams = this._activatedRoute.snapshot.parent.params;
        this.projectKey = parentParams['key'];

        this.loadWorkflowRun();

        // Refresh workflow run when receiving new events for a job
        this.eventSubscription = this._store.select(EventState.last)
            .pipe(
                filter(e => e && this.data && e.type_event === EventType.RUN_WORKFLOW_NODE
                    && e.project_key === this.projectKey
                    && e.workflow_name === this.data.resources.workflow.name
                    && e.workflow_run_num === this.data.number),
                debounce(() => timer(500))
            )
            .subscribe(e => {
                this.loadWorkflowRun();
            });

        this.resizingSubscription = this._store.select(PreferencesState.resizing).subscribe(resizing => {
            this.resizing = resizing;
            this._cd.markForCheck();
        });

        this.infoPanelSize = this._store.selectSnapshot(PreferencesState.panelSize(WorkflowV3RunComponent.INFO_PANEL_KEY));
        this.jobPanelSize = this._store.selectSnapshot(PreferencesState.panelSize(WorkflowV3RunComponent.JOB_PANEL_KEY)) ?? '50%';
    }

    async loadWorkflowRun() {
        const parentParams = this._activatedRoute.snapshot.parent.params;
        const params = this._activatedRoute.snapshot.params;
        const workflowName = parentParams['workflowName'];
        const runNumber = params['number'];

        this.loading = true;
        this._cd.markForCheck();
        this.data = await this._workflowV3RunService.getWorkflowRun(this.projectKey, workflowName, runNumber).toPromise();

        // Create errors entries for failed jobs
        this.errors = [];
        Object.keys(this.data.job_runs).forEach(k => {
            const jrs = this.data.job_runs[k];
            const jr = jrs[jrs.length - 1];
            if (jr.status === 'Fail') {
                let error = { jobName: k, stepNumber: 0 };
                const stepsWithError = (jr.step_status ?? []).filter(s => s.status === 'Fail');
                if (stepsWithError.length > 0) {
                    const step = stepsWithError[stepsWithError.length - 1];
                    error.stepNumber = step.step_order + 1;
                }
                this.errors.push(error);
            }
        });

        // Parse spawn infos
        this.infos = [];
        this.problems = [];
        this.data.infos.forEach(i => {
            switch (i.type) {
                case 'Info':
                    this.infos.push(i.user_message);
                    break;
                default:
                    this.problems.push(i.user_message);
                    break;
            }
        });


        this.loading = false;
        this._cd.markForCheck();

        await this.loadWorkflowRunResults();
    }

    async loadWorkflowRunResults() {
        const parentParams = this._activatedRoute.snapshot.parent.params;
        const params = this._activatedRoute.snapshot.params;
        const workflowName = parentParams['workflowName'];
        const runNumber = params['number'];

        this.loading = true;
        this._cd.markForCheck();

        const rs = await this._workflowRunService.getWorkflowRunResults(this.projectKey, workflowName, runNumber).toPromise();
        const artifactManagerIntegration = this.data.resources.integrations?.find(i => i.model.artifact_manager);
        this.results = WorkflowHelper.toUIArtifact(rs, artifactManagerIntegration);

        this.loading = false;
        this._cd.markForCheck();
    }

    startStreamingLogsForJob() {
        if (!this.cdnFilter) {
            this.cdnFilter = new CDNStreamFilter();
        }

        if (!this.websocket) {
            const protocol = window.location.protocol.replace('http', 'ws');
            const host = window.location.host;
            const href = this._router['location']._basePath;
            this.websocket = webSocket({
                url: `${protocol}//${host}${href}/cdscdn/item/stream`,
                openObserver: {
                    next: value => {
                        if (value.type === 'open') {
                            this.cdnFilter.job_run_id = this.selectJobRun.workflow_node_job_run_id.toString();
                            this.websocket.next(this.cdnFilter);
                        }
                    }
                }
            });

            this.websocketSubscription = this.websocket
                .pipe(retryWhen(errors => errors.pipe(delay(2000))))
                .subscribe((l: CDNLine) => {
                    if (this.v3JobComponent) {
                        this.v3JobComponent.receiveLogs(l);
                    } else {
                        console.log('job component not loaded');
                    }
                }, (err) => {
                    console.error('Error: ', err);
                }, () => {
                    console.warn('Websocket Completed');
                });
        } else {
            // Refresh cdn filter if job changed
            if (this.cdnFilter.job_run_id !== this.selectJobRun.workflow_node_job_run_id.toString()) {
                this.cdnFilter.job_run_id = this.selectJobRun.workflow_node_job_run_id.toString();
                this.websocket.next(this.cdnFilter);
            }
        }
    }

    selectTab(tab: Tab): void {
        this.selectedTab = tab;
    }

    panelStartResize(): void {
        this._store.dispatch(new actionPreferences.SetPanelResize({ resizing: true }));
    }

    panelEndResize(): void {
        this._store.dispatch(new actionPreferences.SetPanelResize({ resizing: false }));
        this._cd.detectChanges(); // force rendering to compute graph container size
        if (this.graph) {
            this.graph.resize();
        }
    }

    infoPanelEndResize(size: string): void {
        this.panelEndResize();
        this._store.dispatch(new actionPreferences.SavePanelSize({ panelKey: WorkflowV3RunComponent.INFO_PANEL_KEY, size: size }));
    }

    jobPanelEndResize(size: string): void {
        this.panelEndResize();
        this._store.dispatch(new actionPreferences.SavePanelSize({ panelKey: WorkflowV3RunComponent.JOB_PANEL_KEY, size: size }));
    }

    clickShowJobLogs(name: string): void {
        if (!this.data.job_runs[name]) {
            this.selectJobRun = null;
            this._cd.markForCheck();
            return;
        }
        this.selectJobRun = this.data.job_runs[name][0];
        this.startStreamingLogsForJob();
        this._cd.markForCheck();
    }

    closeJobPanel(): void {
        this.selectJobRun = null;
        this._cd.detectChanges(); // force rendering to compute graph container size
        if (this.graph) {
            this.graph.resize();
        }
    }

    confirmCopy() {
        this._toast.success('', 'Run result hash copied!');
    }
}
