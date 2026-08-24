import { ChangeDetectionStrategy, ChangeDetectorRef, Component, EventEmitter, inject, Input, OnChanges, Output } from "@angular/core";
import { TimelineActivation, TimelineData } from "../../../../../libs/timeline/src/public-api";
import { V2WorkflowRun, V2WorkflowRunJob, WorkflowRunInfo, WorkflowRunResult } from "../../../../../libs/workflow-graph/src/lib/v2.workflow.run.model";
import { buildRunTimeline, RunTimelineTarget } from "./run-timeline.builder";

/**
 * The run seen as time rather than as structure: how long each job waited, how long it ran and when
 * it produced what. Everything about drawing time belongs to the timeline component, everything about
 * what a lane means belongs here.
 */
@Component({
    standalone: false,
    selector: 'app-run-timeline',
    templateUrl: './run-timeline.html',
    styleUrls: ['./run-timeline.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class RunTimelineComponent implements OnChanges {
    @Input() workflowRun: V2WorkflowRun;
    @Input() jobs: Array<V2WorkflowRunJob>;
    @Input() results: Array<WorkflowRunResult>;
    /** What the run logged. Only its errors are drawn, at the moment each was written. */
    @Input() infos: Array<WorkflowRunInfo>;
    @Input() runAttempt: number;
    /** The run job being pointed at in the graph, so that both views point at the same thing. */
    @Input() hoveredJobRunID: string;
    /** The run job whose panel is open. */
    @Input() selectedJobRunID: string;

    @Output() onSelectJobRun = new EventEmitter<string>();
    @Output() onSelectResult = new EventEmitter<string>();
    /** An error of the run was activated: it is written down in full in the Info tab, not in a panel. */
    @Output() onSelectInfo = new EventEmitter<string>();
    /** What set the run off was activated, named by the hook that fired. */
    @Output() onSelectHook = new EventEmitter<string>();

    data: TimelineData;
    criticalPathEnabled: boolean = false;
    hasCriticalPath: boolean = false;
    /** What the toggle is for, or why it is not available on this run. */
    criticalPathHint: string;

    private targets: { [key: string]: RunTimelineTarget } = {};

    private _cd = inject(ChangeDetectorRef);

    /** The lane of a run job, named the same way the builder names it. */
    get hoveredLaneID(): string {
        return this.hoveredJobRunID ? `job-${this.hoveredJobRunID}` : null;
    }

    get selectedLaneID(): string {
        return this.selectedJobRunID ? `job-${this.selectedJobRunID}` : null;
    }

    ngOnChanges(): void {
        const build = buildRunTimeline(this.workflowRun, this.jobs, this.results, this.infos, this.runAttempt);
        this.data = build.data;
        this.targets = build.targets;

        /**
         * The highlight works by holding back what is *not* on the chain, so it only says something when
         * something is left out. A chain of one says nothing; so does a chain holding every job of the
         * run, which is what a workflow that is one job after another comes to.
         */
        const onTheChain = build.criticalPath.length;
        this.criticalPathHint = onTheChain <= 1
            ? 'No chain to bring out on this run'
            : onTheChain >= build.jobCount
                ? 'Every job of this run is on the critical path, so there is nothing to bring out'
                : 'Highlight the chain of jobs that set the total duration of the run';
        this.hasCriticalPath = onTheChain > 1 && onTheChain < build.jobCount;

        if (!this.hasCriticalPath) {
            this.criticalPathEnabled = false;
        }
        this._cd.markForCheck();
    }

    clickToggleCriticalPath(): void {
        this.criticalPathEnabled = !this.criticalPathEnabled;
        this._cd.markForCheck();
    }

    onActivate(activation: TimelineActivation): void {
        const target = this.targets[activation.markerID ?? activation.laneID];
        if (!target) {
            return;
        }
        switch (target.type) {
            case 'job':
                this.onSelectJobRun.emit(target.id);
                return;
            case 'result':
                this.onSelectResult.emit(target.id);
                return;
            case 'info':
                this.onSelectInfo.emit(target.id);
                return;
            case 'hook':
                this.onSelectHook.emit(target.id);
                return;
        }
    }
}
