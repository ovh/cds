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
import { SemanticDimmerComponent } from 'ng-semantic/ng-semantic';
import { Project } from '../../../model/project.model';
import { WNode, Workflow } from '../../../model/workflow.model';
import { WorkflowRun } from '../../../model/workflow.run.model';
import { WorkflowCoreService } from '../../../service/workflow/workflow.core.service';
import { WorkflowStore } from '../../../service/workflow/workflow.store';
import { AutoUnsubscribe } from '../../../shared/decorator/autoUnsubscribe';
import { WorkflowNodeHookComponent } from '../../../shared/workflow/wnode/hook/hook.component';
import { WorkflowWNodeComponent } from '../../../shared/workflow/wnode/wnode.component';

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
    @Input('workflowData')
    set workflowData(data: Workflow) {
        this.workflow = data;
        this.nodesComponent = new Map<string, ComponentRef<WorkflowWNodeComponent>>();
        this.hooksComponent = new Map<string, ComponentRef<WorkflowNodeHookComponent>>();
        this.changeDisplay();
    }

    _workflowRun: WorkflowRun;
    @Input('workflowRun')
    set workflowRun(data: WorkflowRun) {
        if (data) {
            this._workflowRun = data;
            this.workflow = data.workflow;
            this.previousWorkflowRunId = data.id;
            this.changeDisplay();
        }
    }

    @Input() project: Project;

    @Input('direction')
    set direction(data: string) {
        this._direction = data;
        this._workflowStore.setDirection(this.project.key, this.workflow.name, this.direction);
        this.changeDisplay();
    }
    get direction() { return this._direction; }

    @Output() deleteJoinSrcEvent = new EventEmitter<{ source, target }>();

    ready: boolean;
    _direction: string;

    // workflow graph
    @ViewChild('svgGraph', { read: ViewContainerRef }) svgContainer;
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

    zoom: any;

    constructor(
        private componentFactoryResolver: ComponentFactoryResolver,
        private _cd: ChangeDetectorRef,
        private _workflowStore: WorkflowStore,
        private _workflowCore: WorkflowCoreService
    ) { }

    ngAfterViewInit(): void {
        this.ready = true;
        this.changeDisplay();
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
        this.g = new dagreD3.graphlib.Graph().setGraph({ align: 'UL', rankdir: this.direction, nodesep: 10, ranksep: 15 });

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

        // Run the renderer. This is what draws the final graph.
        let svg = d3.select('svg');
        let oldG = svg.select('g');
        if (oldG) {
            oldG.remove();
        }
        let g = <any>svg.append('g');

        this.render(g, this.g);

        this.zoom = d3.zoom().scaleExtent([1 / 4, 2]).on('zoom', () => {
            g.attr('transform', d3.event.transform);
        });

        svg.call(this.zoom);
    }

    clickOrigin() {
        // TODO implement go to origin
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

    createEdge(from: string, to: string, options: {}): void {
        options['arrowhead'] = 'undirected';
        options['style'] = 'stroke: #B5B7BD;stroke-width: 2px;';
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
                    labelStyle: 'width: 30px; height: 30px',
                    width: 30,
                    height: 30
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
        if (!componentRef || componentRef.instance.node.id !== node.id) {
            componentRef = this.createNodeComponent(node);
            this.nodesComponent.set(node.ref, componentRef);
        }

        let width: number;
        let height: number;
        let shape = 'rect';
        switch (node.type) {
            case 'pipeline':
            case 'outgoinghook':
                width = 180;
                height = 60;
                break;
            case 'join':
                width = 40;
                height = 40;
                shape = 'circle';
                break;
            case 'fork':
                width = 40;
                height = 40;
                break;
        }

        this.svgContainer.insert(componentRef.hostView, 0);
        this.g.setNode('node-' + node.ref, <any>{
            label: () => {
                componentRef.location.nativeElement.style.width = '100%';
                componentRef.location.nativeElement.style.height = '100%';
                return componentRef.location.nativeElement;
            },
            shape: shape,
            labelStyle: 'width: ' + width + 'px; height: ' + height + 'px;',
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
            node.parents.forEach(p => {
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
}
