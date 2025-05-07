import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnInit } from '@angular/core';
import { GraphNode } from '../graph.model';
import { NodeStatus } from './model';

@Component({
    selector: 'app-fork-join-node',
    templateUrl: './fork-join-node.html',
    styleUrls: ['./fork-join-node.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class GraphForkJoinNodeComponent implements OnInit {
    @Input() nodes: Array<GraphNode>;
    @Input() type = 'fork';
    @Input() actionCallback: (type: string, node: GraphNode) => void = () => { };

    highlight = false;
    status: string;
    nodeStatusEnum = NodeStatus;

    constructor(
        private _cd: ChangeDetectorRef
    ) {
        this.setHighlight.bind(this);
        this.selectNode.bind(this);
    }

    ngOnInit() {
        this.status = NodeStatus.sum(this.nodes.map(n => n.run ? n.run.status : null));
    }

    getNodes() {
        return this.nodes;
    }

    onMouseEnter(): void {
        this.nodes.forEach(n => { this.actionCallback('enter', n); });
    }

    onMouseOut(): void {
        this.nodes.forEach(n => { this.actionCallback('out', n); });
    }

    setHighlight(active: boolean, options?: any): void {
        this.highlight = active;
        this._cd.markForCheck();
    }

    selectNode(navigationKey: string): void { }

    activateNode(navigationKey: string): void { }

    setRunActive(active: boolean): void {}   
}
