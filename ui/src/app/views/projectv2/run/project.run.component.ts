import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, ViewChild } from "@angular/core";
import { AutoUnsubscribe } from "app/shared/decorator/autoUnsubscribe";
import { from, interval, lastValueFrom, Subscription } from "rxjs";
import { V2WorkflowRun, V2WorkflowRunJob, WorkflowRunInfo } from "app/model/v2.workflow.run.model";
import { dump } from "js-yaml";
import { V2WorkflowRunService } from "app/service/workflowv2/workflow.service";
import { PreferencesState } from "app/store/preferences.state";
import { Store } from "@ngxs/store";
import * as actionPreferences from "app/store/preferences.action";
import { Tab } from "app/shared/tabs/tabs.component";
import { ProjectV2WorkflowStagesGraphComponent } from "../vcs/repository/workflow/show/graph/stages-graph.component";
import { CDNLine, CDNStreamFilter, PipelineStatus } from "../../../model/pipeline.model";
import { webSocket, WebSocketSubject } from "rxjs/webSocket";
import { concatMap, delay, retryWhen } from "rxjs/operators";
import { ActivatedRoute, Router } from "@angular/router";
import { RunJobComponent } from "./run-job.component";
import { GraphNode } from "../vcs/repository/workflow/show/graph/graph.model";
import { NzMessageService } from "ng-zorro-antd/message";


@Component({
    selector: 'app-projectv2-run',
    templateUrl: './project.run.html',
    styleUrls: ['./project.run.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectV2WorkflowRunComponent implements OnDestroy {
    @ViewChild('graph') graph: ProjectV2WorkflowStagesGraphComponent;
    @ViewChild('runJob') runJobComponent: RunJobComponent

    workflowRun: V2WorkflowRun;
    workflowRunInfos: Array<WorkflowRunInfo>;
    selectedJobRun: V2WorkflowRunJob;
    selectedJobGate: { gate: string, job: string };
    selectedJobRunInfos: Array<WorkflowRunInfo>;
    jobs: Array<V2WorkflowRunJob>;
    workflowGraph: any;
    cdnFilter: CDNStreamFilter;

    // Subs
    sidebarSubs: Subscription;
    resizingSubscription: Subscription;
    websocket: WebSocketSubject<any>;
    websocketSubscription: Subscription;
    pollSubs: Subscription;

    // Panels
    resizing: boolean;
    infoPanelSize: number;
    jobPanelSize: number;

    tabs: Array<Tab>;
    selectedTab: Tab;

    static INFO_PANEL_KEY = 'workflow-run-info';
    static JOB_PANEL_KEY = 'workflow-run-job';

    constructor(
        private _cd: ChangeDetectorRef,
        private _workflowService: V2WorkflowRunService,
        private _store: Store,
        private _router: Router,
        private _route: ActivatedRoute,
        private _messageService: NzMessageService
    ) {
        this._route.params.subscribe(_ => {
            const runIdentifier = this._route.snapshot.params['runIdentifier'];

            if (this.workflowRun && this.workflowRun.id === runIdentifier) {
                return;
            }

            this.load();
        });
        this.resizingSubscription = this._store.select(PreferencesState.resizing).subscribe(resizing => {
            this.resizing = resizing;
            this._cd.markForCheck();
        });
        this.infoPanelSize = this._store.selectSnapshot(PreferencesState.panelSize(ProjectV2WorkflowRunComponent.INFO_PANEL_KEY));
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

    async load() {
        const projectKey = this._route.snapshot.parent.params['key'];
        const runIdentifier = this._route.snapshot.params['runIdentifier'];

        delete this.selectedJobGate;
        delete this.selectedJobRun;
        delete this.selectedJobRunInfos;
        delete this.workflowGraph;
        if (this.pollSubs) {
            this.pollSubs.unsubscribe();
        }

        try {
            this.workflowRun = await lastValueFrom(this._workflowService.getRun(projectKey, runIdentifier));
            this.workflowRunInfos = await lastValueFrom(this._workflowService.getRunInfos(this.workflowRun));
            await this.loadJobs();
        } catch (e) {
            this._messageService.error(`Unable to get workflow run: ${e?.error?.error}`, { nzDuration: 2000 });
        }

        this.pollSubs = interval(5000)
            .pipe(concatMap(_ => from(this.loadJobs())))
            .subscribe();
        this.workflowGraph = dump(this.workflowRun.workflow_data.workflow);

        this._cd.markForCheck();
    }

    async loadJobs() {
        let updatedJobs = await lastValueFrom(this._workflowService.getJobs(this.workflowRun));
        if (this.selectedJobRun) {
            await this.selectJob(this.selectedJobRun.job_id);
        }
        this.jobs = Object.assign([], updatedJobs);
        if (PipelineStatus.isDone(this.workflowRun.status)) {
            this.pollSubs.unsubscribe();
        }
        this._cd.markForCheck();
    }

    onBack(): void { }

    selectTab(tab: Tab): void {
        this.selectedTab = tab;
    }

    panelStartResize(): void {
        this._store.dispatch(new actionPreferences.SetPanelResize({ resizing: true }));
    }

    infoPanelEndResize(size: number): void {
        this.panelEndResize();
        this._store.dispatch(new actionPreferences.SavePanelSize({
            panelKey: ProjectV2WorkflowRunComponent.INFO_PANEL_KEY,
            size: size
        }));
    }

    jobPanelEndResize(size: number): void {
        this.panelEndResize();
        this._store.dispatch(new actionPreferences.SavePanelSize({
            panelKey: ProjectV2WorkflowRunComponent.JOB_PANEL_KEY,
            size: size
        }));
    }

    panelEndResize(): void {
        this._store.dispatch(new actionPreferences.SetPanelResize({ resizing: false }));
        this._cd.detectChanges(); // force rendering to compute graph container size
        if (this.graph) {
            this.graph.resize();
        }
    }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    selectJobGate(gateNode: GraphNode): void {
        delete this.selectedJobRun;
        this.selectedJobGate = { gate: gateNode.gateName, job: gateNode.gateChild };
        this._cd.markForCheck();
    }

    async selectJob(runJobID: string) {
        let jobRun = this.jobs.find(j => j.id === runJobID);
        if (this.selectedJobRun && jobRun && jobRun.id === this.selectedJobRun.id) {
            return;
        }
        delete this.selectedJobGate;
        this.selectedJobRun = jobRun;
        if (!this.selectedJobRun) {
            this._cd.markForCheck();
            return;
        }
        if (!PipelineStatus.isDone(jobRun.status)) {
            this.startStreamingLogsForJob();
        }

        this.selectedJobRunInfos = await lastValueFrom(this._workflowService.getRunJobInfos(this.workflowRun, jobRun.id));
        this._cd.markForCheck();
    }

    unselectJob(): void {
        delete this.selectedJobRunInfos;
        delete this.selectedJobRun;
        if (this.graph) {
            this.graph.resize();
        }
        this._cd.detectChanges(); // force rendering to compute graph container size
    }

    startStreamingLogsForJob() {
        if (!this.cdnFilter) {
            this.cdnFilter = new CDNStreamFilter();
        }
        if (this.cdnFilter.job_run_id === this.selectedJobRun.id) {
            return;
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
                            this.cdnFilter.job_run_id = this.selectedJobRun.id;
                            this.websocket.next(this.cdnFilter);
                        }
                    }
                }
            });

            this.websocketSubscription = this.websocket
                .pipe(retryWhen(errors => errors.pipe(delay(2000))))
                .subscribe((l: CDNLine) => {
                    if (this.runJobComponent) {
                        this.runJobComponent.receiveLogs(l);
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
            if (this.cdnFilter.job_run_id !== this.selectedJobRun.id) {
                this.cdnFilter.job_run_id = this.selectedJobRun.id;
                this.websocket.next(this.cdnFilter);
            }
        }
    }

}
