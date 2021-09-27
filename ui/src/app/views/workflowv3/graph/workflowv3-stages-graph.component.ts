import {
    AfterViewInit,
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    ComponentFactoryResolver,
    ComponentRef,
    EventEmitter,
    HostListener,
    Input,
    OnDestroy,
    Output,
    ViewChild,
    ViewContainerRef
} from '@angular/core';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { GraphNode, JobRun, WorkflowRunV3, WorkflowV3 } from '../workflowv3.model';
import { WorkflowV3ForkJoinNodeComponent } from './workflowv3-fork-join-node.components';
import { GraphDirection, WorkflowV3Graph } from './workflowv3-graph.lib';
import { WorkflowV3JobNodeComponent } from './workflowv3-job-node.component';
import { WorkflowV3JobsGraphComponent } from './workflowv3-jobs-graph.component';

export type WorkflowV3JobsGraphOrNodeComponent = WorkflowV3JobsGraphComponent |
    WorkflowV3ForkJoinNodeComponent | WorkflowV3JobNodeComponent;

@Component({
    selector: 'app-workflowv3-stages-graph',
    templateUrl: './workflowv3-stages-graph.html',
    styleUrls: ['./workflowv3-stages-graph.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowV3StagesGraphComponent implements AfterViewInit, OnDestroy {
    static maxScale = 15;
    static minScale = 1 / 5;

    nodes: Array<GraphNode> = [];
    @Input() set workflow(data: WorkflowV3) {
        this.nodes = [];
        this.hasStages = !!data && !!data.stages;
        if (data && data.stages) {
            this.nodes.push(...Object.keys(data.stages)
                .map(k => <GraphNode>{ name: k, depends_on: data.stages[k].depends_on, sub_graph: [] }));
        }
        if (data && data.jobs) {
            Object.keys(data.jobs).forEach(k => {
                let j = data.jobs[k];
                let node = <GraphNode>{ name: k, depends_on: j.depends_on };
                if (this.jobRuns[k]) {
                    node.run = this.jobRuns[k][0];
                }
                if (j.stage) {
                    for (let i = 0; i < this.nodes.length; i++) {
                        if (this.nodes[i].name === j.stage) {
                            this.nodes[i].sub_graph.push(node);
                            break;
                        }
                    }
                } else {
                    this.nodes.push(node);
                }
            });
        }
        this.changeDisplay();
    }

    jobRuns: { [name: string]: Array<JobRun> } = {};
    @Input() set workflowRun(data: WorkflowRunV3) {
        if (!data) {
            return;
        }
        this.jobRuns = data.job_runs;
        this.workflow = data.workflow;
    }

    @Output() onSelectJob = new EventEmitter<string>();

    direction: GraphDirection = GraphDirection.HORIZONTAL;

    ready: boolean;
    hasStages = false;

    // workflow graph
    @ViewChild('svgGraph', { read: ViewContainerRef }) svgContainer: ViewContainerRef;
    graph: WorkflowV3Graph<WorkflowV3JobsGraphOrNodeComponent>;

    constructor(
        private componentFactoryResolver: ComponentFactoryResolver,
        private _cd: ChangeDetectorRef
    ) { }

    static isJobsGraph = (component: WorkflowV3JobsGraphOrNodeComponent): component is WorkflowV3JobsGraphComponent => {
        if ((component as WorkflowV3JobsGraphComponent).direction) {
            return true;
        }
        return false;
    };

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    ngAfterViewInit(): void {
        this.ready = true;
        this._cd.detectChanges();
        this.changeDisplay();
    }

    @HostListener('window:resize')
    onResize() {
        const element = this.svgContainer.element.nativeElement;
        if (!this.graph) { return; }
        this.graph.resize(element.offsetWidth, element.offsetHeight);
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
            this.graph = new WorkflowV3Graph(this.createForkJoinNodeComponent.bind(this), this.direction,
                WorkflowV3StagesGraphComponent.minScale,
                WorkflowV3StagesGraphComponent.maxScale);
        }

        this.nodes.forEach(n => {
            if (this.hasStages) {
                this.graph.createNode(n.name, this.createSubGraphComponent(n),
                    null, 300, 169);
            } else {
                this.graph.createNode(n.name, this.createJobNodeComponent(n),
                    n.run ? n.run.status : null);
            }
        });

        this.nodes.forEach(n => {
            if (n.depends_on && n.depends_on.length > 0) {
                n.depends_on.forEach(d => {
                    this.graph.createEdge(`node-${d}`, `node-${n.name}`);
                });
            }
        });

        const element = this.svgContainer.element.nativeElement;

        this.graph.draw(element, true);

        this.resize();

        if (!this.graph.transformed) {
            this.clickOrigin();
        }

        this._cd.markForCheck();
    }

    resize() {
        if (!this.svgContainer?.element?.nativeElement?.offsetWidth || !this.svgContainer?.element?.nativeElement?.offsetHeight) {
            return;
        }
        this.graph.resize(this.svgContainer.element.nativeElement.offsetWidth, this.svgContainer.element.nativeElement.offsetHeight);
    }

    clickOrigin() {
        if (!this.svgContainer?.element?.nativeElement?.offsetWidth || !this.svgContainer?.element?.nativeElement?.offsetHeight) {
            return;
        }
        this.graph.center(this.svgContainer.element.nativeElement.offsetWidth, this.svgContainer.element.nativeElement.offsetHeight);
    }

    createJobNodeComponent(node: GraphNode): ComponentRef<WorkflowV3JobNodeComponent> {
        const nodeComponentFactory = this.componentFactoryResolver.resolveComponentFactory(WorkflowV3JobNodeComponent);
        const componentRef = this.svgContainer.createComponent<WorkflowV3JobNodeComponent>(nodeComponentFactory);
        componentRef.instance.node = node;
        componentRef.instance.mouseCallback = this.nodeJobMouseEvent.bind(this);
        componentRef.changeDetectorRef.detectChanges();
        return componentRef;
    }

    createForkJoinNodeComponent(nodes: Array<GraphNode>, type: string): ComponentRef<WorkflowV3ForkJoinNodeComponent> {
        const nodeComponentFactory = this.componentFactoryResolver.resolveComponentFactory(WorkflowV3ForkJoinNodeComponent);
        const componentRef = this.svgContainer.createComponent<WorkflowV3ForkJoinNodeComponent>(nodeComponentFactory);
        componentRef.instance.nodes = nodes;
        componentRef.instance.type = type;
        componentRef.instance.mouseCallback = this.nodeMouseEvent.bind(this);
        componentRef.changeDetectorRef.detectChanges();
        return componentRef;
    }

    createSubGraphComponent(node: GraphNode): ComponentRef<WorkflowV3JobsGraphComponent> {
        const nodeComponentFactory = this.componentFactoryResolver.resolveComponentFactory(WorkflowV3JobsGraphComponent);
        const componentRef = this.svgContainer.createComponent<WorkflowV3JobsGraphComponent>(nodeComponentFactory);
        componentRef.instance.graphNode = node;
        componentRef.instance.direction = this.direction;
        componentRef.instance.centerCallback = this.centerSubGraph.bind(this);
        componentRef.instance.mouseCallback = this.nodeMouseEvent.bind(this);
        componentRef.instance.selectJobCallback = this.subGraphSelectJob.bind(this);
        componentRef.changeDetectorRef.detectChanges();
        return componentRef;
    }

    centerSubGraph(node: GraphNode): void {
        if (!this.svgContainer?.element?.nativeElement?.offsetWidth || !this.svgContainer?.element?.nativeElement?.offsetHeight) {
            return;
        }
        this.graph.centerNode(`node-${node.name}`,
            this.svgContainer.element.nativeElement.offsetWidth,
            this.svgContainer.element.nativeElement.offsetHeight);
    }

    nodeMouseEvent(type: string, n: GraphNode) {
        this.graph.nodeMouseEvent(type, n.name);
    }

    nodeJobMouseEvent(type: string, n: GraphNode) {
        if (type === 'click') {
            this.onSelectJob.emit(n.name);
        }
        this.graph.nodeMouseEvent(type, n.name);
    }

    subGraphSelectJob(name: string): void {
        this.graph.unselectAllNode();
        this.onSelectJob.emit(name);
    }

    changeDirection(): void {
        this.direction = this.direction === GraphDirection.HORIZONTAL ? GraphDirection.VERTICAL : GraphDirection.HORIZONTAL;
        this.changeDisplay();
    }
}
