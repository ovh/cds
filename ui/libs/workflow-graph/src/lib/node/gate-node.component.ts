import {ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnDestroy, OnInit} from '@angular/core';
import {GraphNode} from '../graph.model'
import { NodeStatus } from './status.model';

@Component({
    selector: 'app-gate-node',
    templateUrl: './gate-node.html',
    styleUrls: ['./gate-node.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class GraphGateNodeComponent implements OnInit {
    @Input() node: GraphNode;
    @Input() mouseCallback: (type: string, node: GraphNode) => void;

    highlight = false;
    status: string;
    nodeStatusEnum = NodeStatus;

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
