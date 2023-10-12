import {
    AfterViewInit,
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    ComponentRef, ElementRef,
    EventEmitter,
    HostListener,
    Input,
    OnDestroy,
    Output,
    ViewChild,
    ViewContainerRef
} from '@angular/core';
import {AutoUnsubscribe} from 'app/shared/decorator/autoUnsubscribe';
import {ProjectV2WorkflowJobsGraphComponent} from "./jobs-graph.component";
import {ProjectV2WorkflowForkJoinNodeComponent} from "./node/fork-join-node.components";
import {ProjectV2WorkflowJobNodeComponent} from "./node/job-node.component";
import {GraphNode, GraphNodeTypeJob, GraphNodeTypeStage, JobRun} from "./graph.model";
import {GraphDirection, WorkflowV2Graph} from "./graph.lib";
import {load, LoadOptions} from "js-yaml";
import {V2WorkflowRun, V2WorkflowRunJob} from "../../../../../../../model/v2.workflow.run.model";

export type WorkflowV2JobsGraphOrNodeComponent = ProjectV2WorkflowJobsGraphComponent |
    ProjectV2WorkflowForkJoinNodeComponent | ProjectV2WorkflowJobNodeComponent;

@Component({
    selector: 'app-stages-graph',
    templateUrl: './stages-graph.html',
    styleUrls: ['./stages-graph.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectV2WorkflowStagesGraphComponent implements AfterViewInit, OnDestroy {
    static maxScale = 15;
    static minScale = 1 / 5;

    nodes: Array<GraphNode> = [];
    hooks: Array<any> = [];
    selectedHook: string;
    hooksOn: any;

    @Input() set workflow(data: any) {
        let workflow: any;
        try {
            workflow = load(data && data !== '' ? data : '{}', <LoadOptions>{
                onWarning: (e) => {
                }
            });
        } catch (e) {
            console.error("Invalid workflow:", data, e)
        }
        this.hasStages = !!workflow && !!workflow["stages"];
        this.nodes = [];
        if (workflow && workflow["stages"]) {
            this.nodes.push(...Object.keys(workflow["stages"])
                .map(k => <GraphNode>{
                    name: k,
                    depends_on: workflow["stages"][k]?.needs,
                    sub_graph: [],
                    type: GraphNodeTypeStage
                }));
        }
        if (workflow && workflow["jobs"] && Object.keys(workflow["jobs"]).length > 0) {
            Object.keys(workflow["jobs"]).forEach(k => {
                let job = workflow.jobs[k];
                let node = <GraphNode>{name: k, depends_on: job?.needs, type: GraphNodeTypeJob};
                if (this.jobRuns[k]) {
                    node.run = this.jobRuns[k];
                }

                if (job?.stage) {
                    for (let i = 0; i < this.nodes.length; i++) {
                        if (this.nodes[i].name === job.stage && this.nodes[i].type === GraphNodeTypeStage) {
                            this.nodes[i].sub_graph.push(node);
                            break;
                        }
                    }
                } else {
                    this.nodes.push(node);
                }
            });
        }
        this.hooks = [];
        this.selectedHook = '';
        if (workflow && workflow['on']) {
            this.hooksOn = workflow['on'];
            this.initHooks();
        }
        this.changeDisplay();
        this._cd.markForCheck();
    }

    jobRuns: { [name: string]: V2WorkflowRunJob } = {};

    @Input() set runJobs(data: Array<V2WorkflowRunJob>) {
        if (!data) {
            this.jobRuns = {}
            return;
        }
        this.jobRuns = {};
        data.forEach(j => {
            this.jobRuns[j.job_id] = j;
        });

        if (this.nodes) {
            this.nodes.forEach(n => {
                if (this.jobRuns[n.name]) {
                    n.run = this.jobRuns[n.name];
                }
            })
        }
        this.initGraph();
    }

    _workflowRun: V2WorkflowRun
    @Input() set workflowRun(data: V2WorkflowRun) {
        this._workflowRun = data;
        this.initHooks();
    }

    @Output() onSelectJob = new EventEmitter<string>();

    direction: GraphDirection = GraphDirection.HORIZONTAL;

    ready: boolean;
    hasStages = false;

    // workflow graph
    @ViewChild('svgGraph', {read: ViewContainerRef}) svgContainer: ViewContainerRef;
    graph: WorkflowV2Graph<WorkflowV2JobsGraphOrNodeComponent>;

    constructor(private _cd: ChangeDetectorRef, private host: ElementRef) {
        const observer = new ResizeObserver(entries => {
            this.onResize();
        });
        observer.observe(this.host.nativeElement);
    }

    initHooks(): void {
        this.hooks = [];
        this.selectedHook = '';
        if (this.hooksOn) {
            if(Object.prototype.toString.call(this.hooksOn) === '[object Array]') {
                this.hooks = this.hooksOn;
            } else {
                this.hooks = Object.keys(this.hooksOn);
            }

            if (this._workflowRun) {
                if (this._workflowRun.event.workflow_update) {
                    this.selectedHook = 'workflow_update';
                } else if (this._workflowRun.event.model_update) {
                    this.selectedHook = 'model_update';
                } else if (this._workflowRun.event.git) {
                    this.selectedHook = this._workflowRun.event.git.event_name;
                }
            }
        }
    }

    static isJobsGraph = (component: WorkflowV2JobsGraphOrNodeComponent): component is ProjectV2WorkflowJobsGraphComponent => {
        if ((component as ProjectV2WorkflowJobsGraphComponent).direction) {
            return true;
        }
        return false;
    };

    ngOnDestroy(): void {
    } // Should be set to use @AutoUnsubscribe with AOT

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

    initGraph() {
        if (this.graph) {
            this.graph.clean();
        }
        if (!this.graph || this.graph.direction !== this.direction) {
            this.graph = new WorkflowV2Graph(this.createForkJoinNodeComponent.bind(this), this.direction,
                ProjectV2WorkflowStagesGraphComponent.minScale,
                ProjectV2WorkflowStagesGraphComponent.maxScale);
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

    createJobNodeComponent(node: GraphNode): ComponentRef<ProjectV2WorkflowJobNodeComponent> {
        const componentRef = this.svgContainer.createComponent(ProjectV2WorkflowJobNodeComponent);
        componentRef.instance.node = node;
        componentRef.instance.mouseCallback = this.nodeJobMouseEvent.bind(this);
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

    createSubGraphComponent(node: GraphNode): ComponentRef<ProjectV2WorkflowJobsGraphComponent> {
        const componentRef = this.svgContainer.createComponent(ProjectV2WorkflowJobsGraphComponent);
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
