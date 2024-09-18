import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    ElementRef,
    Input,
    OnDestroy,
    OnChanges,
    ViewChild,
    SimpleChanges,
    OnInit
} from "@angular/core";
import { AutoUnsubscribe } from "app/shared/decorator/autoUnsubscribe";
import { DisplayMode, LogBlock, ScrollTarget } from "../../workflow/run/node/pipeline/workflow-run-job/workflow-run-job.component";
import { PipelineStatus } from "app/model/pipeline.model";
import { V2WorkflowRunService } from "app/service/workflowv2/workflow.service";
import { concatMap, delay, from, interval, lastValueFrom, retryWhen, Subscription } from "rxjs";
import { StepStatus, V2WorkflowRun, V2WorkflowRunJob, V2WorkflowRunJobStatusIsTerminated, WorkflowRunInfo } from "../../../../../libs/workflow-graph/src/lib/v2.workflow.run.model";
import { DurationService } from "../../../../../libs/workflow-graph/src/lib/duration.service";
import moment from "moment";
import { NzMessageService } from "ng-zorro-antd/message";
import { CDNLine, CDNStreamFilter } from "app/model/cdn.model";
import { CDNService } from "app/service/cdn.service";
import { webSocket, WebSocketSubject } from "rxjs/webSocket";
import { Router } from "@angular/router";

export class Tab {
    name: string;
    logBlocks: Array<LogBlock>;
}

@Component({
    selector: 'app-run-job',
    templateUrl: './run-job.html',
    styleUrls: ['./run-job.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class RunJobComponent implements OnInit, OnChanges, OnDestroy {
    @ViewChild('scrollWrapper') scrollWrapper: ElementRef;

    readonly initLoadLinesCount = 10;
    readonly expandLoadLinesCount = 100;
    readonly scrollTargets = ScrollTarget;
    readonly displayModes = DisplayMode;

    @Input() workflowRun: V2WorkflowRun
    @Input() jobRun: V2WorkflowRunJob;

    mode = DisplayMode.ANSI;
    tabs: Array<Tab> = [{ name: 'Job', logBlocks: [new LogBlock('Information')] }];
    currentTabIndex = 0;
    pollRunJobInfosSubs: Subscription;
    websocket: WebSocketSubject<any>;
    websocketSubscription: Subscription;
    jobRunInfos: Array<WorkflowRunInfo> = [];

    constructor(
        private _cd: ChangeDetectorRef,
        private _workflowRunService: V2WorkflowRunService,
        private _messageService: NzMessageService,
        private _cdnService: CDNService,
        private _router: Router
    ) { }

    ngOnDestroy(): void {
        // Should be set to use @AutoUnsubscribe with AOT
        if (this.websocket) { this.stopStreamingLogsForJob(); }
    }

    ngOnInit(): void {
        this.change();
    }

    ngOnChanges(changes: SimpleChanges): void {
        this.change(changes);
    }

    async change(changes: SimpleChanges = null) {
        const isInit = this.jobRun && !changes;
        const jobRunChanged = changes && changes.jobRun;
        const jobRunIDChanged = jobRunChanged && changes.jobRun.previousValue && changes.jobRun.previousValue.id !== changes.jobRun.currentValue.id;
        if (jobRunIDChanged) {
            // Reset view
            this.tabs = [{ name: 'Job', logBlocks: [new LogBlock('Information')] }];
            if (this.pollRunJobInfosSubs) { this.pollRunJobInfosSubs.unsubscribe(); }
            this.stopStreamingLogsForJob();
        }
        if (isInit || jobRunIDChanged) {
            this.currentTabIndex = 0;
            await this.setInfos();
            this._cd.markForCheck();
            await this.setServices();
            this._cd.markForCheck();
        }
        if (isInit || jobRunChanged) {
            await this.setSteps();
            this.computeStepFirstLineNumbers();
            this._cd.markForCheck();
            await this.loadStepsLogs();
        }
        if (isInit) {
            this._cd.detectChanges();
            this.clickScroll(ScrollTarget.BOTTOM);
        } else {
            this._cd.markForCheck();
        }
    }

    async setServices() {
        let promises = [];
        Object.keys(this.jobRun.job.services ?? {}).forEach((serviceName, i) => {
            if (!this.tabs[i + 1]) {
                this.tabs.push({ name: serviceName, logBlocks: [new LogBlock(serviceName)] });
            }
            promises.push(lastValueFrom(this._workflowRunService.getRunJobServiceLogsLink(this.workflowRun, this.jobRun.id, serviceName)));
        });
        const res = await Promise.all(promises);
        res.forEach((link, i) => {
            this.tabs[i + 1].logBlocks[0].link = link;
        });
    }

    getJobStepName(stepID: string, stepIndex: number): string {
        if (stepID) {
            return stepID;
        }
        return `step-${stepIndex}`;
    }

    async refreshInfos() {
        await this.setInfos();
        this.computeStepFirstLineNumbers();
        this._cd.markForCheck();
    }

    async setInfos() {
        try {
            this.jobRunInfos = await lastValueFrom(this._workflowRunService.getRunJobInfos(this.workflowRun, this.jobRun.id));
        } catch (e) {
            this._messageService.error(`Unable to get run job infos: ${e?.error?.error}`, { nzDuration: 2000 });
            return;
        }

        if (!V2WorkflowRunJobStatusIsTerminated(this.jobRun.status) && !this.pollRunJobInfosSubs) {
            this.pollRunJobInfosSubs = interval(5000)
                .pipe(concatMap(_ => from(this.refreshInfos())))
                .subscribe();
        }

        if (V2WorkflowRunJobStatusIsTerminated(this.jobRun.status) && this.pollRunJobInfosSubs) {
            this.pollRunJobInfosSubs.unsubscribe();
        }

        this.tabs[0].logBlocks[0].lines = (this.jobRunInfos ?? [])
            .sort((a, b) => moment(a.issued_at).isBefore(moment(b.issued_at)) ? -1 : 1)
            .map((info, i) => <CDNLine>{
                number: i,
                value: `${info.message}\n`,
                extra: [moment(info.issued_at).format('YYYY-MM-DD HH:mm:ss Z')]
            });
        this.tabs[0].logBlocks[0].totalLinesCount = this.tabs[0].logBlocks[0].lines.length;
    }

    async setSteps() {
        let blockIndex = 1;

        const steps = this.jobRun.job.steps ?? [];

        const setBlockData = (idx: number, stepStatus: StepStatus): void => {
            this.tabs[0].logBlocks[blockIndex].failed = PipelineStatus.FAIL === stepStatus.conclusion;
            this.tabs[0].logBlocks[blockIndex].optional = stepStatus.conclusion === PipelineStatus.SUCCESS && stepStatus.conclusion !== stepStatus.outcome;
            this.tabs[0].logBlocks[blockIndex].startDate = new Date(stepStatus.started);
            if (stepStatus.ended && stepStatus.ended !== '0001-01-01T00:00:00Z') {
                this.tabs[0].logBlocks[blockIndex].duration = DurationService.duration(this.tabs[0].logBlocks[blockIndex].startDate, new Date(stepStatus.ended));
            }
        };

        // Create blocks for steps
        for (let i = 0; i < steps.length; i++) {
            const stepName = this.getJobStepName(steps[i].id, i);
            if (!(this.jobRun.steps_status ?? {})[stepName]) {
                break;
            }
            if (!this.tabs[0].logBlocks[blockIndex]) {
                this.tabs[0].logBlocks.push(new LogBlock(stepName));
            }
            setBlockData(blockIndex, this.jobRun.steps_status[stepName]);
            blockIndex++;
        }

        // Create blocks for post steps
        for (let i = steps.length - 1; i >= 0; i--) {
            const stepName = this.getJobStepName(steps[i].id, i)
            if ((this.jobRun.steps_status ?? {})['Post-' + stepName]) {
                if (!this.tabs[0].logBlocks[blockIndex]) {
                    this.tabs[0].logBlocks.push(new LogBlock('Post-' + stepName));
                }
                setBlockData(blockIndex, this.jobRun.steps_status[stepName]);
                blockIndex++;
            }
        }

        const links = await lastValueFrom(this._workflowRunService.getAllLogsLinks(this.workflowRun, this.jobRun.id));
        links.datas.forEach((link, i) => {
            if (this.tabs[0].logBlocks[i + 1]) {
                this.tabs[0].logBlocks[i + 1].link = link;
            }
        });

        const results = await lastValueFrom(this._cdnService.getLogsLinesCount(links, 'job-step-log'));
        results.forEach(r => {
            const idx = links?.datas?.findIndex(d => d.api_ref === r.api_ref);
            this.tabs[0].logBlocks[idx + 1].totalLinesCount = r.lines_count;
        });

        if (!PipelineStatus.isDone(this.jobRun.status)) {
            this.startStreamingLogsForJob();
        }

        if (PipelineStatus.isDone(this.jobRun.status)) {
            this.stopStreamingLogsForJob();
        }
    }

    async loadStepsLogs() {
        let ps = [];
        for (let i = 0; i < this.tabs[this.currentTabIndex].logBlocks.length; i++) {
            ps.push(this.open(this.tabs[this.currentTabIndex].logBlocks[i]));
        }
        await Promise.all(ps);
    }

    computeStepFirstLineNumbers(): void {
        let nextFirstLineNumber = 1;
        for (let i = 0; i < this.tabs[this.currentTabIndex].logBlocks.length; i++) {
            this.tabs[this.currentTabIndex].logBlocks[i].firstDisplayedLineNumber = nextFirstLineNumber;
            nextFirstLineNumber += this.tabs[this.currentTabIndex].logBlocks[i].totalLinesCount + 1; // add one more line for step name
        }
    }

    trackStepElement(index: number, block: LogBlock): any {
        return index;
    }

    trackLineElement(index: number, element: CDNLine): any {
        return element.number;
    }

    clickScroll(target: ScrollTarget): void {
        this.scrollWrapper.nativeElement.scrollTop = target === ScrollTarget.TOP ?
            0 : this.scrollWrapper.nativeElement.scrollHeight;
    }

    async clickExpandStepDown(stepName: string, event: MouseEvent) {
        let step = this.tabs[this.currentTabIndex].logBlocks.find(s => s.name === stepName);
        if (!step) {
            return;
        }
        let limit = `${this.expandLoadLinesCount}`;
        if (event.shiftKey) {
            limit = '0';
        }
        let result = await lastValueFrom(this._cdnService.getLogLines(step.link,
            { offset: `${step.lines[step.lines.length - 1].number + 1}`, limit }
        ));
        step.totalLinesCount = result.totalCount;
        step.lines = step.lines.concat(result.lines.filter(l => !step.endLines.find(line => line.number === l.number)));
        this._cd.detectChanges();
    }

    async clickExpandStepUp(stepName: string) {
        let step = this.tabs[this.currentTabIndex].logBlocks.find(s => s.name === stepName);
        if (!step) {
            return;
        }
        let result = await lastValueFrom(this._cdnService.getLogLines(step.link,
            { offset: `-${step.endLines.length + this.expandLoadLinesCount}`, limit: `${this.expandLoadLinesCount}` }
        ));
        step.totalLinesCount = result.totalCount;
        step.endLines = result.lines.filter(l => !step.lines.find(line => line.number === l.number)
            && !step.endLines.find(line => line.number === l.number)).concat(step.endLines);
        this._cd.detectChanges();
    }

    async clickOpen(logBlock: LogBlock) {
        if (logBlock.open) {
            logBlock.open = !logBlock.open;
            return;
        }

        await this.open(logBlock);
        this._cd.detectChanges();
    }

    async open(logBlock: LogBlock) {
        if (logBlock.lines.length > 0 || !logBlock.link) {
            logBlock.open = true;
            return;
        }
        logBlock.loading = true;
        let results = await Promise.all([
            lastValueFrom(this._cdnService.getLogLines(logBlock.link, { limit: `${this.initLoadLinesCount}` })),
            lastValueFrom(this._cdnService.getLogLines(logBlock.link, { offset: `-${this.initLoadLinesCount}` }))
        ]);
        logBlock.lines = results[0].lines;
        logBlock.endLines = results[1].lines.filter(l => !results[0].lines.find(line => line.number === l.number));
        logBlock.totalLinesCount = results[0].totalCount;
        logBlock.open = true;
        logBlock.loading = false;
    }

    receiveLogs(l: CDNLine): void {
        for (let i = 0; i < this.tabs.length; i++) {
            for (let j = 0; j < this.tabs[i].logBlocks.length; j++) {
                if (this.tabs[i].logBlocks[j].link?.api_ref === l.api_ref_hash) {
                    if (!this.tabs[i].logBlocks[j].lines.find(line => line.number === l.number)
                        && !this.tabs[i].logBlocks[j].endLines.find(line => line.number === l.number)) {
                        this.tabs[i].logBlocks[j].endLines.push(l);
                        this.tabs[i].logBlocks[j].totalLinesCount++;
                        this._cd.detectChanges();
                    }
                    return;
                }
            }
        }
    }

    startStreamingLogsForJob() {
        if (this.websocket) {
            return;
        }

        const protocol = window.location.protocol.replace('http', 'ws');
        const host = window.location.host;
        const href = this._router['location']._basePath;
        this.websocket = webSocket({
            url: `${protocol}//${host}${href}/cdscdn/v2/item/stream`,
            openObserver: {
                next: value => {
                    if (value.type === 'open') {
                        this.websocket.next(<CDNStreamFilter>{
                            job_run_id: this.jobRun.id
                        });
                    }
                }
            }
        });

        this.websocketSubscription = this.websocket
            .pipe(retryWhen(errors => errors.pipe(delay(2000))))
            .subscribe((l: CDNLine) => {
                this.receiveLogs(l);
            }, (err) => {
                console.error('Error: ', err);
            }, () => {
                console.warn('Websocket Completed');
            });
    }

    stopStreamingLogsForJob(): void {
        if (this.websocketSubscription) { this.websocketSubscription.unsubscribe(); }
        if (this.websocket) { this.websocket.unsubscribe(); this.websocket = null; }
    }

    clickMode(mode: DisplayMode): void {
        this.mode = mode;
        this._cd.markForCheck();
    }

    async selectTab(i: number) {
        this.currentTabIndex = i;
        this.computeStepFirstLineNumbers();
        await this.loadStepsLogs();
        this._cd.detectChanges();
        this.clickScroll(ScrollTarget.BOTTOM);
    }

}

