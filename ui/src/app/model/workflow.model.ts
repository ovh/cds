import {Pipeline} from './pipeline.model';
import {Application} from './application.model';
import {Environment} from './environment.model';
import {intersection} from 'lodash';
import {Parameter} from './parameter.model';
import {WorkflowHookModel} from './workflow.hook.model';
import {GroupPermission} from './group.model';
import {Usage} from './usage.model';

// Workflow represents a pipeline based workflow
export class Workflow {
    id: number;
    name: string;
    description: string;
    project_id: number;
    project_key: string;
    root: WorkflowNode;
    root_id: number;
    joins: Array<WorkflowNodeJoin>;
    last_modified: string;
    groups: Array<GroupPermission>;
    permission: number;
    metadata: Map<string, string>;
    usage: Usage;
    history_length: number;
    purge_tags: Array<string>;

    // UI params
    externalChange: boolean;

    // Do not remove root node
    static removeNodeWithoutChild(workflow: Workflow, node: WorkflowNode): boolean {
        if (node.id === workflow.root.id) {
            if ( (workflow.root.triggers && workflow.root.triggers.length > 1) || (workflow.joins && workflow.joins.length > 1)) {
                return false;
            }
            if (workflow.root.triggers) {
                if (workflow.root.triggers.length === 1) {
                    workflow.root = workflow.root.triggers[0].workflow_dest_node;
                    workflow.root_id = workflow.root.id;
                }
            }
            if (workflow.joins) {
                let joinsIndex = new Array<number>();
                workflow.joins.forEach((j, idx) => {
                    j.source_node_id.forEach(srcId => {
                        if (node.id === srcId) {
                            joinsIndex.push(idx);
                        }
                    });
                });
                if (joinsIndex.length === 1) {
                    // remove id
                    workflow.joins[joinsIndex[0]].source_node_id = workflow.joins[joinsIndex[0]].source_node_id.filter(i => i !== node.id);
                    if ((!node.triggers || node.triggers.length === 0) && workflow.joins[joinsIndex[0]].source_node_id.length === 0) {
                        if (workflow.joins[joinsIndex[0]].triggers && workflow.joins[joinsIndex[0]].triggers.length === 1) {
                            workflow.root = workflow.joins[joinsIndex[0]].triggers[0].workflow_dest_node;
                        }
                    }
                }
            }
            if (workflow.root.id === node.id) {
                return false;
            }
        } else {
            let parentNode: WorkflowNode;
            if (workflow.root.triggers) {
                workflow.root.triggers.forEach((t, idxT) => {
                    parentNode = WorkflowNode.removeNodeWithoutChild(workflow.root, t, node.id, idxT);
                });
            }
            if (workflow.joins) {
                workflow.joins.forEach(j => {
                    j.source_node_id.forEach((srcId, index) => {
                        if (srcId === node.id) {
                            j.source_node_id.splice(index, 1);
                            if (parentNode && j.source_node_id.indexOf(parentNode.id) === -1) {
                                j.source_node_id.push(parentNode.id);
                            }
                        }
                    });
                    j.source_node_ref.forEach((srcRef, index) => {
                        if (srcRef === node.id.toString()) {
                            j.source_node_ref.splice(index, 1);
                            if (parentNode && j.source_node_ref.indexOf(parentNode.id.toString()) === -1) {
                                j.source_node_ref.push(parentNode.id.toString());
                            }
                        }
                    });
                    if (j.triggers) {
                        j.triggers.forEach((t, idxT) => {
                            parentNode = WorkflowNode.removeNodeWithoutChildFromJoinTrigger(j, t, node.id, idxT)
                        });
                    }
                });
            }
        }
        return true;
    }

    static updateHook(workflow: Workflow, h: WorkflowNodeHook) {
        let oldH = WorkflowNode.findHook(workflow.root, h.id);
        if (!oldH) {
            if (workflow.joins) {
                quit: for (let i = 0; i < workflow.joins.length; i++) {
                    let j = workflow.joins[i];
                    if (j.triggers) {
                        for (let k = 0; k < j.triggers.length; k++) {
                            oldH = WorkflowNode.findHook(j.triggers[k].workflow_dest_node, h.id);
                            if (oldH) {
                                break quit;
                            }
                        }
                    }
                }
            }
        }

        if (oldH) {
            oldH.config = h.config;
        }
    };

    static removeHook(workflow: Workflow, h: WorkflowNodeHook) {
        let done = WorkflowNode.removeHook(workflow.root, h.id);
        if (!done) {
            if (workflow.joins) {
                for (let i = 0; i < workflow.joins.length; i++) {
                    let j = workflow.joins[i];
                    if (j.triggers) {
                        for (let k = 0; k < j.triggers.length; k++) {
                            done = WorkflowNode.removeHook(j.triggers[k].workflow_dest_node, h.id);
                            if (done) {
                                return
                            }
                        }
                    }
                }
            }
        }
        return
    }

    static getNodeByID(id: number, w: Workflow): WorkflowNode {
        let node = WorkflowNode.getNodeByID(w.root, id);
        if (!node && w.joins) {
            quit: for (let i = 0; i < w.joins.length; i++) {
                if (w.joins[i].triggers) {
                    for (let j = 0; j < w.joins[i].triggers.length; j++) {
                        node = WorkflowNode.getNodeByID(w.joins[i].triggers[j].workflow_dest_node, id);
                        if (node) {
                            break quit;
                        }
                    }
                }
            }
        }
        return node;
    }

    static removeOldRef(w: Workflow) {
        if (!w.joins) {
            return;
        }
        let refs = new Array<string>();
        WorkflowNode.addRef(refs, w.root);

        w.joins.forEach(j => {
            if (j.triggers) {
                j.triggers.forEach(t => {
                    WorkflowNode.addRef(refs, t.workflow_dest_node);
                });
            }
        });

        w.joins.forEach(j => {
            j.source_node_ref = intersection(j.source_node_ref, refs);
        });
    }

    static getNodeNameImpact(workflow: Workflow, name: string): WorkflowPipelineNameImpact {
        let varName = 'workflow.' + name;
        let warnings = new WorkflowPipelineNameImpact();
        WorkflowNode.getNodeNameImpact(workflow.root, name, warnings);
        if (workflow.joins) {
            workflow.joins.forEach(j => {
                if (j.triggers) {
                    j.triggers.forEach(t => {
                        WorkflowNode.getNodeNameImpact(t.workflow_dest_node, name, warnings);
                    });
                }
            });
        }
        return warnings;
    }

    static getJoinById(id: number, workflow: Workflow): WorkflowNodeJoin {
        if (!workflow || !Array.isArray(workflow.joins)) {
            return null;
        }
        return workflow.joins.find((join) => join.id === id);
    }

    constructor() {
        this.root = new WorkflowNode();
    }
}

export class WorkflowNodeJoin {
    id: number;
    workflow_id: number;
    source_node_id: Array<number>;
    source_node_ref: Array<string>;
    triggers: Array<WorkflowNodeJoinTrigger>;

    constructor() {
        this.source_node_ref = new Array<string>();
    }
}

export class WorkflowNodeJoinTrigger {
    id: number;
    join_id: number;
    workflow_dest_node_id: number;
    workflow_dest_node: WorkflowNode;

    constructor() {
        this.workflow_dest_node = new WorkflowNode();
    }
}

// WorkflowNode represents a node in w workflow tree
export class WorkflowNode {
    id: number;
    name: string;
    ref: string;
    workflow_id: number;
    pipeline_id: number;
    pipeline: Pipeline;
    context: WorkflowNodeContext;
    hooks: Array<WorkflowNodeHook>;
    triggers: Array<WorkflowNodeTrigger>;

    static removeNodeWithoutChild(parentNode: WorkflowNode, trigger: WorkflowNodeTrigger, id: number, triggerInd: number): WorkflowNode {
        if (trigger.workflow_dest_node.id === id) {
            if (trigger.workflow_dest_node.triggers) {
                trigger.workflow_dest_node.triggers.forEach(t => {
                    t.workflow_node_id = parentNode.id;
                    t.workflow_dest_node_id = t.workflow_dest_node.id;
                    parentNode.triggers.push(t);
                });
            }
            parentNode.triggers.splice(triggerInd, 1);
            return parentNode;
        }
        if (trigger.workflow_dest_node.triggers) {
            for (let i = 0; i < trigger.workflow_dest_node.triggers.length; i++) {
                let t = trigger.workflow_dest_node.triggers[i];
                let p = WorkflowNode.removeNodeWithoutChild(trigger.workflow_dest_node, t, id, i);
                if (p) {
                    return p;
                }
            }
        }
        return null;
    }

    static removeNodeWithoutChildFromJoinTrigger(parentNode: WorkflowNodeJoin, trigger: WorkflowNodeJoinTrigger,
                                                 id: number, triggerInd: number) {
        if (trigger.workflow_dest_node.id === id) {
            if (trigger.workflow_dest_node.triggers) {
                trigger.workflow_dest_node.triggers.forEach(t => {
                    let newT = new WorkflowNodeJoinTrigger();
                    newT.workflow_dest_node = t.workflow_dest_node;
                    parentNode.triggers.push(newT);
                });
            }
            parentNode.triggers.splice(triggerInd, 1);
            return;
        }
        if (trigger.workflow_dest_node.triggers) {
            for (let i = 0; i < trigger.workflow_dest_node.triggers.length; i++) {
                let t = trigger.workflow_dest_node.triggers[i];
                let p = WorkflowNode.removeNodeWithoutChild(trigger.workflow_dest_node, t, id, i);
                if (p) {
                    return;
                }
            }
        }
        return null;
    }

    static getNodeByID(node: WorkflowNode, id: number) {
        if (node.id === id) {
            return node;
        }
        let nodeToFind: WorkflowNode;
        if (node.triggers) {
            for (let i = 0; i < node.triggers.length; i++) {
                let n = WorkflowNode.getNodeByID(node.triggers[i].workflow_dest_node, id);
                if (n) {
                    nodeToFind = n;
                    break;
                }
            }
        }
        return nodeToFind;
    }

    static addRef(refs: string[], root: WorkflowNode) {
        refs.push(root.ref);
        if (root.triggers) {
            root.triggers.forEach(t => {
                WorkflowNode.addRef(refs, t.workflow_dest_node);
            });
        }
    }

    static removeHook(n: WorkflowNode, id: number): boolean {
        if (n.hooks) {
            let lengthBefore = n.hooks.length;
            n.hooks = n.hooks.filter(h => h.id !== id);
            if (lengthBefore !== n.hooks.length) {
                return true;
            }
        }
        if (n.triggers) {
            for (let i = 0; i < n.triggers.length; i++) {
                let h = WorkflowNode.removeHook(n.triggers[i].workflow_dest_node, id);
                if (h) {
                    return true;
                }
            }
        }
        return false;
    }

    static findHook(n: WorkflowNode, id: number): WorkflowNodeHook {
        if (n.hooks) {
            for (let i = 0; i < n.hooks.length; i++) {
                if (n.hooks[i].id === id) {
                    return n.hooks[i];
                }
            }
            if (n.triggers) {
                for (let i = 0; i < n.triggers.length; i++) {
                    let h = WorkflowNode.findHook(n.triggers[i].workflow_dest_node, id);
                    if (h) {
                        return h;
                    }
                }
            }
        }
        return null;
    }

    static getNodeNameImpact(n: WorkflowNode, name: string, nodeWarn: WorkflowPipelineNameImpact): void {
        let varName = 'workflow.' + name;
        if (n.context.conditions && n.context.conditions.plain) {
            n.context.conditions.plain.forEach(c => {
                if (c.value.indexOf(varName) !== -1 || c.variable.indexOf(varName) !== -1) {
                    nodeWarn.nodes.push(n);
                }
            });
        }
        if (n.triggers) {
            n.triggers.forEach(t => {
                WorkflowNode.getNodeNameImpact(t.workflow_dest_node, name, nodeWarn);
            });
        }
    }

    constructor() {
        this.context = new WorkflowNodeContext();
    }


}

export class WorkflowPipelineNameImpact {
    nodes = new Array<WorkflowNode>();
}

// WorkflowNodeContext represents a context attached on a node
export class WorkflowNodeContext {
    id: number;
    workflow_node_id: number;
    application_id: number;
    application: Application;
    environment: Environment;
    environment_id: number;
    default_payload: {};
    default_pipeline_parameters: Array<Parameter>;
    conditions: WorkflowNodeConditions;
}

// WorkflowNodeHook represents a hook which cann trigger the workflow from a given node
export class WorkflowNodeHook {
    id: number;
    uuid: string;
    model: WorkflowHookModel;
    config: Map<string, WorkflowNodeHookConfigValue>;
}

export class WorkflowNodeHookConfigValue {
    value: string;
    configurable: boolean;
}

// WorkflowNodeTrigger is a ling betweeb two pipelines in a workflow
export class WorkflowNodeTrigger {
    id: number;
    workflow_node_id: number;
    workflow_dest_node_id: number;
    workflow_dest_node: WorkflowNode;

    constructor() {
        this.workflow_dest_node = new WorkflowNode();
    }
}

// WorkflowTriggerConditions is either a lua script to check conditions or a set of WorkflowTriggerCondition
export class WorkflowNodeConditions {
    lua_script: string;
    plain: Array<WorkflowNodeCondition>;
}

// WorkflowTriggerCondition represents a condition to trigger ot not a pipeline in a workflow. Operator can be =, !=, regex
export class WorkflowNodeCondition {
    variable: string;
    operator: string;
    value: string;
}

export class WorkflowTriggerConditionCache {
    operators: Array<string>;
    names: Array<string>;
}

export class WorkflowNotification {
    source_node_ref: Array<string>;
    notifications: any;

    constructor() {
        this.notifications = {};
    }
}
