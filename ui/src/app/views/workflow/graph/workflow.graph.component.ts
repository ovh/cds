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

    linkWithJoin = false;
    nodeToLink: WorkflowNode;

    nodesComponent = new Array<ComponentRef<WorkflowNodeComponent>>();
    joinsComponent = new Array<ComponentRef<WorkflowJoinComponent>>();

    workflowSubscription: Subscription;

    constructor(private componentFactoryResolver: ComponentFactoryResolver, private _cd: ChangeDetectorRef,
                private _workflowStore: WorkflowStore) {
    }

    ngOnInit(): void {
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
        if (event) {
            this.svgWidth = event.target.innerWidth;
        } else {
            this.svgWidth = window.innerWidth;
        }

        let svg = d3.select('svg');
        let inner = d3.select('svg g');
        let svgWidth = +svg.attr('width');
        let xCenterOffset = (svgWidth - this.g.graph().width) / 2;
        inner.attr('transform', 'translate(' + xCenterOffset + ', 20)');
        svg.attr('height', this.g.graph().height + 40);
    }

    ngAfterViewInit(): void {
        this.initWorkflow();
    }

    initWorkflow() {
        this.svgWidth = window.innerWidth;
        // this.g = new dagreD3.graphlib.Graph().setGraph({ directed: false, rankDir: 'LR'});
        this.g = new dagreD3.graphlib.Graph().setGraph({directed: false});
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

        let inner = d3.select('svg g');
        /* FIXME : resize child
         let zoom = d3.behavior.zoom().on('zoom', () => {
         inner.attr('transform', 'translate(' + (<ZoomEvent>d3.event).translate + ')' + 'scale(' + (<ZoomEvent>d3.event).scale + ')');
         //this.centerGraph(svg, inner);
         });
         svg.call(zoom);
         */
        this.g.transition = (selection) => {
            return selection.transition().duration(500);
        };

        // Add our custom arrow (a hollow-point)
        this.render.arrows()['customArraow'] = (parent, id, edge, type) => {
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
        // Run the renderer. This is what draws the final graph.
        this.render(inner, this.g);

        // Center the graph
        this.onResize(null);

        setTimeout(() => {
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
        this._cd.detectChanges();
    }

    createEdge(from: string, to: string, options: {}): void {
        options['arrowhead'] = 'customArraow';
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
                return componentRef.location.nativeElement;
            }
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
}
