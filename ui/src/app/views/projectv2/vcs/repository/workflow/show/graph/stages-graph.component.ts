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
import {AutoUnsubscribe} from 'app/shared/decorator/autoUnsubscribe';
import {ProjectV2WorkflowJobsGraphComponent} from "./jobs-graph.component";
import {ProjectV2WorkflowForkJoinNodeComponent} from "./node/fork-join-node.components";
import {ProjectV2WorkflowJobNodeComponent} from "./node/job-node.component";
import {GraphNode, GraphNodeTypeGate, GraphNodeTypeJob, GraphNodeTypeStage} from "./graph.model";
import {GraphDirection, WorkflowV2Graph} from "./graph.lib";
import {load, LoadOptions} from "js-yaml";
import {V2WorkflowRun, V2WorkflowRunJob} from "app/model/v2.workflow.run.model";
import {ProjectV2WorkflowGateNodeComponent} from "./node/gate-node.component";

export type WorkflowV2JobsGraphOrNodeComponent = ProjectV2WorkflowJobsGraphComponent |
    ProjectV2WorkflowForkJoinNodeComponent | ProjectV2WorkflowJobNodeComponent | ProjectV2WorkflowGateNodeComponent;

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
            let matrixJobs = new Map<string, any[]>()
            Object.keys(workflow["jobs"]).forEach(jobID => {
                let job = workflow.jobs[jobID];
                let expandMatrixJobs = new Array<string>();
                if (job?.strategy?.matrix) {
                    let keys = Object.keys(job.strategy.matrix);
                    let alls = new Array<Map<string,string>>();
                    this.generateMatrix(job.strategy.matrix, keys, 0, new Map<string,string>(), alls)
                    alls.forEach(m => {
                        let suffix = "";
                        let mapKeys = Array.from(m.keys()).sort();
                        mapKeys.forEach((k, index) => {
                            if (index !== 0) {
                                suffix += ',';
                            }
                            suffix += m.get(k);
                        });
                        let newJob = Object.assign({}, job);
                        newJob.matrixName = jobID + '-' + suffix.replaceAll('/', '-');
                        expandMatrixJobs.push(newJob);
                    });
                    matrixJobs.set(jobID, expandMatrixJobs);
                }
            });
            Object.keys(workflow["jobs"]).forEach(k => {
                let job = workflow.jobs[k];
                let gateNode = undefined;
                if (job?.gate && job.gate !== '') {
                    gateNode = <GraphNode>{name: `${job.gate}-${k}`, type: GraphNodeTypeGate, gateChild: k, gateName: `${job.gate}`}
                }
                if (matrixJobs.has(k)) {
                    matrixJobs.get(k).forEach(j => {
                        let node = <GraphNode>{name: j.matrixName, depends_on: this.getJobNeeds(j, matrixJobs), type: GraphNodeTypeJob};
                        if (job?.stage) {
                            for (let i = 0; i < this.nodes.length; i++) {
                                if (this.nodes[i].name === job.stage && this.nodes[i].type === GraphNodeTypeStage) {
                                    this.nodes[i].sub_graph.push(node);
                                    if (gateNode) {
                                        this.nodes[i].sub_graph.push(gateNode);
                                    }
                                    break;
                                }
                            }
                        } else {
                            this.nodes.push(node);
                            if (gateNode) {
                                this.nodes.push(gateNode);
                            }
                        }
                    });
                } else {
                    let node = <GraphNode>{name: k, depends_on: this.getJobNeeds(job, matrixJobs), type: GraphNodeTypeJob};
                    node.run = this.jobRuns[k];
                    if (job?.stage) {
                        for (let i = 0; i < this.nodes.length; i++) {
                            if (this.nodes[i].name === job.stage && this.nodes[i].type === GraphNodeTypeStage) {
                                this.nodes[i].sub_graph.push(node);
                                if (gateNode) {
                                    this.nodes[i].sub_graph.push(gateNode);
                                }
                                break;
                            }
                        }
                    } else {
                        this.nodes.push(node);
                        if (gateNode) {
                            this.nodes.push(gateNode);
                        }
                    }
                }
            });
            this.initRunJobs();
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
            if (this.nodes) {
                this.nodes.forEach(n => {
                    if (this.hasStages) {
                        n.sub_graph.forEach(sub => {
                            delete sub.run;
                        });
                    } else {
                        delete n.run;
                    }
                })
            }
            return;
        }
        this.jobRuns = {};
        data.forEach(j => {
            if (j.matrix && Object.keys(j.matrix).length > 0) {
                let mapKeys = Object.keys(j.matrix).sort();
                let suffix = "";
                mapKeys.forEach((k, index) => {
                    if (index !== 0) {
                        suffix += ',';
                    }
                    suffix += j.matrix[k];
                });
                this.jobRuns[j.job_id + '-' + suffix.replaceAll('/', '-')] = j;
            } else {
                this.jobRuns[j.job_id] = j;
            }
        });
        this.initRunJobs();
        this.initGraph();
    }

    _workflowRun: V2WorkflowRun
    @Input() set workflowRun(data: V2WorkflowRun) {
        this._workflowRun = data;
        this.initHooks();
    }

    @Output() onSelectJob = new EventEmitter<string>();
    @Output() onSelectJobGate = new EventEmitter<GraphNode>();
    @Output() onSelectJobRun = new EventEmitter<string>();

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

    getJobNeeds(j: {}, matrixJobs: Map<string, Array<{}>>) {
        if (!j['needs']) {
            return [];
        }
        let needs = [];
        <string[]>j['needs'].forEach(n => {
            if (!matrixJobs.has(n)) {
                needs.push(n);
            } else {
                matrixJobs.get(n).forEach(mj => {
                    needs.push(mj['matrixName'].replaceAll('/', '-'));
                });
            }
        });
        return needs;
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

    initRunJobs(): void {
        if (!this.jobRuns || !this.nodes) {
            return;
        }
        this.nodes.forEach(n => {
            if (this.hasStages) {
                n.sub_graph.forEach(sub => {
                    if (this.jobRuns[sub.name]) {
                        sub.run = this.jobRuns[sub.name];
                    }
                });
            } else {
                if (this.jobRuns[n.name]) {
                    n.run = this.jobRuns[n.name];
                }
            }
        })
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
                this.graph.createNode(n.name, GraphNodeTypeStage, this.createSubGraphComponent(n),
                    null, 300, 169);
            } else {
                switch (n.type) {
                    case GraphNodeTypeGate:
                        this.graph.createGate(n, GraphNodeTypeGate, this.createGateNodeComponent(n),
                            n.run ? n.run.status : null);
                        break;
                    default:
                        this.graph.createNode(n.name, GraphNodeTypeJob, this.createJobNodeComponent(n),
                            n.run ? n.run.status : null);
                }
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

    createGateNodeComponent(node: GraphNode): ComponentRef<ProjectV2WorkflowGateNodeComponent> {
        const componentRef = this.svgContainer.createComponent(ProjectV2WorkflowGateNodeComponent);
        componentRef.instance.node = node;
        componentRef.instance.mouseCallback = this.nodeJobMouseEvent.bind(this);
        componentRef.changeDetectorRef.detectChanges();
        return componentRef;
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
            if (n.gateName && n.gateName !== '') {
                this.onSelectJobGate.emit(n);
            }
            if (n.run) {
                this.onSelectJob.emit(n.run.id);
            } else {
                this.onSelectJob.emit(n.name);
            }

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

    generateMatrix(matrix: {[key: string]:string[] }, keys: string[], keyIndex: number, current: Map<string,string>, alls: Map<string,string>[]) {
        if (current.size == keys.length) {
            let combi = new Map<string, string>();
            current.forEach((v, k) => {
                combi.set(k, v);
            });
            alls.push(combi);
            return;
        }
        let key = keys[keyIndex];
        let values = matrix[key];
        values.forEach(v => {
            current.set(key, v);
            this.generateMatrix(matrix, keys, keyIndex+1, current, alls);
            current.delete(key);
        });
    }
}


