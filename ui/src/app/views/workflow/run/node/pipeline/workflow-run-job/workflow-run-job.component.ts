import { HttpClient, HttpHeaders } from '@angular/common/http';
import { NgZone, OnDestroy, Output } from '@angular/core';
import { ChangeDetectionStrategy, ChangeDetectorRef, Component, EventEmitter, Input, OnChanges, OnInit } from '@angular/core';
import { Store } from '@ngxs/store';
import { CDNLogLink, PipelineStatus, SpawnInfo } from 'app/model/pipeline.model';
import { WorkflowNodeJobRun } from 'app/model/workflow.run.model';
import { WorkflowService } from 'app/service/workflow/workflow.service';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { DurationService } from 'app/shared/duration/duration.service';
import { ProjectState } from 'app/store/project.state';
import { WorkflowState } from 'app/store/workflow.state';
import * as moment from 'moment';
import { Observable, Subscription } from 'rxjs';
import { Action } from 'rxjs/internal/scheduler/Action';

export enum DisplayMode {
    ANSI = 'ansi',
    HTML = 'html',
}

export enum ScrollTarget {
    BOTTOM = 'bottom',
    TOP = 'top',
}

export class Tab {
    name: string;
}

export class Step {
    id: number
    name: string;
    lines: Array<Line>;
    open: boolean;
    firstDisplayedLineNumber: number;
    totalLinesCount: number;
    link: CDNLogLink;
    startDate: moment.Moment;
    duration: string;

    constructor(name: string) {
        this.name = name;
    }

    clickOpen(): void {
        this.open = !this.open;
    }
}

export class LinesResponse {
    totalCount: number;
    lines: Array<Line>;
}

export class Line {
    number: number;
    value: string;
    extra: Array<string>;
}

@Component({
    selector: 'app-workflow-run-job',
    templateUrl: './workflow-run-job.html',
    styleUrls: ['workflow-run-job.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowRunJobComponent implements OnInit, OnChanges, OnDestroy {
    @Input() nodeJobRun: WorkflowNodeJobRun;
    @Output() onScroll = new EventEmitter<ScrollTarget>();

    mode = DisplayMode.ANSI;
    displayModes = DisplayMode;
    tabs: Array<Tab>;
    currentTabIndex = 0;
    scrollTargets = ScrollTarget
    pollingSpawnInfoSubscription: Subscription;

    previousNodeJobRun: WorkflowNodeJobRun;
    steps: Array<Step>;

    constructor(
        private _cd: ChangeDetectorRef,
        private _store: Store,
        private _workflowService: WorkflowService,
        private _http: HttpClient,
        private _ngZone: NgZone
    ) { }

    ngOnInit(): void { }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    ngOnChanges(): void {
        if (!this.nodeJobRun) { return; }

        if (this.previousNodeJobRun) {
            let statusChanged = this.previousNodeJobRun.status !== this.nodeJobRun.status;
            let requirementsChanged = this.previousNodeJobRun.job.action.requirements?.length
                !== this.nodeJobRun.job.action.requirements?.length;
            let stepStatusChanged = this.previousNodeJobRun.job.step_status?.length !== this.nodeJobRun.job.step_status?.length;
            let shouldUpdate = statusChanged || requirementsChanged || stepStatusChanged;
            if (!shouldUpdate) {
                return;
            }
        }
        this.previousNodeJobRun = this.nodeJobRun;

        this.tabs = [{ name: 'Job' }];
        if (this.nodeJobRun.job.action.requirements) {
            this.tabs = this.tabs.concat(this.nodeJobRun.job.action.requirements
                .filter(r => r.type === 'service').map(r => <Tab>{ name: r.name }));
        }

        this.steps = [new Step('Informations')].concat(this.nodeJobRun.job.action.actions
            .filter((_, i) => this.nodeJobRun.job.step_status && this.nodeJobRun.job.step_status[i])
            .map(a => new Step(a.step_name ? a.step_name : a.name)));
        this.computeStepsDuration();

        this.stopPollingSpawnInfo();
        this.startPollingSpawnInfo(); // async
        this.loadStepLinks(); // async

        this._cd.markForCheck();
    }

    selectTab(i: number): void {
        this.currentTabIndex = i;
        this._cd.markForCheck();
    }

    clickMode(mode: DisplayMode): void {
        this.mode = mode;
        this._cd.markForCheck();
    }

    async loadStepLinks() {
        let projectKey = this._store.selectSnapshot(ProjectState.projectSnapshot).key;
        let workflowName = this._store.selectSnapshot(WorkflowState.workflowSnapshot).name;
        let nodeRunID = this._store.selectSnapshot(WorkflowState).workflowNodeRun.id;
        let nodeJobRunID = this._store.selectSnapshot(WorkflowState.getSelectedWorkflowNodeJobRun()).id;

        if (!this.nodeJobRun.job.step_status) {
            return;
        }

        for (let i = 1; i < this.steps.length; i++) {
            this.steps[i].link = await this._workflowService.getStepLink(projectKey, workflowName, nodeRunID, nodeJobRunID, i - 1)
                .toPromise();
            let result = await this._http.get(`./cdscdn${this.steps[i].link.lines_path}`, {
                params: { limit: '10' },
                observe: 'response'
            }).map(res => {
                let headers: HttpHeaders = res.headers;
                return <LinesResponse>{
                    totalCount: parseInt(headers.get('X-Total-Count'), 10),
                    lines: res.body as Array<Line>
                }
            }).toPromise();
            this.steps[i].lines = result.lines.map(l => {
                let line = new Line();
                line.number = l.number;
                line.value = l.value;
                return line;
            });
            this.steps[i].totalLinesCount = result.totalCount;
            this.steps[i].open = true;
        }

        this.computeStepFirstLineNumbers();

        this._cd.markForCheck();
    }

    clickScroll(target: ScrollTarget): void { this.onScroll.emit(target); }

    stopPollingSpawnInfo(): void {
        if (this.pollingSpawnInfoSubscription) { this.pollingSpawnInfoSubscription.unsubscribe(); }
    }

    async startPollingSpawnInfo() {
        let projectKey = this._store.selectSnapshot(ProjectState.projectSnapshot).key;
        let workflowName = this._store.selectSnapshot(WorkflowState.workflowSnapshot).name;
        let nodeRunID = this._store.selectSnapshot(WorkflowState).workflowNodeRun.id;
        let nodeJobRunID = this._store.selectSnapshot(WorkflowState.getSelectedWorkflowNodeJobRun()).id;
        let runNumber = this._store.selectSnapshot(WorkflowState).workflowNodeRun.num;

        let callback = (is: Array<SpawnInfo>) => {
            this.steps[0].lines = is.filter(i => !!i.user_message).map((info, i) => <Line>{
                number: i,
                value: `${info.user_message}\n`,
                extra: [moment(info.api_time).format('YYYY-MM-DD hh:mm:ss Z')]
            });
            this.steps[0].totalLinesCount = this.steps[0].lines.length;
            this.steps[0].open = true;
            this.computeStepFirstLineNumbers();
            this._cd.markForCheck();
        }

        if (PipelineStatus.isDone(this.nodeJobRun.status)) {
            callback(this.nodeJobRun.spawninfos);
            return;
        }

        let result = await this._workflowService.getNodeJobRunInfo(projectKey, workflowName,
            runNumber, nodeRunID, nodeJobRunID).toPromise();
        callback(result);

        this._ngZone.runOutsideAngular(() => {
            this.pollingSpawnInfoSubscription = Observable.interval(2000)
                .mergeMap(_ => this._workflowService.getNodeJobRunInfo(projectKey, workflowName,
                    runNumber, nodeRunID, nodeJobRunID)).subscribe(spawnInfos => {
                        this._ngZone.run(() => { callback(spawnInfos); });
                    });
        });
    }

    trackStepElement(index: number, element: Step) { return index; }

    trackLineElement(index: number, element: Line) { return element ? element.number : null; }

    computeStepFirstLineNumbers(): void {
        let nestFirstLineNumber = 1;
        for (let i = 0; i < this.steps.length; i++) {
            this.steps[i].firstDisplayedLineNumber = nestFirstLineNumber;
            nestFirstLineNumber += this.steps[i].totalLinesCount + 1; // add one more line for step name
        }
    }

    computeStepsDuration(): void {
        for (let i = 1; i < this.steps.length; i++) {
            if (!this.nodeJobRun.job?.step_status[i - 1]?.status) {
                continue;
            }
            let stepStatus = this.nodeJobRun.job.step_status[i - 1];
            if (PipelineStatus.neverRun(stepStatus.status) || !stepStatus.start) {
                continue;
            }
            this.steps[i].startDate = moment(stepStatus.start);
            if (stepStatus.done && stepStatus.done !== '0001-01-01T00:00:00Z') {
                this.steps[i].duration = DurationService.duration(this.steps[i].startDate.toDate(), moment(stepStatus.done).toDate());
            }
        }
    }

    formatDuration(from: moment.Moment, to?: moment.Moment): string {
        return DurationService.duration(from.toDate(), to ? to.toDate() : moment().toDate());
    }

    async clickExpandStep(index: number) {
        let step = this.steps[index];

        let result = await this._http.get(`./cdscdn${step.link.lines_path}`, {
            params: { offset: `${step.lines[step.lines.length - 1].number + 1}`, limit: '10' },
            observe: 'response'
        }).map(res => {
            let headers: HttpHeaders = res.headers;
            return <LinesResponse>{
                totalCount: parseInt(headers.get('X-Total-Count'), 10),
                lines: res.body as Array<Line>
            }
        }).toPromise();
        this.steps[index].lines = step.lines.concat(result.lines.map(l => {
            let line = new Line();
            line.number = l.number;
            line.value = l.value;
            return line;
        }));

        this._cd.markForCheck();
    }
}
