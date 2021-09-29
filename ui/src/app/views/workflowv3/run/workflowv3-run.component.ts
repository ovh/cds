import { HttpClient, HttpParams } from '@angular/common/http';
import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, OnInit, ViewChild } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { Store } from '@ngxs/store';
import { EventType } from 'app/model/event.model';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { Tab } from 'app/shared/tabs/tabs.component';
import { EventState } from 'app/store/event.state';
import { Observable, Subscription, timer } from 'rxjs';
import { debounce, filter, finalize } from 'rxjs/operators';
import { GraphDirection } from '../graph/workflowv3-graph.lib';
import { WorkflowV3StagesGraphComponent } from '../graph/workflowv3-stages-graph.component';
import { JobRun, WorkflowRunV3 } from '../workflowv3.model';

@Component({
    selector: 'app-workflowv3-run',
    templateUrl: './workflowv3-run.html',
    styleUrls: ['./workflowv3-run.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowV3RunComponent implements OnInit, OnDestroy {
    @ViewChild('graph') graph: WorkflowV3StagesGraphComponent;

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

    constructor(
        private _cd: ChangeDetectorRef,
        private _http: HttpClient,
        private _activatedRoute: ActivatedRoute,
        private _store: Store
    ) {
        this.tabs = [<Tab>{
            translate: 'common_problems',
            icon: 'exclamation triangle',
            key: 'problems',
            default: true
        }, <Tab>{
            translate: 'common_infos',
            icon: 'exclamation circle',
            key: 'infos'
        }, <Tab>{
            translate: 'common_results',
            icon: 'list',
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
                    && e.workflow_name === this.data.workflow.name
                    && e.workflow_run_num === this.data.number),
                debounce(() => timer(500))
            )
            .subscribe(e => {
                this.loadWorkflowRun();
            });
    }

    loadWorkflowRun(): void {
        const parentParams = this._activatedRoute.snapshot.parent.params;
        const params = this._activatedRoute.snapshot.params;
        const workflowName = parentParams['workflowName'];
        const runNumber = params['number'];

        this.loading = true;
        this._cd.markForCheck();
        this.getWorkflowRun(this.projectKey, workflowName, runNumber)
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(wr => {
                this.data = wr;

                // Create errors entries for failed jobs
                this.errors = [];
                Object.keys(wr.job_runs).forEach(k => {
                    const jrs = wr.job_runs[k];
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
                wr.infos.forEach(i => {
                    switch (i.type) {
                        case 'Info':
                            this.infos.push(i.user_message);
                            break;
                        default:
                            this.problems.push(i.user_message);
                            break;
                    }
                });
            });
    }

    selectTab(tab: Tab): void {
        this.selectedTab = tab;
    }

    panelStartResize(): void {
        this.resizing = true;
        this._cd.markForCheck();
    }

    panelEndResize(): void {
        this.resizing = false;
        this._cd.detectChanges(); // force rendering to compute graph container size
        if (this.graph) {
            this.graph.resize();
        }
    }

    getWorkflowRun(projectKey: string, workflowName: string, runNumber: number): Observable<WorkflowRunV3> {
        let params = new HttpParams();
        params = params.append('format', 'json');
        params = params.append('full', 'true');
        return this._http.get<WorkflowRunV3>(`/project/${projectKey}/workflowv3/${workflowName}/run/${runNumber}`, { params });
    }

    clickShowJobLogs(name: string): void {
        if (!this.data.job_runs[name]) {
            this.selectJobRun = null;
            this._cd.markForCheck();
            return;
        }
        this.selectJobRun = this.data.job_runs[name][0];
        this._cd.markForCheck();
    }

    closeJobPanel(): void {
        this.selectJobRun = null;
        this._cd.detectChanges(); // force rendering to compute graph container size
        if (this.graph) {
            this.graph.resize();
        }
    }
}
