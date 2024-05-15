import { GraphDirection } from "./graph.lib";
import { V2Job, V2JobGate, V2WorkflowRunJobEvent } from "./v2.workflow.run.model";
import { V2WorkflowRunJob } from "./v2.workflow.run.model";

export class StepStatus {
    step_order: number;
    status: string;
    start: string;
    done: string;
}

export class JobRun {
    status: string;
    step_status: Array<StepStatus>;
}

export class GraphNode {
    type: GraphNodeType;
    name: string;
    depends_on: Array<string>;
    sub_graph: Array<GraphNode>;
    job: V2Job;
    gate: V2JobGate;
    run: V2WorkflowRunJob;
    runs: Array<V2WorkflowRunJob>;
    event: V2WorkflowRunJobEvent;

    static generateMatrixOptions(matrix: { [key: string]: Array<string> }): Array<Map<string, string>> {
        const generateMatrix = (matrix: { [key: string]: string[] }, keys: string[], keyIndex: number, current: Map<string, string>, alls: Array<Map<string, string>>) => {
            if (current.size == keys.length) {
                let combi = new Map<string, string>();
                current.forEach((v, k) => {
                    combi.set(k, v);
                });
                alls.push(combi);
                return;
            }
            let key = keys[keyIndex];
            let values = matrix[key];
            values.forEach(v => {
                current.set(key, v);
                generateMatrix(matrix, keys, keyIndex + 1, current, alls);
                current.delete(key);
            });
        };
        let alls = new Array<Map<string, string>>();
        generateMatrix(matrix, Object.keys(matrix), 0, new Map<string, string>(), alls);
        return alls;
    }
}

export enum GraphNodeType {
    Job = 'job',
    Stage = "stage",
    Matrix = "matrix"
}

export class NavigationGraph {
    nodes: { [key: string]: NavigationGraphNode } = {};
    links: Array<NavigationGraphLink> = [];

    constructor(nodes: Array<GraphNode>, direction: GraphDirection) {
        // Create root join
        this.nodes['root'] = new NavigationGraphNode(NavigationGraphNodeType.Join);

        // Create each node
        nodes.forEach(n => {
            this.nodes[`in-${n.name}`] = new NavigationGraphNode(NavigationGraphNodeType.Join);
            this.nodes[`out-${n.name}`] = new NavigationGraphNode(NavigationGraphNodeType.Join);

            // Connect each nodes to its parents or to the root node if no parents defined
            if (n.depends_on && n.depends_on.length > 0) {
                n.depends_on.map(d => `out-${d}`).forEach((d, i) => {
                    const prio = this.getChildren(d).length;
                    this.links.push(new NavigationGraphLink(d, `in-${n.name}`, prio, i));
                });
            } else {
                const prio = this.getChildren('root').length;
                this.links.push(new NavigationGraphLink('root', `in-${n.name}`, prio));
            }

            switch (n.type) {
                case GraphNodeType.Stage:
                    n.sub_graph.forEach(sub => {
                        this.nodes[`in-${n.name}-${sub.name}`] = new NavigationGraphNode(NavigationGraphNodeType.Join);
                        this.nodes[`out-${n.name}-${sub.name}`] = new NavigationGraphNode(NavigationGraphNodeType.Join);

                        // Connect each nodes to its parents or to the subgraph in node if no parents defined
                        if (sub.depends_on && sub.depends_on.length > 0) {
                            sub.depends_on.map(d => `out-${n.name}-${d}`).forEach((d, i) => {
                                const prio = this.getChildren(d).length;
                                this.links.push(new NavigationGraphLink(d, `in-${n.name}-${sub.name}`, prio, i));
                            });
                        } else {
                            const prio = this.getChildren(`in-${n.name}`).length;
                            this.links.push(new NavigationGraphLink(`in-${n.name}`, `in-${n.name}-${sub.name}`, prio));
                        }

                        switch (sub.type) {
                            case GraphNodeType.Matrix:
                                const alls = GraphNode.generateMatrixOptions(sub.job.strategy.matrix);
                                const keys = alls.map(option => Array.from(option.keys()).sort().map(key => `${key}:${option.get(key)}`).join(', '));
                                keys.forEach((k, i) => {
                                    this.nodes[`${n.name}-${sub.name}-${k}`] = new NavigationGraphNode(NavigationGraphNodeType.Job);
                                    if (direction === GraphDirection.HORIZONTAL) {
                                        this.links.push(new NavigationGraphLink(`in-${n.name}-${sub.name}`, `${n.name}-${sub.name}-${k}`, i));
                                        this.links.push(new NavigationGraphLink(`${n.name}-${sub.name}-${k}`, `out-${n.name}-${sub.name}`, i));
                                    } else {
                                        if (i === 0) {
                                            this.links.push(new NavigationGraphLink(`in-${n.name}-${sub.name}`, `${n.name}-${sub.name}-${k}`, 0));
                                        } else {
                                            this.links.push(new NavigationGraphLink(`${n.name}-${sub.name}-${keys[i - 1]}`, `${n.name}-${sub.name}-${k}`, 0));
                                        }
                                        if (i === keys.length - 1) {
                                            this.links.push(new NavigationGraphLink(`${n.name}-${sub.name}-${k}`, `out-${n.name}-${sub.name}`, 0));
                                        }
                                    }
                                });
                                break;
                            default:
                                this.nodes[`${n.name}-${sub.name}`] = new NavigationGraphNode(NavigationGraphNodeType.Job);
                                this.links.push(new NavigationGraphLink(`in-${n.name}-${sub.name}`, `${n.name}-${sub.name}`, 0));
                                this.links.push(new NavigationGraphLink(`${n.name}-${sub.name}`, `out-${n.name}-${sub.name}`, 0));
                                break;
                        }
                    });

                    // Connect subgraph leaf to out node, search for out links on each nodes, if no link found it's a leaf
                    let prio = 0;
                    n.sub_graph.forEach(sub => {
                        if (!this.links.find(l => l.in === `out-${n.name}-${sub.name}`)) {
                            this.links.push(new NavigationGraphLink(`out-${n.name}-${sub.name}`, `out-${n.name}`, prio));
                            prio++
                        };
                    });
                    break;
                case GraphNodeType.Matrix:
                    const alls = GraphNode.generateMatrixOptions(n.job.strategy.matrix);
                    const keys = alls.map(option => Array.from(option.keys()).sort().map(key => `${key}:${option.get(key)}`).join(', '));
                    keys.forEach((k, i) => {
                        this.nodes[`${n.name}-${k}`] = new NavigationGraphNode(NavigationGraphNodeType.Job);
                        if (direction === GraphDirection.HORIZONTAL) {
                            this.links.push(new NavigationGraphLink(`in-${n.name}`, `${n.name}-${k}`, i));
                            this.links.push(new NavigationGraphLink(`${n.name}-${k}`, `out-${n.name}`, i));
                        } else {
                            if (i === 0) {
                                this.links.push(new NavigationGraphLink(`in-${n.name}`, `${n.name}-${k}`, 0));
                            } else {
                                this.links.push(new NavigationGraphLink(`${n.name}-${keys[i - 1]}`, `${n.name}-${k}`, 0));
                            }
                            if (i === keys.length - 1) {
                                this.links.push(new NavigationGraphLink(`${n.name}-${k}`, `out-${n.name}`, 0));
                            }
                        }
                    });
                    break;
                default:
                    this.nodes[`${n.name}`] = new NavigationGraphNode(NavigationGraphNodeType.Job);
                    this.links.push(new NavigationGraphLink(`in-${n.name}`, n.name, 0));
                    this.links.push(new NavigationGraphLink(n.name, `out-${n.name}`, 0));
                    break;
            }
        });

        // Reduce the graph removing all the joins that only have one parent or one child
        let previousNodesCount: number = null;
        while (!previousNodesCount || previousNodesCount < Object.keys(this.nodes).length) {
            previousNodesCount = Object.keys(this.nodes).length;
            Object.keys(this.nodes).forEach(k => {
                if (this.nodes[k].type !== NavigationGraphNodeType.Join) {
                    return;
                }
                const parents = this.getParentsLinks(k);
                const children = this.getChildrenLinks(k)
                if (parents.length === 1) {
                    // Increments priority for parent's children with higher priority than current node
                    this.links = this.links.map(l => {
                        if (l.in !== parents[0].in) {
                            return l;
                        }
                        return new NavigationGraphLink(l.in, l.out, l.priority > parents[0].priority ? l.priority + (children.length - 1) : l.priority, l.reversePriority);
                    });
                    children.forEach(c => { this.links.push(new NavigationGraphLink(parents[0].in, c.out, c.priority + parents[0].priority, c.reversePriority)); });
                } else if (children.length === 1) {
                    // Increments priority for child's parents with higher priority than current node
                    this.links = this.links.map(l => {
                        if (l.out !== children[0].out) {
                            return l;
                        }
                        return new NavigationGraphLink(l.in, l.out, l.priority > children[0].priority ? l.priority + (parents.length - 1) : l.priority, l.reversePriority);
                    });
                    parents.forEach(p => { this.links.push(new NavigationGraphLink(p.in, children[0].out, p.priority + children[0].priority, p.reversePriority)); });
                }
                if (parents.length === 1 || children.length <= 1) {
                    delete this.nodes[k];
                    this.links = this.links.filter(l => l.out !== k && l.in !== k);
                }
            });
        }
    }

    getPrevious(key: string, direction: number = null): string {
        if (!key) { return this.getEntryNode(); }
        const parents = this.links.filter(l => l.out === key).sort((a, b) => a.priority < b.priority ? -1 : 1).map(l => l.in);
        if (parents.length === 0) { return this.nodes[key].type === NavigationGraphNodeType.Join ? null : key; }
        if (this.nodes[key].type !== NavigationGraphNodeType.Join && !direction) {
            if (parents.length !== 1) { direction = 0; } else {
                const neighbours = this.getChildrenLinks(parents[0]);
                const currentPrio = neighbours.find(l => l.out === key).priority ?? 0;
                direction = this.computeDirectionFromPriority(currentPrio, neighbours.length);
            }
        }
        const parentIndex = this.computeIndexFromDirection(direction, parents.length);
        if (this.nodes[parents[parentIndex]].type === NavigationGraphNodeType.Job) {
            return parents[parentIndex];
        }
        return this.getPrevious(parents[parentIndex], direction);
    }

    getNext(key: string, depth: number = 0, direction: number = null): string {
        if (!key) { return this.getEntryNode(); }
        const children = this.getChildren(key);
        if (children.length === 0) { return key; }
        if (this.nodes[key].type !== NavigationGraphNodeType.Join && !direction) {
            if (children.length !== 1) { direction = 0; } else {
                const neighbours = this.getParentsLinks(children[0]);
                const currentPrio = neighbours.find(l => l.in === key).priority ?? 0;
                direction = this.computeDirectionFromPriority(currentPrio, neighbours.length);
            }
        }
        const childrenIndex = this.computeIndexFromDirection(direction, children.length);
        if (this.nodes[children[childrenIndex]].type === NavigationGraphNodeType.Job) {
            return depth == 0 ? children[childrenIndex] : this.getNext(children[childrenIndex], depth - 1, direction);
        }
        return this.getNext(children[childrenIndex], depth > 0 ? depth - 1 : 0, direction);
    }

    getSidePrevious(key: string, depth: number = 0): string {
        if (!key) { return this.getEntryNode(); }
        const parents = this.getParents(key);
        if (parents.length === 0) { return (depth > 0 || this.nodes[key].type === NavigationGraphNodeType.Join) ? null : key; }
        const children = this.getChildren(parents[0]);
        if (children.length === 1 || children[0] === key) {
            return this.getSidePrevious(parents[0], depth + 1) ?? (depth > 0 ? null : key);
        }
        const i = children.findIndex(c => c === key);
        if (this.nodes[children[i - 1]].type === NavigationGraphNodeType.Job) {
            return depth === 0 ? children[i - 1] : this.getNext(children[i - 1], depth - 1);
        }
        return this.getNext(children[i - 1], depth - 1);
    }

    getSideNext(key: string, depth: number = 0): string {
        if (!key) { return this.getEntryNode(); }
        const parents = this.getParents(key);
        if (parents.length === 0) { return (depth > 0 || this.nodes[key].type === NavigationGraphNodeType.Join) ? null : key };
        const children = this.getChildren(parents[parents.length - 1]);
        if (children.length === 1 || children[children.length - 1] === key) {
            return this.getSideNext(parents[parents.length - 1], depth + 1) ?? (depth > 0 ? null : key);
        }
        const i = children.findIndex(c => c === key);
        if (this.nodes[children[i + 1]].type === NavigationGraphNodeType.Job) {
            return depth === 0 ? children[i + 1] : this.getNext(children[i + 1], depth - 1);
        }
        return this.getNext(children[i + 1], depth - 1);
    }

    getParentsLinks(key: string): Array<NavigationGraphLink> {
        return this.links.filter(l => l.out === key).sort((a, b) => a.reversePriority < b.reversePriority ? -1 : 1);
    }

    getParents(key: string): Array<string> {
        return this.getParentsLinks(key).map(l => l.in);
    }

    getChildrenLinks(key: string): Array<NavigationGraphLink> {
        return this.links.filter(l => l.in === key).sort((a, b) => a.priority < b.priority ? -1 : 1);
    }

    getChildren(key: string): Array<string> {
        return this.getChildrenLinks(key).map(l => l.out);
    }

    computeDirectionFromPriority(priority: number, maxPriority: number): number {
        if (priority === maxPriority / 2) { return 0; }
        return priority < maxPriority / 2 ? -1 : 1;
    }

    computeIndexFromDirection(direction: number, length: number): number {
        if (length <= 1) { return 0; }
        if (direction === 0) {
            return Math.round(length / 2) - 1
        }
        return direction < 0 ? 0 : length - 1;
    }

    getEntryNode(): string {
        const key = Object.keys(this.nodes).find(n => this.links.findIndex(l => l.out === n) === -1);
        if(this.nodes[key].type === NavigationGraphNodeType.Job) {
            return key
        }
        return this.getNext(key);
    }
}

export class NavigationGraphNode {
    type: NavigationGraphNodeType;

    constructor(type: NavigationGraphNodeType) {
        this.type = type;
    }
}

export class NavigationGraphLink {
    in: string
    out: string;
    priority: number;
    reversePriority: number;

    constructor(inKey: string, outKey: string, priority: number, reversePriority: number = null) {
        this.in = inKey;
        this.out = outKey;
        this.priority = priority;
        this.reversePriority = reversePriority ?? priority;
    }
}

export enum NavigationGraphNodeType {
    Job = 'job',
    Join = 'join'
}





