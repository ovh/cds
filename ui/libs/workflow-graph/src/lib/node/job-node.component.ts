import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnDestroy, OnInit } from '@angular/core';
import { GraphNode } from '../graph.model'
import { V2WorkflowRunJobStatus } from '../v2.workflow.run.model';
import { Subscription, concatMap, from, interval } from 'rxjs';
import { DurationService } from '../duration.service';
import { GraphNodeAction } from './model';

@Component({
    selector: 'app-job-node',
    templateUrl: './job-node.html',
    styleUrls: ['./job-node.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class GraphJobNodeComponent implements OnInit, OnDestroy {
    @Input() node: GraphNode;
    @Input() actionCallback: (type: GraphNodeAction, node: GraphNode, options?: any) => void = () => { };

    highlight = false;
    selected = false;
    statusEnum = V2WorkflowRunJobStatus;
    duration: string;
    delaySubs: Subscription;
    dates: {
        queued: Date;
        scheduled: Date;
        started: Date;
        ended: Date;
    };
    runActive: boolean = false;

    constructor(
        private _cd: ChangeDetectorRef
    ) {
        this.setHighlight.bind(this);
        this.selectNode.bind(this);
    }

    ngOnDestroy(): void {
        if (this.delaySubs) {
            this.delaySubs.unsubscribe();
        }
    }

    ngOnInit(): void {
        if (!this.node.run) {
            return;
        }
        this.dates = {
            queued: new Date(this.node.run.queued),
            scheduled: this.node.run.scheduled ? new Date(this.node.run.scheduled) : null,
            started: this.node.run.started ? new Date(this.node.run.started) : null,
            ended: this.node.run.ended ? new Date(this.node.run.ended) : null
        };
        const isRunning = this.node.run.status === V2WorkflowRunJobStatus.Waiting ||
            this.node.run.status === V2WorkflowRunJobStatus.Scheduling ||
            this.node.run.status === V2WorkflowRunJobStatus.Building;
        if (isRunning) {
            this.delaySubs = interval(1000)
                .pipe(concatMap(_ => from(this.refreshDelay())))
                .subscribe();
        }
        this.refreshDelay();
    }

    async refreshDelay() {
        const now = new Date();
        switch (this.node.run.status) {
            case V2WorkflowRunJobStatus.Waiting:
            case V2WorkflowRunJobStatus.Scheduling:
                this.duration = DurationService.duration(this.dates.queued, now);
                break;
            case V2WorkflowRunJobStatus.Building:
                this.duration = DurationService.duration(this.dates.started, now);
                break;
            case V2WorkflowRunJobStatus.Fail:
            case V2WorkflowRunJobStatus.Stopped:
            case V2WorkflowRunJobStatus.Success:
                this.duration = DurationService.duration(this.dates.started ?? this.dates.queued, this.dates.ended);
                break;
            default:
                break;
        }
        this._cd.markForCheck();
    }

    getNodes() {
        return [this.node];
    }

    onMouseEnter(): void {
        this.actionCallback(GraphNodeAction.Enter, this.node, { jobRunID: this.node.run ? this.node.run.id : null });
    }

    onMouseOut(): void {
        this.actionCallback(GraphNodeAction.Out, this.node, { jobRunID: this.node.run ? this.node.run.id : null });
    }

    onMouseClick(): void {
        this.actionCallback(GraphNodeAction.Click, this.node, { jobRunID: this.node.run ? this.node.run.id : null });
    }

    setHighlight(active: boolean, options?: any): void {
        this.highlight = active;
        this._cd.markForCheck();
    }

    selectNode(navigationKey: string): void {
        this.selected = navigationKey === (this.node.job.stage ? `${this.node.job.stage}-${this.node.name}` : this.node.name);
        this._cd.markForCheck();
    }

    activateNode(navigationKey: string): void {
        if (navigationKey === (this.node.job.stage ? `${this.node.job.stage}-${this.node.name}` : this.node.name)) {
            this.actionCallback(GraphNodeAction.Click, this.node, { jobRunID: this.node.run ? this.node.run.id : null });
        }
    }

    setRunActive(active: boolean): void {
        this.runActive = active;
        this._cd.markForCheck();
    }

    clickRunGate(event: Event): void {
        this.actionCallback(GraphNodeAction.Click, this.node, { gateName: this.node.gate });
        event.preventDefault();
        event.stopPropagation();
    }

    clickRestart(event: Event): void {
        this.actionCallback(GraphNodeAction.ClickRestart, this.node, { jobRunID: this.node.run.id });
        event.preventDefault();
        event.stopPropagation();
    }

    clickStop(event: Event): void {
        this.actionCallback(GraphNodeAction.ClickStop, this.node, { jobRunID: this.node.run.id });
        event.preventDefault();
        event.stopPropagation();
    }
}
