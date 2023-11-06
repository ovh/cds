import {ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, ViewChild} from "@angular/core";
import {AutoUnsubscribe} from "app/shared/decorator/autoUnsubscribe";
import {SidebarService} from "app/service/sidebar/sidebar.service";
import {from, interval, Subscription} from "rxjs";
import {V2WorkflowRun, V2WorkflowRunJob, WorkflowRunInfo, WorkflowRunResult} from "app/model/v2.workflow.run.model";
import {dump} from "js-yaml";
import {V2WorkflowRunService} from "app/service/workflowv2/workflow.service";
import {PreferencesState} from "app/store/preferences.state";
import {Store} from "@ngxs/store";
import * as actionPreferences from "app/store/preferences.action";
import {Tab} from "app/shared/tabs/tabs.component";
import {ProjectV2WorkflowStagesGraphComponent} from "../vcs/repository/workflow/show/graph/stages-graph.component";
import {CDNLine, CDNStreamFilter, PipelineStatus} from "../../../model/pipeline.model";
import {webSocket, WebSocketSubject} from "rxjs/webSocket";
import {concatMap, delay, retryWhen} from "rxjs/operators";
import {Router} from "@angular/router";
import {RunJobComponent} from "./run-job.component";


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

    selectedRun: V2WorkflowRun;
    selectedJobRun: V2WorkflowRunJob;
    selectedJobRunInfos: Array<WorkflowRunInfo>;
    jobs: Array<V2WorkflowRunJob>;
    selectedRunInfos: Array<WorkflowRunInfo>;
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

    constructor(private _sidebarService: SidebarService, private _cd: ChangeDetectorRef,
                private _workflowService: V2WorkflowRunService, private _store: Store, private _router: Router) {
        this.sidebarSubs = this._sidebarService.getRunObservable().subscribe(r => {
            if (r?.id === this.selectedRun?.id && r?.status === this.selectedRun?.status) {
                return;
            }
            delete this.selectedJobRun;
            delete this.selectedJobRunInfos;
            delete this.selectedRunInfos;
            if (this.pollSubs) {
                this.pollSubs.unsubscribe();
            }
            if (r?.id !== this.selectedRun?.id) {
                delete this.selectedRunInfos;
                delete this.jobs;
            }
            this.selectedRun = r;
            if (r) {
                this.loadJobs();
                this.pollSubs = interval(5000)
                    .pipe(concatMap(_ => from(this.loadJobs())))
                    .subscribe();
                this.workflowGraph = dump(r.workflow_data.workflow);
                this._workflowService.getRunInfos(r).subscribe(infos => {
                    this.selectedRunInfos = infos;
                    this._cd.markForCheck();
                });
            } else {
                delete this.workflowGraph;
            }
            this._cd.markForCheck();
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

    async loadJobs() {
        let updatedJobs = await this._workflowService.getJobs(this.selectedRun).toPromise();
        if (this.selectedJobRun) {
            this.selectJob(this.selectedJobRun.job_id)
        }
        this.jobs = Object.assign([], updatedJobs);
        if (PipelineStatus.isDone(this.selectedRun.status)) {
            this.pollSubs.unsubscribe();
        }
        this._cd.markForCheck();
    }

    selectTab(tab: Tab): void {
        this.selectedTab = tab;
    }

    panelStartResize(): void {
        this._store.dispatch(new actionPreferences.SetPanelResize({resizing: true}));
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
        this._store.dispatch(new actionPreferences.SetPanelResize({resizing: false}));
        this._cd.detectChanges(); // force rendering to compute graph container size
        if (this.graph) {
            this.graph.resize();
        }
    }

    ngOnDestroy(): void {
    }

    selectJob(runJobID: string): void {
        let jobRun = this.jobs.find(j => j.id === runJobID);
        if (this.selectedJobRun && jobRun && jobRun.id === this.selectedJobRun.id) {
            return;
        }
        this.selectedJobRun = jobRun;
        if (!this.selectedJobRun) {
            this._cd.markForCheck();
            return;
        }
        if (!PipelineStatus.isDone(jobRun.status)) {
            this.startStreamingLogsForJob();
        }

        this._workflowService.getRunJobInfos(this.selectedRun, jobRun.id).subscribe(infos => {
            this.selectedJobRunInfos = infos;
            this._cd.markForCheck();
        });
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
            const href = this._router['location']._baseHref;
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
