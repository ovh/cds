import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnInit } from '@angular/core';
import { GraphNode } from '../graph.model';
import { NodeStatus } from './status.model';

@Component({
    selector: 'app-fork-join-node',
    templateUrl: './fork-join-node.html',
    styleUrls: ['./fork-join-node.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class GraphForkJoinNodeComponent implements OnInit {
    @Input() nodes: Array<GraphNode>;
    @Input() type = 'fork';
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
        this.status = NodeStatus.sum(this.nodes.map(n => n.run ? n.run.status : null));
    }

    getNodes() {
        return this.nodes;
    }

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

    setHighlight(active: boolean, options?: any): void {
        this.highlight = active;
        this._cd.markForCheck();
    }

    setSelect(active: boolean, options?: any): void { }
}
