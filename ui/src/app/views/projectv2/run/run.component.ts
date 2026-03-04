import { AfterViewInit, ChangeDetectionStrategy, ChangeDetectorRef, Component, HostListener, inject, OnDestroy, TemplateRef, ViewChild } from "@angular/core";
import { AutoUnsubscribe } from "app/shared/decorator/autoUnsubscribe";
import { from, interval, lastValueFrom, Subscription } from "rxjs";
import { dump } from "js-yaml";
import { V2WorkflowRunService } from "app/service/workflowv2/workflow.service";
import { PreferencesState } from "app/store/preferences.state";
import { Store } from "@ngxs/store";
import * as actionPreferences from "app/store/preferences.action";
import { Tab } from "app/shared/tabs/tabs.component";
import { Tests } from "../../../model/pipeline.model";
import { concatMap, map } from "rxjs/operators";
import { ActivatedRoute, Router } from "@angular/router";
import { NzMessageService } from "ng-zorro-antd/message";
import { NavigationState } from "app/store/navigation.state";
import { V2Job, V2JobGate, V2WorkflowRun, V2WorkflowRunJob, V2WorkflowRunJobStatus, V2WorkflowRunJobStatusIsFailed, V2WorkflowRunStatus, V2WorkflowRunStatusIsTerminated, WorkflowRunInfo, WorkflowRunResult, WorkflowRunResultType, areAllJobVariantsSelected, groupRunJobSelectionsByJobId } from "../../../../../libs/workflow-graph/src/lib/v2.workflow.run.model";
import { RunTriggerComponent } from "./run-trigger.component";
import { RouterService } from "app/service/services.module";
import { ErrorUtils } from "app/shared/error.utils";
import moment from "moment";
import { NzDrawerService } from "ng-zorro-antd/drawer";
import { ProjectV2RunStartComponent, ProjectV2RunStartComponentParams } from "../run-start/run-start.component";
import { HttpClient, HttpHeaders, HttpParams } from "@angular/common/http";
import { Clipboard } from '@angular/cdk/clipboard';
import { GraphComponent } from "../../../../../libs/workflow-graph/src/public-api";
import { Title } from "@angular/platform-browser";
import { WebsocketV2Filter, WebsocketV2FilterType } from "app/model/websocket-v2";
import { EventV2Service } from "app/event-v2.service";
import { EventV2Type } from "app/model/event-v2.model";
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
    animatedRuns: { [key: string]: boolean } = {};
    selectionModeActive: boolean = false;
    gateDrawerOpen: boolean = false;
    /** Selected run job IDs for restart. Both simple jobs and matrix variants are tracked uniformly by their run job UUID. */
    selectedRunJobIds: Array<string> = [];

    // Subs
    paramsSub: Subscription;
    queryParamsSub: Subscription;
    resizingSubscription: Subscription;
    pollSubs: Subscription;
    eventV2Subscription: Subscription;

    // Panels
    resizing: boolean;
    infoPanelSize: string;
    jobPanelSize: string;
    panelExpanded: boolean;

    tabs: Array<Tab>;
    selectedTab: Tab;

    static INFO_PANEL_KEY = 'workflow-run-info';
    static JOB_PANEL_KEY = 'workflow-run-job';

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

    constructor() {
        this.paramsSub = this._route.params.pipe(
            concatMap(_ => {
                const params = this._routerService.getRouteSnapshotParams({}, this._router.routerState.snapshot.root);
                const workflowRunID = params['workflowRunID'];
                if (this.workflowRun && this.workflowRun.id === workflowRunID) {
                    return from([]);
                }
                this.projectKey = params['key'];

                return from(this.load(workflowRunID).then(() => {
                    const params = this._route.snapshot.queryParams;
                    if (params['panel']) {
                        const splitted = params['panel'].split(':');
                        this.openPanel(splitted[0], decodeURI(splitted[1]) ?? null);
                    }
                }));
            })
        ).subscribe(_ => { });

        this.queryParamsSub = this._route.queryParams.subscribe(params => {
            if (params['panel'] && this.workflowRun && this.jobs) {
                const splitted = params['panel'].split(':');
                this.openPanel(splitted[0], splitted[1] ?? null);
            }
        });

        this.resizingSubscription = this._store.select(PreferencesState.resizing).subscribe(resizing => {
            this.resizing = resizing;
            this._cd.markForCheck();
        });
        this.infoPanelSize = this._store.selectSnapshot(PreferencesState.panelSize(ProjectV2RunComponent.INFO_PANEL_KEY));
        this.jobPanelSize = this._store.selectSnapshot(PreferencesState.panelSize(ProjectV2RunComponent.JOB_PANEL_KEY)) ?? '50%';

        this.eventV2Subscription = this._store.select(EventV2State.last).subscribe((event) => {
            if (!event || [EventV2Type.EventRunCrafted, EventV2Type.EventRunBuilding, EventV2Type.EventRunEnded, EventV2Type.EventRunRestart].indexOf(event.type) === -1) { return; }
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

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    /**
     * Load a workflow run and all its associated data.
     * Resets all restart selection state (selected jobs, gate data, selection mode,
     * graph visuals) to prevent stale state from carrying over between runs.
     */
    async load(workflowRunID: string, runAttempt?: number) {
        this.clearPanel();
        delete this.workflowGraph;
        if (this.pollSubs) {
            this.pollSubs.unsubscribe();
            delete this.pollSubs;
        }

        if (this.graph) {
            this.graph.setSelectionModeActive(false);
        }

        await this.loadRun(workflowRunID);
        this.selectedRunAttempt = runAttempt ?? this.workflowRun.run_attempt;
        this._titleService.setTitle(`#${this.workflowRun.run_number} [${this.workflowRun.contexts.git.ref_name}] • ${this.workflowRun.vcs_server}/${this.workflowRun.repository}/${this.workflowRun.workflow_name} • Workflow Run`);

        this.workflowGraph = dump(this.workflowRun.workflow_data.workflow, { lineWidth: -1 });

        this._cd.markForCheck();

        await this.loadRuns();

        this._cd.markForCheck();

        await this.loadJobsAndResults();
    }

    async loadRuns() {
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
            const res = await lastValueFrom(this._http.get(`/v2/project/${this.projectKey}/run`, { params, observe: 'response' })
                .pipe(map(res => {
                    let headers: HttpHeaders = res.headers;
                    return {
                        totalCount: parseInt(headers.get('X-Total-Count'), 10),
                        runs: res.body as Array<V2WorkflowRun>
                    };
                })));
            this.runs = res.runs;
        } catch (e) {
            this._messageService.error(`Unable to list workflow runs: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
        }
    }

    async loadRun(workflowRunID: string) {
        try {
            this.workflowRun = await lastValueFrom(this._workflowService.getRun(this.projectKey, workflowRunID));
            this.workflowRunIsTerminated = V2WorkflowRunStatusIsTerminated(this.workflowRun.status);
            this.workflowRunIsActive = !this.workflowRunIsTerminated;
        } catch (e) {
            this._messageService.error(`Unable to get workflow run: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
        }
    }

    async loadJobsAndResults() {
        try {
            this.jobs = await lastValueFrom(this._workflowService.getJobs(this.workflowRun, this.selectedRunAttempt));
        } catch (e) {
            this._messageService.error(`Unable to get jobs: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
        }

        try {
            this.results = await lastValueFrom(this._workflowService.getResults(this.workflowRun, this.selectedRunAttempt));
            if (!!this.results.find(r => r.type === WorkflowRunResultType.tests)) {
                this.computeTestsReport();
            }
        } catch (e) {
            this._messageService.error(`Unable to get results: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
        }
        try {
            this.workflowRunInfo = await lastValueFrom(this._workflowService.getRunInfos(this.workflowRun));
            this.workflowRunInfo.sort((a, b) => moment(a.issued_at).isBefore(moment(b.issued_at)) ? 1 : -1);
        } catch (e) {
            this._messageService.error(`Unable to get run infos: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
        }

        await this.refreshPanel();

        this.hasJobsFailed = this.jobs.filter(j => V2WorkflowRunJobStatusIsFailed(j.status)).length > 0;
        this.hasSkippedGateJobs = this.getSkippedGateJobIds().length > 0;

        if (this.workflowRunIsActive && !this.pollSubs) {
            this.pollSubs = interval(5000)
                .pipe(concatMap(_ => from(this.pollReload())))
                .subscribe();
        }

        if (this.workflowRunIsTerminated && this.pollSubs) {
            this.pollSubs.unsubscribe();
            delete this.pollSubs;
        }

        this._cd.detectChanges();
    }

    async pollReload() {
        const previousJobsCount = Object.keys(this.workflowRun.workflow_data.workflow.jobs).length;
        await this.loadRun(this.workflowRun.id);

        // Force redraw of the graph if the count of jobs changed in the workflow definition
        if (previousJobsCount !== Object.keys(this.workflowRun.workflow_data.workflow.jobs).length) {
            await this.load(this.workflowRun.id, this.selectedRunAttempt);
            return;
        }

        await this.loadJobsAndResults();
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
        }

        this.selectedItemType = type;

        let params = new HttpParams();
        params = params.append('panel', `${type}:${encodeURIComponent(data)}`);
        this.selectedItemShareLink = `/project/${this.projectKey}/run/${this.workflowRun.id}?${params.toString()}`;

        this._cd.markForCheck();
    }

    async refreshPanel() {
        if (!this.selectedItemType) {
            return;
        }

        switch (this.selectedItemType) {
            case 'job':
                const jobToSelect = this.jobs.find(j => j.id === this.selectedJobRun.id);
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

        const workflowJobs = this.workflowRun.workflow_data.workflow.jobs;
        const selectionsByJobId = groupRunJobSelectionsByJobId(this.selectedRunJobIds, this.jobs);

        // Collect all jobs that require gates and their gate definitions.
        // Only fully-selected jobs can have gate inputs edited.
        // Partially-selected matrix jobs skip gate input — the API handler
        // reuses previous event inputs for UUID-keyed jobs.
        const gateJobs: { [jobId: string]: V2Job } = {};
        const gateDefinitions: { [gateName: string]: V2JobGate } = {};

        for (const [jobId] of selectionsByJobId) {
            if (!areAllJobVariantsSelected(jobId, this.selectedRunJobIds, this.jobs)) {
                continue;
            }
            const jobDef = workflowJobs[jobId];
            if (jobDef && jobDef.gate) {
                const gate = this.workflowRun.workflow_data.workflow.gates[jobDef.gate];
                if (gate && gate.inputs && Object.keys(gate.inputs).length > 0) {
                    gateJobs[jobId] = jobDef;
                    gateDefinitions[jobDef.gate] = gate;
                }
            }
        }

        // If gates are needed, open a drawer; selection is preserved so the
        // user can dismiss the drawer, adjust the selection, and retry.
        if (Object.keys(gateJobs).length > 0) {
            this.openRestartGateDrawer(gateJobs, gateDefinitions);
        } else {
            // No gates needed, restart jobs directly
            await this.triggerRestartJobs();
        }
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

            await lastValueFrom(this._workflowService.startJobs(
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
            await lastValueFrom(this._workflowService.triggerJob(this.projectKey, this.workflowRun.id, runJobId));
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

    openGateDrawer(
        jobs: { [jobId: string]: V2Job },
        gates: { [gateName: string]: V2JobGate },
        additionalJobInputs: { [jobId: string]: { [inputName: string]: any } } | null,
        onSuccess?: () => Promise<void>
    ): void {
        this.gateDrawerOpen = true;
        const drawerRef = this._drawerService.create<RunTriggerComponent, { value: string }, boolean>({
            nzTitle: 'Start Workflow Run Job' + (Object.keys(jobs).length > 1 ? 's' : ''),
            nzContent: RunTriggerComponent,
            nzContentParams: {
                run: this.workflowRun,
                jobs: jobs,
                gates: gates,
                additionalJobInputs: additionalJobInputs,
                runJobs: this.jobs
            },
            nzSize: 'large'
        });
        drawerRef.afterClose.subscribe(async (success) => {
            this.gateDrawerOpen = false;
            if (success && onSuccess) {
                await onSuccess();
            }
        });
    }

    openRestartGateDrawer(
        gateJobs: { [jobId: string]: V2Job },
        gateDefinitions: { [gateName: string]: V2JobGate }
    ): void {
        const additionalJobInputs: { [id: string]: { [inputName: string]: any } } = {};
        const selectionsByJobId = groupRunJobSelectionsByJobId(this.selectedRunJobIds, this.jobs);

        for (const [jobId, selectedIds] of selectionsByJobId) {
            if (gateJobs[jobId]) {
                // This job has gate inputs → handled by the gate form
                continue;
            }
            if (areAllJobVariantsSelected(jobId, this.selectedRunJobIds, this.jobs)) {
                additionalJobInputs[jobId] = {};
            } else {
                selectedIds.forEach(runJobId => {
                    additionalJobInputs[runJobId] = {};
                });
            }
        }

        this.openGateDrawer(
            gateJobs,
            gateDefinitions,
            additionalJobInputs,
            async () => {
                await this.load(this.workflowRun.id);
            }
        );
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
            nzSize: 'large'
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
                await lastValueFrom(this._workflowService.triggerJob(this.workflowRun.project_key, this.workflowRun.id, jobId));
                this._messageService.success(`Job ${jobId} started`);
            } catch (e) {
                this._messageService.error(`Unable to get trigger job gate: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
            }
            await this.load(this.workflowRun.id);
            return;
        }
        this.openGateDrawer(
            { [jobId]: job },
            { [job.gate]: currentGate },
            null,
            async () => {
                await this.load(this.workflowRun.id);
            }
        );
    }

    onMouseEnterRun(id: string): void {
        delete this.animatedRuns[id];
        this._cd.markForCheck();
    }

    trackRunElement(index: number, run: V2WorkflowRun): any {
        return run.id;
    }

}
