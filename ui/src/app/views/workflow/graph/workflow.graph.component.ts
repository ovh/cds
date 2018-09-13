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
import {Subscription} from 'rxjs';
import {Project} from '../../../model/project.model';
import {Workflow, WorkflowNode, WorkflowNodeFork, WorkflowNodeJoin, WorkflowNodeOutgoingHook} from '../../../model/workflow.model';
import {WorkflowNodeRun, WorkflowRun} from '../../../model/workflow.run.model';
import {WorkflowCoreService} from '../../../service/workflow/workflow.core.service';
import {WorkflowStore} from '../../../service/workflow/workflow.store';
import {AutoUnsubscribe} from '../../../shared/decorator/autoUnsubscribe';
import {WorkflowNodeForkComponent} from '../../../shared/workflow/fork/fork.component';
import {WorkflowJoinComponent} from '../../../shared/workflow/join/workflow.join.component';
import {WorkflowNodeHookComponent} from '../../../shared/workflow/node/hook/hook.component';
import { WorkflowNodeOutgoingHookComponent } from '../../../shared/workflow/node/outgoinghook/outgoinghook.component';
import {WorkflowNodeComponent} from '../../../shared/workflow/node/workflow.node.component';

@Component({
    selector: 'app-workflow-graph',
    templateUrl: './workflow.graph.html',
    styleUrls: ['./workflow.graph.scss'],
    entryComponents: [
        WorkflowNodeComponent,
        WorkflowJoinComponent,
        WorkflowNodeHookComponent,
        WorkflowNodeOutgoingHookComponent,
        WorkflowNodeForkComponent
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
            this.outgoingHooksComponent = new Map<string, ComponentRef<WorkflowNodeOutgoingHookComponent>>();
            this.forksComponent = new Map<string, ComponentRef<WorkflowNodeForkComponent>>();
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
    outgoingHooksComponent = new Map<string, ComponentRef<WorkflowNodeOutgoingHookComponent>>();
    forksComponent = new Map<string, ComponentRef<WorkflowNodeForkComponent>>();
    hooksComponent = new Map<number, ComponentRef<WorkflowNodeHookComponent>>();

    linkJoinSubscription: Subscription;

    nodeWidth: number;
    nodeHeight: number;

    constructor(
        private componentFactoryResolver: ComponentFactoryResolver,
        private _cd: ChangeDetectorRef,
        private _workflowStore: WorkflowStore,
        private _workflowCore: WorkflowCoreService
    ) {
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
                .attr('refX', 7) // position of arrow
                .attr('refY', 5) // position of arrow
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
                let mapLevel = new Map<number, number>();
                let mapLevelNode = new Map<string, number>();
                mapLevel.set(1, 1);
                this.getWorkflowMaxNodeByLevel(this.workflow.root, mapLevel, 2, mapLevelNode);
                this.getWorkflowJoinMaxNodeByLevel(nbofNodes, mapLevel, mapLevelNode);
                nbofNodes = Math.max(...Array.from(mapLevel.values()));
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

        if (node.outgoing_hooks) {
            node.outgoing_hooks.forEach(h => {
                this.createOutgoingHook(h);
                let options = {
                    id: 'outgoing-hook-' + h.ref,
                    style: 'stroke: #000000;'
                };
                this.createEdge('node-' + node.ref, 'outgoing-hook-' + h.ref, options);
            });
        }

        if (node.forks) {
            node.forks.forEach(f => {
                this.createFork(f);
                let options = {
                  id: 'fork-' + f.id,
                  style: 'stroke: #000000;top: 20px;',
                    height: 10,
                };
                this.createEdge('node-' + node.ref, 'fork-' + f.name, options);

                if (f.triggers) {
                    f.triggers.forEach(t => {
                        this.createNode(t.workflow_dest_node);
                        let optForkTrig = {
                            id: 'trigger-' + t.id,
                            style: 'stroke: ' + this.getTriggerColor(node, t.id) + ';'
                        };
                        this.createEdge('fork-' + f.name, 'node-' + t.workflow_dest_node.ref, optForkTrig);
                    });
                }
            });
        }
    }

    createFork(f: WorkflowNodeFork): void {
        let componentRef = this.forksComponent.get(f.name);
        if (!componentRef) {
            componentRef = this.createForkComponent(f);
            this.forksComponent.set(f.name, componentRef);
        }

        this.svgContainer.insert(componentRef.hostView, 0);
        this.g.setNode('fork-' + f.name, <any>{
            label: () => {
                componentRef.location.nativeElement.style.width = '97%';
                componentRef.location.nativeElement.style.height = '100%';
                return componentRef.location.nativeElement;
            },
            shape: 'rect',
            labelStyle: 'width: 70px; height: 70px',
            width: 70,
            height: 70
        });
    }

    createForkComponent(f: WorkflowNodeFork): ComponentRef<WorkflowNodeForkComponent> {
        let forkComponentFactory = this.componentFactoryResolver.resolveComponentFactory(WorkflowNodeForkComponent);
        let componentRef = forkComponentFactory.create(this.svgContainer.parentInjector);
        componentRef.instance.fork = f;
        componentRef.instance.workflow = this.workflow;
        return componentRef;
    }

    createOutgoingHook(hook: WorkflowNodeOutgoingHook): void {
        let componentRef = this.outgoingHooksComponent.get(hook.ref);
        if (!componentRef) {
            componentRef = this.createOutgoingHookComponent(hook);
            this.outgoingHooksComponent.set(hook.ref, componentRef);
        }

        this.svgContainer.insert(componentRef.hostView, 0);
        this.g.setNode('outgoing-hook-' + hook.ref, <any>{
            label: () => {
                componentRef.location.nativeElement.style.width = '97%';
                componentRef.location.nativeElement.style.height = '100%';
                return componentRef.location.nativeElement;
            },
            labelStyle: 'width: 210px; height: 68px',
            width: 210,
            height: 68
        });
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

    createOutgoingHookComponent(hook: WorkflowNodeOutgoingHook): ComponentRef<WorkflowNodeOutgoingHookComponent> {
        let nodeComponentFactory = this.componentFactoryResolver.resolveComponentFactory(WorkflowNodeOutgoingHookComponent);
        let componentRef = nodeComponentFactory.create(this.svgContainer.parentInjector);
        componentRef.instance.hook = hook;
        componentRef.instance.workflow = this.workflow;
        componentRef.instance.project = this.project;

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

    private getWorkflowMaxNodeByLevel(node: WorkflowNode, levelMap: Map<number, number>, level: number,
                                      levelNode: Map<string, number>): void {
        levelNode.set(node.ref, level - 1);
        if (node.triggers) {
            node.triggers.forEach(t => {
                this.getWorkflowMaxNodeByLevel(t.workflow_dest_node, levelMap, level + 1, levelNode);
                if (levelMap.get(level)) {
                    levelMap.set(level, levelMap.get(level) + 1);
                } else {
                    levelMap.set(level, 1);
                }
            });
        }
        if (node.outgoing_hooks) {
            node.outgoing_hooks.forEach(o => {
                if (o.triggers) {
                    o.triggers.forEach(t => {
                        this.getWorkflowMaxNodeByLevel(t.workflow_dest_node, levelMap, level + 1, levelNode);
                        if (levelMap.get(level)) {
                            levelMap.set(level, levelMap.get(level) + 1);
                        } else {
                            levelMap.set(level, 1);
                        }
                    });
                }
            });
        }
        if (node.forks) {
            node.forks.forEach(f => {
                if (f.triggers) {
                    f.triggers.forEach(t => {
                        this.getWorkflowMaxNodeByLevel(t.workflow_dest_node,  levelMap, level + 1, levelNode);
                        if (levelMap.get(level)) {
                            levelMap.set(level, levelMap.get(level) + 1);
                        } else {
                            levelMap.set(level, 1);
                        }
                    });
                }
            });
        }
    }

    private getWorkflowNodeDeep(node: WorkflowNode, maxDeep: Map<string, number>) {
        if (node.triggers) {
            node.triggers.forEach(t => {
                maxDeep.set(t.workflow_dest_node.ref, maxDeep.get(node.ref) + 1);
                this.getWorkflowNodeDeep(t.workflow_dest_node, maxDeep);
            });
        }
        if (node.outgoing_hooks) {
            node.outgoing_hooks.forEach(o => {
                if (o.triggers) {
                    o.triggers.forEach(t => {
                        maxDeep.set(t.workflow_dest_node.ref, maxDeep.get(node.ref) + 1);
                        this.getWorkflowNodeDeep(t.workflow_dest_node, maxDeep);
                    });
                }
            });
        }
        if (node.forks) {
            node.forks.forEach(f => {
                if (f.triggers) {
                    f.triggers.forEach(t => {
                        maxDeep.set(t.workflow_dest_node.ref, maxDeep.get(node.ref) + 1);
                        this.getWorkflowNodeDeep(t.workflow_dest_node, maxDeep);
                    });
                }
            });
        }
    }

    private getWorkflowJoinMaxNodeByLevel(maxNode: number, mapLevel: Map<number, number>, levelNode: Map<string, number>): number {
        if (this.workflow.joins) {
            this.workflow.joins.forEach(j => {
                let maxLevel = 0;
                if (j.source_node_ref) {
                    j.source_node_ref.forEach( r => {
                       if (levelNode.get(r) > maxLevel) {
                           maxLevel = levelNode.get(r);
                       }
                    });
                }
                maxLevel++;
                if (j.triggers) {
                    j.triggers.forEach(t => {
                        this.getWorkflowMaxNodeByLevel(t.workflow_dest_node,  mapLevel, maxLevel + 1, levelNode);
                        if (mapLevel.get(maxLevel)) {
                            mapLevel.set(maxLevel, mapLevel.get(maxLevel) + 1);
                        } else {
                            mapLevel.set(maxLevel, 1);
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
