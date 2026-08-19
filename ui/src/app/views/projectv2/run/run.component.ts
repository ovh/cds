import { AfterViewInit, ChangeDetectionStrategy, ChangeDetectorRef, Component, HostListener, inject, OnDestroy, TemplateRef, ViewChild } from "@angular/core";
import { AutoUnsubscribe } from "app/shared/decorator/autoUnsubscribe";
import { from, interval, lastValueFrom, Subscription } from "rxjs";
import { RUN_SUMMARY_HEADERS, V2WorkflowRunService } from "app/service/workflowv2/workflow.service";
import { PreferencesState } from "app/store/preferences.state";
import { Store } from "@ngxs/store";
import * as actionPreferences from "app/store/preferences.action";
import { Tab } from "app/shared/tabs/tabs.component";
import { Tests } from "../../../model/pipeline.model";
import { concatMap, map, switchMap } from "rxjs/operators";
import { ActivatedRoute, Router } from "@angular/router";
import { NzMessageService } from "ng-zorro-antd/message";
import { NavigationState } from "app/store/navigation.state";
import { V2JobGate, V2WorkflowRun, V2WorkflowRunJob, V2WorkflowRunJobStatus, V2WorkflowRunJobStatusIsFailed, V2WorkflowRunStatus, V2WorkflowRunStatusIsTerminated, WorkflowRunInfo, WorkflowRunResult, WorkflowRunResultType, areAllJobVariantsSelected, groupRunJobSelectionsByJobId } from "../../../../../libs/workflow-graph/src/lib/v2.workflow.run.model";
import { RunTriggerComponent } from "./run-trigger.component";
import { RouterService } from "app/service/services.module";
import { ErrorUtils } from "app/shared/error.utils";
import moment from "moment";
import { NzDrawerService } from "ng-zorro-antd/drawer";
import { ProjectV2RunStartComponent, ProjectV2RunStartComponentParams } from "../run-start/run-start.component";
import { HttpClient, HttpHeaders, HttpParams } from "@angular/common/http";
import { Clipboard } from '@angular/cdk/clipboard';
import { LiveAnnouncer } from '@angular/cdk/a11y';
import { GraphComponent } from "../../../../../libs/workflow-graph/src/public-api";
import { Title } from "@angular/platform-browser";
import { WebsocketV2Filter, WebsocketV2FilterType } from "app/model/websocket-v2";
import { EventV2Service } from "app/event-v2.service";
import { EventV2Type, FullEventV2 } from "app/model/event-v2.model";
import { EventV2State } from "app/store/event-v2.state";
import { animate, keyframes, state, style, transition, trigger } from "@angular/animations";

@Component({
    standalone: false,
    selector: 'app-projectv2-run',
    templateUrl: './run.html',
    styleUrls: ['./run.scss'],
    animations: [
        trigger('appendToList', [
            state('active', style({
                opacity: 1
            })),
            state('append', style({
                opacity: 1
            })),
            transition('append => active', animate('0ms')),
            transition('active => append', animate('1000ms', keyframes([
                style({ opacity: 1 }),
                style({ opacity: 0.5 }),
                style({ opacity: 1 })
            ])))
        ]),
        // Two runs of the same workflow in the same state look alike: without this, switching from
        // one to the other moves nothing but a number, and nothing tells the user it happened.
        trigger('runSwitch', [
            transition('* => *', [
                style({ opacity: 0.25 }),
                animate('200ms ease-out', style({ opacity: 1 }))
            ])
        ])
    ],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectV2RunComponent implements AfterViewInit, OnDestroy {
    @ViewChild('graph') graph: GraphComponent;
    @ViewChild('tabResultsTemplate') tabResultsTemplate: TemplateRef<any>;
    @ViewChild('tabTestsTemplate') tabTestsTemplate: TemplateRef<any>;
    @ViewChild('shareLink') shareLink: any;

    runs: Array<V2WorkflowRun>;
    workflowRun: V2WorkflowRun;
    workflowRunInfo: Array<WorkflowRunInfo>;
    selectedItemType: string;
    selectedItemShareLink: string;
    selectedSourceKey: string;
    selectedJobRun: V2WorkflowRunJob;
    selectedHookName: string;
    selectedRunResult: WorkflowRunResult;
    selectedTest: string;
    jobs: Array<V2WorkflowRunJob>;
    workflowGraph: any;
    selectedRunAttempt: number;
    results: Array<WorkflowRunResult>;
    tests: Tests;
    projectKey: string;
    workflowRunIsTerminated: boolean = false;
    workflowRunIsActive: boolean = false;
    hasJobsFailed: boolean = false;
    hasSkippedGateJobs: boolean = false;
    loading: { restart: boolean, stop: boolean } = {
        restart: false,
        stop: false
    };
    /** The run itself is being read: the view shows its shape, waiting for what fills it. */
    loadingRun: boolean = false;
    /**
     * The last run asked for. Reads are not cancellable, so a read that is not for this run anymore
     * is thrown away instead of being shown: going through runs quickly must land on the last one,
     * never on whichever answered last.
     */
    private requestedRunID: string;
    /** Holds the place of the runs of the sidebar while they are being read. */
    readonly runsPlaceholders = Array.from({ length: 8 }, (_, i) => i);
    /** Someone who asked for less movement gets no transition when the run changes. */
    readonly reduceMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    animatedRuns: { [key: string]: boolean } = {};
    selectionModeActive: boolean = false;
    gateDrawerOpen: boolean = false;
    /** Selected run job IDs for restart. Both simple jobs and matrix variants are tracked uniformly by their run job UUID. */
    selectedRunJobIds: Array<string> = [];

    // Subs
    paramsSub: Subscription;
    queryParamsSub: Subscription;
    resizingSubscription: Subscription;
    refreshSubs: Subscription;
    eventV2Subscription: Subscription;
    connectedSubscription: Subscription;

    /** Last applied event per subject, events are not ordered on the wire. */
    private lastEventTimestamps: { [key: string]: number } = {};
    private refreshTimers: { [key: string]: any } = {};
    /** Bumped by every event changing the state of the run, to spot a read racing with it. */
    private eventSequence: number = 0;

    // Panels
    resizing: boolean;
    infoPanelSize: string;
    jobPanelSize: string;
    panelExpanded: boolean;

    tabs: Array<Tab>;
    selectedTab: Tab;

    static INFO_PANEL_KEY = 'workflow-run-info';
    static JOB_PANEL_KEY = 'workflow-run-job';
    /**
     * The view is driven by the events of the run. This refresh is only a safety net for what events
     * cannot carry (run infos written without any status change) and for what a disconnection may have
     * dropped, hence its low frequency.
     */
    static SAFETY_REFRESH_DELAY = 30000;

    /** Panels without a target are serialized as "type" alone, others as "type:encodedData". */
    static parsePanelParam(value: string): { type: string, data: string } {
        const separator = value.indexOf(':');
        if (separator === -1) {
            return { type: value, data: null };
        }
        return { type: value.substring(0, separator), data: decodeURIComponent(value.substring(separator + 1)) };
    }

    private _cd = inject(ChangeDetectorRef);
    private _workflowService = inject(V2WorkflowRunService);
    private _store = inject(Store);
    private _router = inject(Router);
    private _route = inject(ActivatedRoute);
    private _messageService = inject(NzMessageService);
    private _routerService = inject(RouterService);
    private _drawerService = inject(NzDrawerService);
    private _clipboard = inject(Clipboard);
    private _titleService = inject(Title);
    private _http = inject(HttpClient);
    private _eventV2Service = inject(EventV2Service);
    private _liveAnnouncer = inject(LiveAnnouncer);

    constructor() {
        // switchMap, not concatMap: going through several runs quickly must not read them one after
        // the other and show each of them on the way. Only the last one asked for is applied, see the
        // guard in load().
        this.paramsSub = this._route.params.pipe(
            switchMap(_ => {
                const params = this._routerService.getRouteSnapshotParams({}, this._router.routerState.snapshot.root);
                const workflowRunID = params['workflowRunID'];
                if (this.workflowRun && this.workflowRun.id === workflowRunID) {
                    return from([]);
                }
                this.projectKey = params['key'];

                return from(this.load(workflowRunID).then(() => {
                    const params = this._route.snapshot.queryParams;
                    if (params['panel']) {
                        const panel = ProjectV2RunComponent.parsePanelParam(params['panel']);
                        this.openPanel(panel.type, panel.data);
                    }
                }));
            })
        ).subscribe(_ => { });

        this.queryParamsSub = this._route.queryParams.subscribe(params => {
            if (params['panel'] && this.workflowRun && this.jobs) {
                const panel = ProjectV2RunComponent.parsePanelParam(params['panel']);
                this.openPanel(panel.type, panel.data);
            }
        });

        this.resizingSubscription = this._store.select(PreferencesState.resizing).subscribe(resizing => {
            this.resizing = resizing;
            this._cd.markForCheck();
        });
        this.infoPanelSize = this._store.selectSnapshot(PreferencesState.panelSize(ProjectV2RunComponent.INFO_PANEL_KEY));
        this.jobPanelSize = this._store.selectSnapshot(PreferencesState.panelSize(ProjectV2RunComponent.JOB_PANEL_KEY)) ?? '50%';

        this.eventV2Subscription = this._store.select(EventV2State.last).subscribe((event) => this.onEvent(event));

        // Events sent while the websocket was down are lost, so read everything again when it opens.
        this.connectedSubscription = this._eventV2Service.connected$.subscribe(connected => {
            if (connected && this.workflowRun) {
                this.scheduleReload();
            }
        });
    }

    ngAfterViewInit(): void {
        this.tabs = [<Tab>{
            title: 'Info',
            key: 'info',
            default: true
        }, <Tab>{
            title: 'Results',
            key: 'results',
            template: this.tabResultsTemplate
        }, <Tab>{
            title: 'Tests',
            key: 'tests',
            template: this.tabTestsTemplate
        }];
        this._cd.markForCheck();
    }

    ngOnDestroy(): void {
        this.clearScheduledRefresh();
    }

    // A tab left in the background has its timers throttled and its websocket frames delayed, so what
    // it displays cannot be trusted when the user comes back to it.
    @HostListener('document:visibilitychange')
    onVisibilityChange(): void {
        if (document.visibilityState === 'visible' && this.workflowRun && !this.workflowRunIsTerminated) {
            this.scheduleReload();
        }
    }

    onEvent(event: FullEventV2): void {
        if (!event) {
            return;
        }

        this.updateRunList(event);

        // An event for the run currently being read: what is being read is already behind it, and
        // load() will notice through the sequence.
        if (this.loadingRun && event.workflow_run_id === this.requestedRunID) {
            this.eventSequence++;
        }

        if (!this.workflowRun || event.workflow_run_id !== this.workflowRun.id) {
            return;
        }

        switch (event.type) {
            case EventV2Type.EventRunCrafted:
            case EventV2Type.EventRunBuilding:
            case EventV2Type.EventRunUpdated:
            case EventV2Type.EventRunEnded:
            case EventV2Type.EventRunRestart:
                this.applyRunEvent(event);
                break;
            case EventV2Type.EventRunDeleted:
                this._messageService.warning('This workflow run has been deleted', { nzDuration: 4000 });
                break;
            case EventV2Type.EventRunJobManualTriggered:
                // The inputs of the gate are kept on the run, which this event does not carry.
                this.scheduleReload();
                break;
            case EventV2Type.EventRunJobRunResultAdded:
            case EventV2Type.EventRunJobRunResultUpdated:
                this.applyRunResultEvent(event);
                break;
            default:
                if (event.type.indexOf('RunJob') === 0) {
                    this.applyRunJobEvent(event);
                }
        }
    }

    /** The sidebar lists the last runs of the same workflow, fed by the run events of the project. */
    private updateRunList(event: FullEventV2): void {
        if ([EventV2Type.EventRunCrafted, EventV2Type.EventRunBuilding, EventV2Type.EventRunEnded, EventV2Type.EventRunRestart].indexOf(event.type) === -1) { return; }
        if (!this.runs) { return; }
        const idx = this.runs.findIndex(run => run.id === event.workflow_run_id);
        delete (this.animatedRuns[event.payload.id]);
        this._cd.detectChanges();
        if (idx !== -1) {
            this.runs[idx] = event.payload;
        } else {
            this.runs = [event.payload].concat(...this.runs);
            if (this.runs.length > 50) {
                this.runs.pop();
            }
        }
        this.animatedRuns[event.payload.id] = true;
        this._cd.markForCheck();
    }

    private applyRunEvent(event: FullEventV2): void {
        if (!this.isNewerEvent('run', event)) { return; }

        const run = event.payload as V2WorkflowRun;
        const previousStatus = this.workflowRun.status;
        const previousAttempt = this.workflowRun.run_attempt;
        const previousShape = ProjectV2RunComponent.workflowShape(this.workflowRun.workflow_data?.workflow);
        const wasOnLastAttempt = this.selectedRunAttempt === previousAttempt;

        // The payload of the event carries the results of the jobs in the contexts, the run read from
        // the API does not: drop them so that the run stays the same object whatever refreshed it.
        const { jobs, ...contexts } = run.contexts ?? {};
        this.workflowRun = { ...run, contexts };
        this.workflowRunIsTerminated = V2WorkflowRunStatusIsTerminated(run.status);
        this.workflowRunIsActive = !this.workflowRunIsTerminated;

        if (previousStatus !== run.status) {
            this._liveAnnouncer.announce(`Run ${run.run_number} ${run.status}`, 'polite');
        }

        // The run was restarted from somewhere else: follow its new attempt, unless the user was
        // deliberately looking at a previous one.
        if (run.run_attempt > previousAttempt && wasOnLastAttempt) {
            this.selectedRunAttempt = run.run_attempt;
            this.scheduleJobsRefresh();
        }

        // Jobs coming from a template or a matrix replace the job that declared them while the run is
        // going: the graph has to be drawn again from the new definition, and the new jobs read.
        if (previousShape !== ProjectV2RunComponent.workflowShape(run.workflow_data?.workflow)) {
            this.workflowGraph = run.workflow_data.workflow;
            this.scheduleJobsRefresh();
        }

        this.scheduleRunInfoRefresh();

        if (this.workflowRunIsTerminated) {
            this.stopSafetyRefresh();
            // Last word on the run: read everything the events may have missed while it was running.
            this.scheduleJobsRefresh();
        } else {
            this.startSafetyRefresh();
        }

        this.eventSequence++;
        this._cd.markForCheck();
    }

    private applyRunJobEvent(event: FullEventV2): void {
        if (!this.jobs || event.run_attempt !== this.selectedRunAttempt) { return; }
        if (!this.isNewerEvent(`job-${event.run_job_id}`, event)) { return; }

        const job = event.payload as V2WorkflowRunJob;
        if (!job?.id) { return; }

        // A job the displayed definition does not know about: it was added by a template or a matrix
        // expanded after the graph was drawn, the run has to be read again to get its new definition.
        if (!this.workflowRun.workflow_data?.workflow?.jobs?.[job.job_id]) {
            this.scheduleReload();
            return;
        }

        const idx = this.jobs.findIndex(j => j.id === job.id);
        if (idx !== -1) {
            this.jobs = this.jobs.map((j, i) => i === idx ? job : j);
        } else {
            // A retry takes the place of the run job it retries: the list holds the last retry of each
            // job, as the API returns it.
            const retried = this.jobs.find(j => ProjectV2RunComponent.isSameJob(j, job) && j.retry < job.retry);
            this.jobs = retried ? this.jobs.map(j => j === retried ? job : j) : this.jobs.concat(job);

            // The panel was open on the run job that has just been retried: it stays open on the job,
            // now holding its new retry. Whether it shows that retry or the one being read is the
            // panel's own decision.
            if (retried && this.selectedItemType === 'job' && this.selectedJobRun?.id === retried.id) {
                this.openPanel('job', job.id);
            }
        }

        if (this.selectedJobRun?.id === job.id) {
            this.selectedJobRun = job;
        }

        this.hasJobsFailed = this.jobs.filter(j => V2WorkflowRunJobStatusIsFailed(j.status)).length > 0;
        this.hasSkippedGateJobs = this.getSkippedGateJobIds().length > 0;

        // Steps moving forward do not change the state of the run: no need to read its infos, and a
        // read racing with them can be trusted.
        if (event.type !== EventV2Type.EventRunJobStepUpdated) {
            this.scheduleRunInfoRefresh();
            this.eventSequence++;
        }

        this._cd.markForCheck();
    }

    private applyRunResultEvent(event: FullEventV2): void {
        if (!this.results || event.run_attempt !== this.selectedRunAttempt) { return; }

        const result = event.payload as WorkflowRunResult;
        if (!result?.id || !this.isNewerEvent(`result-${result.id}`, event)) { return; }

        const idx = this.results.findIndex(r => r.id === result.id);
        this.results = idx !== -1 ? this.results.map((r, i) => i === idx ? result : r) : this.results.concat(result);

        if (this.selectedRunResult?.id === result.id) {
            this.selectedRunResult = result;
        }
        if (this.results.find(r => r.type === WorkflowRunResultType.tests)) {
            this.computeTestsReport();
        }

        this.eventSequence++;
        this._cd.markForCheck();
    }

    /** Events are not ordered on the wire, an outdated one must not overwrite a fresher state. */
    private isNewerEvent(key: string, event: FullEventV2): boolean {
        const timestamp = Date.parse(event.timestamp);
        if (isNaN(timestamp)) {
            return true;
        }
        if (this.lastEventTimestamps[key] > timestamp) {
            return false;
        }
        this.lastEventTimestamps[key] = timestamp;
        return true;
    }

    private static workflowPath(run: V2WorkflowRun): string {
        return `${run.vcs_server}/${run.repository}/${run.workflow_name}`;
    }

    /** Two run jobs of the same job of the workflow: same name, and same matrix variant if any. */
    private static isSameJob(a: V2WorkflowRunJob, b: V2WorkflowRunJob): boolean {
        if (a.job_id !== b.job_id) {
            return false;
        }
        const variant = (matrix: { [key: string]: string }) => Object.keys(matrix ?? {}).sort().map(k => `${k}=${matrix[k]}`).join(',');
        return variant(a.matrix) === variant(b.matrix);
    }

    /**
     * What the graph is drawn from: the jobs of the run, their stage, their gate and their
     * dependencies. Two runs sharing that shape draw the same graph.
     */
    private static workflowShape(workflow: any): string {
        if (!workflow) {
            return '';
        }
        const jobs = workflow.jobs ?? {};
        const stages = workflow.stages ?? {};
        return JSON.stringify({
            jobs: Object.keys(jobs).sort().map(k => [k, jobs[k]?.stage ?? '', jobs[k]?.gate ?? '', (jobs[k]?.needs ?? []).slice().sort()]),
            stages: Object.keys(stages).sort().map(k => [k, (stages[k]?.needs ?? []).slice().sort()])
        });
    }

    private scheduleJobsRefresh(): void {
        this.scheduleRefresh('jobs', 300, () => this.loadJobsAndResults());
    }

    private scheduleReload(): void {
        this.scheduleRefresh('reload', 300, () => this.reload());
    }

    private scheduleRunInfoRefresh(): void {
        this.scheduleRefresh('infos', 1000, () => this.loadRunInfos());
    }

    private scheduleRefresh(key: string, delayMs: number, refresh: () => Promise<void>): void {
        if (this.refreshTimers[key]) {
            clearTimeout(this.refreshTimers[key]);
        }
        this.refreshTimers[key] = setTimeout(() => {
            delete this.refreshTimers[key];
            refresh();
        }, delayMs);
    }

    private clearScheduledRefresh(): void {
        Object.keys(this.refreshTimers).forEach(k => clearTimeout(this.refreshTimers[k]));
        this.refreshTimers = {};
    }

    private startSafetyRefresh(): void {
        if (this.refreshSubs) {
            return;
        }
        this.refreshSubs = interval(ProjectV2RunComponent.SAFETY_REFRESH_DELAY)
            .pipe(concatMap(_ => from(this.reload())))
            .subscribe();
    }

    private stopSafetyRefresh(): void {
        if (!this.refreshSubs) {
            return;
        }
        this.refreshSubs.unsubscribe();
        delete this.refreshSubs;
    }

    /**
     * Read a workflow run and everything shown around it. Selection mode and the open panel are
     * dropped on the way, so that nothing of the previous run carries over to this one.
     */
    async load(workflowRunID: string, runAttempt?: number) {
        this.clearPanel();
        // The graph of the run being left is kept until the next one is read: emptying it here would
        // show a blank frame between two runs.
        this.stopSafetyRefresh();
        this.clearScheduledRefresh();
        this.lastEventTimestamps = {};
        this.loadingRun = true;
        this.requestedRunID = workflowRunID;

        if (this.graph) {
            this.graph.setSelectionModeActive(false);
        }

        // Watch every event of this run: the run itself, its jobs, their steps and their results.
        this._eventV2Service.updateFilter(<WebsocketV2Filter>{
            type: WebsocketV2FilterType.PROJECT_RUN,
            project_key: this.projectKey,
            workflow_run_id: workflowRunID
        });

        // The jobs, the results and the infos of a run are keyed by its id, not by anything the run
        // carries: they are read at the same time as the run rather than one after the other, so the
        // whole view is drawn once, complete, after a single round trip.
        const sequence = this.eventSequence;
        const [run, jobs, results, infos] = await Promise.all([
            this.fetchRun(workflowRunID),
            this.fetchJobs(workflowRunID, runAttempt),
            this.fetchResults(workflowRunID, runAttempt),
            this.fetchRunInfos(workflowRunID)
        ]);

        // Another run was asked for while this one was being read: it is the one the user is waiting
        // for, this answer is dropped rather than shown on the way.
        if (this.requestedRunID !== workflowRunID) {
            return;
        }

        this.loadingRun = false;

        if (!run) {
            this._cd.markForCheck();
            return;
        }

        // The sidebar lists the runs of one workflow: coming from another one, what it holds has
        // nothing to do with this run.
        if (this.workflowRun && ProjectV2RunComponent.workflowPath(this.workflowRun) !== ProjectV2RunComponent.workflowPath(run)) {
            delete this.runs;
        }

        // The definition of the run and everything filling it are set together, without an await
        // between them: no rendering can catch the graph of this run holding the jobs of another,
        // which would paint its nodes with the statuses of the previous one.
        this.applyRun(run);
        this.selectedRunAttempt = runAttempt ?? run.run_attempt;
        this._titleService.setTitle(`#${run.run_number} [${run.contexts.git.ref_name}] • ${run.vcs_server}/${run.repository}/${run.workflow_name} • Workflow Run`);
        this.workflowGraph = run.workflow_data.workflow;

        // Nothing of the run being left is kept: what could not be read is shown empty rather than
        // with the values of the previous run.
        this.applyJobs(jobs ?? []);
        this.applyResults(results ?? []);
        this.applyRunInfos(infos ?? []);

        await this.refreshPanel();
        this._cd.markForCheck();

        if (sequence !== this.eventSequence) {
            this.scheduleJobsRefresh();
        }

        // The sibling runs of the sidebar are the only thing that needs the run to be read first.
        // Nothing else waits for them.
        this.loadRuns();
    }

    async loadRuns() {
        const runID = this.workflowRun.id;

        let params = new HttpParams();
        params = params.appendAll({
            workflow: `${this.workflowRun.vcs_server}/${this.workflowRun.repository}/${this.workflowRun.workflow_name}`,
            offset: 0,
            limit: 50
        });

        this._eventV2Service.updateFilter(<WebsocketV2Filter>{
            type: WebsocketV2FilterType.PROJECT_RUNS,
            project_key: this.projectKey,
            project_runs_params: params.toString()
        });

        try {
            const res = await lastValueFrom(this._http.get(`/v2/project/${this.projectKey}/run`, {
                params,
                headers: RUN_SUMMARY_HEADERS,
                observe: 'response'
            })
                .pipe(map(res => {
                    let headers: HttpHeaders = res.headers;
                    return {
                        totalCount: parseInt(headers.get('X-Total-Count'), 10),
                        runs: res.body as Array<V2WorkflowRun>
                    };
                })));
            if (this.isStale(runID)) {
                return;
            }
            this.runs = res.runs;
        } catch (e) {
            if (this.isStale(runID)) {
                return;
            }
            this._messageService.error(`Unable to list workflow runs: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
            this.runs = [];
        }

        // Nothing awaits this read: without marking the view, the runs would only appear on the next
        // interaction.
        this._cd.markForCheck();
    }

    private async fetchRun(workflowRunID: string): Promise<V2WorkflowRun> {
        try {
            return await lastValueFrom(this._workflowService.getRun(this.projectKey, workflowRunID));
        } catch (e) {
            this._messageService.error(`Unable to get workflow run: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
            return null;
        }
    }

    private async fetchJobs(workflowRunID: string, attempt: number): Promise<Array<V2WorkflowRunJob>> {
        try {
            return await lastValueFrom(this._workflowService.getJobs(this.projectKey, workflowRunID, attempt));
        } catch (e) {
            this._messageService.error(`Unable to get jobs: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
            return null;
        }
    }

    private async fetchResults(workflowRunID: string, attempt: number): Promise<Array<WorkflowRunResult>> {
        try {
            return await lastValueFrom(this._workflowService.getResults(this.projectKey, workflowRunID, attempt));
        } catch (e) {
            this._messageService.error(`Unable to get results: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
            return null;
        }
    }

    private async fetchRunInfos(workflowRunID: string): Promise<Array<WorkflowRunInfo>> {
        try {
            return await lastValueFrom(this._workflowService.getRunInfos(this.projectKey, workflowRunID));
        } catch (e) {
            this._messageService.error(`Unable to get run infos: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
            return null;
        }
    }

    private applyRun(run: V2WorkflowRun): void {
        this.workflowRun = run;
        this.workflowRunIsTerminated = V2WorkflowRunStatusIsTerminated(run.status);
        this.workflowRunIsActive = !this.workflowRunIsTerminated;
        if (this.workflowRunIsActive) {
            this.startSafetyRefresh();
        } else {
            this.stopSafetyRefresh();
        }
    }

    private applyJobs(jobs: Array<V2WorkflowRunJob>): void {
        if (!jobs) {
            return;
        }
        this.jobs = jobs;
        this.hasJobsFailed = this.jobs.filter(j => V2WorkflowRunJobStatusIsFailed(j.status)).length > 0;
        this.hasSkippedGateJobs = this.getSkippedGateJobIds().length > 0;
    }

    private applyResults(results: Array<WorkflowRunResult>): void {
        if (!results) {
            return;
        }
        this.results = results;
        if (this.results.find(r => r.type === WorkflowRunResultType.tests)) {
            this.computeTestsReport();
        } else {
            // A run without test result must not keep showing the report of the previous one.
            delete this.tests;
        }
    }

    private applyRunInfos(infos: Array<WorkflowRunInfo>): void {
        if (!infos) {
            return;
        }
        this.workflowRunInfo = infos.sort((a, b) => moment(a.issued_at).isBefore(moment(b.issued_at)) ? 1 : -1);
    }

    async loadJobsAndResults() {
        const sequence = this.eventSequence;
        const runID = this.workflowRun.id;

        const [jobs, results, infos] = await Promise.all([
            this.fetchJobs(runID, this.selectedRunAttempt),
            this.fetchResults(runID, this.selectedRunAttempt),
            this.fetchRunInfos(runID)
        ]);

        if (this.isStale(runID)) {
            return;
        }

        this.applyJobs(jobs);
        this.applyResults(results);
        this.applyRunInfos(infos);

        await this.refreshPanel();

        this._cd.markForCheck();

        // An event landed while this was being read: what has just been read is already outdated.
        if (sequence !== this.eventSequence) {
            this.scheduleJobsRefresh();
        }
    }

    async loadRunInfos(): Promise<void> {
        const runID = this.workflowRun.id;
        const infos = await this.fetchRunInfos(runID);
        if (this.isStale(runID)) {
            return;
        }
        this.applyRunInfos(infos);
        this._cd.markForCheck();
    }

    /** Whether what was read is not for the run the view must show anymore. */
    private isStale(runID: string): boolean {
        return this.requestedRunID !== runID;
    }

    /** Read the run and everything shown around it again, the events being trusted for nothing here. */
    async reload(): Promise<void> {
        const previousShape = ProjectV2RunComponent.workflowShape(this.workflowRun.workflow_data?.workflow);

        const sequence = this.eventSequence;
        const runID = this.workflowRun.id;
        const [run, jobs, results, infos] = await Promise.all([
            this.fetchRun(runID),
            this.fetchJobs(runID, this.selectedRunAttempt),
            this.fetchResults(runID, this.selectedRunAttempt),
            this.fetchRunInfos(runID)
        ]);

        if (this.isStale(runID)) {
            return;
        }

        if (run) {
            this.applyRun(run);
            // Draw the graph again when the definition of the run changed, a template or a matrix
            // having replaced the job that declared them.
            if (previousShape !== ProjectV2RunComponent.workflowShape(run.workflow_data?.workflow)) {
                this.workflowGraph = run.workflow_data.workflow;
            }
        }
        this.applyJobs(jobs);
        this.applyResults(results);
        this.applyRunInfos(infos);

        await this.refreshPanel();

        this._cd.markForCheck();

        if (sequence !== this.eventSequence) {
            this.scheduleJobsRefresh();
        }
    }

    computeTestsReport(): void {
        this.tests = <Tests>{
            ko: 0,
            ok: 0,
            skipped: 0,
            total: 0,
            test_suites: []
        };

        (this.results ?? []).filter(r => r.type === WorkflowRunResultType.tests).forEach(r => {
            const suites = r.detail.data.tests_suites;
            if (!suites.test_suites) { return; }
            this.tests.test_suites.push(...suites.test_suites);
            const stats = r.detail.data.tests_stats;
            this.tests.ko += stats.ko ?? 0;
            this.tests.ok += stats.ok ?? 0;
            this.tests.skipped += stats.skipped ?? 0;
            this.tests.total += stats.total ?? 0;
        });
    }

    onBack(): void {
        const lastFilters = this._store.selectSnapshot(NavigationState.selectActivityRunLastFilters(this.projectKey));
        if (lastFilters) {
            this._router.navigateByUrl(lastFilters);
        } else {
            this._router.navigate(['/project', this.projectKey, 'run']);
        }
    }

    selectTab(tab: Tab): void {
        this.selectedTab = tab;
    }

    panelStartResize(): void {
        this._store.dispatch(new actionPreferences.SetPanelResize({ resizing: true }));
    }

    infoPanelEndResize(size: string): void {
        this.panelEndResize();
        this._store.dispatch(new actionPreferences.SavePanelSize({
            panelKey: ProjectV2RunComponent.INFO_PANEL_KEY,
            size: size
        }));
    }

    jobPanelEndResize(size: string): void {
        this.panelEndResize();
        this._store.dispatch(new actionPreferences.SavePanelSize({
            panelKey: ProjectV2RunComponent.JOB_PANEL_KEY,
            size: size
        }));
    }

    panelEndResize(): void {
        this._store.dispatch(new actionPreferences.SetPanelResize({ resizing: false }));
    }

    async openPanel(type: string, data: string = null) {
        this.clearPanel();

        switch (type) {
            case 'hook':
                this.selectedHookName = data;
                break;
            case 'result':
                this.selectedRunResult = this.results.find(r => r.id === data);
                break;
            case 'job':
                const selectedJobRun = this.jobs.find(j => j.id === data);
                this.selectedJobRun = selectedJobRun;
                break;
            case 'test':
                this.selectedTest = data;
                break;
            case 'sources':
                this.selectedSourceKey = data;
                break;
        }

        this.selectedItemType = type;

        this.refreshShareLink(type, data);

        this._cd.markForCheck();
    }

    /** Keep the share link pointed at the source file currently opened in the panel. */
    onSourceFileChange(key: string): void {
        this.selectedSourceKey = key;
        if (this.selectedItemType === 'sources') {
            this.refreshShareLink('sources', key);
            this._cd.markForCheck();
        }
    }

    private refreshShareLink(type: string, data: string): void {
        let params = new HttpParams();
        params = params.append('panel', data === null || data === undefined ? type : `${type}:${encodeURIComponent(data)}`);
        this.selectedItemShareLink = `/project/${this.projectKey}/run/${this.workflowRun.id}?${params.toString()}`;
    }

    async refreshPanel() {
        if (!this.selectedItemType) {
            return;
        }

        switch (this.selectedItemType) {
            case 'job':
                // The run job on screen may have been retried since, in which case the list holds its
                // retry instead of it: the panel follows the job rather than closing.
                const jobToSelect = this.jobs.find(j => j.id === this.selectedJobRun.id)
                    ?? this.jobs.find(j => ProjectV2RunComponent.isSameJob(j, this.selectedJobRun) && j.retry > this.selectedJobRun.retry);
                if (jobToSelect) {
                    this.openPanel('job', jobToSelect.id);
                } else {
                    this.clearPanel();
                }
                break;
            case 'result':
                if (!this.selectedRunResult.detail.data.name) {
                    break;
                }
                const resultToSelect = this.results.find(r => r.detail.data.name && r.detail.data.name === this.selectedRunResult.detail.data.name);
                if (resultToSelect) {
                    this.openPanel('result', resultToSelect.id);
                } else {
                    this.clearPanel();
                }
                break;
        }
    }

    clearPanel(): void {
        delete this.selectedItemType;
        delete this.selectedItemShareLink;
        delete this.selectedHookName;
        delete this.selectedRunResult;
        delete this.selectedJobRun;
        delete this.selectedTest;
        delete this.selectedSourceKey;
    }

    async changeRunAttempt(value: number) {
        this.selectedRunAttempt = value;
        this._cd.markForCheck();
        await this.load(this.workflowRun.id, this.selectedRunAttempt);
    }

    clickRestartJobs(): void {
        this.graph.setSelectionModeActive(true);
    }

    clickSelectAllFailedJobs(): void {
        // Enter selection mode first if not already active
        if (!this.selectionModeActive) {
            this.clickRestartJobs();
        }

        // Reset selection
        let selectedRunJobIds = [];

        const jobsByJobId = new Map<string, { failed: V2WorkflowRunJob[], total: number }>();
        for (const job of this.jobs) {
            if (!jobsByJobId.has(job.job_id)) {
                jobsByJobId.set(job.job_id, { failed: [], total: 0 });
            }
            const entry = jobsByJobId.get(job.job_id);
            entry.total++;
            if (V2WorkflowRunJobStatusIsFailed(job.status)) {
                entry.failed.push(job);
            }
        }

        for (const [jobId, { failed, total }] of jobsByJobId) {
            if (failed.length === 0) continue;

            // Add all failed run job IDs to selection
            failed.forEach(j => {
                if (!selectedRunJobIds.includes(j.id)) {
                    selectedRunJobIds.push(j.id);
                }
            });
        }

        // Push selection to graph (handles blocking, pruning, and emitting)
        this.graph.updateSelection([...selectedRunJobIds]);

        this._cd.markForCheck();
    }

    clickSelectAllJobsWithGates(): void {
        // Enter selection mode first if not already active
        if (!this.selectionModeActive) {
            this.clickRestartJobs();
        }

        let selectedRunJobIds = [];

        // Select skipped gate jobs that have at least one succeeded parent
        const gateJobIds = this.getSkippedGateJobIds();
        for (const jobId of gateJobIds) {
            // Add all run job IDs for this job definition
            (this.jobs ?? []).filter(j => j.job_id === jobId).forEach(j => {
                if (!selectedRunJobIds.includes(j.id)) {
                    selectedRunJobIds.push(j.id);
                }
            });
            // After each selection, recompute blocked descendants so that
            // later gate jobs that are descendants of earlier ones get blocked
            this.graph.updateSelection([...selectedRunJobIds]);
        }

        this._cd.markForCheck();
    }

    /**
     * Returns the names of jobs with gates that were skipped and have at least
     * one succeeded parent (direct dependency).  Only applicable when the
     * workflow run itself succeeded.
     */
    getSkippedGateJobIds(): string[] {
        if (this.workflowRun?.status !== V2WorkflowRunStatus.Success) {
            return [];
        }

        const workflowJobs = this.workflowRun.workflow_data.workflow.jobs;
        const stages = this.workflowRun.workflow_data.workflow.stages;

        const succeededJobIds: Array<string> = [];
        const skippedJobIds: Array<string> = [];
        for (const j of this.jobs) {
            if (j.status === V2WorkflowRunJobStatus.Success) {
                if (!succeededJobIds.includes(j.job_id)) { succeededJobIds.push(j.job_id); }
            } else if (j.status === V2WorkflowRunJobStatus.Skipped) {
                if (!skippedJobIds.includes(j.job_id)) { skippedJobIds.push(j.job_id); }
            }
        }

        const result: string[] = [];

        for (const jobId of Object.keys(workflowJobs)) {
            const jobDef = workflowJobs[jobId];
            if (!jobDef?.gate || !skippedJobIds.includes(jobId)) {
                continue;
            }

            let hasSucceededParent = false;

            // 1. Direct job-level parents: jobs listed in this job's needs
            if (jobDef.needs && jobDef.needs.length > 0) {
                for (const parentName of jobDef.needs) {
                    if (succeededJobIds.includes(parentName)) {
                        hasSucceededParent = true;
                        break;
                    }
                }
            }

            // 2. Stage-level parents: if job is in a stage with needs,
            //    check if any job in a parent stage succeeded
            if (!hasSucceededParent && jobDef.stage && stages) {
                const myStage = stages[jobDef.stage];
                if (myStage?.needs && myStage.needs.length > 0) {
                    for (const parentStageName of myStage.needs) {
                        for (const otherJobId of Object.keys(workflowJobs)) {
                            const other = workflowJobs[otherJobId];
                            if (other?.stage === parentStageName && succeededJobIds.includes(otherJobId)) {
                                hasSucceededParent = true;
                                break;
                            }
                        }
                        if (hasSucceededParent) { break; }
                    }
                }
            }

            // 3. Root job (no needs, no stage needs) — always eligible if skipped
            if (!hasSucceededParent && (!jobDef.needs || jobDef.needs.length === 0)) {
                const inStageWithNeeds = jobDef.stage && stages?.[jobDef.stage]?.needs?.length > 0;
                if (!inStageWithNeeds) {
                    hasSucceededParent = true;
                }
            }

            if (hasSucceededParent) {
                result.push(jobId);
            }
        }

        return result;
    }

    clickCancelSelection(): void {
        this.graph.setSelectionModeActive(false);
    }

    async clickValidateRestartJobs(): Promise<void> {
        if (this.selectedRunJobIds.length === 0) {
            this._messageService.warning('No jobs selected for restart', { nzDuration: 2000 });
            return;
        }

        if (this.selectionRequiresGate) {
            this.openRunTriggerDrawer(this.selectedRunJobIds);
            return;
        }

        // No gates needed, restart jobs directly
        await this.triggerRestartJobs();
    }

    /** Receive the latest selection from the graph component. */
    onSelectionChange(selectedRunJobIds: Array<string>): void {
        this.selectedRunJobIds = selectedRunJobIds;
        this._cd.markForCheck();
    }

    /** Receive selection mode changes from the graph component. */
    onSelectionModeChange(active: boolean): void {
        this.selectionModeActive = active;
        if (!active) {
            this.selectedRunJobIds = [];
        }
        this._cd.markForCheck();
    }

    /** Whether the current selection includes at least one fully-selected gated job that requires drawer input. */
    get selectionRequiresGate(): boolean {
        if (!this.workflowRun?.workflow_data?.workflow) { return false; }
        const workflowJobs = this.workflowRun.workflow_data.workflow.jobs;
        const gates = this.workflowRun.workflow_data.workflow.gates;
        const selectionsByJobId = groupRunJobSelectionsByJobId(this.selectedRunJobIds, this.jobs ?? []);
        for (const [jobId] of selectionsByJobId) {
            if (!areAllJobVariantsSelected(jobId, this.selectedRunJobIds, this.jobs ?? [])) {
                continue;
            }
            const jobDef = workflowJobs[jobId];
            if (jobDef?.gate) {
                const gate = gates[jobDef.gate];
                if (gate?.inputs && Object.keys(gate.inputs).length > 0) {
                    return true;
                }
            }
        }
        return false;
    }

    /**
     * Restart selected jobs via a single batch API call.
     *
     * Groups selections by job_id using the shared utility:
     * - Fully-selected jobs → keyed by job_id, can include gate inputs.
     * - Partially-selected matrix jobs → keyed by individual run job UUID,
     *   no gate inputs (API reuses previous event inputs for UUID keys).
     */
    async triggerRestartJobs(): Promise<void> {
        this.loading.restart = true;
        this._cd.markForCheck();

        try {
            const jobInputs: { [jobIdentifier: string]: { [inputName: string]: any } } = {};
            const selectionsByJobId = groupRunJobSelectionsByJobId(this.selectedRunJobIds, this.jobs);

            for (const [jobId, selectedIds] of selectionsByJobId) {
                if (areAllJobVariantsSelected(jobId, this.selectedRunJobIds, this.jobs)) {
                    // Full selection → key by job_id
                    jobInputs[jobId] = {};
                } else {
                    // Partial selection → individual run job UUIDs
                    selectedIds.forEach(runJobId => {
                        jobInputs[runJobId] = {};
                    });
                }
            }

            await lastValueFrom(this._workflowService.triggerJobs(
                this.projectKey,
                this.workflowRun.id,
                { job_inputs: jobInputs }
            ));

            const count = this.selectedRunJobIds.length;
            this._messageService.success(
                `${count} job${count > 1 ? 's' : ''} restarted successfully`,
                { nzDuration: 2000 }
            );

            // Clear selection and reload to reflect changes
            await this.load(this.workflowRun.id);
        } catch (e) {
            this._messageService.error(`Unable to restart jobs: ${ErrorUtils.print(e)}`, { nzDuration: 4000 });
        }

        this.loading.restart = false;
        this._cd.markForCheck();
    }

    async clickStopRun() {
        this.loading.stop = true;
        this._cd.markForCheck();
        try {
            await lastValueFrom(this._workflowService.stop(this.projectKey, this.workflowRun.id));
            this._messageService.success('Workflow run will be stopped', { nzDuration: 2000 });
            await this.load(this.workflowRun.id);
        } catch (e) {
            this._messageService.error(`Unable to stop run: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
        }
        this.loading.stop = false;
        this._cd.markForCheck();
    }

    clickClosePanel(): void {
        const panelOpened = !!this.selectedItemType;
        this.clearPanel();
        this.jobPanelSize = this._store.selectSnapshot(PreferencesState.panelSize(ProjectV2RunComponent.JOB_PANEL_KEY)) ?? '50%';
        this.panelExpanded = false;

        if (this.graph) {
            this.graph.unSelect();
            if (!panelOpened) {
                // Force resize to restore the previous transformation
                this.graph.resize();
            }
        }

        this._router.navigate(['/project', this.projectKey, 'run', this.workflowRun.id], {
            queryParams: {
                panel: null
            },
            queryParamsHandling: "merge"
        });
    }

    clickExpandPanel(): void {
        if (this.panelExpanded) {
            this.jobPanelSize = this._store.selectSnapshot(PreferencesState.panelSize(ProjectV2RunComponent.JOB_PANEL_KEY)) ?? '50%';
            this.panelExpanded = false;
        } else {
            this.jobPanelSize = '90%';
            this.panelExpanded = true;
        }
        this._cd.markForCheck();
    }

    @HostListener('window:keydown.escape', ['$event'])
    handleEscapeKey(event: Event) {
        if (this.gateDrawerOpen) {
            return;
        }
        if (this.selectionModeActive) {
            this.graph.setSelectionModeActive(false);
            return;
        }
        this.clickClosePanel();
    }

    dblClickOnPanel(): void {
        this.clickExpandPanel();
    }

    generateAnnotationQueryParams(annotation: { key: string, value: string }): any {
        let queryParams = {
            'workflow': this.workflowRun.vcs_server + '/' + this.workflowRun.repository + '/' + this.workflowRun.workflow_name
        };
        queryParams[annotation.key] = annotation.value;
        return queryParams;
    }

    async restartJob(runJobId: string) {
        try {
            await lastValueFrom(this._workflowService.triggerJobs(this.projectKey, this.workflowRun.id, { job_inputs: { [runJobId]: {} } }));
            this._messageService.success('Workflow run job restarted', { nzDuration: 2000 });
            await this.load(this.workflowRun.id);
        } catch (e) {
            this._messageService.error(`Unable to restart job: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
        }
    }

    async stopJob(runJobId: string) {
        try {
            await lastValueFrom(this._workflowService.stopJob(this.projectKey, this.workflowRun.id, runJobId));
            this._messageService.success('Workflow run job stopped', { nzDuration: 2000 });
            await this.load(this.workflowRun.id);
        } catch (e) {
            this._messageService.error(`Unable to stop job: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
        }
    }

    openRunTriggerDrawer(jobRunIDs: Array<string>): void {
        this.gateDrawerOpen = true;
        const drawerRef = this._drawerService.create<RunTriggerComponent, { value: string }, boolean>({
            nzTitle: 'Trigger Workflow Run Job' + (jobRunIDs.length > 1 ? 's' : ''),
            nzContent: RunTriggerComponent,
            nzContentParams: {
                run: this.workflowRun,
                runJobs: this.jobs,
                jobRunIDs
            },
            nzSize: 'large',
            nzBodyStyle: { 'padding': '0' }
        });
        drawerRef.afterClose.subscribe(async (success) => {
            this.gateDrawerOpen = false;
            if (success) {
                await this.load(this.workflowRun.id);
            }
        });
    }

    openRunStartDrawer(): void {
        const drawerRef = this._drawerService.create<ProjectV2RunStartComponent, { value: string }, string>({
            nzTitle: 'Start new Workflow Run',
            nzContent: ProjectV2RunStartComponent,
            nzContentParams: {
                params: <ProjectV2RunStartComponentParams>{
                    workflow_repository: this.workflowRun.contexts.cds.workflow_vcs_server + '/' + this.workflowRun.contexts.cds.workflow_repository,
                    repository: this.workflowRun.contexts.git.server + '/' + this.workflowRun.contexts.git.repository,
                    workflow_ref: this.workflowRun.contexts.cds.workflow_ref,
                    ref: this.workflowRun.contexts.git.ref,
                    workflow: this.workflowRun.contexts.cds.workflow_vcs_server + '/' + this.workflowRun.contexts.cds.workflow_repository + '/' + this.workflowRun.contexts.cds.workflow
                }
            },
            nzSize: 'large',
            nzBodyStyle: { 'padding': '0' }
        });
        drawerRef.afterClose.subscribe(data => { });
    }

    confirmCopy(event: any) {
        event.stopPropagation();
        event.preventDefault();
        this._clipboard.copy(this.shareLink.nativeElement.href);
        this._messageService.success('Share link copied!');
    }

    confirmCopyAnnotationValue(event: any, value: string) {
        event.stopPropagation();
        event.preventDefault();
        this._clipboard.copy(value);
        this._messageService.success('Annotation value copied!');
    }

    /**
     * Trigger a gated job from the graph (single-job gate interaction).
     *
     * - Gate without inputs: triggers the job API directly, no drawer.
     * - Gate with inputs: opens a drawer via openGateDrawer().
     *
     * Drawer dismiss behavior: if the drawer is closed without submitting
     * (gateInputs is null), no API call is made and the graph is NOT refreshed.
     * The graph only refreshes (load()) when the user explicitly submits.
     */
    async triggerGatedJob(jobId: string) {
        const job = this.workflowRun.workflow_data.workflow.jobs[jobId];
        const currentGate = <V2JobGate>this.workflowRun.workflow_data.workflow.gates[job.gate];
        if (!currentGate.inputs) {
            try {
                await lastValueFrom(this._workflowService.triggerJobs(this.workflowRun.project_key, this.workflowRun.id, { job_inputs: { [jobId]: {} } }));
                this._messageService.success(`Job ${jobId} started`);
            } catch (e) {
                this._messageService.error(`Unable to get trigger job gate: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
            }
            await this.load(this.workflowRun.id);
            return;
        }
        this.openRunTriggerDrawer(this.jobs.filter(j => j.job_id === jobId).map(j => j.id));
    }

    onMouseEnterRun(id: string): void {
        delete this.animatedRuns[id];
        this._cd.markForCheck();
    }

    trackRunElement(index: number, run: V2WorkflowRun): any {
        return run.id;
    }

}
