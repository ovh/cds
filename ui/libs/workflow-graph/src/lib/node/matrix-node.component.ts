import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnDestroy, OnInit } from '@angular/core';
import { GraphNode } from '../graph.model'
import { V2WorkflowRunJobStatus } from '../v2.workflow.run.model';
import { concatMap, from, interval, Subscription } from 'rxjs';
import { DurationService } from '../duration/duration.service';

@Component({
    selector: 'app-matrix-node',
    templateUrl: './matrix-node.html',
    styleUrls: ['./matrix-node.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class GraphMatrixNodeComponent implements OnInit, OnDestroy {
    @Input() node: GraphNode;
    @Input() mouseCallback: (type: string, node: GraphNode, options?: any) => void;

    highlightKey: string;
    selectedKey: string;
    statusEnum = V2WorkflowRunJobStatus;
    durations: { [key: string]: string } = {};
    delaySubs: Subscription;
    dates: {
        [key: string]: {
            queued: Date;
            scheduled: Date;
            started: Date;
            ended: Date;
        }
    } = {};
    keys: Array<string> = [];
    status: { [key: string]: string } = {};
    jobRunIDs: { [key: string]: string } = {};

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
        const alls = GraphNode.generateMatrixOptions(this.node.job.strategy.matrix);
        this.keys = alls.map(option => {
            return Array.from(option.keys()).sort().map(key => {
                return `${key}:${option.get(key)}`;
            }).join(', ');
        });
        (this.node.runs ?? []).forEach(r => {
            const key = Object.keys(r.matrix).sort().map(key => {
                return `${key}:${r.matrix[key]}`;
            }).join(', ');
            this.dates[key] = {
                queued: new Date(r.queued),
                scheduled: r.scheduled ? new Date(r.scheduled) : null,
                started: r.started ? new Date(r.started) : null,
                ended: r.ended ? new Date(r.ended) : null
            };
            this.status[key] = r.status;
            this.jobRunIDs[key] = r.id;
        });
        const isRunning = Object.keys(this.status).findIndex(key => this.status[key] === V2WorkflowRunJobStatus.Waiting ||
            this.status[key] === V2WorkflowRunJobStatus.Scheduling ||
            this.status[key] === V2WorkflowRunJobStatus.Building) !== -1;
        if (isRunning) {
            this.delaySubs = interval(1000)
                .pipe(concatMap(_ => from(this.refreshDelay())))
                .subscribe();
        }
        this.refreshDelay();
    }

    async refreshDelay() {
        const now = new Date();
        (this.node.runs ?? []).forEach(r => {
            const key = Object.keys(r.matrix).sort().map(key => {
                return `${key}:${r.matrix[key]}`;
            }).join(', ');
            switch (r.status) {
                case V2WorkflowRunJobStatus.Waiting:
                case V2WorkflowRunJobStatus.Scheduling:
                    this.durations[key] = DurationService.duration(this.dates[key].queued, now);
                    break;
                case V2WorkflowRunJobStatus.Building:
                    this.durations[key] = DurationService.duration(this.dates[key].started, now);
                    break;
                case V2WorkflowRunJobStatus.Fail:
                case V2WorkflowRunJobStatus.Stopped:
                case V2WorkflowRunJobStatus.Success:
                    if (this.dates[key].started) {
                        this.durations[key] = DurationService.duration(this.dates[key].started, this.dates[key].ended);
                    }
                    break;
                default:
                    break;
            }
        });
        this._cd.markForCheck();
    }

    getNodes() {
        return [this.node];
    }

    onMouseEnter(key: string): void {
        if (this.mouseCallback) {
            this.mouseCallback('enter', this.node, {
                jobRunID: this.jobRunIDs[key] ?? null,
                jobMatrixKey: key
            });
        }
    }

    onMouseOut(key: string): void {
        if (this.mouseCallback) {
            this.mouseCallback('out', this.node, {
                jobRunID: this.jobRunIDs[key] ?? null,
                jobMatrixKey: key
            });
        }
    }

    onMouseClick(key: string): void {
        if (this.mouseCallback) {
            this.mouseCallback('click', this.node, {
                jobRunID: this.jobRunIDs[key] ?? null,
                jobMatrixKey: key
            });
        }
    }

    setHighlight(active: boolean, options?: any): void {
        if (options && options['jobMatrixKey'] && active) {
            this.highlightKey = options['jobMatrixKey'];
        } else {
            this.highlightKey = null;
        }
        this._cd.markForCheck();
    }

    selectNode(navigationKey: string): void {
        const baseKey = this.node.job.stage ? `${this.node.job.stage}-${this.node.name}` : this.node.name;
        this.selectedKey = null;
        for (let i = 0; i < this.keys.length; i++) {
            if (`${baseKey}-${this.keys[i]}` === navigationKey) {
                this.selectedKey = this.keys[i];
                break;
            }
        }
        this._cd.markForCheck();
    }

    activateNode(navigationKey: string): void {
        const baseKey = this.node.job.stage ? `${this.node.job.stage}-${this.node.name}` : this.node.name;
        if (this.mouseCallback && this.selectedKey && `${baseKey}-${this.selectedKey}` === navigationKey) {
            this.mouseCallback('click', this.node, {
                jobRunID: this.jobRunIDs[this.selectedKey] ?? null,
                jobMatrixKey: this.selectedKey
            });
        }
    }

    clickRunGate(event: Event): void {
        if (this.mouseCallback) {
            this.mouseCallback('click', this.node, { gateName: this.node.gate });
        }
        event.preventDefault();
        event.stopPropagation();
    }
}
