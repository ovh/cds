import {
    AfterViewInit, ChangeDetectorRef,
    Component,
    ComponentFactoryResolver,
    ComponentRef,
    EventEmitter,
    HostListener,
    Input, OnInit,
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
import {Subscription} from 'rxjs/Subscription';
import {AutoUnsubscribe} from '../../../shared/decorator/autoUnsubscribe';
import {WorkflowStore} from '../../../service/workflow/workflow.store';
import {CDSWorker} from '../../../shared/worker/worker';
import {SemanticDimmerComponent} from 'ng-semantic/ng-semantic';

@Component({
    selector: 'app-workflow-graph',
    templateUrl: './workflow.graph.html',
    styleUrls: ['./workflow.graph.scss'],
    entryComponents: [
        WorkflowNodeComponent,
        WorkflowJoinComponent
    ]
})
@AutoUnsubscribe()
export class WorkflowGraphComponent implements AfterViewInit, OnInit {

    @Input() workflow: Workflow;
    @Input() project: Project;
    @Input() webworker: CDSWorker;

    @Output() editTriggerEvent = new EventEmitter<{source, target}>();
    @Output() editTriggerJoinEvent = new EventEmitter<{source, target}>();
    @Output() deleteJoinSrcEvent = new EventEmitter<{source, target}>();
    @Output() addSrcToJoinEvent = new EventEmitter<{source, target}>();

    // workflow graph
    @ViewChild('svgGraph', {read: ViewContainerRef}) svgContainer;
    g: dagreD3.graphlib.Graph;
    render = new dagreD3.render();
    svgWidth: number;
    svgHeight: number;
    direction: string;

    @ViewChild('dimmer')
    dimmer: SemanticDimmerComponent;

    linkWithJoin = false;
    nodeToLink: WorkflowNode;

    nodesComponent = new Array<ComponentRef<WorkflowNodeComponent>>();
    joinsComponent = new Array<ComponentRef<WorkflowJoinComponent>>();

    workflowSubscription: Subscription;

    readonly minSvgWidth = 155;
    readonly minPipelineWidth = 177;
    currentSvgWidth = 155;


    constructor(private componentFactoryResolver: ComponentFactoryResolver, private _cd: ChangeDetectorRef,
                private _workflowStore: WorkflowStore) {
    }

    ngOnInit(): void {
        this.direction = this._workflowStore.getDirection(this.project.key, this.workflow.name);
        this.workflowSubscription = this._workflowStore.getWorkflows(this.project.key, this.workflow.name).subscribe(ws => {
            if (ws) {
                let updatedWorkflow = ws.get(this.project.key + '-' + this.workflow.name);
                if (updatedWorkflow && !updatedWorkflow.externalChange
                    && (new Date(updatedWorkflow.last_modified)).getTime() > (new Date(this.workflow.last_modified)).getTime()) {
                    this.workflow = updatedWorkflow;
                    this.initWorkflow();
                }
            }
        }, () => {
            console.log('Error getting workflow');
        });
    }

    @HostListener('window:resize', ['$event'])
    onResize(event) {
        /*
        let svg = d3.select('svg');
        let inner = d3.select('svg g');
        if (this.direction === 'LR') {
            let w = 0;
            inner.each(function () {
                w = this.getBBox().width;
            });
            this.svgWidth = w + 40;
            inner.attr('transform', 'translate(20, 0)');
        } else {
            inner.attr('transform', 'translate(20, 0)');
            // Horizontal center
            /*
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
        */
    }

    ngAfterViewInit(): void {
        this.initWorkflow();
    }

    changeDisplay(): void {
        this._workflowStore.setDirection(this.project.key, this.workflow.name, this.direction);
        this.joinsComponent.forEach( j => {
            j.destroy();
        });
        this.nodesComponent.forEach( j => {
            j.destroy();
        });
        this.initWorkflow();
    }

    initWorkflow() {
        this.svgWidth = window.innerWidth;
        this.svgHeight = window.innerHeight;
        // https://github.com/cpettitt/dagre/wiki#configuring-the-layout
        this.g = new dagreD3.graphlib.Graph().setGraph({rankdir: this.direction});

        let mapDeep = new Map<number, number>();
        mapDeep.set(this.workflow.root.id, 1);
        this.getWorkflowNodeDeep(this.workflow.root, mapDeep);
        this.getWorkflowJoinDeep(mapDeep);

        this.currentSvgWidth = Math.floor(this.svgWidth * .75 / Math.max(...Array.from(mapDeep.values())));
        if (this.currentSvgWidth < this.minSvgWidth) {
            this.currentSvgWidth = this.minSvgWidth;
        }

        if (this.workflow.root) {
            this.createNode(this.workflow.root);
        }
        if (this.workflow.joins) {
            this.workflow.joins.forEach(j => {
                this.createJoin(j);
            });

        }

        // Set up an SVG group so that we can translate the final graph.
        let svg = d3.select('svg');
        svg.attr('width', this.svgWidth);
        svg.attr('height', this.svgHeight);
        let inner = d3.select('svg g');

        this.g.transition = (selection) => {
            return selection.transition().duration(500);
        };

        // Add our custom arrow (a hollow-point)
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

        this.g.graph().transition = function(selection) {
            return selection.transition().duration(500);
        };
        debugger;
        // Run the renderer. This is what draws the final graph.
        this.render(inner, this.g);

        this._cd.detectChanges();

        setTimeout(() => {
            // Center the graph
            this.onResize(null);

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
        }, 1);


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

        this.joinsComponent.push(componentRef);
        this.svgContainer.insert(componentRef.hostView);

        this.g.setNode('join-' + join.id, {
            shape: 'circle',
            label: () => {
                return componentRef.location.nativeElement;
            },
            class: 'join'
        });

        if (join.source_node_id) {
            join.source_node_id.forEach(nodeID => {
                this.createEdge('node-' + nodeID, 'join-' + join.id, {});
            });
        }

        if (join.triggers) {
            join.triggers.forEach(t => {
                this.createNode(t.workflow_dest_node);
                this.createEdge('join-' + join.id, 'node-' + t.workflow_dest_node.id, {id: 'trigger-' + t.id});
            });
        }
    }

    createNode(node: WorkflowNode): void {
        let componentRef = this.createNodeComponent(node);
        this.svgContainer.insert(componentRef.hostView);
        this.g.setNode('node-' + node.id, {
            label: () => {
                componentRef.location.nativeElement.style.width = this.currentSvgWidth + 'px';
                return componentRef.location.nativeElement;
            },
            labelStyle: "width: " + this.currentSvgWidth + "px",
            width: this.currentSvgWidth
        });
        if (node.triggers) {
            node.triggers.forEach(t => {
                this.createNode(t.workflow_dest_node);
                this.createEdge('node-' + node.id, 'node-' + t.workflow_dest_node.id, {id: 'trigger-' + t.id});
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
        componentRef.instance.workflowPipelineWidth = (this.currentSvgWidth + 22) + 'px';
        componentRef.instance.workflowSvgNodeWidth = this.currentSvgWidth + 'px';
        this.nodesComponent.push(componentRef);
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

    private getWorkflowNodeDeep(node: WorkflowNode, maxDeep: Map<number, number>){
        if (node.triggers) {
            node.triggers.forEach( t => {
                maxDeep.set(t.workflow_dest_node.id, maxDeep.get(node.id) + 1);
                this.getWorkflowNodeDeep(t.workflow_dest_node, maxDeep);
            });
        }
    }


    private getWorkflowJoinDeep(maxDeep: Map<number, number>) {
        if (this.workflow.joins) {
            for(let i=0; i<this.workflow.joins.length; i++) {
                this.workflow.joins.forEach(j => {

                    let canCheck = true;
                    let joinMaxDeep = 0;
                    j.source_node_id.forEach( id => {
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
