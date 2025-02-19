import { AfterViewInit, ChangeDetectionStrategy, ChangeDetectorRef, Component, HostListener, OnDestroy, TemplateRef, ViewChild } from "@angular/core";
import { AutoUnsubscribe } from "app/shared/decorator/autoUnsubscribe";
import { from, interval, lastValueFrom, Subscription } from "rxjs";
import { dump } from "js-yaml";
import { V2WorkflowRunService } from "app/service/workflowv2/workflow.service";
import { PreferencesState } from "app/store/preferences.state";
import { Store } from "@ngxs/store";
import * as actionPreferences from "app/store/preferences.action";
import { Tab } from "app/shared/tabs/tabs.component";
import { Tests } from "../../../model/pipeline.model";
import { concatMap } from "rxjs/operators";
import { ActivatedRoute, Router } from "@angular/router";
import { NzMessageService } from "ng-zorro-antd/message";
import { WorkflowV2StagesGraphComponent } from "../../../../../libs/workflow-graph/src/public-api";
import { NavigationState } from "app/store/navigation.state";
import { V2WorkflowRun, V2WorkflowRunJob, V2WorkflowRunJobStatusIsFailed, V2WorkflowRunStatusIsTerminated, WorkflowRunInfo, WorkflowRunResult, WorkflowRunResultType } from "../../../../../libs/workflow-graph/src/lib/v2.workflow.run.model";
import { RouterService } from "app/service/services.module";
import { ErrorUtils } from "app/shared/error.utils";
import moment from "moment";
import { NzDrawerService } from "ng-zorro-antd/drawer";
import { ProjectV2RunStartComponent, ProjectV2RunStartComponentParams } from "../run-start/run-start.component";
import { HttpParams } from "@angular/common/http";
import { ToastService } from "app/shared/toast/ToastService";
import { Clipboard } from '@angular/cdk/clipboard';

@Component({
    selector: 'app-projectv2-run',
    templateUrl: './run.html',
    styleUrls: ['./run.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectV2RunComponent implements AfterViewInit, OnDestroy {
    @ViewChild('graph') graph: WorkflowV2StagesGraphComponent;
    @ViewChild('tabResultsTemplate') tabResultsTemplate: TemplateRef<any>;
    @ViewChild('tabTestsTemplate') tabTestsTemplate: TemplateRef<any>;
    @ViewChild('shareLink') shareLink: any;

    workflowRun: V2WorkflowRun;
    workflowRunInfo: Array<WorkflowRunInfo>;
    selectedItemType: string;
    selectedItemShareLink: string;
    selectedJobRun: V2WorkflowRunJob;
    selectedJob: string;
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

    // Subs
    paramsSub: Subscription;
    queryParamsSub: Subscription;
    sidebarSubs: Subscription;
    resizingSubscription: Subscription;
    pollSubs: Subscription;
    pollRunJobInfosSubs: Subscription;

    // Panels
    resizing: boolean;
    infoPanelSize: string;
    jobPanelSize: string;
    panelExpanded: boolean;

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
        private _messageService: NzMessageService,
        private _routerService: RouterService,
        private _drawerService: NzDrawerService,
        private _toast: ToastService,
        private _clipboard: Clipboard
    ) {
        this.paramsSub = this._route.params.subscribe(_ => {
            const params = this._routerService.getRouteSnapshotParams({}, this._router.routerState.snapshot.root);
            const workflowRunID = params['workflowRunID'];
            if (this.workflowRun && this.workflowRun.id === workflowRunID) {
                return;
            }
            this.projectKey = params['key'];
            this.load(workflowRunID).then(() => {
                const params = this._route.snapshot.queryParams;
                if (params['panel']) {
                    const splitted = params['panel'].split(':');
                    this.openPanel(splitted[0], decodeURI(splitted[1]) ?? null);
                }
            });
        });

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

    async load(workflowRunID: string, runAttempt?: number) {
        this.clearPanel();
        delete this.workflowGraph;
        if (this.pollSubs) {
            this.pollSubs.unsubscribe();
            delete this.pollSubs;
        }

        await this.loadRun(workflowRunID);
        this.selectedRunAttempt = runAttempt ?? this.workflowRun.run_attempt;

        this.workflowGraph = dump(this.workflowRun.workflow_data.workflow, { lineWidth: -1 });

        this._cd.markForCheck();

        await this.loadJobsAndResults();
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

        // Reload workflow run if we received a runjob from an unknown job
        if (this.jobs) {
            for (let i = 0; i < this.jobs.length; i++) {
                if (!this.workflowRun.workflow_data.workflow.jobs[this.jobs[i].job_id]) {
                    await this.load(this.jobs[i].workflow_run_id, this.selectedRunAttempt);
                    return;
                }
            }
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
        this.loadRun(this.workflowRun.id);
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
        this._cd.detectChanges(); // force rendering to compute graph container size
        if (this.graph) {
            this.graph.resize();
        }
    }

    async openPanel(type: string, data: string = null) {
        this.clearPanel();

        this.selectedItemType = type;

        let params = new HttpParams();
        params = params.append('panel', `${type}:${encodeURI(data)}`);
        this.selectedItemShareLink = `/project/${this.projectKey}/run/${this.workflowRun.id}?${params.toString()}`;

        switch (type) {
            case 'hook':
                this.selectedHookName = data;
                break;
            case 'gate':
                this.selectedJob = data;
                break;
            case 'result':
                this.selectedRunResult = this.results.find(r => r.id === data);
                break;
            case 'job':
                this.selectedJobRun = this.jobs.find(j => j.id === data);
                break;
            case 'test':
                this.selectedTest = data;
                break;
        }

        this._cd.detectChanges(); // force rendering to compute graph container size
        if (this.graph) {
            this.graph.resize();
        }
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
        delete this.selectedJob;
        delete this.selectedJobRun;
        delete this.selectedTest;
    }

    async changeRunAttempt(value: number) {
        this.selectedRunAttempt = value;
        this._cd.markForCheck();
        await this.load(this.workflowRun.id, this.selectedRunAttempt);
    }

    async clickRestartJobs() {
        await lastValueFrom(this._workflowService.restart(this.projectKey, this.workflowRun.id));
        this._messageService.success('Workflow run jobs restarted', { nzDuration: 2000 });
        await this.load(this.workflowRun.id);
    }

    async clickStopRun() {
        await lastValueFrom(this._workflowService.stop(this.projectKey, this.workflowRun.id));
        this._messageService.success('Workflow run stopped', { nzDuration: 2000 });
        await this.load(this.workflowRun.id);
    }

    clickClosePanel(): void {
        this.clearPanel();
        this.jobPanelSize = this._store.selectSnapshot(PreferencesState.panelSize(ProjectV2RunComponent.JOB_PANEL_KEY)) ?? '50%';
        this.panelExpanded = false;

        this._cd.detectChanges(); // force rendering to compute graph container size
        if (this.graph) {
            this.graph.unSelect();
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
        this._cd.detectChanges();
        if (this.graph) {
            this.graph.resize();
        }
    }

    @HostListener('window:keydown.escape', ['$event'])
    handleKeyDown(event: KeyboardEvent) {
        this.clickClosePanel();
    }

    dblClickOnPanel(): void {
        this.clickExpandPanel();
    }

    generateAnnotationQueryParams(annotation: { key: string, value: string }): any {
        let queryParams = {
            'workflow': this.workflowRun.vcs_server + '/' + this.workflowRun.repository + '/' + this.workflowRun.workflow_name
        };
        queryParams[annotation.key] = encodeURI(annotation.value);
        return queryParams;
    }

    async restartJob(id: string) {
        await lastValueFrom(this._workflowService.triggerJob(this.projectKey, this.workflowRun.id, id));
        this._messageService.success('Workflow run job restarted', { nzDuration: 2000 });
        await this.load(this.workflowRun.id);
    }

    async stopJob(id: string) {
        await lastValueFrom(this._workflowService.stopJob(this.projectKey, this.workflowRun.id, id));
        this._messageService.success('Workflow run job stop', { nzDuration: 2000 });
        await this.load(this.workflowRun.id);
    }

    async onGateSubmit() {
        this.clickClosePanel();
        await this.load(this.workflowRun.id);
    }

    openRunStartDrawer(): void {
        const drawerRef = this._drawerService.create<ProjectV2RunStartComponent, { value: string }, string>({
            nzTitle: 'Start new worklfow run',
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
        this._toast.success('', 'Share link copied!');
    }

	confirmCopyAnnotationValue(event: any, value: string) {
		event.stopPropagation();
		event.preventDefault();
		this._clipboard.copy(value);
		this._toast.success('', 'Annotation value copied!');
	}
}