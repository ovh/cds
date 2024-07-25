import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    ElementRef,
    Input,
    OnDestroy,
    ViewChild
} from "@angular/core";
import { AutoUnsubscribe } from "app/shared/decorator/autoUnsubscribe";
import { LogBlock, ScrollTarget } from "../../workflow/run/node/pipeline/workflow-run-job/workflow-run-job.component";
import { CDNLine, CDNLogLinkData, CDNLogLinks, PipelineStatus } from "app/model/pipeline.model";
import { V2WorkflowRunService } from "app/service/workflowv2/workflow.service";
import { WorkflowService } from "app/service/workflow/workflow.service";
import { lastValueFrom } from "rxjs";
import { V2WorkflowRun, V2WorkflowRunJob } from "../../../../../libs/workflow-graph/src/lib/v2.workflow.run.model";
import { DurationService } from "../../../../../libs/workflow-graph/src/lib/duration.service";

@Component({
    selector: 'app-run-job-logs',
    templateUrl: './run-job-logs.html',
    styleUrls: ['./run-job-logs.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class RunJobLogsComponent implements OnDestroy {
    @ViewChild('scrollWrapper') scrollWrapper: ElementRef;

    readonly initLoadLinesCount = 10;
    readonly expandLoadLinesCount = 100;
    readonly scrollTargets = ScrollTarget;

    @Input() workflowRun: V2WorkflowRun

    _runJob: V2WorkflowRunJob;
    @Input() set runJob(data: V2WorkflowRunJob) {
        this.changeRunJob(data);
    }

    get runJob(): V2WorkflowRunJob {
        return this._runJob;
    }

    logBlocks: Array<LogBlock>;
    currentTabIndex = 0;

    constructor(
        private _cd: ChangeDetectorRef,
        private _workflowRunService: V2WorkflowRunService,
        private _workflowService: WorkflowService,
    ) { }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    async changeRunJob(data: V2WorkflowRunJob) {
        if (this._runJob && this._runJob.id === data.id) {
            return;
        }
        this.logBlocks = new Array<LogBlock>();
        this._runJob = data;

        if (this._runJob.job['services']) {
            for (const serviceName in this._runJob.job['services']) {
                let block = new LogBlock('service ' + serviceName);
                this.logBlocks.push(block);
            }
        }

        if (this._runJob.job['steps']) {
            this._runJob.job['steps'].forEach((v, index) => {
                if (!v['id']) {
                    v['id'] = 'step-' + index;
                }
                if (this._runJob.steps_status && this._runJob.steps_status[v['id']]) {
                    let block = new LogBlock(v['id']);
                    block.failed = PipelineStatus.FAIL === this._runJob.steps_status[v['id']].conclusion
                    if (this._runJob.steps_status[v['id']].conclusion === PipelineStatus.SUCCESS && this._runJob.steps_status[v['id']].conclusion !== this._runJob.steps_status[v['id']].outcome) {
                        block.optional = true
                    }
                    this.logBlocks.push(block);
                }
            });
        }

        this.computeStepsDuration();

        this._cd.markForCheck();

        await this.loadEndedSteps();
    }

    async loadEndedSteps() {
        let types = new Map<string, Array<CDNLogLinkData>>();
        let linksServices = new Array<CDNLogLinkData>();
        let linksSteps = new Array<CDNLogLinkData>();
        let links = await lastValueFrom(this._workflowRunService.getAllLogsLinks(this.workflowRun, this.runJob.id));
        links.datas.forEach(link => {
            let logBlockStep = this.logBlocks.find(s => s.name === link.step_name)
            if (logBlockStep) {
                logBlockStep.link = <CDNLogLinkData>{ api_ref: link.api_ref, item_type: link.item_type };
                linksSteps.push(link);
                types.set(link.item_type, linksSteps);
            }
            let logBlockService = this.logBlocks.find(s => s.name === 'service ' + link.service_name)
            if (logBlockService) {
                logBlockService.link = <CDNLogLinkData>{ api_ref: link.api_ref, item_type: link.item_type };
                linksServices.push(link);
                types.set(link.item_type, linksServices);
            }
        });

        if (links?.datas?.length > 0) {
            for (let type of Array.from(types.entries())) {
                let itemType = type[0];
                let itemLinks = type[1];
                let links = new CDNLogLinks();
                links.item_type = itemType;
                links.datas = itemLinks;
                let results = await lastValueFrom(this._workflowService.getLogsLinesCount(links, this.initLoadLinesCount, itemType))
                if (results) {
                    results.forEach(r => {
                        let logBlock = this.logBlocks.find(s => s.link.api_ref === r.api_ref);
                        if (logBlock) {
                            logBlock.totalLinesCount = r.lines_count;
                            logBlock.open = false;
                            logBlock.loading = false;
                        }
                    });
                }
            };
        }

        this.computeStepFirstLineNumbers();
        this._cd.markForCheck();
        await this.loadStepsLogs();
        this._cd.detectChanges();
        this.clickScroll(ScrollTarget.BOTTOM);
    }

    async loadStepsLogs() {
        let ps = [];
        for (let i = 0; i < this.logBlocks.length; i++) {
            ps.push(this.open(this.logBlocks[i]));
        }
        await Promise.all(ps);
    }

    computeStepsDuration(): void {
        if (this.logBlocks) {
            this.logBlocks.forEach(s => {
                if (this._runJob.steps_status && this._runJob.steps_status[s.name]) {
                    let stepStatus = this._runJob.steps_status[s.name];
                    if (!stepStatus) {
                        return;
                    }
                    s.startDate = new Date(stepStatus.started);
                    if (stepStatus.ended && stepStatus.ended !== '0001-01-01T00:00:00Z') {
                        s.duration = DurationService.duration(s.startDate, new Date(stepStatus.ended));
                    }
                }
            });
        }
    }

    computeStepFirstLineNumbers(): void {
        let nextFirstLineNumber = 1;
        for (let i = 0; i < this.logBlocks.length; i++) {
            this.logBlocks[i].firstDisplayedLineNumber = nextFirstLineNumber;
            nextFirstLineNumber += this.logBlocks[i].totalLinesCount + 1; // add one more line for step name
        }
    }

    trackStepElement(index: number, _: LogBlock) {
        return index;
    }

    trackLineElement(index: number, element: CDNLine) {
        return element ? element.number : null;
    }

    clickScroll(target: ScrollTarget): void {
        this.scrollWrapper.nativeElement.scrollTop = target === ScrollTarget.TOP ?
            0 : this.scrollWrapper.nativeElement.scrollHeight;
    }

    async clickExpandStepDown(stepName: string, event: MouseEvent) {
        let step = this.logBlocks.find(s => s.name === stepName);
        if (!step) {
            return;
        }
        let limit = `${this.expandLoadLinesCount}`;
        if (event.shiftKey) {
            limit = '0';
        }
        let result = await lastValueFrom(this._workflowService.getLogLines(step.link,
            { offset: `${step.lines[step.lines.length - 1].number + 1}`, limit }
        ));
        step.totalLinesCount = result.totalCount;
        step.lines = step.lines.concat(result.lines.filter(l => !step.endLines.find(line => line.number === l.number)));
        this._cd.markForCheck();
    }

    async clickExpandStepUp(stepName: string) {
        let step = this.logBlocks.find(s => s.name === stepName);
        if (!step) {
            return;
        }
        let result = await lastValueFrom(this._workflowService.getLogLines(step.link,
            { offset: `-${step.endLines.length + this.expandLoadLinesCount}`, limit: `${this.expandLoadLinesCount}` }
        ));
        step.totalLinesCount = result.totalCount;
        step.endLines = result.lines.filter(l => !step.lines.find(line => line.number === l.number)
            && !step.endLines.find(line => line.number === l.number)).concat(step.endLines);
        this._cd.markForCheck();
    }

    async clickOpen(logBlock: LogBlock) {
        await this.open(logBlock);
        this._cd.markForCheck();
    }

    async open(logBlock: LogBlock) {
        if (logBlock?.lines?.length > 0 || logBlock.open) {
            logBlock.clickOpen();
            return;
        }

        logBlock.loading = true;
        let results = await Promise.all([
            lastValueFrom(this._workflowService.getLogLines(logBlock.link, { limit: `${this.initLoadLinesCount}` })),
            lastValueFrom(this._workflowService.getLogLines(logBlock.link, { offset: `-${this.initLoadLinesCount}` }))
        ]);
        logBlock.lines = results[0].lines;
        logBlock.endLines = results[1].lines.filter(l => !results[0].lines.find(line => line.number === l.number));
        logBlock.totalLinesCount = results[0].totalCount;
        logBlock.open = true;
        logBlock.loading = false;
    }

    receiveLogs(l: CDNLine): void {
        if (this.logBlocks) {
            this.logBlocks.forEach(v => {
                if (v?.link?.api_ref === l.api_ref_hash) {
                    if (!v.lines.find(line => line.number === l.number)
                        && !v.endLines.find(line => line.number === l.number)) {
                        v.endLines.push(l);
                        v.totalLinesCount++;
                    }
                }
            });
        }
        this._cd.markForCheck();
    }
}
