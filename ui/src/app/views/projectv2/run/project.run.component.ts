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
import { PipelineStatus } from "../../../model/pipeline.model";
import { concatMap } from "rxjs/operators";
import { ActivatedRoute, Router } from "@angular/router";
import { NzMessageService } from "ng-zorro-antd/message";
import { WorkflowV2StagesGraphComponent } from "../../../../../libs/workflow-graph/src/public-api";
import { NavigationState } from "app/store/navigation.state";
import { NsAutoHeightTableDirective } from "app/shared/directives/ns-auto-height-table.directive";

@Component({
    selector: 'app-projectv2-run',
    templateUrl: './project.run.html',
    styleUrls: ['./project.run.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectV2WorkflowRunComponent implements OnDestroy {
    @ViewChild('graph') graph: WorkflowV2StagesGraphComponent;
    @ViewChild('autoHeightDirective') autoHeightDirective: NsAutoHeightTableDirective;

    workflowRun: V2WorkflowRun;
    workflowRunInfos: Array<WorkflowRunInfo>;
    workflowRunInfosContainsProblems: boolean = false;
    selectedItemType: string;
    selectedJobRun: V2WorkflowRunJob;
    selectedJobGate: { gate: string, job: string };
    selectedJobRunInfos: Array<WorkflowRunInfo>;
    selectedHookName: string;
    selectedRunResultID: string;
    jobs: Array<V2WorkflowRunJob>;
    workflowGraph: any;

    // Subs
    sidebarSubs: Subscription;
    resizingSubscription: Subscription;
    pollSubs: Subscription;
    pollRunJobInfosSubs: Subscription;

    // Panels
    resizing: boolean;
    infoPanelSize: number;
    jobPanelSize: number;

    defaultTabs: Array<Tab>;
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
        this.jobPanelSize = this._store.selectSnapshot(PreferencesState.panelSize(ProjectV2WorkflowRunComponent.JOB_PANEL_KEY));
        this.defaultTabs = [<Tab>{
            title: 'Infos',
            key: 'infos'
        }, <Tab>{
            title: 'Results',
            key: 'results'
        }];
        this.tabs = [...this.defaultTabs];
        this.tabs[0].default = true;
    }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

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
            if (!!this.workflowRunInfos.find(i => i.level === 'warning' || i.level === 'error')) {
                this.tabs = [<Tab>{
                    title: 'Problems',
                    key: 'problems',
                    default: true
                }, ...this.defaultTabs];
            }
        } catch (e) {
            this._messageService.error(`Unable to get workflow run: ${e?.error?.error}`, { nzDuration: 2000 });
        }

        this.workflowGraph = dump(this.workflowRun.workflow_data.workflow);

        this._cd.markForCheck();

        this.loadJobs();
    }

    async loadJobs() {
        try {
            this.jobs = await lastValueFrom(this._workflowService.getJobs(this.workflowRun));
        } catch (e) {
            this._messageService.error(`Unable to get jobs: ${e?.error?.error}`, { nzDuration: 2000 });
        }

        if (!PipelineStatus.isDone(this.workflowRun.status) && !this.pollSubs) {
            this.pollSubs = interval(5000)
                .pipe(concatMap(_ => from(this.loadJobs())))
                .subscribe();
        }

        if (PipelineStatus.isDone(this.workflowRun.status) && this.pollSubs) {
            this.pollSubs.unsubscribe();
        }

        this._cd.markForCheck();
    }

    onBack(): void {
        const projectKey = this._route.snapshot.parent.params['key'];
        const lastFilters = this._store.selectSnapshot(NavigationState.selectActivityRunLastFilters(projectKey));
        if (lastFilters) {
            this._router.navigateByUrl(lastFilters);
        } else {
            this._router.navigate(['/projectv2', projectKey, 'run']);
        }
    }

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
        if (this.autoHeightDirective) {
            this.autoHeightDirective.onResize(null);
        }
    }


    async selectJob(runJobID: string) {

        try {
            this.selectedJobRunInfos = await lastValueFrom(this._workflowService.getRunJobInfos(this.workflowRun, this.selectedJobRun.id));
        } catch (e) {
            this._messageService.error(`Unable to get run job infos: ${e?.error?.error}`, { nzDuration: 2000 });
        }

        if (!PipelineStatus.isDone(this.selectedJobRun.status) && !this.pollRunJobInfosSubs) {
            this.pollRunJobInfosSubs = interval(5000)
                .pipe(concatMap(_ => from(this.selectJob(runJobID))))
                .subscribe();
        }

        if (PipelineStatus.isDone(this.selectedJobRun.status) && this.pollRunJobInfosSubs) {
            this.pollRunJobInfosSubs.unsubscribe();
        }
    }

    async openPanel(type: string, data: any) {
        this.clearPanel();

        this.selectedItemType = type;

        switch (type) {
            case 'hook':
                this.selectedHookName = data;
                break;
            case 'gate':
                this.selectedJobGate = { gate: data.gateName, job: data.gateChild };
                break;
            case 'result':
                this.selectedRunResultID = data;
                break;
            case 'job':
                this.selectedJobRun = this.jobs.find(j => j.id === data);
                await this.selectJob(data);
                break;
        }


        this._cd.markForCheck();
    }

    clearPanel(): void {
        if (this.pollRunJobInfosSubs) {
            this.pollRunJobInfosSubs.unsubscribe();
        }
        delete this.selectedItemType;
        delete this.selectedHookName;
        delete this.selectedRunResultID;
        delete this.selectedJobGate;
        delete this.selectedJobRunInfos;
        delete this.selectedJobRun;
    }

    closePanel(): void {
        this.clearPanel();

        this._cd.detectChanges(); // force rendering to compute graph container size
        if (this.graph) {
            this.graph.resize();
        }
    }

}