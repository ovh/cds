import {
    AfterViewInit,
    ChangeDetectorRef,
    Component,
    ComponentFactoryResolver,
    ComponentRef,
    EventEmitter,
    HostListener,
    Input,
    OnInit,
    Output,
    ViewChild,
    ViewContainerRef
} from '@angular/core';
import {Workflow, WorkflowNode, WorkflowNodeJoin} from '../../../model/workflow.model';
import {WorkflowJoinComponent} from '../../../shared/workflow/join/workflow.join.component';
import {WorkflowNodeComponent} from '../../../shared/workflow/node/workflow.node.component';
import {Project} from '../../../model/project.model';
import * as d3 from 'd3';
import * as dagreD3 from 'dagre-d3';
import {AutoUnsubscribe} from '../../../shared/decorator/autoUnsubscribe';
import {WorkflowStore} from '../../../service/workflow/workflow.store';
import {CDSWorker} from '../../../shared/worker/worker';
import {SemanticDimmerComponent} from 'ng-semantic/ng-semantic';
import {WorkflowNodeHookComponent} from '../../../shared/workflow/node/hook/hook.component';

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
export class WorkflowGraphComponent implements AfterViewInit, OnInit {

    workflow: Workflow;

    @Input('workflowData')
    set workflowData(data: Workflow) {
        this.workflow = data;
        this.changeDisplay(true);
    }

    @Input() project: Project;
    @Input() webworker: CDSWorker;
    @Input() status: string;
    @Input('direction')
    set direction(data: string) {
        this._direction = data;
        this.changeDisplay(false);
    }

    @Output() editTriggerEvent = new EventEmitter<{ source, target }>();
    @Output() editTriggerJoinEvent = new EventEmitter<{ source, target }>();
    @Output() deleteJoinSrcEvent = new EventEmitter<{ source, target }>();
    @Output() addSrcToJoinEvent = new EventEmitter<{ source, target }>();

    ready: boolean;
    _direction: string;
    displayDirection = false;

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

    nodesComponent = new Map<number, ComponentRef<WorkflowNodeComponent>>();
    joinsComponent = new Map<number, ComponentRef<WorkflowJoinComponent>>();
    hooksComponent = new Map<number, ComponentRef<WorkflowNodeHookComponent>>();

    nodeWidth: number;
    nodeHeight: number;

    constructor(private componentFactoryResolver: ComponentFactoryResolver, private _cd: ChangeDetectorRef,
                private _workflowStore: WorkflowStore) {
    }

    ngOnInit(): void {
        if (!this._direction) {
            this.displayDirection = true;
            this._direction = this._workflowStore.getDirection(this.project.key, this.workflow.name);
        }
    }

    @HostListener('window:resize', ['$event'])
    onResize(event) {
        // Resize svg
        let svg = d3.select('svg');
        let inner = d3.select('svg g');
        if (this._direction === 'LR') {
            let w = 0;
            inner.each(function () {
                w = this.getBBox().width;
            });
            this.svgWidth = w + 30;
            inner.attr('transform', 'translate(20, 0)');
        } else {
            inner.attr('transform', 'translate(20, 0)');
            // Horizontal center
            if (event) {
                this.svgWidth = event.target.innerWidth;
            } else {
                this.svgWidth = window.innerWidth;
            }
            let svgWidth = +svg.attr('width');
            let xCenterOffset = (svgWidth - this.g.graph().width) / 2;
            inner.attr('transform', 'translate(' + xCenterOffset + ', 0)');
        }
        this.svgHeight = this.g.graph().height + 40;
        svg.attr('height', this.svgHeight);
        this.changeDisplay(false);
    }

    ngAfterViewInit(): void {
        this.ready = true;
        this.changeDisplay(true);
    }

    changeDisplay(resize: boolean): void {
        if (!this.ready) {
            return;
        }
        this._workflowStore.setDirection(this.project.key, this.workflow.name, this._direction);
        this.joinsComponent.forEach(j => {
            j.destroy();
        });
        this.nodesComponent.forEach(j => {
            j.destroy();
        });
        this.hooksComponent.forEach(h => {
            h.destroy();
        });
        this.joinsComponent.clear();
        this.nodesComponent.clear();
        this.hooksComponent.clear();

        this.initWorkflow(resize);
    }

    initWorkflow(resize: boolean) {
        // https://github.com/cpettitt/dagre/wiki#configuring-the-layout
        this.g = new dagreD3.graphlib.Graph().setGraph({rankdir: this._direction});

        // Calculate node width
        this.nodeHeight = 78;
        this.calculateDynamicWidth();
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
        this.createCustomArraow();

        // Setup transition
        this.g.graph().transition = function (selection) {
            return selection.transition().duration(100);
        };

        // Run the renderer. This is what draws the final graph.
        this.render(d3.select('svg g'), this.g);

        // Add listener on graph element
        this.addListener(d3.select('svg'));
        this.svgHeight = this.g.graph().height + 40;
        this._cd.detectChanges();
    }

    private createCustomArraow() {
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
            // Trigger between node and node
            if (d.v.indexOf('node-') === 0 && d.w.indexOf('node-') === 0) {
                this.editTriggerEvent.emit({source: d.v, target: d.w});
            }
            // Join Trigger
            if (d.v.indexOf('join-') === 0) {
                this.editTriggerJoinEvent.emit({source: d.v, target: d.w});
            }

            // Node Join Src
            if (d.v.indexOf('node-') === 0 && d.w.indexOf('join-') === 0) {
                this.deleteJoinSrcEvent.emit({source: d.v, target: d.w});
            }
        });
    }

    private calculateDynamicWidth() {
        let mapDeep = new Map<number, number>();
        mapDeep.set(this.workflow.root.id, 1);
        this.getWorkflowNodeDeep(this.workflow.root, mapDeep);
        this.getWorkflowJoinDeep(mapDeep);

        this.nodeWidth = Math.floor(window.innerWidth * .85 / Math.max(...Array.from(mapDeep.values())));
        if (this.nodeWidth < 155) {
            this.nodeWidth = 155;
        }

    }

    createEdge(from: string, to: string, options: {}): void {
        options['arrowhead'] = 'customArrow';
        this.g.setEdge(from, to, options);
    }

    createJoin(join: WorkflowNodeJoin): void {
        let nodeComponentFactory = this.componentFactoryResolver.resolveComponentFactory(WorkflowJoinComponent);
        let componentRef = nodeComponentFactory.create(this.svgContainer.parentInjector);
        componentRef.instance.workflow = this.workflow;
        componentRef.instance.join = join;
        componentRef.instance.project = this.project;
        componentRef.instance.disabled = this.linkWithJoin;

        if (this.webworker) {
            componentRef.instance.readonly = true;
        }

        componentRef.instance.selectEvent.subscribe(j => {
            if (this.linkWithJoin && this.nodeToLink) {
                this.addSrcToJoinEvent.emit({source: this.nodeToLink, target: j});
            }
        });
        this.joinsComponent.set(join.id, componentRef);


        this.svgContainer.insert(componentRef.hostView);

        this.g.setNode('join-' + join.id, {
            shape: 'circle',
            label: () => {
                return componentRef.location.nativeElement;
            }
        });

        if (join.source_node_id) {
            join.source_node_id.forEach(nodeID => {
                this.createEdge('node-' + nodeID, 'join-' + join.id, {});
            });
        }

        if (join.triggers) {
            join.triggers.forEach(t => {
                this.createNode(t.workflow_dest_node);
                let options = {
                    id: 'trigger-' + t.id
                };
                if (t.manual) {
                    options['style'] = 'stroke-dasharray: 5, 5';
                }
                this.createEdge('join-' + join.id, 'node-' + t.workflow_dest_node.id, options);
            });
        }
    }

    createHookNode(node: WorkflowNode): void {
        if (!node.hooks || node.hooks.length === 0) {
            return;
        }

        node.hooks.forEach(h => {
            let hookComponent = this.componentFactoryResolver.resolveComponentFactory(WorkflowNodeHookComponent);
            let componentRef = hookComponent.create(this.svgContainer.parentInjector);
            componentRef.instance.hook = h;
            componentRef.instance.workflow = this.workflow;
            componentRef.instance.project = this.project;
            componentRef.instance.node = node;

            if (this.webworker) {
                componentRef.instance.readonly = true;
            }

            this.svgContainer.insert(componentRef.hostView);

            this.hooksComponent.set(h.id, componentRef);

            this.g.setNode(
                'hook-' + node.id + '-' + h.id, {
                    label: () => {
                        return componentRef.location.nativeElement;
                    }

                }
            );

            let options = {
                id: 'hook-' + node.id + '-' + h.id
            };
            this.createEdge('hook-' + node.id + '-' + h.id, 'node-' + node.id, options);
        });
    }

    createNode(node: WorkflowNode): void {
        let componentRef = this.createNodeComponent(node);
        this.svgContainer.insert(componentRef.hostView);
        this.g.setNode('node-' + node.id, {
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
                    id: 'trigger-' + t.id
                };
                if (t.manual) {
                    options['style'] = 'stroke-dasharray: 5, 5';
                }
                this.createEdge('node-' + node.id, 'node-' + t.workflow_dest_node.id, options);
            });
        }
    }

    createNodeComponent(node: WorkflowNode): ComponentRef<WorkflowNodeComponent> {
        let nodeComponentFactory = this.componentFactoryResolver.resolveComponentFactory(WorkflowNodeComponent);
        let componentRef = nodeComponentFactory.create(this.svgContainer.parentInjector);
        componentRef.instance.node = node;
        componentRef.instance.workflow = this.workflow;
        componentRef.instance.project = this.project;
        componentRef.instance.disabled = this.linkWithJoin;
        componentRef.instance.webworker = this.webworker;
        componentRef.instance.workflowRunStatus = this.status;
        this.nodesComponent.set(node.id, componentRef);
        componentRef.instance.linkJoinEvent.subscribe(n => {
            this.nodeToLink = n;
            this.toggleLinkJoin(true);

        });

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

    private getWorkflowNodeDeep(node: WorkflowNode, maxDeep: Map<number, number>) {
        if (node.triggers) {
            node.triggers.forEach(t => {
                maxDeep.set(t.workflow_dest_node.id, maxDeep.get(node.id) + 1);
                this.getWorkflowNodeDeep(t.workflow_dest_node, maxDeep);
            });
        }
    }


    private getWorkflowJoinDeep(maxDeep: Map<number, number>) {
        if (this.workflow.joins) {
            for (let i = 0; i < this.workflow.joins.length; i++) {
                this.workflow.joins.forEach(j => {

                    let canCheck = true;
                    let joinMaxDeep = 0;
                    j.source_node_id.forEach(id => {
                        let deep = maxDeep.get(id);
                        if (!maxDeep.get(id)) {
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
                            maxDeep.set(t.workflow_dest_node.id, joinMaxDeep + 1);
                            this.getWorkflowNodeDeep(t.workflow_dest_node, maxDeep);
                        })
                    }
                });
            }

        }
    }
}
