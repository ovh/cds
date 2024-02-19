import {ChangeDetectionStrategy, ChangeDetectorRef, Component, Input} from '@angular/core';
import {GraphNode} from '../graph.model'
import { NodeStatus } from './status.model';

@Component({
    selector: 'app-job-node',
    templateUrl: './job-node.html',
    styleUrls: ['./job-node.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class GraphJobNodeComponent {
    @Input() node: GraphNode;
    @Input() mouseCallback: (type: string, node: GraphNode) => void;

    highlight = false;
    selected = false;
    nodeStatusEnum = NodeStatus;

    constructor(
        private _cd: ChangeDetectorRef
    ) {
        this.setHighlight.bind(this);
        this.setSelect.bind(this);
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

    setSelect(active: boolean): void {
        this.selected = active;
        this._cd.markForCheck();
    }
}
