import {ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnDestroy, OnInit} from '@angular/core';
import {PipelineStatus} from 'app/model/pipeline.model';
import {AutoUnsubscribe} from 'app/shared/decorator/autoUnsubscribe';
import {GraphNode} from "../graph.model";

@Component({
    selector: 'app-gate-node',
    templateUrl: './gate-node.html',
    styleUrls: ['./gate-node.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectV2WorkflowGateNodeComponent implements OnInit, OnDestroy {
    @Input() node: GraphNode;
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
        this.status = this.node.gateStatus;
    }

    getNodes() {
        return [this.node];
    }

    ngOnDestroy(): void {
    } // Should be set to use @AutoUnsubscribe with AOT

    onMouseEnter(): void {
        if (this.mouseCallback) {
            this.mouseCallback('enter', this.node);
        }
    }

    onMouseOut(): void {
        if (this.mouseCallback) {
            this.mouseCallback('out', this.node);
        }
    }

    onMouseClick(): void {
        if (this.mouseCallback) {
            this.mouseCallback('click', this.node);
        }
    }

    setHighlight(active: boolean): void {
        this.highlight = active;
        this._cd.markForCheck();
    }

    setSelect(_: boolean): void {
    }
}
