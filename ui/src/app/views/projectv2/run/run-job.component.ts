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
import { DisplayMode, ScrollTarget } from "../../workflow/run/node/pipeline/workflow-run-job/workflow-run-job.component";
import { PipelineStatus } from "app/model/pipeline.model";
import { V2WorkflowRunService } from "app/service/workflowv2/workflow.service";
import { delay, interval, lastValueFrom, retryWhen, Subscription } from "rxjs";
import { StepStatus, V2WorkflowRun, V2WorkflowRunJob, V2WorkflowRunJobStatus, V2WorkflowRunJobStatusIsTerminated, WorkflowRunInfo } from "../../../../../libs/workflow-graph/src/lib/v2.workflow.run.model";
import { DurationService } from "../../../../../libs/workflow-graph/src/lib/duration.service";
import moment from "moment";
import { NzMessageService } from "ng-zorro-antd/message";
import { CDNLine, CDNLogLink, CDNStreamFilter } from "app/model/cdn.model";
import { CDNService } from "app/service/cdn.service";
import { webSocket, WebSocketSubject } from "rxjs/webSocket";
import { Router } from "@angular/router";
import { ErrorUtils } from "app/shared/error.utils";

export class Tab {
    name: string;
    logBlocks: Array<LogBlock>;
}
export class LogBlock {
    id: number;
    name: string;
    /** The block carrying the infos of the job, which is not a step and has nothing to show without them. */
    information: boolean;
    lines: Array<CDNLine>;
    endLines: Array<CDNLine>;
    closed: boolean;
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
    standalone: false,
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

    /** Run infos are not carried by the events of the run, they are read again at most that often. */
    static INFOS_REFRESH_DELAY = 5000;

    mode = DisplayMode.ANSI;
    tabs: Array<Tab> = [];
    currentTabIndex = 0;
    websocket: WebSocketSubject<any>;
    websocketSubscription: Subscription;
    jobRunInfos: Array<WorkflowRunInfo> = [];
    autoScrolling: boolean = true;

    jobRetries: Array<V2WorkflowRunJob>;
    selectedJobRetry: V2WorkflowRunJob;
    selectedRetry: number = 0;
    /**
     * Whether the panel moves on when the job is retried. It does while showing the last retry, and
     * stops as soon as an earlier one is picked: someone reading it is not to be moved away from it.
     */
    private followLastRetry: boolean = true;

    /** What a job that has not run yet shows in place of logs it does not have. */
    pending: {
        label: string;
        icon: string;
        since: string;
    };
    pendingSubs: Subscription;

    /**
     * The placeholder only stands as long as the panel has nothing else to show: the infos of the job,
     * or the first step to report a status, take over as soon as they land.
     */
    get showPending(): boolean {
        if (!this.pending) {
            return false;
        }
        const logBlocks = this.tabs[this.currentTabIndex]?.logBlocks ?? [];
        return !logBlocks.some(block => !block.information || block.lines.length > 0);
    }

    private infosRefreshedAt: number = 0;
    private infosRefreshTimer: any;

    constructor(
        private _cd: ChangeDetectorRef,
        private _workflowRunService: V2WorkflowRunService,
        private _messageService: NzMessageService,
        private _cdnService: CDNService,
        private _router: Router
    ) {
        this.reset();
    }

    ngOnDestroy(): void {
        if (this.websocket) { this.stopStreamingLogsForJob(); }
        this.cancelInfosRefresh();
    }

    ngOnInit(): void {
        this.selectedRetry = this.jobRun?.retry;
        this.change();
    }

    ngOnChanges(changes: SimpleChanges): void {
        this.change(changes);
    }

    reset(): void {
        const informations = new LogBlock('Information');
        informations.information = true;
        this.tabs = [{ name: 'Job', logBlocks: [informations] }];
        this.currentTabIndex = 0;
        this.cancelInfosRefresh();
        this.stopStreamingLogsForJob();
    }

    async change(changes: SimpleChanges = null) {
        const isInit = this.jobRun && !changes;
        const jobRunChanged = !!changes && !!changes.jobRun;
        const jobRunIDChanged = jobRunChanged && changes.jobRun.previousValue && changes.jobRun.previousValue.id !== changes.jobRun.currentValue.id;
        const displayedID = this.selectedJobRetry?.id;

        // The panel is handed another run job of the same job: it was retried while being read.
        const sameJobRetried = jobRunIDChanged && changes.jobRun.previousValue.job_id === changes.jobRun.currentValue.job_id;

        if (jobRunIDChanged && !sameJobRetried) {
            this.reset();
            // The retry that was selected belongs to the job being left, this one has its own.
            this.selectedRetry = this.jobRun.retry;
            this.followLastRetry = true;
        } else if (sameJobRetried && this.followLastRetry) {
            this.selectedRetry = this.jobRun.retry;
        }

        if (this.jobRun.retry > 0) {
            // The list holds every retry of the job, it only changes when it is retried again.
            if (this.jobRetries?.length !== this.jobRun.retry + 1) {
                await this.getRetry();
            }
            // The last retry is the one the run view keeps up to date, the others are done and never
            // change: taking it from the input rather than from the list is what keeps the steps of a
            // retried job moving.
            this.selectedJobRetry = this.selectedRetry === this.jobRun.retry
                ? this.jobRun
                : (this.jobRetries.find(r => r.retry === this.selectedRetry) ?? this.jobRun);
            this._cd.markForCheck();
        } else {
            this.selectedJobRetry = this.jobRun;
        }

        // Another run job is on screen: another job, or another retry of the same one. Its steps and
        // its logs are not the ones displayed, and a log block already holding lines is never read
        // again, so the blocks are dropped rather than reused.
        const displayedJobChanged = !!displayedID && displayedID !== this.selectedJobRetry?.id;
        if (displayedJobChanged) {
            this.reset();
        }

        if (isInit || jobRunIDChanged || displayedJobChanged) {
            await this.setInfos();
            this._cd.markForCheck();
            await this.setServices();
            this._cd.markForCheck();
        } else if (jobRunChanged) {
            this.scheduleInfosRefresh();
        }
        this.setPending();

        if (isInit || jobRunChanged || displayedJobChanged) {
            await this.setSteps();
            this.computeStepFirstLineNumbers();
            this._cd.markForCheck();
            await this.loadStepsLogs();
        }

        this._cd.detectChanges();
        this.autoScroll();
    }

    /**
     * A job that has not started has no log to show, and used to open on an empty panel. What it is
     * waiting for, and for how long, is shown instead.
     */
    private setPending(): void {
        if (this.pendingSubs) {
            this.pendingSubs.unsubscribe();
            this.pendingSubs = null;
        }

        const job = this.selectedJobRetry;
        if (!job || job.started) {
            this.pending = null;
            return;
        }

        this.pending = { ...RunJobComponent.pendingState(job.status), since: null };
        // How long it has been waiting only means something while it still is: a job that will never
        // run is not "queued for three days".
        if (!V2WorkflowRunJobStatusIsTerminated(job.status)) {
            this.refreshPendingSince();
            this.pendingSubs = interval(1000).subscribe(() => this.refreshPendingSince());
        }
    }

    private refreshPendingSince(): void {
        // Hidden behind what the panel has to show: nothing to keep counting.
        if (!this.showPending || !this.selectedJobRetry?.queued) {
            return;
        }
        this.pending.since = DurationService.duration(new Date(this.selectedJobRetry.queued), new Date());
        this._cd.markForCheck();
    }

    private static pendingState(status: V2WorkflowRunJobStatus): { label: string, icon: string } {
        switch (status) {
            case V2WorkflowRunJobStatus.Waiting:
                return { label: 'Waiting for a worker', icon: 'clock-circle' };
            case V2WorkflowRunJobStatus.Scheduling:
                return { label: 'A worker is starting', icon: 'loading' };
            case V2WorkflowRunJobStatus.Blocked:
                return { label: 'Waiting for its turn', icon: 'pause-circle' };
            case V2WorkflowRunJobStatus.Skipped:
                return { label: 'Skipped, it never ran', icon: 'stop' };
            case V2WorkflowRunJobStatus.Cancelled:
                return { label: 'Cancelled before it ran', icon: 'close-circle' };
            case V2WorkflowRunJobStatus.Stopped:
                return { label: 'Stopped before it ran', icon: 'close-circle' };
            default:
                return { label: 'Not started yet', icon: 'clock-circle' };
        }
    }

    async getRetry() {
        this.jobRetries = [];
         try {
            this.jobRetries = await lastValueFrom(this._workflowRunService.getRetries(this.jobRun.project_key, this.jobRun.workflow_run_id, this.jobRun.id));
        } catch (e) {
            this._messageService.error(`Unable to get run job retries: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
            return;
        }
    }

    async setServices() {
        let promises = [];
        Object.keys(this.selectedJobRetry.job.services ?? {}).forEach((serviceName, i) => {
            if (!this.tabs[i + 1]) {
                this.tabs.push({ name: serviceName, logBlocks: [new LogBlock(serviceName)] });
            }
            promises.push(lastValueFrom(this._workflowRunService.getRunJobServiceLogsLink(this.workflowRun, this.selectedJobRetry.id, serviceName)));
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

    /**
     * The job moved: its infos may have been completed. They are read again without exceeding one call
     * per interval, whatever the number of steps the job goes through in the meantime.
     */
    private scheduleInfosRefresh(): void {
        if (this.infosRefreshTimer) {
            return;
        }
        const delayMs = Math.max(0, RunJobComponent.INFOS_REFRESH_DELAY - (Date.now() - this.infosRefreshedAt));
        this.infosRefreshTimer = setTimeout(() => {
            this.infosRefreshTimer = null;
            this.refreshInfos();
        }, delayMs);
    }

    private cancelInfosRefresh(): void {
        if (this.infosRefreshTimer) {
            clearTimeout(this.infosRefreshTimer);
            this.infosRefreshTimer = null;
        }
    }

    async setInfos() {
        this.infosRefreshedAt = Date.now();

        try {
            this.jobRunInfos = await lastValueFrom(this._workflowRunService.getRunJobInfos(this.workflowRun, this.selectedJobRetry.id));
        } catch (e) {
            this._messageService.error(`Unable to get run job infos: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
            return;
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

        const steps = this.selectedJobRetry.job.steps ?? [];

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
            if (!(this.selectedJobRetry.steps_status ?? {})[stepName]) {
                break;
            }
            if (!this.tabs[0].logBlocks[blockIndex]) {
                this.tabs[0].logBlocks.push(new LogBlock(stepName));
            }
            setBlockData(blockIndex, this.selectedJobRetry.steps_status[stepName]);
            blockIndex++;
        }

        // Create blocks for post steps
        for (let i = steps.length - 1; i >= 0; i--) {
            const stepName = this.getJobStepName(steps[i].id, i)
            if ((this.selectedJobRetry.steps_status ?? {})['Post-' + stepName]) {
                if (!this.tabs[0].logBlocks[blockIndex]) {
                    this.tabs[0].logBlocks.push(new LogBlock('Post-' + stepName));
                }
                setBlockData(blockIndex, this.selectedJobRetry.steps_status['Post-' + stepName]);
                blockIndex++;
            }
        }

        // No step has run: there is no log to link and no line to count.
        if (this.tabs[0].logBlocks.length > 1) {
            const links = await lastValueFrom(this._workflowRunService.getAllLogsLinks(this.workflowRun, this.selectedJobRetry.id));
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
        }

        if (!PipelineStatus.isDone(this.selectedJobRetry.status)) {
            this.startStreamingLogsForJob();
        }

        if (PipelineStatus.isDone(this.selectedJobRetry.status)) {
            this.stopStreamingLogsForJob();
        }
    }

    async loadStepsLogs() {
        let ps = [];
        for (let i = 0; i < this.tabs[this.currentTabIndex].logBlocks.length; i++) {
            ps.push(this.loadStepLogs(this.tabs[this.currentTabIndex].logBlocks[i]));
        }
        await Promise.all(ps);
    }

    async loadStepLogs(logBlock: LogBlock) {
        if (logBlock.lines.length > 0 || !logBlock.link) {
            return;
        }
        logBlock.loading = true;
        const results = await Promise.all([
            lastValueFrom(this._cdnService.getLogLines(logBlock.link, { limit: `${this.initLoadLinesCount}` })),
            lastValueFrom(this._cdnService.getLogLines(logBlock.link, { offset: `-${this.initLoadLinesCount}` }))
        ]);
        logBlock.lines = results[0].lines;
        logBlock.endLines = results[1].lines.filter(l => !results[0].lines.find(line => line.number === l.number));
        logBlock.totalLinesCount = results[0].totalCount;
        logBlock.loading = false;
    }

    computeStepFirstLineNumbers(): void {
        let nextFirstLineNumber = 1;
        for (let i = 0; i < this.tabs[this.currentTabIndex].logBlocks.length; i++) {
            const logBlock = this.tabs[this.currentTabIndex].logBlocks[i];
            // The infos of the job are not shown when there are none: they take no line either.
            if (logBlock.information && logBlock.lines.length === 0) {
                continue;
            }
            logBlock.firstDisplayedLineNumber = nextFirstLineNumber;
            nextFirstLineNumber += logBlock.totalLinesCount + 1; // add one more line for step name
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
        const step = this.tabs[this.currentTabIndex].logBlocks.find(s => s.name === stepName);
        if (!step) {
            return;
        }
        let limit = `${this.expandLoadLinesCount}`;
        if (event.shiftKey) {
            limit = '0';
        }
        const result = await lastValueFrom(this._cdnService.getLogLines(step.link,
            { offset: `${step.lines[step.lines.length - 1].number + 1}`, limit }
        ));
        step.totalLinesCount = result.totalCount;
        step.lines = step.lines.concat(result.lines.filter(l => !step.endLines.find(line => line.number === l.number)));
        this._cd.detectChanges();
    }

    async clickExpandStepUp(stepName: string) {
        const step = this.tabs[this.currentTabIndex].logBlocks.find(s => s.name === stepName);
        if (!step) {
            return;
        }
        const result = await lastValueFrom(this._cdnService.getLogLines(step.link,
            { offset: `-${step.endLines.length + this.expandLoadLinesCount}`, limit: `${this.expandLoadLinesCount}` }
        ));
        step.totalLinesCount = result.totalCount;
        step.endLines = result.lines.filter(l => !step.lines.find(line => line.number === l.number)
            && !step.endLines.find(line => line.number === l.number)).concat(step.endLines);
        this._cd.detectChanges();
    }

    async clickOpenClose(logBlock: LogBlock) {
        if (!logBlock.closed) {
            logBlock.closed = true;
            return;
        }

        logBlock.closed = false;
        await this.loadStepLogs(logBlock);
        this._cd.detectChanges();
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
                        this.autoScroll();
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
            url: `${protocol}//${host}${href}/cdscdn/item/stream`,
            openObserver: {
                next: value => {
                    if (value.type === 'open') {
                        this.websocket.next(<CDNStreamFilter>{
                            job_run_id: this.selectedJobRetry.id
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

    onScroll(e: any): void {
        // If the scroll is nearly complete, activate auto scroll for the view.
        this.autoScrolling = Math.abs(this.scrollWrapper.nativeElement.scrollHeight - this.scrollWrapper.nativeElement.clientHeight - this.scrollWrapper.nativeElement.scrollTop) <= 50;
    }

    autoScroll(): void {
        if (this.autoScrolling) { this.clickScroll(ScrollTarget.BOTTOM); }
    }

    onRetryChange(retry: number): void {
        this.selectedRetry = retry;
        // Reading an earlier retry pins the panel to it, coming back to the last one lets it move on.
        this.followLastRetry = retry === this.jobRun.retry;
        this.change();
    }

}

