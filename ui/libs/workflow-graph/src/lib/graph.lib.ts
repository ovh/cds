import { ComponentRef } from '@angular/core';
import * as d3 from 'd3';
import * as dagreD3 from 'dagre-d3';
import { GraphNode, GraphNodeType } from './graph.model';
import { NodeStatus } from './node/model';

export enum GraphDirection {
    HORIZONTAL = 'horizontal',
    VERTICAL = 'vertical'
}

export enum NodeMouseEvent {
    Enter = 'enter',
    Out = 'out'
}

export enum SelectionMode {
    Disabled = 'disabled',
    Active = 'active',
    Blocked = 'blocked'
}

export interface InteractiveNode {
    getNodes(): Array<GraphNode>;
    setHighlight(active: boolean, options?: any): void;
    selectNode(navigationKey: string): void;
    activateNode(navigationKey: string): void;
    setRunActive(active: boolean): void;
    match(navigateKey: string): boolean;
    setSelectionMode(navigationKey: string, mode: SelectionMode): void;
    setSelected(selectedRunJobIds: Array<string>): void;
}

export type ComponentFactory<T> = (nodes: Array<GraphNode>, type: string) => ComponentRef<T>;

export class Node {
    type: string;
    key: string;
    width: number;
    height: number;
}

export class Edge {
    from: string;
    to: string;
    options: {};
}

export class WorkflowV2Graph<T extends InteractiveNode> {
    static margin = 40; // let 40px on top and bottom of the graph
    static marginSubGraph = 20; // let 20px on top and bottom of the sub graph
    static maxOriginScale = 1;

    nodesComponent = new Map<string, ComponentRef<T>>();
    nodes = new Array<Node>();
    edges = new Array<Edge>();
    direction: GraphDirection = GraphDirection.HORIZONTAL;
    zoom: d3.ZoomBehavior<Element, {}>;
    svg: d3.Selection<any, any, any, any>;
    g: d3.Selection<any, any, any, any>;
    graph: dagreD3.graphlib.Graph;
    render: dagreD3.Render;
    minScale = 1;
    maxScale = 1;
    transformed: any = null;
    previousTransformed: any = null;
    componentFactory: ComponentFactory<T>;
    nodeOutNames: { [key: string]: string } = {};
    nodeInNames: { [key: string]: string } = {};
    forks: { [key: string]: { parents: Array<string>, children: Array<string> } } = {};
    joins: { [key: string]: { parents: Array<string>, children: Array<string> } } = {};
    nodeStatus: { [key: string]: string } = {};
    centeredNode: string = null;
    currentScale: number = 1;
    currentWidth: number = 0;
    currentHeight: number = 0;
    lassoEnabled: boolean = false;
    lassoRect: d3.Selection<any, any, any, any> = null;
    lassoStart: { x: number, y: number } = null;
    lassoMouseHandlers: { mousedown: any, mousemove: any, mouseup: any } = null;
    lassoSelectionCallback: (diff: {
        added: string[], removed: string[], covered: Array<string>
    }) => void = null;
    lassoCurrentSelection: Array<string> = [];

    constructor(
        factory: ComponentFactory<T>,
        direction: GraphDirection,
        minScale: number = 1,
        maxScale: number = 1
    ) {
        this.componentFactory = factory;
        this.direction = direction;
        this.graph = new dagreD3.graphlib.Graph().setGraph({
            rankdir: this.direction === GraphDirection.VERTICAL ? 'TB' : 'LR',
            nodesep: 10,  // minimum separation (px) between nodes on the same rank
            ranksep: 10,  // minimum separation (px) between ranks
            edgesep: 0
        });
        this.minScale = minScale;
        this.maxScale = maxScale;
        this.render = new dagreD3.render();
        this.render.shapes().customRectH = WorkflowV2Graph.customRect(GraphDirection.HORIZONTAL);
        this.render.shapes().customRectV = WorkflowV2Graph.customRect(GraphDirection.VERTICAL);
        this.render.shapes().customRectForMatrixH = WorkflowV2Graph.customRectForMatrix(GraphDirection.HORIZONTAL);
        this.render.shapes().customRectForMatrixV = WorkflowV2Graph.customRectForMatrix(GraphDirection.VERTICAL);
        this.render.shapes().customCircle = WorkflowV2Graph.customCircle;
        this.render.arrows().customArrow = WorkflowV2Graph.customArrow;
    }

    static customRect = (direction: GraphDirection) => (parent, bbox, node) => {
        let shapeSvg = parent.insert('rect', ':first-child')
            .attr('rx', node.rx)
            .attr('ry', node.ry)
            .attr('x', -bbox.width / 2)
            .attr('y', -bbox.height / 2)
            .attr('width', bbox.width)
            .attr('height', bbox.height);

        node.intersect = (point) => {
            if (direction === GraphDirection.VERTICAL) {
                const h = ((node.height) / 2);
                return { x: node.x, y: node.y + (point.y < node.y ? -h : h) };
            }
            const w = ((node.width) / 2);
            return { x: node.x + (point.x < node.x ? -w : w), y: node.y };
        };

        return shapeSvg;
    };

    static customRectForMatrix = (direction: GraphDirection) => (parent, bbox, node) => {
        let shapeSvg = parent.insert('rect', ':first-child')
            .attr('rx', node.rx)
            .attr('ry', node.ry)
            .attr('x', -bbox.width / 2)
            .attr('y', -bbox.height / 2)
            .attr('width', bbox.width)
            .attr('height', bbox.height);

        node.intersect = (point) => {
            if (direction === GraphDirection.VERTICAL) {
                const h = ((node.height) / 2);
                return { x: node.x, y: (node.y + (point.y < node.y ? -h : h)) - 30 };
            }
            const w = ((node.width) / 2);
            return { x: node.x + (point.x < node.x ? -w : w), y: node.y };
        };

        return shapeSvg;
    };

    static customCircle = (parent, bbox, node) => {
        let r = Math.max(bbox.width, bbox.height) / 2;
        let shapeSvg = parent.insert('circle', ':first-child')
            .attr('x', -bbox.width / 2)
            .attr('y', -bbox.height / 2)
            .attr('r', r);

        node.intersect = point => ({ x: node.x, y: node.y });

        return shapeSvg;
    };

    static customArrow = (parent, id, edge, type) => {
        let markerHead = parent.append('marker')
            .attr('id', id)
            .attr('viewBox', '0 0 10 10')
            .attr('refX', 0)
            .attr('refY', 5)
            .attr('markerUnits', 'strokeWidth')
            .attr('markerWidth', 5)
            .attr('markerHeight', 5)
            .attr('orient', 'auto');
        let pathHead = markerHead.append('path')
            .attr('d', 'M 0 5 L 10 5');
        pathHead.attr('style', edge['style']);
        pathHead.attr('class', edge['class']);

        const arrowEndId = id.replace('head', 'start');
        let markerEnd = parent.append('marker')
            .attr('id', arrowEndId)
            .attr('viewBox', '0 0 10 10')
            .attr('refX', 10)
            .attr('refY', 5)
            .attr('markerUnits', 'strokeWidth')
            .attr('markerWidth', 5)
            .attr('markerHeight', 5)
            .attr('orient', 'auto');
        let pathEnd = markerEnd.append('path')
            .attr('d', 'M 0 5 L 10 5');
        pathEnd.attr('style', edge['style']);
        pathEnd.attr('class', edge['class']);

        const makeFragmentRef = (url, fragmentId) => {
            const baseUrl = url.split('#')[0];
            return `${baseUrl}#${fragmentId}`;
        };

        const defs = parent._groups[0][0];
        const edgePath = d3.select(defs.parentNode.childNodes[0]);
        edgePath.attr('marker-start', () => `url(${makeFragmentRef(location.href, arrowEndId)})`);
    };

    resize(width: number, height: number): void {
        if (!this.svg) {
            return;
        }
        const diffWidth = (this.currentWidth - width);
        const diffHeight = (this.currentHeight - height);
        this.currentHeight = height;
        this.currentWidth = width;
        this.svg.attr('width', width);
        this.svg.attr('height', height);
        if (this.centeredNode) {
            this.centerNode(this.centeredNode);
        } else if (this.previousTransformed) {
            this.svg.call(this.zoom.transform,
                d3.zoomIdentity.translate(this.previousTransformed.x, this.previousTransformed.y).scale(this.previousTransformed.k));
        } else if (this.transformed) {
            this.svg.call(this.zoom.transform,
                d3.zoomIdentity.translate(this.transformed.x - diffWidth / 2, this.transformed.y - diffHeight / 2).scale(this.transformed.k));
        } else {
            this.center();
        }
    }

    clean(): void {
        this.graph.edges().forEach(e => this.graph.removeEdge(e.v, e.w));
        this.graph.nodes().forEach(n => this.graph.removeNode(n));
        this.nodesComponent.forEach(c => c.destroy());
        this.nodesComponent = new Map<string, ComponentRef<T>>();
        this.nodes = new Array<Node>();
        this.edges = new Array<Edge>();
        this.nodeStatus = {};
    }

    draw(element: any, withZoom: boolean): void {
        d3.select(element).selectAll('svg').remove();
        this.svg = d3.select(element).insert('svg');
        this.g = this.svg.insert('g');

        this.graph.edges().forEach(e => this.graph.removeEdge(e.v, e.w));
        this.graph.nodes().forEach(n => this.graph.removeNode(n));

        this.drawNodes();
        this.drawEdges();

        this.render(this.g, <any>this.graph);

        if (withZoom) {
            this.zoom = d3.zoom().scaleExtent([this.minScale, this.maxScale]).on('zoom', (event) => {
                if (event.transform && event.transform.x && event.transform.x !== Number.POSITIVE_INFINITY
                    && event.transform.y && event.transform.y !== Number.POSITIVE_INFINITY) {
                    this.g.attr('transform', event.transform);
                    this.transformed = event.transform;
                    this.currentScale = event.transform.k;
                    this.centeredNode = null;
                    this.previousTransformed = null;
                }
            });
            if (this.transformed) {
                this.svg.call(this.zoom.transform,
                    d3.zoomIdentity.translate(this.transformed.x, this.transformed.y).scale(this.transformed.k));
            }
            this.svg.call(this.zoom);
        }
    }

    /**
     * Reset the viewport to show the entire graph centered at optimal scale.
     * Scale is capped at maxOriginScale (1 = 100%) so the graph never appears zoomed in.
     * Clears centeredNode, transformed, and previousTransformed state.
     * Without zoom (sub-graphs inside stages), applies the transform directly.
     */
    center(): void {
        if (!this.zoom) {
            const w = this.currentWidth - WorkflowV2Graph.marginSubGraph * 2;
            const h = this.currentHeight - WorkflowV2Graph.marginSubGraph * 2;
            const gw = this.graph.graph().width;
            const gh = this.graph.graph().height;
            const oScale = Math.min(w / gw, h / gh); // calculate optimal scale for current graph
            const centerX = (w - gw * oScale + WorkflowV2Graph.marginSubGraph * 2) / 2;
            const centerY = (h - gh * oScale + WorkflowV2Graph.marginSubGraph * 2) / 2;
            this.g.attr('transform', `translate(${centerX}, ${centerY}) scale(${oScale})`);
            return;
        }
        const w = this.currentWidth - WorkflowV2Graph.margin * 2;
        const h = this.currentHeight - WorkflowV2Graph.margin * 2;
        const gw = this.graph.graph().width;
        const gh = this.graph.graph().height;
        const oScale = Math.min(w / gw, h / gh); // calculate optimal scale for current graph
        // calculate final scale that fit min and max scale values
        const scale = Math.min(
            WorkflowV2Graph.maxOriginScale,
            Math.max(this.minScale, oScale)
        );
        const centerX = (w - gw * scale + WorkflowV2Graph.margin * 2) / 2;
        const centerY = (h - gh * scale + WorkflowV2Graph.margin * 2) / 2;
        this.svg.call(this.zoom.transform, d3.zoomIdentity.translate(centerX, centerY).scale(scale));
        this.centeredNode = null;
        this.transformed = null;
        this.previousTransformed = null;
    }

    /**
     * Center and zoom the viewport on a specific stage node.
     * Computes optimal scale for the stage bounding box, capped at maxOriginScale.
     * Clears centeredNode so the stage won't persist as focus target on resize.
     */
    centerStage(nodeName: string): void {
        if (!this.zoom) {
            return;
        }
        const node = this.graph.node(nodeName);
        // calculate optimal scale for current graph
        const oScale = Math.min((this.currentWidth - WorkflowV2Graph.margin * 2) / node.width,
            (this.currentHeight - WorkflowV2Graph.margin * 2) / node.height);
        // calculate final scale that fit min and max scale values
        const scale = Math.min(WorkflowV2Graph.maxOriginScale, Math.max(this.minScale, oScale));
        const nodeDeltaCenterX = this.currentWidth / 2 - node.x * scale;
        const nodeDeltaCenterY = this.currentHeight / 2 - node.y * scale;
        this.svg.call(this.zoom.transform, d3.zoomIdentity.translate(nodeDeltaCenterX, nodeDeltaCenterY).scale(scale));
        this.centeredNode = null;
    }

    /**
     * Resolve a navigation key to dagre node coordinates.
     * For top-level nodes, matches directly. For stage-nested nodes, iterates stage
     * sub-graphs and returns coordinates offset by the stage's top-left corner.
     */
    getSVGNodeForNavigationKey(navigationKey: string): any {
        let node;
        for (let i = 0; i < this.nodes.length; i++) {
            if (this.nodes[i].type === GraphNodeType.Stage) {
                continue;
            }
            if (this.nodesComponent.get(`node-${this.nodes[i].key}`).instance.match(navigationKey)) {
                return this.graph.node(`node-${this.nodes[i].key}`);
            }
        }
        for (let i = 0; i < this.nodes.length; i++) {
            if (this.nodes[i].type === GraphNodeType.Stage) {
                const subNode = (this.nodesComponent.get(`node-${this.nodes[i].key}`).instance as any).graph.getSVGNodeForNavigationKey(navigationKey);
                if (subNode) {
                    const stageNode = this.graph.node(`node-${this.nodes[i].key}`);
                    node = {
                        ...subNode,
                        x: (stageNode.x - stageNode.width / 2) + subNode.x,
                        y: (stageNode.y - stageNode.height / 2) + subNode.y
                    };
                    break;
                }
            }
        }
        return node;
    }

    /**
     * Center the viewport on a specific node, preserving current zoom scale.
     * Resolves node coordinates via getSVGNodeForNavigationKey (handles stage-nested nodes).
     *
     * @param transform If true (hard reset): clears centeredNode and previousTransformed.
     *   Used by keyboard navigation — no return-to-previous position.
     *   If false (default, soft center): saves the current transform for future restore,
     *   sets centeredNode so the node stays centered on resize.
     *   Used by node click — allows returning to previous viewport.
     */
    centerNode(navigationKey: string, transform: boolean = false): void {
        if (!this.zoom) {
            return;
        }
        const node = this.getSVGNodeForNavigationKey(navigationKey);
        if (!node) {
            return;
        }
        const nodeDeltaCenterX = this.currentWidth / 2 - node.x * this.currentScale;
        const nodeDeltaCenterY = this.currentHeight / 2 - node.y * this.currentScale;
        if (transform) {
            this.svg.call(this.zoom.transform, d3.zoomIdentity.translate(nodeDeltaCenterX, nodeDeltaCenterY).scale(this.currentScale));
            this.centeredNode = null;
            this.previousTransformed = null;
            return;
        }
        const previousTransformation = this.previousTransformed ?? (this.transformed ? { ...this.transformed } : null);
        this.svg.call(this.zoom.transform, d3.zoomIdentity.translate(nodeDeltaCenterX, nodeDeltaCenterY).scale(this.currentScale));
        this.previousTransformed = previousTransformation;
        this.centeredNode = navigationKey;
        this.transformed = null;
    }

    drawNodes(): void {
        this.nodes.forEach(n => {
            switch (n.type) {
                case GraphNodeType.Matrix:
                    this.createGNode(`node-${n.key}`, this.nodesComponent.get(`node-${n.key}`), n.width, n.height, {
                        class: n.key,
                        shape: this.direction === GraphDirection.VERTICAL ? 'customRectForMatrixV' : 'customRectForMatrixH'
                    });
                    break;
                default:
                    this.createGNode(`node-${n.key}`, this.nodesComponent.get(`node-${n.key}`), n.width, n.height, {
                        class: n.key
                    });
                    break;
            }
        });
    }

    uniqueStrings(a: Array<string>): Array<string> {
        let o = {};
        a.forEach(s => o[s] = true);
        return Object.keys(o);
    }

    drawEdges(): void {
        let nodesChildren: { [key: string]: Array<string> } = {};
        let nodesParents: { [key: string]: Array<string> } = {};

        this.edges.forEach(e => {
            if (!nodesChildren[e.from]) {
                nodesChildren[e.from] = [e.to];
            } else {
                nodesChildren[e.from] = this.uniqueStrings([...nodesChildren[e.from], e.to]);
            }
            if (!nodesParents[e.to]) {
                nodesParents[e.to] = [e.from];
            } else {
                nodesParents[e.to] = this.uniqueStrings([...nodesParents[e.to], e.from]);
            }
        });

        // Create fork
        this.forks = {};
        this.nodeOutNames = {};
        Object.keys(nodesChildren).forEach(c => {
            if (nodesChildren[c].length > 1) {
                const children = nodesChildren[c].map(n => n.split('node-')[1]).sort();
                const keyFork = children.join('-');
                if (this.forks[keyFork]) {
                    this.forks[keyFork].parents = this.forks[keyFork].parents.concat(c);
                    this.forks[keyFork].children = this.uniqueStrings(this.forks[keyFork].children.concat(nodesChildren[c]));
                } else {
                    this.forks[keyFork] = {
                        parents: [c],
                        children: nodesChildren[c]
                    };
                }
            }
        });
        Object.keys(this.forks).forEach(f => {
            let nodes = this.forks[f].parents.map(n => this.nodesComponent.get(n).instance.getNodes()).reduce((p, c) => p.concat(c));
            const componentRef = this.componentFactory(nodes, 'fork');
            let nodeKeys = this.forks[f].parents.map(n => n.split('node-')[1]).sort().join(' ');
            this.createGFork(f, componentRef, { class: `${nodeKeys}` });
            this.nodeStatus[`fork-${f}`] = NodeStatus.sum(nodes.map(n => n.run ? n.run.status : null));
            this.forks[f].parents.forEach(n => {
                let edge = <Edge>{
                    from: n,
                    to: `fork-${f}`,
                    options: {
                        class: `${f} ${n.split('node-')[1]}`
                    }
                };
                if (this.nodeStatus[`fork-${f}`]) {
                    const color = this.nodeStatusToColor(this.nodeStatus[`fork-${f}`]);
                    edge.options['class'] += ' ' + color;
                    edge.options['style'] = 'stroke-width: 2px;';
                }
                this.createGEdge(edge);
                this.nodeOutNames[n] = `fork-${f}`;
            });
        });

        // Create join
        this.joins = {};
        this.nodeInNames = {};
        Object.keys(nodesParents).forEach(p => {
            if (nodesParents[p].length > 1) {
                const parents = nodesParents[p].map(n => n.split('node-')[1]).sort();
                const keyJoin = parents.join('-');
                if (this.joins[keyJoin]) {
                    this.joins[keyJoin].children = this.joins[keyJoin].children.concat(p);
                    this.joins[keyJoin].parents = this.uniqueStrings(this.joins[keyJoin].children.concat(nodesParents[p]));
                } else {
                    this.joins[keyJoin] = {
                        children: [p],
                        parents: nodesParents[p]
                    };
                }
            }
        });
        Object.keys(this.joins).forEach(j => {
            let nodes = this.joins[j].parents.map(n => this.nodesComponent.get(n).instance.getNodes()).reduce((p, c) => p.concat(c));
            const componentRef = this.componentFactory(nodes, 'join');
            let nodeKeys = this.joins[j].children.map(n => n.split('node-')[1]).sort().join(' ');
            this.createGJoin(j, componentRef, { class: `${nodeKeys}` });
            this.nodeStatus[`join-${j}`] = NodeStatus.sum(nodes.map(n => n.run ? n.run.status : null));
            this.joins[j].children.forEach(n => {
                let edge = <Edge>{
                    from: `join-${j}`,
                    to: n,
                    options: {
                        class: `${j} ${n.split('node-')[1]}`
                    }
                };
                if (this.nodeStatus[`join-${j}`]) {
                    const color = this.nodeStatusToColor(this.nodeStatus[`join-${j}`]);
                    edge.options['class'] += ' ' + color;
                    edge.options['style'] = 'stroke-width: 2px;';
                }
                this.createGEdge(edge);
                this.nodeInNames[n] = `join-${j}`;
            });
        });

        let uniqueEdges: { [key: string]: { from: string, to: string, classes: Array<string> } } = {};
        this.edges.forEach(e => {
            let from = this.nodeOutNames[e.from] ?? e.from;
            let to = this.nodeInNames[e.to] ?? e.to;
            let edgeKey = `${from}-${to}`;
            let nodeKeyFrom = e.from.split('node-')[1];
            let nodeKeyTo = e.to.split('node-')[1];
            if (!uniqueEdges[edgeKey]) {
                uniqueEdges[edgeKey] = { from, to, classes: [nodeKeyFrom, nodeKeyTo] };
                return;
            }
            uniqueEdges[edgeKey].classes = this.uniqueStrings(uniqueEdges[edgeKey].classes.concat(nodeKeyFrom, nodeKeyTo));
        });

        Object.keys(uniqueEdges).forEach(edgeKey => {
            let e = uniqueEdges[edgeKey];
            let options = {
                class: e.classes.join(' ')
            };
            if (this.nodeStatus[e.from]) {
                const color = this.nodeStatusToColor(this.nodeStatus[e.from]);
                options['class'] += ' ' + color;
                options['style'] = 'stroke-width: 2px;';
            }
            this.createGEdge(<Edge>{
                from: e.from, to: e.to, options
            });
        });
    }

    nodeStatusToColor(s: string): string {
        switch (s) {
            case NodeStatus.SUCCESS:
                return 'color-success';
            case NodeStatus.FAIL:
                return 'color-fail';
            case NodeStatus.WAITING:
            case NodeStatus.SCHEDULING:
            case NodeStatus.BUILDING:
            case NodeStatus.STOPPED:
                return 'color-inactive';
            default:
                return '';
        }
    }

    createNode(key: string, node: GraphNode, componentRef: ComponentRef<T>, h?: number, w?: number): void {
        // Default dimensions for a standard job node (200 × 60 px)
        let width = 200
        let height = 60;
        switch (node.type) {
            case GraphNodeType.Stage:
                width = w;
                height = h;
                break;
            case GraphNodeType.Matrix:
                // Matrix node: 240 px wide, height = 30 per variant row + 10 px gap between rows + 40 header + 40 footer + 20 padding
                width = 240;
                const alls = GraphNode.generateMatrixOptions(node.job.strategy.matrix);
                height = 30 * alls.length + 10 * (alls.length - 1) + 40 + 40 + 20;
                break;
        }
        this.nodes.push(<Node>{ type: node.type, key, width, height });
        this.nodesComponent.set(`node-${key}`, componentRef);
    }

    setNodeStatus(key: string, status: string): void {
        this.nodeStatus[`node-${key}`] = status;
    }

    createGNode(name: string, componentRef: ComponentRef<T>, width: number, height: number, options: {}): void {
        this.graph.setNode(name, <any>{
            shape: this.direction === GraphDirection.VERTICAL ? 'customRectV' : 'customRectH',
            label: () => componentRef.location.nativeElement,
            labelStyle: `width: ${width}px;height: ${height}px;`,
            width,
            height,
            ...options
        });
    }

    /** Creates a fork node — a small circle (60 × 60 px) inserted where edges diverge. */
    createGFork(key: string, componentRef: ComponentRef<T>, options: {}): void {
        this.nodesComponent.set(`fork-${key}`, componentRef);
        this.createGNode(`fork-${key}`, componentRef, 20, 20, {
            shape: 'customCircle',
            width: 60,
            height: 60,
            ...options
        });
    }

    /** Creates a join node — a small circle (60 × 60 px) inserted where edges converge. */
    createGJoin(key: string, componentRef: ComponentRef<T>, options: {}): void {
        this.nodesComponent.set(`join-${key}`, componentRef);
        this.createGNode(`join-${key}`, componentRef, 20, 20, {
            shape: 'customCircle',
            width: 60,
            height: 60,
            ...options
        });
    }

    createEdge(from: string, to: string): void {
        this.edges.push(<Edge>{ from, to });
    }

    createGEdge(e: Edge): void {
        this.graph.setEdge(e.from, e.to, {
            arrowhead: 'customArrow',
            style: 'stroke: #B5B7BD;stroke-width: 2px;',
            curve: d3.curveBasis,
            ...e.options
        });
    }

    nodeMouseEvent(type: NodeMouseEvent, key: string, options?: any): void {
        switch (type) {
            case NodeMouseEvent.Enter:
                this.highlightNode(true, key, options);
                break;
            case NodeMouseEvent.Out:
                this.highlightNode(false, key, options);
                break;
        }
    }

    selectNode(navigationKey: string): void {
        this.nodesComponent.forEach(n => n.instance.selectNode(navigationKey));
    }

    activateNode(navigationKey: string): void {
        this.nodesComponent.forEach(n => n.instance.activateNode(navigationKey));
    }

    setRunActive(active: boolean): void {
        this.nodesComponent.forEach(n => n.instance.setRunActive(active));
    }

    highlightNode(active: boolean, key: string, options?: any) {
        const keyEscape = key.replace('.', '\\.');
        let selectionEdges = d3.selectAll(`.${keyEscape} > .path`);
        if (selectionEdges.size() > 0) {
            selectionEdges.attr('class', active ? 'path highlight' : 'path');
        }
        let selectionEdgeMarkers = d3.selectAll(`.${keyEscape} > defs > marker > path`);
        if (selectionEdgeMarkers.size() > 0) {
            selectionEdgeMarkers.attr('class', active ? 'highlight' : '');
        }
        if (this.nodesComponent.has(`node-${key}`)) {
            this.nodesComponent.get(`node-${key}`).instance.setHighlight(active, options);
        }
        let inName = this.nodeInNames[`node-${key}`];
        if (inName !== `node-${key}`) {
            if (this.nodesComponent.has(inName)) {
                this.nodesComponent.get(inName).instance.setHighlight(active, options);
            }
        }
        let outName = this.nodeOutNames[`node-${key}`];
        if (outName !== `node-${key}`) {
            if (this.nodesComponent.has(outName)) {
                this.nodesComponent.get(outName).instance.setHighlight(active, options);
            }
        }
    }

    /** Broadcast the selection mode to every node. */
    setNodeSelectionMode(navigationKey: string, mode: SelectionMode): void {
        this.nodesComponent.forEach(n => n.instance.setSelectionMode(navigationKey, mode));
    }

    /**
     * Broadcast the full set of selected run job IDs to every node.
     * Each node self-determines its selection state from its own run job ID(s).
     * Stage nodes forward to their sub-graph automatically.
     */
    updateSelection(selectedRunJobIds: Array<string>): void {
        this.nodesComponent.forEach(n => n.instance.setSelected(selectedRunJobIds));
    }

    /**
     * Compute which nodes/matrix rows intersect a lasso rectangle.
     * Uses DOM getBoundingClientRect() on the actual rendered elements, which
     * inherently accounts for all SVG transforms (zoom, pan, sub-graph scale).
     * No manual coordinate math or layout-constant assumptions needed.
     *
     * @param lx1,ly1,lx2,ly2 Lasso rectangle in SVG-local pixel coords.
     */
    computeLassoHits(lx1: number, ly1: number, lx2: number, ly2: number): Array<string> {
        const hitRunJobIds: Array<string> = [];

        // Convert lasso SVG-local coords to viewport coords
        const svgEl = this.svg.node() as SVGSVGElement;
        const svgRect = svgEl.getBoundingClientRect();
        const vLx1 = lx1 + svgRect.left;
        const vLy1 = ly1 + svgRect.top;
        const vLx2 = lx2 + svgRect.left;
        const vLy2 = ly2 + svgRect.top;

        const rectsIntersect = (r: DOMRect) =>
            r.width > 0 && r.height > 0 &&
            r.left < vLx2 && r.right > vLx1 &&
            r.top < vLy2 && r.bottom > vLy1;

        /**
         * Check a single node component against the lasso.
         * For matrix nodes, drills into individual .job-wrapper rows.
         * For simple jobs, uses the node's run job ID.
         * All hits are collected as run job IDs uniformly.
         */
        const checkNode = (node: Node, comp: ComponentRef<T>) => {
            const el = comp.location.nativeElement as HTMLElement;
            const rect = el.getBoundingClientRect();
            if (!rectsIntersect(rect)) { return; }

            if (node.type === 'matrix') {
                const matrixComp = comp.instance as any;
                const keys: string[] = matrixComp.keys || [];
                const jobRunIDs: { [key: string]: string } = matrixComp.jobRunIDs || {};
                const rowEls = el.querySelectorAll('.job-wrapper');
                rowEls.forEach((rowEl, i) => {
                    if (i < keys.length && rectsIntersect(rowEl.getBoundingClientRect())) {
                        const uuid = jobRunIDs[keys[i]];
                        if (uuid) { hitRunJobIds.push(uuid); }
                    }
                });
            } else {
                const runJobId = (comp.instance as any).node?.run?.id;
                if (runJobId) { hitRunJobIds.push(runJobId); }
            }
        };

        // Check all nodes (top-level + stage sub-graphs)
        this.nodes.forEach(n => {
            if (n.type === 'stage') {
                // Iterate sub-graph nodes
                const stageComp = this.nodesComponent.get(`node-${n.key}`);
                if (!stageComp) { return; }
                const subGraph = (stageComp.instance as any).graph as WorkflowV2Graph<T>;
                if (!subGraph) { return; }
                const stagePrefix = n.key + '-';
                subGraph.nodes.forEach(sn => {
                    if (sn.type === 'stage') { return; }
                    if (!sn.key.startsWith(stagePrefix)) { return; }
                    const comp = subGraph.nodesComponent.get(`node-${sn.key}`);
                    if (!comp) { return; }
                    checkNode(sn, comp);
                });
                return;
            }
            const comp = this.nodesComponent.get(`node-${n.key}`);
            if (!comp) { return; }
            checkNode(n, comp);
        });

        return hitRunJobIds;
    }

    /**
     * Enable rectangular lasso selection on the graph.
     *
     * Visual: dashed blue rectangle with semi-transparent fill, crosshair cursor.
     * Disables all d3 zoom interactions (pan and scroll-wheel zoom) during lasso.
     *
     * On mousemove, performs live hit-testing via computeLassoHits() and emits
     * incremental diffs (added, removed, covered) to the callback. The parent
     * component uses these diffs to manage selection state with descendant constraints.
     */
    enableLasso(callback: (diff: {
        added: string[], removed: string[], covered: Array<string>
    }) => void): void {
        if (!this.svg || this.lassoEnabled) return;
        this.lassoEnabled = true;
        this.lassoSelectionCallback = callback;
        this.lassoCurrentSelection = [];

        // Add crosshair cursor class to the SVG wrapper
        const svgEl = this.svg.node() as SVGSVGElement;
        if (svgEl?.parentElement) {
            svgEl.parentElement.classList.add('lasso-active');
        }

        // Disable d3 zoom pan (keep scroll-wheel zoom)
        this.svg.on('.zoom', null);

        const self = this;

        this.lassoMouseHandlers = {
            mousedown(event: MouseEvent) {
                if (event.button !== 0) return; // left-click only
                event.preventDefault();
                event.stopPropagation();
                const svgEl = self.svg.node() as SVGSVGElement;
                const rect = svgEl.getBoundingClientRect();
                self.lassoStart = {
                    x: event.clientX - rect.left,
                    y: event.clientY - rect.top
                };
                self.lassoCurrentSelection = [];
                // Create the lasso rectangle in SVG screen space (on top of the <g> transform)
                self.lassoRect = self.svg.append('rect')
                    .attr('class', 'lasso-selection')
                    .attr('x', self.lassoStart.x)
                    .attr('y', self.lassoStart.y)
                    .attr('width', 0)
                    .attr('height', 0)
                    .attr('fill', 'rgba(24, 144, 255, 0.1)')
                    .attr('stroke', '#1890ff')
                    .attr('stroke-width', 1)
                    .attr('stroke-dasharray', '4,2');
            },
            mousemove(event: MouseEvent) {
                if (!self.lassoStart || !self.lassoRect) return;
                event.preventDefault();
                const svgEl = self.svg.node() as SVGSVGElement;
                const rect = svgEl.getBoundingClientRect();
                const currentX = event.clientX - rect.left;
                const currentY = event.clientY - rect.top;
                const x = Math.min(self.lassoStart.x, currentX);
                const y = Math.min(self.lassoStart.y, currentY);
                const w = Math.abs(currentX - self.lassoStart.x);
                const h = Math.abs(currentY - self.lassoStart.y);
                self.lassoRect
                    .attr('x', x)
                    .attr('y', y)
                    .attr('width', w)
                    .attr('height', h);

                // Live hit-testing: compute intersecting nodes and emit diffs
                if (w > 5 || h > 5) {
                    const hits = self.computeLassoHits(x, y, x + w, y + h);
                    const added = hits.filter(id => !self.lassoCurrentSelection.includes(id));
                    const removed = self.lassoCurrentSelection.filter(id => !hits.includes(id));

                    if (added.length > 0 || removed.length > 0) {
                        self.lassoCurrentSelection = hits;
                        if (self.lassoSelectionCallback) {
                            self.lassoSelectionCallback({
                                added, removed, covered: [...hits]
                            });
                        }
                    }
                }
            },
            mouseup(event: MouseEvent) {
                if (!self.lassoStart || !self.lassoRect) return;
                event.preventDefault();

                // Remove lasso rect
                self.lassoRect.remove();
                self.lassoRect = null;
                self.lassoStart = null;
                self.lassoCurrentSelection = [];
            }
        };

        const svgNode = this.svg.node() as SVGSVGElement;
        svgNode.addEventListener('mousedown', this.lassoMouseHandlers.mousedown);
        window.addEventListener('mousemove', this.lassoMouseHandlers.mousemove);
        window.addEventListener('mouseup', this.lassoMouseHandlers.mouseup);
    }

    disableLasso(): void {
        if (!this.lassoEnabled) return;
        this.lassoEnabled = false;
        this.lassoSelectionCallback = null;

        // Remove crosshair cursor class from the SVG wrapper
        const svgEl = this.svg?.node() as SVGSVGElement;
        if (svgEl?.parentElement) {
            svgEl.parentElement.classList.remove('lasso-active');
        }

        // Clean up lasso rect if in progress
        if (this.lassoRect) {
            this.lassoRect.remove();
            this.lassoRect = null;
            this.lassoStart = null;
        }

        // Remove event listeners
        if (this.lassoMouseHandlers) {
            const svgNode = this.svg?.node() as SVGSVGElement;
            if (svgNode) {
                svgNode.removeEventListener('mousedown', this.lassoMouseHandlers.mousedown);
            }
            window.removeEventListener('mousemove', this.lassoMouseHandlers.mousemove);
            window.removeEventListener('mouseup', this.lassoMouseHandlers.mouseup);
            this.lassoMouseHandlers = null;
        }

        // Re-enable d3 zoom
        if (this.zoom && this.svg) {
            this.svg.call(this.zoom);
        }
    }
}
