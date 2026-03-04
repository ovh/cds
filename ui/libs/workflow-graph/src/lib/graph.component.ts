import {
    AfterViewInit,
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    ComponentRef,
    ElementRef,
    EventEmitter,
    HostListener,
    inject,
    Input,
    OnDestroy,
    Output,
    ViewChild,
    ViewContainerRef
} from '@angular/core';
import { GraphStageNodeComponent } from './node/stage-node.component';
import { GraphForkJoinNodeComponent } from './node/fork-join-node.components';
import { GraphJobNodeComponent } from './node/job-node.component';
import { GraphNode, GraphNodeType, NavigationGraph } from './graph.model';
import { GraphDirection, NodeMouseEvent, SelectionMode, WorkflowV2Graph } from './graph.lib';
import { load, LoadOptions } from 'js-yaml';
import { V2Workflow, V2WorkflowRun, V2WorkflowRunJob, V2WorkflowRunJobStatusIsActive } from './v2.workflow.run.model';
import { GraphMatrixNodeComponent } from './node/matrix-node.component';
import { GraphNodeAction } from './node/model';

export type WorkflowV2JobsGraphOrNodeOrMatrixComponent = GraphStageNodeComponent | GraphForkJoinNodeComponent | GraphJobNodeComponent | GraphMatrixNodeComponent;

@Component({
    standalone: false,
    selector: 'app-graph',
    templateUrl: './graph.html',
    styleUrls: ['./graph.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class GraphComponent implements AfterViewInit, OnDestroy {
    static maxScale = 15;
    static minScale = 1 / 5;

    @ViewChild('svgGraph', { read: ViewContainerRef }) svgContainer: ViewContainerRef;

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
                    type: GraphNodeType.Job,
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

    @Input() navigationDisabled: boolean = false;
    /** Whether Shift can enter selection mode (e.g. workflow run is terminated). */
    @Input() selectionDisabled: boolean = false;

    @Output() onSelectJobGate = new EventEmitter<string>();
    @Output() onConfirmJobGate = new EventEmitter<string>();
    @Output() onSelectJobRun = new EventEmitter<string>();
    @Output() onSelectJobRunRestart = new EventEmitter<string>();
    @Output() onSelectJobRunStop = new EventEmitter<string>();
    @Output() onSelectHook = new EventEmitter<string>();
    /** Emitted whenever the selection changes (toggle, lasso, or programmatic). */
    @Output() onSelectionChange = new EventEmitter<Array<string>>();
    /** Emitted when the user presses Enter while selection mode is active. */
    @Output() onSelectionValidate = new EventEmitter<Array<string>>();
    /** Emitted when selection mode is entered or exited. */
    @Output() onSelectionModeChange = new EventEmitter<boolean>();

    /** Current selection of run job IDs, managed internally. */
    selectedRunJobIds: Array<string> = [];
    /** Whether the graph is currently in selection (restart) mode. */
    selectionModeActive: boolean = false;
    nodes: Array<GraphNode> = [];
    hooks: Array<any> = [];
    selectedHook: string;
    hooksOn: any;
    selectedNodeNavigationKey: string;
    navigationGraph: NavigationGraph;
    direction: GraphDirection = GraphDirection.HORIZONTAL;
    ready: boolean;
    hasStages = false;
    graph: WorkflowV2Graph<WorkflowV2JobsGraphOrNodeOrMatrixComponent>;

    private _cd = inject(ChangeDetectorRef);
    private host = inject(ElementRef);

    constructor() {
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
        }
        this.selectedHook = this._workflowRun?.event?.event_name;
    }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    ngAfterViewInit(): void {
        this.ready = true;
        this._cd.detectChanges();
        this.changeDisplay();
    }

    @HostListener('window:keydown', ['$event'])
    handleKeyDown(event: KeyboardEvent) {
        // Selection-mode shortcuts (handled regardless of navigationDisabled)
        switch (event.key) {
            case 'Shift':
                if (this.selectionDisabled) { return; }
                if (!this.selectionModeActive) {
                    this.setSelectionModeActive(true);
                }
                this.enableLassoSelection();
                return;
            case 'Enter':
                if (this.selectionModeActive) {
                    this.onSelectionValidate.emit([...this.selectedRunJobIds]);
                    return;
                }
                break;
        }

        // Arrow / Enter navigation (guarded)
        if (!this.navigationGraph || this.navigationDisabled) { return; }
        let newSelected: string = null;
        switch (event.key) {
            case 'ArrowDown':
                newSelected = this.direction === GraphDirection.HORIZONTAL ? this.navigationGraph.getSideNext(this.selectedNodeNavigationKey) : this.navigationGraph.getNext(this.selectedNodeNavigationKey);
                break;
            case 'ArrowUp':
                newSelected = this.direction === GraphDirection.HORIZONTAL ? this.navigationGraph.getSidePrevious(this.selectedNodeNavigationKey) : this.navigationGraph.getPrevious(this.selectedNodeNavigationKey);
                break;
            case 'ArrowLeft':
                newSelected = this.direction === GraphDirection.HORIZONTAL ? this.navigationGraph.getPrevious(this.selectedNodeNavigationKey) : this.navigationGraph.getSidePrevious(this.selectedNodeNavigationKey);
                break;
            case 'ArrowRight':
                newSelected = this.direction === GraphDirection.HORIZONTAL ? this.navigationGraph.getNext(this.selectedNodeNavigationKey) : this.navigationGraph.getSideNext(this.selectedNodeNavigationKey);
                break;
            case 'Enter':
                if (this.selectedNodeNavigationKey) {
                    this.graph.activateNode(this.selectedNodeNavigationKey);
                }
                return;
            default:
                return;
        }
        if (newSelected) {
            this.selectedNodeNavigationKey = newSelected;
            this.graph.selectNode(this.selectedNodeNavigationKey);
            this.graph.centerNode(this.selectedNodeNavigationKey, true);
        }
    }

    @HostListener('window:keyup', ['$event'])
    handleKeyUp(event: KeyboardEvent) {
        if (event.key === 'Shift' && this.selectionModeActive) {
            this.disableLassoSelection();
            if (this.selectedRunJobIds.length === 0) {
                this.setSelectionModeActive(false);
            }
        }
    }

    unSelect() {
        this.graph.selectNode(null);
        this.graph.centeredNode = null;
    }

    onResize() {
        this.resize();
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
                    sub.type = GraphNodeType.Job;
                    delete sub.run;
                    delete sub.runs;
                });
            } else {
                n.type = GraphNodeType.Job;
                delete n.run;
                delete n.runs;
            }
        });

        // Add run job data on nodes
        this._runJobs.forEach(j => {
            let isMatrix = false;
            if (j.matrix) {
                isMatrix = true
            }
            this.nodes.forEach(n => {
                if (n.sub_graph) {
                    n.sub_graph.forEach(sub => {
                        if (sub.name === j.job_id) {
                            if (isMatrix) {
                                sub.type = GraphNodeType.Matrix;
                                sub.runs = (sub.runs ?? []).concat(j);
                            } else {
                                sub.run = j;
                            }
                        }
                    });
                } else {
                    if (n.name === j.job_id) {
                        if (isMatrix) {
                            n.type = GraphNodeType.Matrix;
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
                GraphComponent.minScale,
                GraphComponent.maxScale);
        }
        this.navigationGraph = new NavigationGraph(this.nodes, this.direction);

        this.nodes.forEach(n => {
            let component: ComponentRef<WorkflowV2JobsGraphOrNodeOrMatrixComponent>;
            switch (n.type) {
                case GraphNodeType.Stage:
                    const r = this.createSubGraphComponent(n);
                    component = r;
                    this.graph.createNode(n.name, n, component, r.instance.graph.graph.graph().height + 2 * WorkflowV2Graph.marginSubGraph, r.instance.graph.graph.graph().width + 2 * WorkflowV2Graph.marginSubGraph);
                    break;
                case GraphNodeType.Matrix:
                    component = this.createJobMatrixComponent(n);
                    this.graph.createNode(n.name, n, component);
                    break;
                default:
                    component = this.createJobNodeComponent(n);
                    this.graph.createNode(n.name, n, component);
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

        if (this.selectedNodeNavigationKey) {
            this.graph.selectNode(this.selectedNodeNavigationKey);
        }

        const runActive = (this._runJobs ?? []).filter(j => V2WorkflowRunJobStatusIsActive(j.status)).length > 0;
        this.graph.setRunActive(runActive);

        this._cd.markForCheck();
    }

    public resize() {
        if (!this.svgContainer?.element?.nativeElement?.offsetWidth || !this.svgContainer?.element?.nativeElement?.offsetHeight) {
            return;
        }
        this.graph.resize(this.svgContainer.element.nativeElement.offsetWidth, this.svgContainer.element.nativeElement.offsetHeight);
    }

    clickOrigin() {
        this.graph.center();
    }

    clickHook(type: string): void {
        this.onSelectHook.emit(type);
    }

    createJobNodeComponent(node: GraphNode): ComponentRef<GraphJobNodeComponent> {
        const componentRef = this.svgContainer.createComponent(GraphJobNodeComponent);
        componentRef.instance.node = node;
        componentRef.instance.actionCallback = this.onNodeAction.bind(this);
        componentRef.changeDetectorRef.detectChanges();
        return componentRef;
    }

    createJobMatrixComponent(node: GraphNode): ComponentRef<GraphMatrixNodeComponent> {
        const componentRef = this.svgContainer.createComponent(GraphMatrixNodeComponent);
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

    createSubGraphComponent(node: GraphNode): ComponentRef<GraphStageNodeComponent> {
        const componentRef = this.svgContainer.createComponent(GraphStageNodeComponent);
        componentRef.instance.graphNode = node;
        componentRef.instance.direction = this.direction;
        componentRef.instance.centerCallback = (n: GraphNode) => { this.graph.centerStage(`node-${node.name}`); };
        componentRef.instance.actionCallback = this.onNodeAction.bind(this);
        componentRef.changeDetectorRef.detectChanges();
        return componentRef;
    }

    onNodeAction(type: GraphNodeAction, n: GraphNode, options?: any) {
        switch (type) {
            case GraphNodeAction.Enter:
                this.graph.nodeMouseEvent(NodeMouseEvent.Enter, n.name, options);
                break;
            case GraphNodeAction.Out:
                this.graph.nodeMouseEvent(NodeMouseEvent.Out, n.name, options);
                break;
            case GraphNodeAction.Click:
                const baseKey = (n.job && n.job.stage) ? `${n.job.stage}-${n.name}` : n.name;
                this.selectedNodeNavigationKey = baseKey
                if (n.type === GraphNodeType.Matrix) { this.selectedNodeNavigationKey += '-' + options.jobMatrixKey; }
                this.graph.selectNode(this.selectedNodeNavigationKey);
                this.graph.centerNode(baseKey);
                if (options && options['jobRunID']) {
                    this.onSelectJobRun.emit(options['jobRunID']);
                }
                break;
            case GraphNodeAction.ClickGate:
                if (options && options['gateName']) {
                    this.onSelectJobGate.emit(n.name);
                }
                break;
            case GraphNodeAction.ClickConfirmGate:
                this.onConfirmJobGate.emit(n.name);
                break;
            case GraphNodeAction.ClickRestart:
                this.onSelectJobRunRestart.emit(options['jobRunID']);
                break;
            case GraphNodeAction.ClickStop:
                this.onSelectJobRunStop.emit(options['jobRunID']);
                break;
            case GraphNodeAction.ToggleSelection:
                this.handleToggleSelection(options['runJobId'], options['selected']);
                break;
        }
    }

    changeDirection(): void {
        this.direction = this.direction === GraphDirection.HORIZONTAL ? GraphDirection.VERTICAL : GraphDirection.HORIZONTAL;
        this.changeDisplay();
    }

    setSelectionModeActive(active: boolean): void {
        this.selectionModeActive = active;
        if (!active) {
            this.selectedRunJobIds = [];
            this.navigationGraph.getAllJobNavigationKeys().forEach(navKey => {
                this.graph.setNodeSelectionMode(navKey, SelectionMode.Disabled);
            });
            this.onSelectionChange.emit([]);
        } else {
            this.navigationGraph.getAllJobNavigationKeys().forEach(navKey => {
                this.graph.setNodeSelectionMode(navKey, SelectionMode.Active);
            });
        }
        this.onSelectionModeChange.emit(active);
    }

    /**
     * Apply a selection of run job IDs, automatically computing which nodes
     * should be blocked (descendants of selected jobs) and pruning blocked
     * IDs from the selection.
     *
     * Steps:
     *  1. Compute blocked descendant navigation keys via the NavigationGraph.
     *  2. Remove blocked run job IDs from the selection.
     *  3. Reset all nodes to Active, then mark blocked nodes as Blocked.
     *  4. Broadcast the pruned selection to every node via setSelected().
     *  5. Store the pruned selection and emit onSelectionChange.
     */
    updateSelection(selectedRunJobIds: Array<string>) {
        // 1. Compute blocked descendant navigation keys
        const disabledNavigationKeys: Array<string> = [];
        for (const runJobId of selectedRunJobIds) {
            for (const descendantKey of this.navigationGraph.getDescendantNavigationKeys(runJobId)) {
                if (!disabledNavigationKeys.includes(descendantKey)) {
                    disabledNavigationKeys.push(descendantKey);
                }
            }
        }

        // 2. Prune blocked run job IDs from the selection
        this.selectedRunJobIds = selectedRunJobIds.filter(id => {
            const navKey = this.navigationGraph.getNavigationKey(id);
            return !navKey || !disabledNavigationKeys.includes(navKey);
        });

        // 3. Reset all nodes to Active, then block descendants
        this.navigationGraph.getAllJobNavigationKeys().forEach(navKey => {
            this.graph.setNodeSelectionMode(navKey, SelectionMode.Active);
        });
        for (const navKey of disabledNavigationKeys) {
            this.graph.setNodeSelectionMode(navKey, SelectionMode.Blocked);
        }

        // 4. Broadcast the pruned selection
        this.graph.updateSelection(this.selectedRunJobIds);

        // 5. Notify parent
        this.onSelectionChange.emit([...this.selectedRunJobIds]);
    }

    /** Handle a toggle selection event from a node (internal). */
    private handleToggleSelection(runJobId: string, selected: boolean): void {
        let ids = [...this.selectedRunJobIds];
        if (selected) {
            if (!ids.includes(runJobId)) { ids.push(runJobId); }
        } else {
            ids = ids.filter(id => id !== runJobId);
        }
        this.updateSelection(ids);
    }

    /** Handle lasso selection diffs (internal). */
    private handleLassoSelection(diff: {
        added: string[], removed: string[], covered: Array<string>
    }): void {
        // Apply additions
        let ids = [...this.selectedRunJobIds];
        diff.added.forEach(id => {
            if (!ids.includes(id)) { ids.push(id); }
        });
        // Apply removals
        ids = ids.filter(id => !diff.removed.includes(id));

        this.updateSelection(ids);

        // Reconcile: run jobs still covered by the lasso that are neither
        // selected nor blocked should be selected.
        const disabledNavigationKeys: Array<string> = [];
        for (const runJobId of this.selectedRunJobIds) {
            for (const descendantKey of this.navigationGraph.getDescendantNavigationKeys(runJobId)) {
                if (!disabledNavigationKeys.includes(descendantKey)) {
                    disabledNavigationKeys.push(descendantKey);
                }
            }
        }
        let reconciled = false;
        diff.covered.forEach(runJobId => {
            if (this.selectedRunJobIds.includes(runJobId)) { return; }
            if (!this.navigationGraph) { return; }
            const navKey = this.navigationGraph.getNavigationKey(runJobId);
            if (!navKey || disabledNavigationKeys.includes(navKey)) { return; }
            this.selectedRunJobIds.push(runJobId);
            reconciled = true;
        });

        if (reconciled) {
            this.updateSelection([...this.selectedRunJobIds]);
        }
    }

    enableLassoSelection(): void {
        this.graph.enableLasso((diff) => {
            this.handleLassoSelection(diff);
        });
    }

    disableLassoSelection(): void {
        this.graph.disableLasso();
    }
}


