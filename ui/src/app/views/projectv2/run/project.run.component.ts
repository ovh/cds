import {ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, ViewChild} from "@angular/core";
import {AutoUnsubscribe} from "app/shared/decorator/autoUnsubscribe";
import {SidebarService} from "app/service/sidebar/sidebar.service";
import {forkJoin, Subscription} from "rxjs";
import {V2WorkflowRun, V2WorkflowRunJob, WorkflowRunInfo} from "app/model/v2.workflow.run.model";
import {dump} from "js-yaml";
import {V2WorkflowRunService} from "app/service/workflowv2/workflow.service";
import {PreferencesState} from "app/store/preferences.state";
import {Store} from "@ngxs/store";
import * as actionPreferences from "app/store/preferences.action";
import {Tab} from "app/shared/tabs/tabs.component";
import {ProjectV2WorkflowStagesGraphComponent} from "../vcs/repository/workflow/show/graph/stages-graph.component";


@Component({
    selector: 'app-projectv2-run',
    templateUrl: './project.run.html',
    styleUrls: ['./project.run.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectV2WorkflowRunComponent implements OnDestroy {

    @ViewChild('graph') graph: ProjectV2WorkflowStagesGraphComponent;

    selectedRun: V2WorkflowRun;
    selectedJobRun: V2WorkflowRunJob;
    selectedJobRunInfos: Array<WorkflowRunInfo>;
    jobs: Array<V2WorkflowRunJob>;
    selectedRunInfos: Array<WorkflowRunInfo>;
    workflowGraph: any;

    // Subs
    sidebarSubs: Subscription;
    resizingSubscription: Subscription;

    // Panels
    resizing: boolean;
    infoPanelSize: number;
    jobPanelSize: number;

    tabs: Array<Tab>;
    selectedTab: Tab;

    static INFO_PANEL_KEY = 'workflow-run-info';
    static JOB_PANEL_KEY = 'workflow-run-job';

    constructor(private _sidebarService: SidebarService, private _cd: ChangeDetectorRef,
                private _workflowService: V2WorkflowRunService, private _store: Store) {
        this.sidebarSubs = this._sidebarService.getRunObservable().subscribe(r => {
            if (r?.id === this.selectedRun?.id) {
                return;
            }
            delete this.selectedJobRun;
            delete this.selectedJobRunInfos;
            delete this.selectedRunInfos;
            this.selectedRun = r;
            if (r) {
                this.workflowGraph = dump(r.workflow_data.workflow);
                forkJoin([
                    this._workflowService.getJobs(r),
                    this._workflowService.getRunInfos(r)
                ]).subscribe(result => {
                    this.jobs = result[0];
                    this.selectedRunInfos = result[1];
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
            console.log('bim');
            this.graph.resize();
        }
    }

    ngOnDestroy(): void {
    }

    selectJob(jobName: string): void {
        if (this.selectedJobRun?.job_id === jobName) {
            return;
        }
        forkJoin([
            this._workflowService.getRunJob(this.selectedRun, jobName),
            this._workflowService.getRunJobInfos(this.selectedRun, jobName)
        ]).subscribe(result => {
            this.selectedJobRun = result[0];
            this.selectedJobRunInfos = result[1];
            this._cd.markForCheck();
        });
    }

    unselectJob(): void {
        delete this.selectedJobRunInfos;
        delete this.selectedJobRun;
        if (this.graph) {
            this.graph.resize();
        }
        this._cd.detectChanges(); // force rendering to compute graph container size
    }

}
