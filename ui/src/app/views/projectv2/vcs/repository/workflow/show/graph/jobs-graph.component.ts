import {
    AfterViewInit,
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    ComponentRef,
    Input,
    OnDestroy,
    ViewChild,
    ViewContainerRef
} from '@angular/core';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import {GraphNode} from "./graph.model";
import {GraphDirection, WorkflowNodeComponent, WorkflowV2Graph} from "./graph.lib";
import {ProjectV2WorkflowForkJoinNodeComponent} from "./node/fork-join-node.components";
import {ProjectV2WorkflowJobNodeComponent} from "./node/job-node.component";

@Component({
    selector: 'app-jobs-graph',
    templateUrl: './jobs-graph.html',
    styleUrls: ['./jobs-graph.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectV2WorkflowJobsGraphComponent implements AfterViewInit, OnDestroy {
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
    @Input() mouseCallback: (type: string, node: GraphNode) => void;
    @Input() selectJobCallback: (name: string) => void;

    ready: boolean;
    highlight = false;

    // workflow graph
    @ViewChild('svgSubGraph', { read: ViewContainerRef }) svgContainer: ViewContainerRef;
    graph: WorkflowV2Graph<WorkflowNodeComponent>;

    constructor(
        private _cd: ChangeDetectorRef
    ) { }

    getNodes() { return [this.node]; }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

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

    setHighlight(active: boolean): void {
        this.highlight = active;
        this._cd.markForCheck();
    }

    setSelect(active: boolean): void {
        this.graph.unselectAllNode();
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
                ProjectV2WorkflowJobsGraphComponent.minScale, ProjectV2WorkflowJobsGraphComponent.maxScale);
        }

        this.nodes.forEach(n => {
            this.graph.createNode(`${this.node.name}-${n.name}`, this.createJobNodeComponent(n),
                n.run ? n.run.status : null);
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

    createJobNodeComponent(node: GraphNode): ComponentRef<ProjectV2WorkflowJobNodeComponent> {
        const componentRef = this.svgContainer.createComponent(ProjectV2WorkflowJobNodeComponent);
        componentRef.instance.node = node;
        componentRef.instance.mouseCallback = this.nodeMouseEvent.bind(this);
        componentRef.changeDetectorRef.detectChanges();
        return componentRef;
    }

    createForkJoinNodeComponent(nodes: Array<GraphNode>, type: string): ComponentRef<ProjectV2WorkflowForkJoinNodeComponent> {
        const componentRef = this.svgContainer.createComponent(ProjectV2WorkflowForkJoinNodeComponent);
        componentRef.instance.nodes = nodes;
        componentRef.instance.type = type;
        componentRef.instance.mouseCallback = this.nodeMouseEvent.bind(this);
        componentRef.changeDetectorRef.detectChanges();
        return componentRef;
    }

    nodeMouseEvent(type: string, n: GraphNode) {
        if (type === 'click' && this.selectJobCallback) {
            this.selectJobCallback(n.name);
        }
        this.graph.nodeMouseEvent(type, `${this.node.name}-${n.name}`);
    }

    clickCenter(): void {
        if (this.centerCallback) { this.centerCallback(this.node); }
    }
}
