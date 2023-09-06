import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    ElementRef,
    EventEmitter,
    Input,
    Output
} from "@angular/core";
import {AutoUnsubscribe} from "app/shared/decorator/autoUnsubscribe";
import {V2WorkflowRun, V2WorkflowRunJob} from "app/model/v2.workflow.run.model";
import {LogBlock, ScrollTarget} from "../../workflow/run/node/pipeline/workflow-run-job/workflow-run-job.component";
import * as moment from "moment/moment";
import {DurationService} from "app/shared/duration/duration.service";
import {CDNLine, CDNLogLink, PipelineStatus} from "app/model/pipeline.model";
import {V2WorkflowRunService} from "app/service/workflowv2/workflow.service";
import {WorkflowService} from "app/service/workflow/workflow.service";

@Component({
    selector: 'app-run-job-logs',
    templateUrl: './run-job-logs.html',
    styleUrls: ['./run-job-logs.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class RunJobLogsComponent {
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

    @Output() onScroll = new EventEmitter<ScrollTarget>();

    steps: Array<LogBlock>;
    currentTabIndex = 0;

    constructor(private ref: ElementRef, private _cd: ChangeDetectorRef, private _workflowRunService: V2WorkflowRunService, private _workflowService: WorkflowService) {
    }

    async changeRunJob(data: V2WorkflowRunJob) {
        this.steps = new Array<LogBlock>();
        this._runJob = data;

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
                    this.steps.push(block);
                }
            });
        }

        this.computeStepsDuration();

        this._cd.markForCheck();

        await this.loadEndedSteps();
    }

    async loadEndedSteps() {
        let links = await this._workflowRunService
            .getAllStepsLinks(this.workflowRun, this.runJob.job_id).toPromise();
        links.datas.forEach(d => {
            let step = this.steps.find(s => s.name === d.step_name)
            if (step) {
                step.link = <CDNLogLink>{api_ref: d.api_ref, item_type: links.item_type};
            }
        });

        if (links?.datas?.length > 0) {
            let results = await this._workflowService.getLogsLinesCount(links, this.initLoadLinesCount).toPromise();
            if (results) {
                results.forEach(r => {
                    let step = this.steps.find(s => s.link.api_ref === r.api_ref);
                    if (step) {
                        step.totalLinesCount = r.lines_count;
                        step.open = false;
                        step.loading = false;
                    }
                });
            }
        }


        this.computeStepFirstLineNumbers();

        if (PipelineStatus.isDone(this.runJob.status)) {
            await this.loadFirstFailedOrLastStep();
        } else {
            // TODO
        }

        this._cd.markForCheck();
    }

    async loadFirstFailedOrLastStep() {
        if (this.steps.length <= 1) {
            return;
        }
        if (PipelineStatus.SUCCESS === this.runJob.status) {
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


    computeStepsDuration(): void {
        // TODO : no data on step start/end for now
    }

    computeStepFirstLineNumbers(): void {
        let nextFirstLineNumber = 1;
        for (let i = 0; i < this.steps.length; i++) {
            this.steps[i].firstDisplayedLineNumber = nextFirstLineNumber;
            nextFirstLineNumber += this.steps[i].totalLinesCount + 1; // add one more line for step name
        }
    }

    trackStepElement(index: number, _: LogBlock) {
        return index;
    }

    trackLineElement(index: number, element: CDNLine) {
        return element ? element.number : null;
    }

    formatDuration(fromM: moment.Moment, to?: moment.Moment): string {
        return DurationService.duration(fromM.toDate(), to ? to.toDate() : moment().toDate());
    }

    clickScroll(target: ScrollTarget): void {
        this.onScroll.emit(target);
    }

    async clickExpandStepDown(stepName: string) {
        let step = this.steps.find(s => s.name === stepName);
        if (!step) {
            return;
        }
        let result = await this._workflowService.getLogLines(step.link,
            {offset: `${step.lines[step.lines.length - 1].number + 1}`, limit: `${this.expandLoadLinesCount}`}
        ).toPromise();
        step.totalLinesCount = result.totalCount;
        step.lines = step.lines.concat(result.lines.filter(l => !step.endLines.find(line => line.number === l.number)));
        this._cd.markForCheck();

    }

    async clickExpandStepUp(stepName: string) {
        let step = this.steps.find(s => s.name === stepName);
        if (!step) {
            return;
        }
        let result = await this._workflowService.getLogLines(step.link,
            {offset: `-${step.endLines.length + this.expandLoadLinesCount}`, limit: `${this.expandLoadLinesCount}`}
        ).toPromise();
        step.totalLinesCount = result.totalCount;
        step.endLines = result.lines.filter(l => !step.lines.find(line => line.number === l.number)
            && !step.endLines.find(line => line.number === l.number)).concat(step.endLines);
        this._cd.markForCheck();
    }

    async clickOpen(step: LogBlock) {
        if (step?.lines?.length > 0 || step.open) {
            step.clickOpen();
            return;
        }

        step.loading = true;
        let results = await Promise.all([
            this._workflowService.getLogLines(step.link, {limit: `${this.initLoadLinesCount}`}).toPromise(),
            this._workflowService.getLogLines(step.link, {offset: `-${this.initLoadLinesCount}`}).toPromise()
        ]);
        step.lines = results[0].lines;
        step.endLines = results[1].lines.filter(l => !results[0].lines.find(line => line.number === l.number));
        step.totalLinesCount = results[0].totalCount;
        step.open = true;
        step.loading = false;
        this._cd.markForCheck();
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
        this._cd.markForCheck();
    }

    onJobScroll(target: ScrollTarget) {
        this.ref.nativeElement.children[0].scrollTop = target === ScrollTarget.TOP ?
            0 : this.ref.nativeElement.children[0].scrollHeight;
    }
}
