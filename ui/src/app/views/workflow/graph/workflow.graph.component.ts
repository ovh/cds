// eslint-disable-next-line max-len
import { AfterViewInit, ChangeDetectionStrategy, ChangeDetectorRef, Component, ComponentFactoryResolver, ComponentRef, EventEmitter, HostListener, Input, OnDestroy, Output, ViewChild, ViewContainerRef } from '@angular/core';
import * as d3 from 'd3';
import * as dagreD3 from 'dagre-d3';
import { Project } from 'app/model/project.model';
import { WNode, Workflow } from 'app/model/workflow.model';
import { WorkflowCoreService } from 'app/service/workflow/workflow.core.service';
import { WorkflowStore } from 'app/service/workflow/workflow.store';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { WorkflowNodeHookComponent } from 'app/shared/workflow/wnode/hook/hook.component';
import { WorkflowWNodeComponent } from 'app/shared/workflow/wnode/wnode.component';

@Component({
    selector: 'app-workflow-graph',
    templateUrl: './workflow.graph.html',
    styleUrls: ['./workflow.graph.scss'],
    entryComponents: [
        WorkflowWNodeComponent,
        WorkflowNodeHookComponent
    ],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowGraphComponent implements AfterViewInit, OnDestroy {
    static margin = 80; // let 40px on top and bottom of the graph
    static maxScale = 2;
    static minScale = 1 / 4;
    static maxOriginScale = 1;

    workflow: Workflow;
    @Input()
    set workflowData(data: Workflow) {
        this.workflow = data;
        this.nodesComponent = new Map<string, ComponentRef<WorkflowWNodeComponent>>();
        this.hooksComponent = new Map<string, ComponentRef<WorkflowNodeHookComponent>>();
        this.changeDisplay();
    }

    @Input() project: Project;

    @Input()
    set direction(data: string) {
        this._direction = data;
        this._workflowStore.setDirection(this.project?.key, this.workflow?.name, this.direction);
        this.changeDisplay();
    }
    get direction() {
        return this._direction;
    }

    @Output() deleteJoinSrcEvent = new EventEmitter<{ source: any, target: any }>();

    ready: boolean;
    _direction: string;

    // workflow graph
    @ViewChild('svgGraph', { read: ViewContainerRef }) svgContainer: ViewContainerRef;
    g: dagreD3.graphlib.Graph;
    render = new dagreD3.render();

    linkWithJoin = false;

    nodesComponent = new Map<string, ComponentRef<WorkflowWNodeComponent>>();
    hooksComponent = new Map<string, ComponentRef<WorkflowNodeHookComponent>>();

    zoom: d3.ZoomBehavior<Element, {}>;
    svg: any;

    constructor(
        private componentFactoryResolver: ComponentFactoryResolver,
        private _cd: ChangeDetectorRef,
        private _workflowStore: WorkflowStore,
        private _workflowCore: WorkflowCoreService,
    ) { }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    ngAfterViewInit(): void {
        this.ready = true;
        this._cd.detectChanges();
        this.changeDisplay();
    }

    @HostListener('window:resize')
    onResize() {
        if (!this.svg) { return; }
        const element = this.svgContainer.element.nativeElement;
        this.svg.attr('width', element.offsetWidth);
        this.svg.attr('height', element.offsetHeight);
    }

    changeDisplay(): void {
        if (!this.ready) {
            return;
        }
        this.initWorkflow();
    }

    initWorkflow() {
        if (!this.workflow) {
            return;
        }
        // https://github.com/cpettitt/dagre/wiki#configuring-the-layout
        this.g = new dagreD3.graphlib.Graph().setGraph({ rankdir: this.direction, nodesep: 10, ranksep: 15, edgesep: 5 });
        // Create all nodes
        if (this.workflow.workflow_data && this.workflow.workflow_data.node && this.workflow.workflow_data.node.id > 0) {
            this.createNode(this.workflow.workflow_data.node);
        }
        if (this.workflow.workflow_data && this.workflow.workflow_data.joins) {
            this.workflow.workflow_data.joins.forEach(j => {
                this.createNode(j);
            });
        }

        const element = this.svgContainer.element.nativeElement;
        d3.select(element).selectAll('svg').remove();
        this.svg = d3.select(element).append('svg');

        let g = this.svg.append('g');
        this.render(g, this.g);

        this.zoom = d3.zoom().scaleExtent([
            WorkflowGraphComponent.minScale,
            WorkflowGraphComponent.maxScale
        ]).on('zoom', () => {
            if (d3.event.transform && d3.event.transform.x && d3.event.transform.x !== Number.POSITIVE_INFINITY
                && d3.event.transform.y && d3.event.transform.y !== Number.POSITIVE_INFINITY) {
                g.attr('transform', d3.event.transform);
            }
        });

        this.svg.call(this.zoom);

        this.clickOrigin();
        this._cd.markForCheck();
    }

    clickOrigin() {
        this.onResize();
        if (!this.svgContainer?.element?.nativeElement?.offsetWidth || !this.svgContainer?.element?.nativeElement?.offsetHeight) {
            return;
        }
        let w = this.svgContainer.element.nativeElement.offsetWidth - WorkflowGraphComponent.margin;
        let h = this.svgContainer.element.nativeElement.offsetHeight - WorkflowGraphComponent.margin;
        let gw = this.g.graph().width;
        let gh = this.g.graph().height;
        let oScale = Math.min(w / gw, h / gh); // calculate optimal scale for current graph
        // calculate final scale that fit min and max scale values
        let scale = Math.min(
            WorkflowGraphComponent.maxOriginScale,
            Math.max(WorkflowGraphComponent.minScale, oScale)
        );
        let centerX = (w - gw * scale + WorkflowGraphComponent.margin) / 2;
        let centerY = (h - gh * scale + WorkflowGraphComponent.margin) / 2;
        this.svg.call(this.zoom.transform, d3.zoomIdentity.translate(centerX, centerY).scale(scale));
    }

    createEdge(from: string, to: string, options: {}): void {
        this.g.setEdge(from, to, {
            ...options,
            arrowhead: 'undirected',
            style: 'stroke: #B5B7BD;stroke-width: 2px;'
        });
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
                componentRef = this.svgContainer.createComponent<WorkflowNodeHookComponent>(hookComponent);
            }
            componentRef.instance.hook = h;
            componentRef.instance.workflow = this.workflow;
            componentRef.instance.node = node;
            componentRef.changeDetectorRef.detectChanges();
            this.hooksComponent.set(hookId, componentRef);

            this.g.setNode(
                'hook-' + node.ref + '-' + hookId, <any>{
                    label: () => componentRef.location.nativeElement,
                    labelStyle: 'width: 25px;height: 25px;'
                }
            );

            this.createEdge(`hook-${node.ref}-${hookId}`, `node-${node.ref}`, {
                id: `hook-${node.ref}-${hookId}`
            });
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
                width = 42;
                height = 42;
                break;
        }

        this.g.setNode('node-' + node.ref, <any>{
            label: () => componentRef.location.nativeElement,
            shape,
            labelStyle: `width: ${width}px;height: ${height}px;`
        });

        this.createHookNode(node);

        if (node.triggers) {
            node.triggers.forEach(t => {
                this.createNode(t.child_node);
                this.createEdge('node-' + node.ref, 'node-' + t.child_node.ref, {
                    id: 'trigger-' + t.id,
                    style: 'stroke: #000000;'
                });
            });
        }

        // Create parent trigger
        if (node.type === 'join') {
            node.parents.forEach(p => {
                this.createEdge('node-' + p.parent_name, 'node-' + node.ref, {
                    id: 'join-trigger-' + p.parent_name,
                    style: 'stroke: #000000;'
                });
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
        const nodeComponentFactory = this.componentFactoryResolver.resolveComponentFactory(WorkflowWNodeComponent);
        const componentRef = this.svgContainer.createComponent<WorkflowWNodeComponent>(nodeComponentFactory);
        componentRef.instance.node = node;
        componentRef.instance.workflow = this.workflow;
        componentRef.instance.project = this.project;
        componentRef.changeDetectorRef.detectChanges();
        return componentRef;
    }
}
