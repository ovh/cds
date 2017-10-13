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
    last_modified: Date;
    groups: Array<GroupPermission>;
    permission: number;
    usage: Usage;

    // UI params
    externalChange: boolean;

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
            oldH.conditions = h.conditions;
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

    static getNodeNameImpact(workflow: Workflow, name: string) {
        let varName = 'workflow.' + name;
        let warnings = new WorkflowPipelineNameImpact();
        WorkflowNode.getNodeNameImpact(workflow.root, name, warnings);
        if (workflow.joins) {
            workflow.joins.forEach(j => {
                if (j.triggers) {
                    j.triggers.forEach(t => {
                        if (t.conditions) {
                            t.conditions.forEach(c => {
                                if (c.value.indexOf(varName) !== -1 || c.variable.indexOf(varName) !== -1) {
                                    warnings.joinTriggers.push(t);
                                }
                            });
                        }
                        WorkflowNode.getNodeNameImpact(t.workflow_dest_node, name, warnings);
                    });
                }
            });
        }
        return warnings;
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
    conditions: Array<WorkflowTriggerCondition>;
    manual: boolean;
    continue_on_error: boolean;

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
        if (n.hooks) {
            n.hooks.forEach(h => {
                if (h.conditions) {
                    h.conditions.forEach(c => {
                        if (c.value.indexOf(varName) !== -1 || c.variable.indexOf(varName) !== -1) {
                            nodeWarn.hooks.push(h);
                        }
                    });
                }
            });
        }
        if (n.triggers) {
            n.triggers.forEach(t => {
                if (t.conditions) {
                    t.conditions.forEach(c => {
                        if (c.value.indexOf(varName) !== -1 || c.variable.indexOf(varName) !== -1) {
                            nodeWarn.triggers.push(t);
                        }
                    });
                }
                WorkflowNode.getNodeNameImpact(t.workflow_dest_node, name, nodeWarn);
            });
        }
    }

    constructor() {
        this.context = new WorkflowNodeContext();
    }


}

export class WorkflowPipelineNameImpact {
    triggers = new Array<WorkflowNodeTrigger>();
    joinTriggers = new Array<WorkflowNodeJoinTrigger>();
    hooks = new Array<WorkflowNodeHook>();
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
}

// WorkflowNodeHook represents a hook which cann trigger the workflow from a given node
export class WorkflowNodeHook {
    id: number;
    uuid: string;
    model: WorkflowHookModel;
    conditions: Array<WorkflowTriggerCondition>;
    config: {};
}

// WorkflowNodeTrigger is a ling betweeb two pipelines in a workflow
export class WorkflowNodeTrigger {
    id: number;
    workflow_node_id: number;
    workflow_dest_node_id: number;
    workflow_dest_node: WorkflowNode;
    conditions: Array<WorkflowTriggerCondition>;
    manual: boolean;
    continue_on_error: boolean;

    constructor() {
        this.workflow_dest_node = new WorkflowNode();
    }
}

// WorkflowTriggerCondition represents a condition to trigger ot not a pipeline in a workflow. Operator can be =, !=, regex
export class WorkflowTriggerCondition {
    variable: string;
    operator: string;
    value: string;
}

export class WorkflowTriggerConditionCache {
    operators: Array<string>;
    names: Array<string>;
}
