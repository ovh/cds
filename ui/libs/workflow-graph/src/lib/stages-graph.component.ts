import {
    AfterViewInit,
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    ComponentRef, ElementRef,
    EventEmitter,
    Input,
    OnDestroy,
    Output,
    ViewChild,
    ViewContainerRef
} from '@angular/core';
import { WorkflowV2JobsGraphComponent } from './jobs-graph.component';
import { GraphForkJoinNodeComponent } from './node/fork-join-node.components';
import { GraphJobNodeComponent } from './node/job-node.component';
import { GraphNode, GraphNodeType } from './graph.model';
import { GraphDirection, WorkflowV2Graph } from './graph.lib';
import { load, LoadOptions } from 'js-yaml';
import { V2Workflow, V2WorkflowRun, V2WorkflowRunJob } from './v2.workflow.run.model';
import { GraphMatrixNodeComponent } from './node/matrix-node.component';

export type WorkflowV2JobsGraphOrNodeOrMatrixComponent = WorkflowV2JobsGraphComponent | GraphForkJoinNodeComponent | GraphJobNodeComponent | GraphMatrixNodeComponent;

@Component({
    selector: 'app-stages-graph',
    templateUrl: './stages-graph.html',
    styleUrls: ['./stages-graph.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class WorkflowV2StagesGraphComponent implements AfterViewInit, OnDestroy {
    static maxScale = 15;
    static minScale = 1 / 5;

    nodes: Array<GraphNode> = [];
    hooks: Array<any> = [];
    selectedHook: string;
    hooksOn: any;

    @Input() set workflow(data: any) {
        // Parse the workflow
        let workflow: V2Workflow;
        try {
            workflow = load(data && data !== '' ? data : '{}', <LoadOptions>{
                onWarning: (e) => { }
            });
        } catch (e) {
            console.error("Invalid workflow:", data, e)
        }

        this.hasStages = !!workflow && !!workflow.stages;

        this.nodes = [];
        if (this.hasStages) {
            this.nodes.push(...Object.keys(workflow.stages).map(k => <GraphNode>{
                type: GraphNodeType.Stage,
                name: k,
                depends_on: workflow.stages[k]?.needs,
                sub_graph: []
            }));
        }

        if (workflow && workflow.jobs) {
            Object.keys(workflow.jobs).forEach(jobName => {
                const jobSpec = workflow.jobs[jobName];

                let node = <GraphNode>{
                    type: jobSpec?.strategy?.matrix ? GraphNodeType.Matrix : GraphNodeType.Job,
                    name: jobName,
                    depends_on: jobSpec?.needs,
                    job: jobSpec
                };
                if (jobSpec.gate) {
                    node.gate = workflow.gates[jobSpec.gate];
                }

                if (jobSpec?.stage) {
                    for (let i = 0; i < this.nodes.length; i++) {
                        if (this.nodes[i].name === jobSpec.stage && this.nodes[i].type === GraphNodeType.Stage) {
                            this.nodes[i].sub_graph.push(node);
                            break;
                        }
                    }
                } else {
                    this.nodes.push(node);
                }
            });
        }

        this.initRunJobs();
        this.initGate();

        this.hooks = [];
        this.selectedHook = '';
        if (workflow && workflow.on) {
            this.hooksOn = workflow.on;
            this.initHooks();
        }

        this.changeDisplay();
        this._cd.markForCheck();
    }

    _runJobs: Array<V2WorkflowRunJob> = [];

    @Input() set runJobs(data: Array<V2WorkflowRunJob>) {
        this._runJobs = data ?? [];
        if (!this.svgContainer) {
            return;
        }
        this.initRunJobs();
        this.initGraph();
    }

    _workflowRun: V2WorkflowRun
    @Input() set workflowRun(data: V2WorkflowRun) {
        this._workflowRun = data;
        this.initHooks();
        this.initGate();
    }

    @Output() onSelectJob = new EventEmitter<string>();
    @Output() onSelectJobGate = new EventEmitter<GraphNode>();
    @Output() onSelectJobRun = new EventEmitter<string>();
    @Output() onSelectHook = new EventEmitter<string>();

    direction: GraphDirection = GraphDirection.HORIZONTAL;

    ready: boolean;
    hasStages = false;

    // workflow graph
    @ViewChild('svgGraph', { read: ViewContainerRef }) svgContainer: ViewContainerRef;
    graph: WorkflowV2Graph<WorkflowV2JobsGraphOrNodeOrMatrixComponent>;

    constructor(
        private _cd: ChangeDetectorRef,
        private host: ElementRef
    ) {
        const observer = new ResizeObserver(entries => {
            this.onResize();
        });
        observer.observe(this.host.nativeElement);
    }

    initHooks(): void {
        this.hooks = [];
        this.selectedHook = '';
        if (this.hooksOn) {
            if (Object.prototype.toString.call(this.hooksOn) === '[object Array]') {
                this.hooks = this.hooksOn;
            } else {
                this.hooks = Object.keys(this.hooksOn);
            }
            this.selectedHook = this._workflowRun?.event?.event_name;
        }
    }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    ngAfterViewInit(): void {
        this.ready = true;
        this._cd.detectChanges();
        this.changeDisplay();
    }

    onResize() {
        const element = this.svgContainer.element.nativeElement;
        if (!this.graph) {
            return;
        }
        this.graph.resize(element.offsetWidth, element.offsetHeight);
    }

    changeDisplay(): void {
        if (!this.ready) {
            return;
        }
        this.initGraph();
    }

    initGate(): void {
        if (!this._workflowRun || !this.nodes) {
            return;
        }
        if (!this._workflowRun.job_events || this._workflowRun.job_events.length === 0) {
            return;
        }
        this._workflowRun.job_events.forEach(e => {
            this.nodes.forEach(n => {
                if (n.sub_graph) {
                    n.sub_graph.forEach(sub => {
                        if (sub.name === e.job_id) {
                            sub.event = e;
                        }
                    });
                } else {
                    if (n.name === e.job_id) {
                        n.event = e;
                    }
                };
            });
        });
    }

    initRunJobs(): void {
        // Clean run job data on nodes
        this.nodes.forEach(n => {
            if (n.sub_graph) {
                n.sub_graph.forEach(sub => {
                    delete sub.run;
                    delete sub.runs;
                });
            } else {
                delete n.run;
                delete n.runs;
            }
        });

        // Add run job data on nodes
        this._runJobs.forEach(j => {
            this.nodes.forEach(n => {
                if (n.sub_graph) {
                    n.sub_graph.forEach(sub => {
                        if (sub.name === j.job_id) {
                            if (sub.type === GraphNodeType.Matrix) {
                                sub.runs = (sub.runs ?? []).concat(j);
                            } else {
                                sub.run = j;
                            }
                        }
                    });
                } else {
                    if (n.name === j.job_id) {
                        if (n.type === GraphNodeType.Matrix) {
                            n.runs = (n.runs ?? []).concat(j);
                        } else {
                            n.run = j;
                        }
                    }
                };
            });
        });
    }

    initGraph() {
        if (this.graph) {
            this.graph.clean();
        }
        if (!this.graph || this.graph.direction !== this.direction) {
            this.graph = new WorkflowV2Graph(this.createForkJoinNodeComponent.bind(this), this.direction,
                WorkflowV2StagesGraphComponent.minScale,
                WorkflowV2StagesGraphComponent.maxScale);
        }

        this.nodes.forEach(n => {
            let component: ComponentRef<WorkflowV2JobsGraphOrNodeOrMatrixComponent>;
            switch (n.type) {
                case GraphNodeType.Stage:
                    component = this.createSubGraphComponent(n);
                    this.graph.createNode(n.name, n.type, component, 300, 170);
                    break;
                case GraphNodeType.Matrix:
                    component = this.createJobMatrixComponent(n);
                    const alls = GraphNode.generateMatrixOptions(n.job.strategy.matrix);
                    let height = 30 * alls.length + 10 * (alls.length - 1) + 60 + 20;
                    this.graph.createNode(n.name, n.type, component, 240, height);
                    break;
                default:
                    component = this.createJobNodeComponent(n);
                    this.graph.createNode(n.name, n.type, component);
                    if (n.run) {
                        this.graph.setNodeStatus(n.name, n.run ? n.run.status : null);
                    }
                    break;
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

    clickHook(type: string): void {
        this.onSelectHook.emit(type);
    }

    createJobNodeComponent(node: GraphNode): ComponentRef<GraphJobNodeComponent> {
        const componentRef = this.svgContainer.createComponent(GraphJobNodeComponent);
        componentRef.instance.node = node;
        componentRef.instance.mouseCallback = this.nodeJobMouseEvent.bind(this);
        componentRef.changeDetectorRef.detectChanges();
        return componentRef;
    }

    createJobMatrixComponent(node: GraphNode): ComponentRef<GraphMatrixNodeComponent> {
        const componentRef = this.svgContainer.createComponent(GraphMatrixNodeComponent);
        componentRef.instance.node = node;
        componentRef.instance.mouseCallback = this.nodeJobMouseEvent.bind(this);
        componentRef.changeDetectorRef.detectChanges();
        return componentRef;
    }

    createForkJoinNodeComponent(nodes: Array<GraphNode>, type: string): ComponentRef<GraphForkJoinNodeComponent> {
        const componentRef = this.svgContainer.createComponent(GraphForkJoinNodeComponent);
        componentRef.instance.nodes = nodes;
        componentRef.instance.type = type;
        componentRef.instance.mouseCallback = this.nodeMouseEvent.bind(this);
        componentRef.changeDetectorRef.detectChanges();
        return componentRef;
    }

    createSubGraphComponent(node: GraphNode): ComponentRef<WorkflowV2JobsGraphComponent> {
        const componentRef = this.svgContainer.createComponent(WorkflowV2JobsGraphComponent);
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

    nodeMouseEvent(type: string, n: GraphNode, options?: any) {
        this.graph.nodeMouseEvent(type, n.name, options);
    }

    nodeJobMouseEvent(type: string, n: GraphNode, options?: any) {
        if (type === 'click') {
            if (options && options['jobRunID']) {
                this.onSelectJobRun.emit(options['jobRunID']);
            } else if (options && options['gateName']) {
                this.onSelectJobGate.emit(n);
            } else {
                this.onSelectJob.emit(n.name);
            }
        }
        this.graph.nodeMouseEvent(type, n.name, options);
    }

    subGraphSelectJob(type: string, n: GraphNode, options?: any): void {
        if (type === 'click') {
            this.graph.unselectAllNode();
            if (options && options['jobRunID']) {
                this.onSelectJobRun.emit(options['jobRunID']);
            } else if (options && options['gateName']) {
                this.onSelectJobGate.emit(n);
            } else {
                this.onSelectJob.emit(n.name);
            }
        }
    }

    changeDirection(): void {
        this.direction = this.direction === GraphDirection.HORIZONTAL ? GraphDirection.VERTICAL : GraphDirection.HORIZONTAL;
        this.changeDisplay();
    }
}


