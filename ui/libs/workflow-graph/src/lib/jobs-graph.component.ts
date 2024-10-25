import {
    AfterViewInit,
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    ComponentRef,
    Input,
    ViewChild,
    ViewContainerRef
} from '@angular/core';
import { GraphNode, GraphNodeType } from './graph.model';
import { GraphDirection, NodeMouseEvent, WorkflowV2Graph } from './graph.lib';
import { GraphForkJoinNodeComponent } from './node/fork-join-node.components';
import { GraphJobNodeComponent } from './node/job-node.component';
import { GraphMatrixNodeComponent } from './node/matrix-node.component';
import { GraphNodeAction } from './node/model';

export type WorkflowV2JobsNodeOrMatrixComponent = GraphForkJoinNodeComponent | GraphJobNodeComponent | GraphMatrixNodeComponent;

@Component({
    selector: 'app-jobs-graph',
    templateUrl: './jobs-graph.html',
    styleUrls: ['./jobs-graph.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class WorkflowV2JobsGraphComponent implements AfterViewInit {
    static maxScale = 2;
    static minScale = 0.1;

    node: GraphNode;
    nodes: Array<GraphNode> = [];

    @Input() set graphNode(data: GraphNode) {
        this.node = data;
        this.nodes = data.sub_graph;
        this.changeDisplay();
    }

    @Input() direction: GraphDirection;
    @Input() centerCallback: any;
    @Input() actionCallback: (type: string, node: GraphNode, options?: any) => void;

    ready: boolean;
    highlight = false;

    // workflow graph
    @ViewChild('svgSubGraph', { read: ViewContainerRef }) svgContainer: ViewContainerRef;
    graph: WorkflowV2Graph<WorkflowV2JobsNodeOrMatrixComponent>;

    constructor(
        private _cd: ChangeDetectorRef
    ) { }

    getNodes() {
        return [this.node];
    }

    onMouseEnter(): void {
        this.actionCallback('enter', this.node);
    }

    onMouseOut(): void {
        this.actionCallback('out', this.node);
    }

    setHighlight(active: boolean): void {
        this.highlight = active;
        this._cd.markForCheck();
    }

    selectNode(navigationKey: string): void {
        this.graph.selectNode(navigationKey);
    }

    activateNode(navigationKey: string): void {
        this.graph.activateNode(navigationKey);
    }

    setRunActive(active: boolean): void {
        this.graph.setRunActive(active);
    }

    ngAfterViewInit(): void {
        this.ready = true;
        this._cd.detectChanges();
        this.changeDisplay();
    }

    changeDisplay(): void {
        if (!this.ready) {
            return;
        }
        this.initGraph();
    }

    initGraph() {
        if (this.graph) {
            this.graph.clean();
        }
        if (!this.graph || this.graph.direction !== this.direction) {
            this.graph = new WorkflowV2Graph(this.createForkJoinNodeComponent.bind(this), this.direction,
                WorkflowV2JobsGraphComponent.minScale, WorkflowV2JobsGraphComponent.maxScale);
        }

        this.nodes.forEach(n => {
            let component: ComponentRef<WorkflowV2JobsNodeOrMatrixComponent>;
            switch (n.type) {
                case GraphNodeType.Matrix:
                    component = this.createJobMatrixComponent(n);
                    this.graph.createNode(`${this.node.name}-${n.name}`, n, component);
                    break;
                default:
                    component = this.createJobNodeComponent(n);
                    this.graph.createNode(`${this.node.name}-${n.name}`, n, component);
                    if (n.run) {
                        this.graph.setNodeStatus(`${this.node.name}-${n.name}`, n.run ? n.run.status : null);
                    }
                    break;
            }
        });

        this.nodes.forEach(n => {
            if (n.depends_on && n.depends_on.length > 0) {
                n.depends_on.forEach(d => {
                    this.graph.createEdge(`node-${this.node.name}-${d}`, `node-${this.node.name}-${n.name}`);
                });
            }
        });

        const element = this.svgContainer.element.nativeElement;
        this.graph.draw(element, false);
        this.graph.center(300, 169);
        this._cd.markForCheck();
    }

    createJobMatrixComponent(node: GraphNode): ComponentRef<GraphMatrixNodeComponent> {
        const componentRef = this.svgContainer.createComponent(GraphMatrixNodeComponent);
        componentRef.instance.node = node;
        componentRef.instance.actionCallback = this.onNodeAction.bind(this);
        componentRef.changeDetectorRef.detectChanges();
        return componentRef;
    }

    createJobNodeComponent(node: GraphNode): ComponentRef<GraphJobNodeComponent> {
        const componentRef = this.svgContainer.createComponent(GraphJobNodeComponent);
        componentRef.instance.node = node;
        componentRef.instance.actionCallback = this.onNodeAction.bind(this);
        componentRef.changeDetectorRef.detectChanges();
        return componentRef;
    }

    createForkJoinNodeComponent(nodes: Array<GraphNode>, type: string): ComponentRef<GraphForkJoinNodeComponent> {
        const componentRef = this.svgContainer.createComponent(GraphForkJoinNodeComponent);
        componentRef.instance.nodes = nodes;
        componentRef.instance.type = type;
        componentRef.instance.actionCallback = this.onNodeAction.bind(this);
        componentRef.changeDetectorRef.detectChanges();
        return componentRef;
    }

    onNodeAction(type: GraphNodeAction, n: GraphNode, options?: any) {
        switch (type) {
            case GraphNodeAction.Enter:
                this.graph.nodeMouseEvent(NodeMouseEvent.Enter, `${this.node.name}-${n.name}`, options);
                break;
            case GraphNodeAction.Out:
                this.graph.nodeMouseEvent(NodeMouseEvent.Out, `${this.node.name}-${n.name}`, options);
                break;
        }
        this.actionCallback(type, n, options);
    }

    clickCenter(): void {
        if (this.centerCallback) {
            this.centerCallback(this.node);
        }
    }
}
