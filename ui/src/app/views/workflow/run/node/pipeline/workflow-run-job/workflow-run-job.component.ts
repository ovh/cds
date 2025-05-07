import { ElementRef, OnDestroy, Output } from '@angular/core';
import { ChangeDetectionStrategy, ChangeDetectorRef, Component, EventEmitter, Input, OnInit } from '@angular/core';
import { PipelineStatus, SpawnInfo } from 'app/model/pipeline.model';
import { WorkflowNodeJobRun } from 'app/model/workflow.run.model';
import { WorkflowService } from 'app/service/workflow/workflow.service';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import moment from 'moment';
import { from, interval, lastValueFrom, Subject, Subscription } from 'rxjs';
import { concatMap } from 'rxjs/operators';
import { WorkflowRunJobVariableComponent } from '../variables/job.variables.component';
import { NzModalService } from 'ng-zorro-antd/modal';
import { DurationService } from '../../../../../../../../libs/workflow-graph/src/lib/duration.service';
import { CDNLine, CDNLogLink } from 'app/model/cdn.model';
import { CDNService } from 'app/service/cdn.service';

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
    startDate: Date;
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

    @Input() projectKey: string;
    @Input() workflowName: string;
    @Input() workflowRunNumber: number;

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
    previousNodeJobRun: WorkflowNodeJobRun;
    steps: Array<LogBlock>;
    services: Array<LogBlock>;

    // The following subject and subscription are used as a channel to serialize changes on polling.
    subjectChannel: Subject<WorkflowNodeJobRun>;
    subscriptionChannel: Subscription;

    constructor(
        private ref: ElementRef,
        private _cd: ChangeDetectorRef,
        private _workflowService: WorkflowService,
        private _modalService: NzModalService,
        private _cdnService: CDNService
    ) {
        this.subjectChannel = new Subject<WorkflowNodeJobRun>();
        this.subscriptionChannel = this.subjectChannel.pipe(
            concatMap(data => from(this.onNodeJobRunChange(data)))
        ).subscribe();
    }

    ngOnInit(): void {
        this.pollingSpawnInfoSubscription = interval(2000)
            .pipe(concatMap(_ => from(this.loadSpawnInfo())))
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
            this.steps = [new LogBlock('Information')];
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
    }

    selectTab(i: number): void {
        this.currentTabIndex = i;
        this._cd.markForCheck();
        this.loadDataForCurrentTab().then(() => { });
    }

    clickMode(mode: DisplayMode): void {
        this.mode = mode;
        this._cd.markForCheck();
    }

    async loadEndedSteps() {
        let projectKey = this.projectKey;
        let workflowName = this.workflowName;

        if (!this.nodeJobRun.job.step_status) {
            return;
        }

        let links = await lastValueFrom(this._workflowService
            .getAllStepsLinks(projectKey, workflowName, this.nodeJobRun.workflow_node_run_id, this.nodeJobRun.id));
        links.datas.forEach((d, i) => {
            this.steps[i + 1].link = <CDNLogLink>{ api_ref: d.api_ref, item_type: d.item_type };
        });

        const results = await lastValueFrom(this._cdnService.getLogsLinesCount(links, 'step-log'));
        results.forEach(r => {
            const idx = links?.datas?.findIndex(d => d.api_ref === r.api_ref);
            this.steps[idx + 1].totalLinesCount = r.lines_count;
            this.steps[idx + 1].open = false;
            this.steps[idx + 1].loading = false;
        });

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
            if (PipelineStatus.isDone(this.nodeJobRun?.status)) {
                // cancel the interval
                this.pollingSpawnInfoSubscription.unsubscribe();
            }
            return;
        }

        let projectKey = this.projectKey;
        let workflowName = this.workflowName;
        let runNumber = this.workflowRunNumber;

        let result = await lastValueFrom(this._workflowService.getNodeJobRunInfo(projectKey, workflowName,
            runNumber, this.nodeJobRun.workflow_node_run_id, this.nodeJobRun.id));
        this.setSpawnInfos(result);

    }

    setSpawnInfos(is: Array<SpawnInfo>): void {
        this.steps[0].lines = is.filter(i => !!i.user_message).map((info, i) => <CDNLine>{
            number: i,
            value: `${info.user_message}\n`,
            extra: [moment(info.api_time).format('YYYY-MM-DD HH:mm:ss Z')]
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
            this.steps[i].startDate = new Date(stepStatus.start);
            if (stepStatus.done && stepStatus.done !== '0001-01-01T00:00:00Z') {
                this.steps[i].duration = DurationService.duration(this.steps[i].startDate, new Date(stepStatus.done));
            }
        }
    }

    async clickExpandStepDown(index: number, event: MouseEvent) {
        let step = this.steps[index];

        let limit = `${this.expandLoadLinesCount}`;
        if (event.shiftKey) {
            limit = '0';
        }

        let result = await lastValueFrom(this._cdnService.getLogLines(step.link,
            { offset: `${step.lines[step.lines.length - 1].number + 1}`, limit: limit }
        ));
        this.steps[index].totalLinesCount = result.totalCount;
        this.steps[index].lines = step.lines.concat(result.lines.filter(l => !step.endLines.find(line => line.number === l.number)));
        this._cd.markForCheck();
    }

    async clickExpandStepUp(index: number, event: MouseEvent) {
        let step = this.steps[index];

        let offset = `-${step.endLines.length + this.expandLoadLinesCount}`;
        let limit = `${this.expandLoadLinesCount}`;
        if (event.shiftKey) {
            offset = `${step.lines[step.lines.length - 1].number + 1}`;
            limit = '0';
        }

        let result = await lastValueFrom(this._cdnService.getLogLines(step.link,
            { offset: offset, limit: limit }
        ));
        this.steps[index].totalLinesCount = result.totalCount;
        this.steps[index].endLines = result.lines.filter(l => !step.lines.find(line => line.number === l.number)
            && !step.endLines.find(line => line.number === l.number)).concat(step.endLines);
        this._cd.markForCheck();
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

    receiveLogs(l: CDNLine): void {
        if (this.steps) {
            this.steps.forEach(v => {
                if (v?.link?.api_ref === l.api_ref_hash) {
                    if (!v.lines.find(line => line.number === l.number)
                        && !v.endLines.find(line => line.number === l.number)) {
                        v.endLines.push(l);
                        v.totalLinesCount++;
                    }
                }
            });
        }
        if (this.services) {
            this.services.forEach(v => {
                if (v?.link?.api_ref === l.api_ref_hash) {
                    if (!v.lines.find(line => line.number === l.number)
                        && !v.endLines.find(line => line.number === l.number)) {
                        v.endLines.push(l);
                        v.totalLinesCount++;
                        this._cd.markForCheck();
                    }
                }
            });
        }
        this._cd.markForCheck();
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

        let projectKey = this.projectKey;
        let workflowName = this.workflowName;

        let link = await lastValueFrom(this._workflowService.getStepLink(projectKey, workflowName,
            this.nodeJobRun.workflow_node_run_id, this.nodeJobRun.id, this.nodeJobRun.job.step_status.length - 1))
        let result = await lastValueFrom(this._cdnService.getLogLines(link, { limit: `${this.initLoadLinesCount}` }));
        this.steps[this.steps.length - 1].link = link;

        // Websocket may have already sent endlines
        if (result.lines) {
            this.steps[this.steps.length - 1].lines = [];
            for (let i = 0; i < result.lines.length; i++) {
                let lineFound = this.steps[this.steps.length - 1].endLines.find(line => line.number === result.lines[i].number);
                if (lineFound) {
                    break;
                }
                this.steps[this.steps.length - 1].lines.push(result.lines[i]);
            }
        }
        this.steps[this.steps.length - 1].totalLinesCount = result.totalCount;
        this.steps[this.steps.length - 1].open = true;
        this.steps[this.steps.length - 1].loading = false;
        this._cd.markForCheck();
    }

    async loadOrListenService() {
        let projectKey = this.projectKey;
        let workflowName = this.workflowName;

        this.services[this.currentTabIndex - 1].link = await lastValueFrom(this._workflowService.getServiceLink(projectKey, workflowName,
            this.nodeJobRun.workflow_node_run_id, this.nodeJobRun.id, this.services[this.currentTabIndex - 1].name));

        if (!PipelineStatus.isActive(this.nodeJobRun.status)) {
            let results = await Promise.all([
                lastValueFrom(this._cdnService.getLogLines(this.services[this.currentTabIndex - 1].link,
                    { limit: `${this.initLoadLinesCount}` }
                )),
                lastValueFrom(this._cdnService.getLogLines(this.services[this.currentTabIndex - 1].link,
                    { offset: `-${this.initLoadLinesCount}` }
                )),
            ]);
            this.services[this.currentTabIndex - 1].lines = results[0].lines;
            this.services[this.currentTabIndex - 1].endLines = results[1].lines
                .filter(l => !results[0].lines.find(line => line.number === l.number));
            this.services[this.currentTabIndex - 1].totalLinesCount = results[0].totalCount;
            this._cd.markForCheck();
            return;
        }

        let result = await lastValueFrom(this._cdnService.getLogLines(this.services[this.currentTabIndex - 1].link,
            { limit: `${this.initLoadLinesCount}` }
        ));
        this.services[this.currentTabIndex - 1].lines = result.lines;
        this.services[this.currentTabIndex - 1].totalLinesCount = result.totalCount;
        this._cd.markForCheck();
    }

    async clickExpandServiceDown(index: number) {
        let service = this.services[index];
        let result = await lastValueFrom(this._cdnService.getLogLines(service.link, {
            offset: `${service.lines[service.lines.length - 1].number + 1}`,
            limit: `${this.expandLoadLinesCount}`
        }));
        this.services[index].totalLinesCount = result.totalCount;
        this.services[index].lines = service.lines.concat(result.lines
            .filter(l => !service.endLines.find(line => line.number === l.number)));
        this._cd.markForCheck();
    }

    async clickExpandServiceUp(index: number) {
        let service = this.services[index];
        let result = await lastValueFrom(this._cdnService.getLogLines(service.link, {
            offset: `-${service.endLines.length + this.expandLoadLinesCount}`,
            limit: `${this.expandLoadLinesCount}`
        }));
        this.services[index].totalLinesCount = result.totalCount;
        this.services[index].endLines = result.lines.filter(l => !service.lines.find(line => line.number === l.number)
            && !service.endLines.find(line => line.number === l.number)).concat(service.endLines);
        this._cd.markForCheck();
    }

    clickVariables(): void {
        this._modalService.create({
            nzWidth: '900px',
            nzTitle: 'Job variables',
            nzContent: WorkflowRunJobVariableComponent,
            nzData: {
                variables: this._nodeJobRun?.parameters
            }
        })
    }

    async clickOpen(step: LogBlock) {
        if (step?.lines?.length > 0 || step.open) {
            step.open = !step.open;
            return;
        }

        step.loading = true;
        let results = await Promise.all([
            lastValueFrom(this._cdnService.getLogLines(step.link, { limit: `${this.initLoadLinesCount}` })),
            lastValueFrom(this._cdnService.getLogLines(step.link, { offset: `-${this.initLoadLinesCount}` }))
        ]);
        step.lines = results[0].lines;
        step.endLines = results[1].lines.filter(l => !results[0].lines.find(line => line.number === l.number));
        step.totalLinesCount = results[0].totalCount;
        step.open = true;
        step.loading = false;
        this._cd.markForCheck();
    }

    onJobScroll(target: ScrollTarget) {
        this.ref.nativeElement.children[0].scrollTop = target === ScrollTarget.TOP ?
            0 : this.ref.nativeElement.children[0].scrollHeight;
    }
}
