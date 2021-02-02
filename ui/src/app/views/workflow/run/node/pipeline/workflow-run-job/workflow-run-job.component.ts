import { OnDestroy, Output, ViewChild } from '@angular/core';
import { ChangeDetectionStrategy, ChangeDetectorRef, Component, EventEmitter, Input, OnInit } from '@angular/core';
import { Router } from '@angular/router';
import { Store } from '@ngxs/store';
import { CDNLine, CDNLogLink, CDNStreamFilter, PipelineStatus, SpawnInfo } from 'app/model/pipeline.model';
import { WorkflowNodeJobRun } from 'app/model/workflow.run.model';
import { WorkflowService } from 'app/service/workflow/workflow.service';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { DurationService } from 'app/shared/duration/duration.service';
import { ProjectState } from 'app/store/project.state';
import { WorkflowState } from 'app/store/workflow.state';
import * as moment from 'moment';
import { from, interval, Subject, Subscription } from 'rxjs';
import { delay, mergeMap, retryWhen } from 'rxjs/operators';
import { webSocket, WebSocketSubject } from 'rxjs/webSocket';
import { WorkflowRunJobVariableComponent } from '../variables/job.variables.component';

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

export class LogBlock {
    id: number;
    name: string;
    lines: Array<CDNLine>;
    endLines: Array<CDNLine>;
    open: boolean;
    firstDisplayedLineNumber: number;
    totalLinesCount: number;
    link: CDNLogLink;
    startDate: moment.Moment;
    duration: string;
    optional: boolean;
    disabled: boolean;
    failed: boolean;
    loading: boolean;

    constructor(name: string) {
        this.name = name;
        this.lines = [];
        this.endLines = [];
        this.firstDisplayedLineNumber = 0;
        this.totalLinesCount = 0;
    }

    clickOpen(): void {
        this.open = !this.open;
    }
}

@Component({
    selector: 'app-workflow-run-job',
    templateUrl: './workflow-run-job.html',
    styleUrls: ['workflow-run-job.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowRunJobComponent implements OnInit, OnDestroy {
    readonly initLoadLinesCount = 10;
    readonly expandLoadLinesCount = 100;
    readonly displayModes = DisplayMode;
    readonly scrollTargets = ScrollTarget

    @ViewChild('jobVariable') jobVariable: WorkflowRunJobVariableComponent;

    @Input() set nodeJobRun(data: WorkflowNodeJobRun) {
        this.subjectChannel.next(data);
    }
    get nodeJobRun(): WorkflowNodeJobRun {
        return this._nodeJobRun;
    }
    _nodeJobRun: WorkflowNodeJobRun;
    @Output() onScroll = new EventEmitter<ScrollTarget>();

    mode = DisplayMode.ANSI;
    tabs: Array<Tab>;
    currentTabIndex = 0;
    pollingSpawnInfoSubscription: Subscription;
    websocket: WebSocketSubject<any>;
    websocketSubscription: Subscription;
    previousNodeJobRun: WorkflowNodeJobRun;
    steps: Array<LogBlock>;
    services: Array<LogBlock>;

    // The following subject and subscription are used as a channel to serialize changes on polling and websocket subscription.
    subjectChannel: Subject<WorkflowNodeJobRun>;
    subscriptionChannel: Subscription;

    constructor(
        private _cd: ChangeDetectorRef,
        private _store: Store,
        private _workflowService: WorkflowService,
        private _router: Router
    ) {
        this.subjectChannel = new Subject<WorkflowNodeJobRun>();
        this.subscriptionChannel = this.subjectChannel.pipe(
            mergeMap(data => from(this.onNodeJobRunChange(data)))
        ).subscribe();
    }

    ngOnInit(): void {
        this.pollingSpawnInfoSubscription = interval(2000)
            .pipe(mergeMap(_ => from(this.loadSpawnInfo())))
            .subscribe();
    }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    async onNodeJobRunChange(data: WorkflowNodeJobRun) {
        if (!data) {
            return;
        }
        this._nodeJobRun = data;

        if (this.previousNodeJobRun && this.previousNodeJobRun.id !== this.nodeJobRun.id) {
            this.reset();
        }

        if (this.previousNodeJobRun) {
            let statusChanged = this.previousNodeJobRun.status !== this.nodeJobRun.status;
            let requirementsChanged = this.previousNodeJobRun.job.action.requirements?.length
                !== this.nodeJobRun.job.action.requirements?.length;
            let stepStatusChanged = this.previousNodeJobRun.job.step_status?.length !== this.nodeJobRun.job.step_status?.length;
            let lastStepStatusChanged = this.previousNodeJobRun.job.step_status && this.nodeJobRun.job.step_status &&
                this.previousNodeJobRun.job.step_status.length === this.nodeJobRun.job.step_status.length &&
                this.previousNodeJobRun.job.step_status.length > 0 &&
                (this.previousNodeJobRun.job.step_status[this.previousNodeJobRun.job.step_status.length - 1].status
                    !== this.nodeJobRun.job.step_status[this.nodeJobRun.job.step_status.length - 1].status);
            let parametersChanged = this.previousNodeJobRun?.parameters?.length !== this.nodeJobRun?.parameters?.length;
            let shouldUpdate = statusChanged || requirementsChanged || stepStatusChanged || parametersChanged || lastStepStatusChanged;
            if (!shouldUpdate) {
                return;
            }
            if (this.previousNodeJobRun.id !== this.nodeJobRun.id) {
                this.currentTabIndex = 0;
            }
        }
        this.previousNodeJobRun = this.nodeJobRun;

        let requirements = (this.nodeJobRun.job.action.requirements ? this.nodeJobRun.job.action.requirements : []);
        if (!this.tabs) {
            this.tabs = [{ name: 'Job' }].concat(requirements
                .filter(r => r.type === 'service').map(r => <Tab>{ name: r.name }));
        }
        if (!this.services) {
            this.services = requirements.filter(r => r.type === 'service').map(r => new LogBlock(r.name));
        }

        if (!this.steps) {
            this.steps = [new LogBlock('Informations')];
        }
        let steps = (this.nodeJobRun.job.action.actions ? this.nodeJobRun.job.action.actions : []);
        steps.forEach((a, i) => {
            if (!this.nodeJobRun.job.step_status || !this.nodeJobRun.job.step_status[i]) {
                return;
            }
            let exists = this.steps[i + 1];
            if (!exists) {
                let block = new LogBlock(a.step_name ? a.step_name : a.name);
                block.disabled = !a.enabled;
                block.optional = a.optional;
                block.loading = a.enabled;
                this.steps.push(block);
            }
            this.steps[i + 1].failed = PipelineStatus.FAIL === this.nodeJobRun.job.step_status[i].status;
        });
        this.computeStepsDuration();

        this._cd.markForCheck();

        await this.loadDataForCurrentTab();
    }

    async loadDataForCurrentTab() {
        this.stopWebsocketSubscription();

        if (this.currentTabIndex === 0) {
            if (PipelineStatus.isDone(this.nodeJobRun.status)) {
                this.setSpawnInfos(this.nodeJobRun.spawninfos);
            } else {
                await this.loadSpawnInfo();
            }
            await this.loadEndedSteps();
        } else {
            await this.loadOrListenService();
        }
    }

    reset(): void {
        this.previousNodeJobRun = null;
        this.tabs = null;
        this.steps = null;
        this.services = null;
        this.currentTabIndex = 0;
        this.stopWebsocketSubscription();
    }

    selectTab(i: number): void {
        this.currentTabIndex = i;
        this._cd.markForCheck();
        this.loadDataForCurrentTab();
    }

    clickMode(mode: DisplayMode): void {
        this.mode = mode;
        this._cd.markForCheck();
    }

    async loadEndedSteps() {
        let projectKey = this._store.selectSnapshot(ProjectState.projectSnapshot).key;
        let workflowName = this._store.selectSnapshot(WorkflowState.workflowSnapshot).name;

        if (!this.nodeJobRun.job.step_status) {
            return;
        }

        let links = await this._workflowService
            .getAllStepsLinks(projectKey, workflowName, this.nodeJobRun.workflow_node_run_id, this.nodeJobRun.id).toPromise();
        links.datas.forEach(d => {
            this.steps[d.step_order + 1].link = <CDNLogLink>{ api_ref: d.api_ref, item_type: links.item_type };
        });

        let results = await this._workflowService.getLogsLinesCount(links, this.initLoadLinesCount).toPromise();
        if (results) {
            results.forEach(r => {
                let steporder = links?.datas?.find(d => d.api_ref === r.api_ref)?.step_order + 1;
                if (!steporder) {
                    return
                }
                this.steps[steporder].totalLinesCount = r.lines_count;
                this.steps[steporder].open = false;
                this.steps[steporder].loading = false;
            })
        }

        this.computeStepFirstLineNumbers();

        if (PipelineStatus.isDone(this.nodeJobRun.status)) {
            await this.loadFirstFailedOrLastStep();
        } else {
            await this.startListenLastActiveStep();
        }

        this._cd.markForCheck();
    }

    clickScroll(target: ScrollTarget): void {
        this.onScroll.emit(target);
    }

    async loadSpawnInfo() {
        if (!this.nodeJobRun || PipelineStatus.isDone(this.nodeJobRun.status)) {
            return;
        }

        let projectKey = this._store.selectSnapshot(ProjectState.projectSnapshot).key;
        let workflowName = this._store.selectSnapshot(WorkflowState.workflowSnapshot).name;
        let runNumber = this._store.selectSnapshot(WorkflowState).workflowNodeRun.num;

        let result = await this._workflowService.getNodeJobRunInfo(projectKey, workflowName,
            runNumber, this.nodeJobRun.workflow_node_run_id, this.nodeJobRun.id).toPromise();
        this.setSpawnInfos(result);
    }

    setSpawnInfos(is: Array<SpawnInfo>): void {
        this.steps[0].lines = is.filter(i => !!i.user_message).map((info, i) => <CDNLine>{
            number: i,
            value: `${info.user_message}\n`,
            extra: [moment(info.api_time).format('YYYY-MM-DD hh:mm:ss Z')]
        });
        this.steps[0].totalLinesCount = this.steps[0].lines.length;
        this.steps[0].open = true;
        this.computeStepFirstLineNumbers();
        this._cd.markForCheck();
    }

    trackStepElement(index: number, _: LogBlock) {
        return index;
    }

    trackLineElement(index: number, element: CDNLine) {
        return element ? element.number : null;
    }

    computeStepFirstLineNumbers(): void {
        let nextFirstLineNumber = 1;
        for (let i = 0; i < this.steps.length; i++) {
            this.steps[i].firstDisplayedLineNumber = nextFirstLineNumber;
            nextFirstLineNumber += this.steps[i].totalLinesCount + 1; // add one more line for step name
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

    async clickExpandStepDown(index: number) {
        let step = this.steps[index];
        let result = await this._workflowService.getLogLines(step.link,
            { offset: `${step.lines[step.lines.length - 1].number + 1}`, limit: `${this.expandLoadLinesCount}` }
        ).toPromise()
        this.steps[index].totalLinesCount = result.totalCount;
        this.steps[index].lines = step.lines.concat(result.lines.filter(l => !step.endLines.find(line => line.number === l.number)));
        this._cd.markForCheck();
    }

    async clickExpandStepUp(index: number) {
        let step = this.steps[index];
        let result = await this._workflowService.getLogLines(step.link,
            { offset: `-${step.endLines.length + this.expandLoadLinesCount}`, limit: `${this.expandLoadLinesCount}` }
        ).toPromise();
        this.steps[index].totalLinesCount = result.totalCount;
        this.steps[index].endLines = result.lines.filter(l => !step.lines.find(line => line.number === l.number) && !step.endLines.find(line => line.number === l.number)).concat(step.endLines);
        this._cd.markForCheck();
    }

    stopWebsocketSubscription(): void {
        if (this.websocketSubscription) {
            this.websocketSubscription.unsubscribe();
        }
    }

    async loadFirstFailedOrLastStep() {
        if (this.steps.length <= 1) {
            return;
        }
        if (PipelineStatus.SUCCESS === this.nodeJobRun.status) {
            await this.clickOpen(this.steps[this.steps.length - 1]);
            return;
        }
        for (let i = 1; i < this.steps.length; i++) {
            if (this.steps[i].failed) {
                await this.clickOpen(this.steps[i]);
                return;
            }
        }
    }

    async startListenLastActiveStep() {
        // Skip if only informations step exists
        if (this.steps.length <= 1) {
            return;
        }
        let lastStepStatus = this.nodeJobRun.job.step_status[this.nodeJobRun.job.step_status.length - 1];
        let action = this.nodeJobRun.job.action.actions[this.nodeJobRun.job.step_status.length - 1];
        if (!PipelineStatus.isActive(lastStepStatus.status) || !action.enabled) {
            return;
        }

        let projectKey = this._store.selectSnapshot(ProjectState.projectSnapshot).key;
        let workflowName = this._store.selectSnapshot(WorkflowState.workflowSnapshot).name;

        let link = await this._workflowService.getStepLink(projectKey, workflowName,
            this.nodeJobRun.workflow_node_run_id, this.nodeJobRun.id, this.nodeJobRun.job.step_status.length - 1).toPromise();
        let result = await this._workflowService.getLogLines(link, { limit: `${this.initLoadLinesCount}` }).toPromise();
        this.steps[this.steps.length - 1].link = link;
        this.steps[this.steps.length - 1].lines = result.lines;
        this.steps[this.steps.length - 1].totalLinesCount = result.totalCount;
        this.steps[this.steps.length - 1].open = true;
        this.steps[this.steps.length - 1].loading = false;
        this._cd.markForCheck();

        const protocol = window.location.protocol.replace('http', 'ws');
        const host = window.location.host;
        const href = this._router['location']._baseHref;

        this.websocket = webSocket({
            url: `${protocol}//${host}${href}/cdscdn/item/stream`,
            openObserver: {
                next: value => {
                    if (value.type === 'open') {
                        this.websocket.next(<CDNStreamFilter>{
                            item_type: link.item_type,
                            api_ref: link.api_ref,
                            offset: result.totalCount > 0 ? -5 : 0
                        });
                    }
                }
            }
        });

        this.websocketSubscription = this.websocket
            .pipe(retryWhen(errors => errors.pipe(delay(2000))))
            .subscribe((l: CDNLine) => {
                if (!this.steps[this.steps.length - 1].lines.find(line => line.number === l.number)
                    && !this.steps[this.steps.length - 1].endLines.find(line => line.number === l.number)) {
                    this.steps[this.steps.length - 1].endLines.push(l);
                    this.steps[this.steps.length - 1].totalLinesCount++;
                    this._cd.markForCheck();
                }
            }, (err) => {
                console.error('Error: ', err)
            }, () => {
                console.warn('Websocket Completed');
            });
    }

    async loadOrListenService() {
        let projectKey = this._store.selectSnapshot(ProjectState.projectSnapshot).key;
        let workflowName = this._store.selectSnapshot(WorkflowState.workflowSnapshot).name;

        this.services[this.currentTabIndex - 1].link = await this._workflowService.getServiceLink(projectKey, workflowName,
            this.nodeJobRun.workflow_node_run_id, this.nodeJobRun.id, this.services[this.currentTabIndex - 1].name).toPromise();

        if (!PipelineStatus.isActive(this.nodeJobRun.status)) {
            let results = await Promise.all([
                this._workflowService.getLogLines(this.services[this.currentTabIndex - 1].link,
                    { limit: `${this.initLoadLinesCount}` }
                ).toPromise(),
                this._workflowService.getLogLines(this.services[this.currentTabIndex - 1].link,
                    { offset: `-${this.initLoadLinesCount}` }
                ).toPromise(),
            ]);
            this.services[this.currentTabIndex - 1].lines = results[0].lines;
            this.services[this.currentTabIndex - 1].endLines = results[1].lines
                .filter(l => !results[0].lines.find(line => line.number === l.number));
            this.services[this.currentTabIndex - 1].totalLinesCount = results[0].totalCount;
            this._cd.markForCheck();
            return;
        }

        let result = await this._workflowService.getLogLines(this.services[this.currentTabIndex - 1].link,
            { limit: `${this.initLoadLinesCount}` }
        ).toPromise();
        this.services[this.currentTabIndex - 1].lines = result.lines;
        this.services[this.currentTabIndex - 1].totalLinesCount = result.totalCount;
        this._cd.markForCheck();

        const protocol = window.location.protocol.replace('http', 'ws');
        const host = window.location.host;
        const href = this._router['location']._baseHref;


        this.websocket = webSocket({
            url: `${protocol}//${host}${href}/cdscdn/item/stream`,
            openObserver: {
                next: value => {
                    if (value.type === 'open') {
                        this.websocket.next(<CDNStreamFilter>{
                            item_type: this.services[this.currentTabIndex - 1].link.item_type,
                            api_ref: this.services[this.currentTabIndex - 1].link.api_ref,
                            offset: this.services[this.currentTabIndex - 1].totalLinesCount > 0 ? -5 : 0
                        });
                    }
                }
            }
        });

        this.websocketSubscription = this.websocket
            .pipe(retryWhen(errors => errors.pipe(delay(2000))))
            .subscribe((l: CDNLine) => {
                if (!this.services[this.currentTabIndex - 1].lines.find(line => line.number === l.number)
                    && !this.services[this.currentTabIndex - 1].endLines.find(line => line.number === l.number)) {
                    this.services[this.currentTabIndex - 1].endLines.push(l);
                    this.services[this.currentTabIndex - 1].totalLinesCount++;
                    this._cd.markForCheck();
                }
            }, (err) => {
                console.error('Error: ', err)
            }, () => {
                console.warn('Websocket Completed');
            });
    }

    async clickExpandServiceDown(index: number) {
        let service = this.services[index];
        let result = await this._workflowService.getLogLines(service.link, {
            offset: `${service.lines[service.lines.length - 1].number + 1}`,
            limit: `${this.expandLoadLinesCount}`
        }).toPromise();
        this.services[index].totalLinesCount = result.totalCount;
        this.services[index].lines = service.lines.concat(result.lines
            .filter(l => !service.endLines.find(line => line.number === l.number)));
        this._cd.markForCheck();
    }

    async clickExpandServiceUp(index: number) {
        let service = this.services[index];
        let result = await this._workflowService.getLogLines(service.link, {
            offset: `-${service.endLines.length + this.expandLoadLinesCount}`,
            limit: `${this.expandLoadLinesCount}`
        }).toPromise();
        this.services[index].totalLinesCount = result.totalCount;
        this.services[index].endLines = result.lines.filter(l => !service.lines.find(line => line.number === l.number) && !service.endLines.find(line => line.number === l.number)).concat(service.endLines);
        this._cd.markForCheck();
    }

    clickVariables(): void {
        if (this.jobVariable) {
            this.jobVariable.show();
        }
    }

    async clickOpen(step: LogBlock) {
        if (step?.lines?.length > 0 || step.open) {
            step.clickOpen();
            return
        }

        step.loading = true;
        let results = await Promise.all([
            this._workflowService.getLogLines(step.link, { limit: `${this.initLoadLinesCount}` }).toPromise(),
            this._workflowService.getLogLines(step.link, { offset: `-${this.initLoadLinesCount}` }).toPromise()
        ]);
        step.lines = results[0].lines;
        step.endLines = results[1].lines.filter(l => !results[0].lines.find(line => line.number === l.number));
        step.totalLinesCount = results[0].totalCount;
        step.open = true;
        step.loading = false;
        this._cd.markForCheck()
    }
}
