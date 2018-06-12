import {
    AfterViewInit,
    ChangeDetectorRef,
    Component,
    ComponentFactoryResolver,
    ComponentRef,
    EventEmitter,
    HostListener,
    Input,
    Output,
    ViewChild,
    ViewContainerRef
} from '@angular/core';
import * as d3 from 'd3';
import * as dagreD3 from 'dagre-d3';
import {SemanticDimmerComponent} from 'ng-semantic/ng-semantic';
import {Project} from '../../../model/project.model';
import {Workflow, WorkflowNode, WorkflowNodeJoin} from '../../../model/workflow.model';
import {WorkflowStore} from '../../../service/workflow/workflow.store';
import {AutoUnsubscribe} from '../../../shared/decorator/autoUnsubscribe';
import {WorkflowJoinComponent} from '../../../shared/workflow/join/workflow.join.component';
import {WorkflowNodeHookComponent} from '../../../shared/workflow/node/hook/hook.component';
import {WorkflowNodeComponent} from '../../../shared/workflow/node/workflow.node.component';
import {WorkflowCoreService} from '../../../service/workflow/workflow.core.service';
import {WorkflowNodeRun, WorkflowRun} from '../../../model/workflow.run.model';
import {Subscription} from 'rxjs';

@Component({
    selector: 'app-workflow-graph',
    templateUrl: './workflow.graph.html',
    styleUrls: ['./workflow.graph.scss'],
    entryComponents: [
        WorkflowNodeComponent,
        WorkflowJoinComponent,
        WorkflowNodeHookComponent
    ]
})
@AutoUnsubscribe()
export class WorkflowGraphComponent implements AfterViewInit {

    workflow: Workflow;
    _workflowRun: WorkflowRun;
    creationMode = 'graphical';

    @Input('workflowData')
    set workflowData(data: Workflow) {
        // check if nodes have to be updated
        this.workflow = data;
        this.nodeHeight = 78;
        if (data.forceRefresh) {
            this.nodesComponent = new Map<string, ComponentRef<WorkflowNodeComponent>>();
            this.joinsComponent = new Map<string, ComponentRef<WorkflowJoinComponent>>();
            this.hooksComponent = new Map<number, ComponentRef<WorkflowNodeHookComponent>>();
        }
        this.calculateDynamicWidth();
        this.changeDisplay();
    }

    @Input('workflowRun')
    set workflowRun(data: WorkflowRun) {
        if (data) {
            // check if nodes have to be updated
            this._workflowRun = data;
            this.workflow = data.workflow;
            this.nodeHeight = 78;
            if (!this.previousWorkflowRunId || this.previousWorkflowRunId !== data.id) {
                this.calculateDynamicWidth();
            }
            this.previousWorkflowRunId = data.id;
            this.changeDisplay();
        }
    }

    @Input() project: Project;

    @Input('direction')
    set direction(data: string) {
        this._direction = data;
        this._workflowStore.setDirection(this.project.key, this.workflow.name, this.direction);
        this.calculateDynamicWidth();
        this.changeDisplay();
    }

    get direction() {
        return this._direction
    }

    @Output() deleteJoinSrcEvent = new EventEmitter<{ source, target }>();
    @Output() addSrcToJoinEvent = new EventEmitter<{ source, target }>();

    ready: boolean;
    _direction: string;

    // workflow graph
    @ViewChild('svgGraph', {read: ViewContainerRef}) svgContainer;
    g: dagreD3.graphlib.Graph;
    render = new dagreD3.render();
    svgWidth: number = window.innerWidth;
    svgHeight: number = window.innerHeight;

    @ViewChild('dimmer')
    dimmer: SemanticDimmerComponent;

    linkWithJoin = false;
    nodeToLink: WorkflowNode;
    previousWorkflowRunId = 0;

    nodesComponent = new Map<string, ComponentRef<WorkflowNodeComponent>>();
    joinsComponent = new Map<string, ComponentRef<WorkflowJoinComponent>>();
    hooksComponent = new Map<number, ComponentRef<WorkflowNodeHookComponent>>();

    linkJoinSubscription: Subscription;

    nodeWidth: number;
    nodeHeight: number;

    constructor(private componentFactoryResolver: ComponentFactoryResolver, private _cd: ChangeDetectorRef,
                private _workflowStore: WorkflowStore, private _workflowCore: WorkflowCoreService) {
        this.linkJoinSubscription = this._workflowCore.getLinkJoinEvent().subscribe(n => {
            if (n) {
                this.nodeToLink = n;
                this.toggleLinkJoin(true);
            } else {
              this.toggleLinkJoin(false);
            }
        });
    }

    @HostListener('window:resize', ['$event'])
    onResize(event) {
        this.resize(event);
    }

    resize(event) {
        // Resize svg
        let svg = d3.select('svg');
        let inner = d3.select('svg g');

        let w = 0;
        inner.each(function () {
            w = this.getBBox().width;
        });
        this.svgWidth = w + 30;

        this.svgHeight = this.g.graph().height + 40;
        svg.attr('height', this.svgHeight);
    }

    ngAfterViewInit(): void {
        this.ready = true;
        this.changeDisplay();
        this.resize(null);
        this._cd.detectChanges();
    }

    changeDisplay(): void {
        if (!this.ready && this.workflow) {
            return;
        }
        this.initWorkflow();
    }

    initWorkflow() {
        // https://github.com/cpettitt/dagre/wiki#configuring-the-layout
        this.g = new dagreD3.graphlib.Graph().setGraph({align: 'UL', rankdir: this.direction, nodesep: 10});

        // Create all nodes
        if (this.workflow.root) {
            this.createNode(this.workflow.root);
        }
        if (this.workflow.joins) {
            this.workflow.joins.forEach(j => {
                this.createJoin(j);
            });

        }

        // Add our custom arrow (a hollow-point)
        this.createCustomArrow();

        // Setup transition
        this.g.graph().transition = function (selection) {
            return selection.transition().duration(100);
        };

        // Run the renderer. This is what draws the final graph.
        this.render(d3.select('svg g'), this.g);

        // Add listener on graph element
        this.addListener(d3.select('svg'));
        this.svgHeight = this.g.graph().height + 40;
        this.svgWidth = this.g.graph().width;
    }

    private createCustomArrow() {
        this.render.arrows()['customArrow'] = (parent, id, edge, type) => {
            let marker = parent.append('marker')
                .attr('id', id)
                .attr('viewBox', '0 0 10 10')
                .attr('refX', 7)
                .attr('refY', 5)
                .attr('markerUnits', 'strokeWidth')
                .attr('markerWidth', 4)
                .attr('markerHeight', 3)
                .attr('orient', 'auto');

            let path = marker.append('path')
                .attr('d', 'M 0 0 L 10 5 L 0 10 z')
                .style('stroke-width', 1)
                .style('stroke-dasharray', '1,0');
            dagreD3['util'].applyStyle(path, edge[type + 'Style']);
        };
    }

    private addListener(svg: d3.Selection<any>) {
        svg.selectAll('g.edgePath').on('click', d => {
            if (this.linkWithJoin) {
                return;
            }

            // Node Join Src
            if (d.v.indexOf('node-') === 0 && d.w.indexOf('join-') === 0) {
                this.deleteJoinSrcEvent.emit({source: d.v, target: d.w});
            }
        });
    }

    private calculateDynamicWidth() {
        let nbofNodes = 1;
        switch (this.direction) {
            case 'LR':
                let mapDeep = new Map<string, number>();
                mapDeep.set(this.workflow.root.ref, 1);
                this.getWorkflowNodeDeep(this.workflow.root, mapDeep);
                this.getWorkflowJoinDeep(mapDeep);
                nbofNodes = Math.max(...Array.from(mapDeep.values()));
                break;
            default:
                nbofNodes = this.getWorkflowMaxNodeByLevel(this.workflow.root, nbofNodes);
                nbofNodes = this.getWorkflowJoinMaxNodeByLevel(nbofNodes);
                break;
        }

        let windowsWidth = window.innerWidth - 250; // sidebar width

        this.nodeWidth = Math.floor(windowsWidth * .75 / nbofNodes);
        if (this.nodeWidth < 200) {
            this.nodeWidth = 200;
        }
    }

    createEdge(from: string, to: string, options: {}): void {
        options['arrowhead'] = 'customArrow';
        this.g.setEdge(from, to, options);
    }

    createJoin(join: WorkflowNodeJoin): void {
        let componentRef = this.joinsComponent.get(join.ref);
        if (!componentRef) {
            let nodeComponentFactory = this.componentFactoryResolver.resolveComponentFactory(WorkflowJoinComponent);
            componentRef = nodeComponentFactory.create(this.svgContainer.parentInjector);
            componentRef.instance.workflow = this.workflow;
            componentRef.instance.join = join;
            componentRef.instance.project = this.project;
            componentRef.instance.disabled = this.linkWithJoin;

            if (this._workflowRun) {
                componentRef.instance.readonly = true;
            }

            componentRef.instance.selectEvent.subscribe(j => {
                if (this.linkWithJoin && this.nodeToLink) {
                    this.addSrcToJoinEvent.emit({source: this.nodeToLink, target: j});
                }
            });
            this.joinsComponent.set(join.ref, componentRef);
        }

        this.svgContainer.insert(componentRef.hostView, 0);
        this.g.setNode('join-' + join.ref, <any>{
            shape: 'circle',
            label: () => {
                componentRef.location.nativeElement.style.width = '100%';
                componentRef.location.nativeElement.style.height = '100%';
                return componentRef.location.nativeElement;
            },
            labelStyle: 'width: 40px; height: 40px',
            width: 20,
            height: 20
        });

        if (join.source_node_ref) {
            join.source_node_ref.forEach((nodeRef, id) => {
                let style =  'black;';
                if (Array.isArray(join.source_node_id) && join.source_node_id.length && join.source_node_id[id]) {
                    style = this.getJoinSrcStyle(join.source_node_id[id]) + ';';
                }
                this.createEdge('node-' + nodeRef, 'join-' + join.ref, { style: 'stroke: ' + style});
            });
        }

        if (join.triggers) {
            join.triggers.forEach((t, id) => {
                this.createNode(t.workflow_dest_node);
                let options = {
                    id: 'trigger-' + id,
                    style: 'stroke: ' + this.getJoinTriggerColor(t.id) + ';'
                };
                this.createEdge('join-' + join.ref, 'node-' + t.workflow_dest_node.ref, options);
            });
        }
    }

    getJoinSrcStyle(nodeID: number): string {
        if (this._workflowRun && this._workflowRun.nodes && this._workflowRun.nodes[nodeID] && this._workflowRun.nodes[nodeID].length > 0) {
            switch (this._workflowRun.nodes[nodeID][0].status) {
                case 'Success':
                case 'Fail':
                    return '#21BA45';
            }
        }
        return 'black';
    }

    createHookNode(node: WorkflowNode): void {
        if (!node.hooks || node.hooks.length === 0) {
            return;
        }

        node.hooks.forEach((h, index) => {
            let hookId = index;
            if (h.id) {
              hookId = h.id;
            }
            let componentRef = this.hooksComponent.get(hookId);
            if (!componentRef) {
                let hookComponent = this.componentFactoryResolver.resolveComponentFactory(WorkflowNodeHookComponent);
                componentRef = hookComponent.create(this.svgContainer.parentInjector);
                componentRef.instance.hook = h;
                componentRef.instance.workflow = this.workflow;
                componentRef.instance.project = this.project;
                componentRef.instance.node = node;
                this.hooksComponent.set(hookId, componentRef);
            }

            this.svgContainer.insert(componentRef.hostView, 0);
            this.g.setNode(
                'hook-' + node.ref + '-' + hookId, <any>{
                    label: () => {
                        componentRef.location.nativeElement.style.width = '100%';
                        componentRef.location.nativeElement.style.height = '100%';
                        return componentRef.location.nativeElement;
                    },
                    labelStyle: 'width: 40px; height: 40px',
                    width: 20,
                    height: 20
                }
            );

            let options = {
                id: 'hook-' + node.ref + '-' + hookId
            };
            this.createEdge('hook-' + node.ref + '-' + hookId, 'node-' + node.ref, options);
        });
    }

    createNode(node: WorkflowNode): void {
        let componentRef = this.nodesComponent.get(node.ref);
        if (!componentRef) {
            componentRef = this.createNodeComponent(node);
            this.nodesComponent.set(node.ref, componentRef);
        }

        this.svgContainer.insert(componentRef.hostView, 0);
        this.g.setNode('node-' + node.ref, <any>{
            label: () => {
                componentRef.location.nativeElement.style.width = '97%';
                componentRef.location.nativeElement.style.height = '100%';
                return componentRef.location.nativeElement;
            },
            labelStyle: 'width: ' + this.nodeWidth + 'px; height: ' + this.nodeHeight + 'px',
            width: this.nodeWidth,
            height: this.nodeHeight
        });

        this.createHookNode(node);

        if (node.triggers) {
            node.triggers.forEach(t => {
                this.createNode(t.workflow_dest_node);
                let options = {
                    id: 'trigger-' + t.id,
                    style: 'stroke: ' + this.getTriggerColor(node, t.id) + ';'
                };
                this.createEdge('node-' + node.ref, 'node-' + t.workflow_dest_node.ref, options);
            });
        }
    }

    getJoinTriggerColor(triggerID: number): string {
        if (this._workflowRun && this._workflowRun.join_triggers_run) {
            if (this._workflowRun.join_triggers_run[triggerID]) {
                switch (this._workflowRun.join_triggers_run[triggerID].status) {
                    case 'Success':
                    case 'Warning':
                        return '#21BA45';
                    case 'Fail':
                        return '#FF4F60';
                }
            }
        }
        return '#000000';
    }

    getTriggerColor(node: WorkflowNode, triggerId: number): string {
        if (this._workflowRun && this._workflowRun.nodes && node) {
            if (this._workflowRun.nodes[node.ref]) {
                let lastRun = <WorkflowNodeRun>this._workflowRun.nodes[node.ref][0];
                if (lastRun.triggers_run && lastRun.triggers_run[triggerId]) {
                    switch (lastRun.triggers_run[triggerId].status) {
                        case 'Success':
                        case 'Warning':
                            return '#21BA45';
                        case 'Fail':
                            return '#FF4F60';
                    }
                }
            }
        }
        return '#000000';
    }

    @HostListener('document:keydown', ['$event'])
    handleKeyboardEvent(event: KeyboardEvent) {
        if (event.code === 'Escape' && this.linkWithJoin) {
            this._workflowCore.linkJoinEvent(null);
        }
    }

    createNodeComponent(node: WorkflowNode): ComponentRef<WorkflowNodeComponent> {
        let nodeComponentFactory = this.componentFactoryResolver.resolveComponentFactory(WorkflowNodeComponent);
        let componentRef = nodeComponentFactory.create(this.svgContainer.parentInjector);
        componentRef.instance.node = node;
        componentRef.instance.workflow = this.workflow;
        componentRef.instance.project = this.project;
        componentRef.instance.disabled = this.linkWithJoin;

        return componentRef;
    }

    toggleLinkJoin(b: boolean): void {

        this.linkWithJoin = b;
        this.nodesComponent.forEach(c => {
            (<WorkflowNodeComponent>c.instance).disabled = this.linkWithJoin;
        });
        this.joinsComponent.forEach(c => {
            (<WorkflowJoinComponent>c.instance).disabled = this.linkWithJoin;
        });
    }

    private getWorkflowMaxNodeByLevel(node: WorkflowNode, maxNode: number): number {
        if (node.triggers) {
            let nb = node.triggers.length;
            if (nb > maxNode) {
                maxNode = nb;
            }

            node.triggers.forEach(t => {
                let nb2 = this.getWorkflowMaxNodeByLevel(t.workflow_dest_node, maxNode);
                if (nb2 > maxNode) {
                    maxNode = nb2;
                }
            });
        }
        return maxNode;
    }

    private getWorkflowNodeDeep(node: WorkflowNode, maxDeep: Map<string, number>) {
        if (node.triggers) {
            node.triggers.forEach(t => {
                maxDeep.set(t.workflow_dest_node.ref, maxDeep.get(node.ref) + 1);
                this.getWorkflowNodeDeep(t.workflow_dest_node, maxDeep);
            });
        }
    }

    private getWorkflowJoinMaxNodeByLevel(maxNode: number): number {
        if (this.workflow.joins) {
            this.workflow.joins.forEach(j => {
                if (j.triggers) {
                    let nb = j.triggers.length;
                    if (nb > maxNode) {
                        maxNode = nb;
                    }
                    j.triggers.forEach(t => {
                        let n = this.getWorkflowMaxNodeByLevel(t.workflow_dest_node, maxNode);
                        if (n > maxNode) {
                            maxNode = n;
                        }
                    });
                }
            });
        }
        return maxNode;
    }

    private getWorkflowJoinDeep(maxDeep: Map<string, number>) {
        if (this.workflow.joins) {
            for (let i = 0; i < this.workflow.joins.length; i++) {
                this.workflow.joins.forEach(j => {

                    let canCheck = true;
                    let joinMaxDeep = 0;
                    j.source_node_ref.forEach(ref => {
                        let deep = maxDeep.get(ref);
                        if (!maxDeep.get(ref)) {
                            canCheck = false;
                        } else {
                            if (deep > joinMaxDeep) {
                                joinMaxDeep = deep;
                            }
                        }
                    });
                    if (canCheck && j.triggers) {
                        // get maxdeep
                        j.triggers.forEach(t => {
                            maxDeep.set(t.workflow_dest_node.ref, joinMaxDeep + 1);
                            this.getWorkflowNodeDeep(t.workflow_dest_node, maxDeep);
                        })
                    }
                });
            }

        }
    }
}
