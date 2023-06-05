import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnDestroy, OnInit } from '@angular/core';
import { PipelineStatus } from 'app/model/pipeline.model';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import {GraphNode} from "../graph.model";

@Component({
    selector: 'app-fork-join-node',
    templateUrl: './fork-join-node.html',
    styleUrls: ['./fork-join-node.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectV2WorkflowForkJoinNodeComponent implements OnInit, OnDestroy {
    @Input() nodes: Array<GraphNode>;
    @Input() type = 'fork';
    @Input() mouseCallback: (type: string, node: GraphNode) => void;

    highlight = false;
    status: string;
    pipelineStatusEnum = PipelineStatus;

    constructor(
        private _cd: ChangeDetectorRef
    ) {
        this.setHighlight.bind(this);
        this.setSelect.bind(this);
    }

    ngOnInit() {
        this.status = PipelineStatus.sum(this.nodes.map(n => n.run ? n.run.status : null));
    }

    getNodes() { return this.nodes; }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    onMouseEnter(): void {
        if (this.mouseCallback) {
            this.nodes.forEach(n => {
                this.mouseCallback('enter', n);
            });
        }
    }

    onMouseOut(): void {
        if (this.mouseCallback) {
            this.nodes.forEach(n => {
                this.mouseCallback('out', n);
            });
        }
    }

    setHighlight(active: boolean): void {
        this.highlight = active;
        this._cd.markForCheck();
    }

    setSelect(_: boolean): void { }
}
