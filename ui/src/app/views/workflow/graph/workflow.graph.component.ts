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
import {WNode, Workflow} from '../../../model/workflow.model';
import {WorkflowRun} from '../../../model/workflow.run.model';
import {WorkflowCoreService} from '../../../service/workflow/workflow.core.service';
import {WorkflowStore} from '../../../service/workflow/workflow.store';
import {AutoUnsubscribe} from '../../../shared/decorator/autoUnsubscribe';
import {WorkflowNodeHookComponent} from '../../../shared/workflow/wnode/hook/hook.component';
import {WorkflowWNodeComponent} from '../../../shared/workflow/wnode/wnode.component';

@Component({
    selector: 'app-workflow-graph',
    templateUrl: './workflow.graph.html',
    styleUrls: ['./workflow.graph.scss'],
    entryComponents: [
        WorkflowWNodeComponent,
        WorkflowNodeHookComponent
    ]
})
@AutoUnsubscribe()
export class WorkflowGraphComponent implements AfterViewInit {

    workflow: Workflow;
    _workflowRun: WorkflowRun;

    @Input('workflowData')
    set workflowData(data: Workflow) {
        this.workflow = data;
        this.nodeHeight = 78;
        if (data.forceRefresh) {
            this.nodesComponent = new Map<string, ComponentRef<WorkflowWNodeComponent>>();
            this.hooksComponent = new Map<string, ComponentRef<WorkflowNodeHookComponent>>();
        } else {
            let nodesRef = Workflow.getMapNodesRef(this.workflow);
            // Update node reference inside component
            this.nodesComponent.forEach((v, k, m) => {
                let n = nodesRef.get(v.instance.node.ref);
                if (n) {
                    v.instance.node = n;
                    v.instance.workflow = this.workflow;
                }
            });
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
    nodeToLink: WNode;
    previousWorkflowRunId = 0;

    nodesComponent = new Map<string, ComponentRef<WorkflowWNodeComponent>>();
    hooksComponent = new Map<string, ComponentRef<WorkflowNodeHookComponent>>();

    nodeWidth: number;
    nodeHeight: number;

    constructor(
        private componentFactoryResolver: ComponentFactoryResolver,
        private _cd: ChangeDetectorRef,
        private _workflowStore: WorkflowStore,
        private _workflowCore: WorkflowCoreService
    ) {}

    @HostListener('window:resize', ['$event'])
    onResize() {
        this.resize();
    }

    resize() {
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
        this.resize();
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
        if (this.workflow.workflow_data && this.workflow.workflow_data.node) {
            this.createNode(this.workflow.workflow_data.node);
        }
        if (this.workflow.workflow_data && this.workflow.workflow_data.joins) {
            this.workflow.workflow_data.joins.forEach(j => {
                this.createNode(j);
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

    private calculateDynamicWidth() {
        let nbofNodes = 1;
        switch (this.direction) {
            case 'LR':
                let mapDeep = new Map<string, number>();
                mapDeep.set(this.workflow.workflow_data.node.ref, 1);
                this.getWorkflowNodeDeep(this.workflow.workflow_data.node, mapDeep);
                this.getWorkflowJoinDeep(mapDeep);
                nbofNodes = Math.max(...Array.from(mapDeep.values()));
                break;
            default:
                let mapLevel = new Map<number, number>();
                let mapLevelNode = new Map<string, number>();
                mapLevel.set(1, 1);
                this.getWorkflowMaxNodeByLevel(this.workflow.workflow_data.node, mapLevel, 2, mapLevelNode);
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

    createHookNode(node: WNode): void {
        if (!node.hooks || node.hooks.length === 0) {
            return;
        }

        node.hooks.forEach(h => {
            let hookId = h.uuid;
            let componentRef = this.hooksComponent.get(hookId);
            if (!componentRef) {
                let hookComponent = this.componentFactoryResolver.resolveComponentFactory(WorkflowNodeHookComponent);
                componentRef = hookComponent.create(this.svgContainer.parentInjector);

            }
            componentRef.instance.hook = h;
            componentRef.instance.workflow = this.workflow;
            componentRef.instance.project = this.project;
            componentRef.instance.node = node;
            this.hooksComponent.set(hookId, componentRef);

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

    createNode(node: WNode): void {
        let componentRef = this.nodesComponent.get(node.ref);
        if (!componentRef) {
            componentRef = this.createNodeComponent(node);
            this.nodesComponent.set(node.ref, componentRef);
        }

        let width: number;
        let height: number;
        let componentRefWidth = '97%';
        switch (node.type) {
            case 'pipeline':
                width = this.nodeWidth;
                height = this.nodeHeight;
                break;
            case 'join':
                width = 70;
                height = 70;
                break;
            case 'outgoinghook':
                componentRefWidth = '98%';
                width = 210;
                height = 68;
                break;
            case 'fork':
                width = 70;
                height = 70;
                break;
        }
        this.svgContainer.insert(componentRef.hostView, 0);
        this.g.setNode('node-' + node.ref, <any>{
            label: () => {
                componentRef.location.nativeElement.style.width = componentRefWidth;
                componentRef.location.nativeElement.style.height = '100%';
                return componentRef.location.nativeElement;
            },
            labelStyle: 'width: ' + width + 'px; height: ' + height + 'px',
            width: width,
            height: height
        });

        this.createHookNode(node);

        if (node.triggers) {
            node.triggers.forEach(t => {
                this.createNode(t.child_node);
                let options = {
                    id: 'trigger-' + t.id,
                    style: 'stroke: #000000;'
                };
                this.createEdge('node-' + node.ref, 'node-' + t.child_node.ref, options);
            });
        }

        // Create parent trigger
        if (node.type === 'join') {
            node.parents.forEach( p => {
                let options = {
                    id: 'join-trigger-' + p.parent_name,
                    style: 'stroke: #000000;'
                };
               this.createEdge('node-' + p.parent_name, 'node-' + node.ref, options);
            });
        }

    }

    @HostListener('document:keydown', ['$event'])
    handleKeyboardEvent(event: KeyboardEvent) {
        if (event.code === 'Escape' && this.linkWithJoin) {
            this._workflowCore.linkJoinEvent(null);
        }
    }

    createNodeComponent(node: WNode): ComponentRef<WorkflowWNodeComponent> {
        let nodeComponentFactory = this.componentFactoryResolver.resolveComponentFactory(WorkflowWNodeComponent);
        let componentRef = nodeComponentFactory.create(this.svgContainer.parentInjector);
        componentRef.instance.node = node;
        componentRef.instance.workflow = this.workflow;
        componentRef.instance.project = this.project;

        return componentRef;
    }

    private getWorkflowMaxNodeByLevel(node: WNode, levelMap: Map<number, number>, level: number,
                                      levelNode: Map<string, number>): void {
        levelNode.set(node.ref, level - 1);
        if (node.triggers) {
            node.triggers.forEach(t => {
                this.getWorkflowMaxNodeByLevel(t.child_node, levelMap, level + 1, levelNode);
                if (levelMap.get(level)) {
                    levelMap.set(level, levelMap.get(level) + 1);
                } else {
                    levelMap.set(level, 1);
                }
            });
        }
    }

    private getWorkflowNodeDeep(node: WNode, maxDeep: Map<string, number>) {
        if (node.triggers) {
            node.triggers.forEach(t => {
                maxDeep.set(t.child_node.ref, maxDeep.get(node.ref) + 1);
                this.getWorkflowNodeDeep(t.child_node, maxDeep);
            });
        }
    }

    private getWorkflowJoinMaxNodeByLevel(maxNode: number, mapLevel: Map<number, number>, levelNode: Map<string, number>): number {
        if (this.workflow.workflow_data && this.workflow.workflow_data.joins) {
            this.workflow.workflow_data.joins.forEach(j => {
                let maxLevel = 0;
                if (j.parents) {
                    j.parents.forEach( r => {
                       if (levelNode.get(r.parent_name) > maxLevel) {
                           maxLevel = levelNode.get(r.parent_name);
                       }
                    });
                }
                maxLevel++;
                if (j.triggers) {
                    j.triggers.forEach(t => {
                        this.getWorkflowMaxNodeByLevel(t.child_node,  mapLevel, maxLevel + 1, levelNode);
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
        if (this.workflow.workflow_data && this.workflow.workflow_data.joins) {
            for (let i = 0; i < this.workflow.workflow_data.joins.length; i++) {
                this.workflow.workflow_data.joins.forEach(j => {

                    let canCheck = true;
                    let joinMaxDeep = 0;
                    j.parents.forEach(r => {
                        let deep = maxDeep.get(r.parent_name);
                        if (!maxDeep.get(r.parent_name)) {
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
                            maxDeep.set(t.child_node.ref, joinMaxDeep + 1);
                            this.getWorkflowNodeDeep(t.child_node, maxDeep);
                        })
                    }
                });
            }

        }
    }
}
