import {notificationTypes, UserNotificationSettings} from 'app/model/notification.model';
import {Application} from './application.model';
import {Environment} from './environment.model';
import {GroupPermission} from './group.model';
import {Parameter} from './parameter.model';
import {Pipeline} from './pipeline.model';
import { ProjectPlatform } from './platform.model';
import { Label } from './project.model';
import {Usage} from './usage.model';
import {WorkflowHookModel} from './workflow.hook.model';

// Workflow represents a pipeline based workflow
export class Workflow {
    id: number;
    name: string;
    description: string;
    icon: string;
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
    notifications: Array<WorkflowNotification>;
    from_repository: string;
    favorite: boolean;
    pipelines: {[key: number]: Pipeline; };
    applications: {[key: number]: Application; };
    environments: {[key: number]: Environment; };
    project_platforms: {[key: number]: ProjectPlatform; };
    hook_models: {[key: number]: WorkflowHookModel; };
    outgoing_hook_models: {[key: number]: WorkflowHookModel; };
    labels: Label[];
    workflow_data: WorkflowData;

    // UI params
    externalChange: boolean;
    forceRefresh: boolean;
    previewMode: boolean;

    static retroMigrate(w: Workflow) {
        w.root =  WNode.retroMigrateNode(w, w.workflow_data.node);
        if (w.workflow_data.joins && w.workflow_data.joins.length > 0) {
            w.joins = new Array<WorkflowNodeJoin>();
            w.workflow_data.joins.forEach(j => {
               w.joins.push(WNode.retroMigrateJoin(w, j));
            });
        }
    }

    static getAllNodes(data: Workflow): Array<WNode> {
        let nodes = new Array<WNode>();

        nodes.push(...WNode.getAllNodes(data.workflow_data.node));

        if (data.workflow_data.joins) {
            data.workflow_data.joins.forEach(j => {
                nodes.push(...WNode.getAllNodes(j));
            });
        }
        return nodes;
    }

    static getNodeByRef(ref: string, w: Workflow): WNode {
        let node = WNode.getNodeByRef(w.workflow_data.node, ref);
        if (node) {
            return node;
        }
        if (w.workflow_data.joins) {
            for (let i = 0; i < w.workflow_data.joins.length; i++) {
                let n = WNode.getNodeByRef(w.workflow_data.joins[i], ref);
                if (n) {
                    return n;
                }
            }
        }
        return null;
    }

    static getNodeByID(id: number, w: Workflow): WNode {
        let node = WNode.getNodeByID(w.workflow_data.node, id);
        if (node) {
            return node;
        }
        if (w.workflow_data.joins) {
            for (let i = 0; i < w.workflow_data.joins.length; i++) {
                let n = WNode.getNodeByID(w.workflow_data.joins[i], id);
                if (n) {
                    return n;
                }
            }
        }
        return null;
    }

    static removeNodeWithChild(w: Workflow, nodeID: number): boolean {
        let result = false;
        // Cannot remove root node
        if (nodeID === w.workflow_data.node.id) {
            return false;
        }
        let b = WNode.removeNodeWithChild(null, w.workflow_data.node, nodeID, 0);
        if (!b) {
            if (w.workflow_data.joins) {
                for (let i = 0; i < w.workflow_data.joins.length; i++) {
                    if (w.workflow_data.joins[i].id === nodeID) {
                        w.workflow_data.joins.splice(i, 1);
                        result = true;
                        break;
                    }
                    let bb = WNode.removeNodeWithChild(null, w.workflow_data.joins[i], nodeID, i);
                    if (bb) {
                        result = true;
                        break;
                    }
                }
            }
        } else {
            result = true;
        }
        if (result) {
            let nodes = Workflow.getAllNodes(w);
            Workflow.cleanJoin(w, nodes);
            Workflow.cleanNotifications(w, nodes);
        }
        return result;
    }

    static removeNodeOnly(w: Workflow, nodeID: number): boolean {
        let result = false;
        if (nodeID === w.workflow_data.node.id && w.workflow_data.node.triggers.length > 0) {
            // Replace node by a fork
            let newRoot = new WNode();
            newRoot.triggers = w.workflow_data.node.triggers;
            newRoot.type = WNodeType.FORK;
            newRoot.hooks = w.workflow_data.node.hooks;
            newRoot.workflow_id = w.workflow_data.node.workflow_id;
            w.workflow_data.node = newRoot;
            result = true;
        }
        if (!result) {
            let b = WNode.removeNodeOnly(null, w.workflow_data.node, nodeID);
            if (b) {
                result = true;
            }
            if (!result && w.workflow_data.joins) {
                for (let i = 0; i < w.workflow_data.joins.length; i++) {
                    let bb = WNode.removeNodeOnly(null, w.workflow_data.joins[i], nodeID)
                    if (bb) {
                        result = true;
                        break;
                    }
                }
            }
        }
        if (result) {
            let nodes = Workflow.getAllNodes(w);
            Workflow.cleanJoin(w, nodes);
            Workflow.cleanNotifications(w, nodes);
        }

        return result;
    }

    static cleanNotifications(workflow: Workflow, nodes: Array<WNode>) {
        if (workflow.notifications && workflow.notifications.length > 0) {
            for (let i = 0; i < workflow.notifications.length; i++) {
                if (workflow.notifications[i].source_node_ref) {
                    for (let j = 0; j < workflow.notifications[i].source_node_ref.length; j++) {
                        if (-1 === nodes.findIndex(n => n.ref === workflow.notifications[i].source_node_ref[j])) {
                            workflow.notifications[i].source_node_ref.splice(j, 1);
                            j--;
                        }
                    }
                    if (workflow.notifications[i].source_node_ref.length === 0) {
                        workflow.notifications.splice(i, 1);
                        i--;
                    }
                }
            }
        }
    }

    static cleanJoin(workflow: Workflow, nodes: Array<WNode>) {
        if (workflow.workflow_data.joins) {
            for (let i = 0; i < workflow.workflow_data.joins.length; i ++) {
                if (workflow.workflow_data.joins[i].parents && workflow.workflow_data.joins[i].parents.length > 0) {
                    for (let j = 0; j < workflow.workflow_data.joins[i].parents.length; j++) {
                        if (-1 === nodes.findIndex(n => n.ref === workflow.workflow_data.joins[i].parents[j].parent_name)) {
                            workflow.joins[i].source_node_ref.splice(j, 1);
                            j--;
                        }
                    }
                }
                if (workflow.joins[i].source_node_ref.length === 0) {
                    workflow.joins.splice(i, 1);
                    i--;
                }
            }
        }
    }

    static getMapNodesRef(data: Workflow): Map<string, WNode> {
        let nodes = new Map<string, WNode>();
        nodes = WNode.getMapNodesRef(nodes, data.workflow_data.node);

        if (data.workflow_data.joins) {
            data.workflow_data.joins.forEach(j => {
                nodes = WNode.getMapNodesRef(nodes, j);
            });
        }
        return nodes;
    }

    static prepareRequestForAPI(workflow: Workflow) {
        WNode.prepareRequestForAPI(workflow.workflow_data.node);
        if (workflow.workflow_data.joins) {
            workflow.workflow_data.joins.forEach(j => {
                j.id = 0;
                if (j.triggers) {
                    j.triggers.forEach(t => {
                        WNode.prepareRequestForAPI(t.child_node);
                    });
                }
            });
        }
        delete workflow.root;
        delete workflow.joins;
        delete workflow.usage;
        delete workflow.applications;
        delete workflow.environments;
        delete workflow.pipelines;
        delete workflow.project_platforms;
        delete workflow.hook_models;
        delete workflow.outgoing_hook_models;
    }

    static getPipeline(workflow: Workflow, node: WNode): Pipeline {
        if (node.context && node.context.pipeline_id) {
            return workflow.pipelines[node.context.pipeline_id];
        }
    }
    static getApplication(workflow: Workflow, node: WNode): Application {
        if (node.context && node.context.application_id) {
            return workflow.applications[node.context.application_id];
        }
    }
    static getEnvironment(workflow: Workflow, node: WNode): Environment {
        if (node.context && node.context.environment_id) {
            return workflow.environments[node.context.environment_id];
        }
    }
    static getPlatform(workflow: Workflow, node: WNode): ProjectPlatform {
        if (node.context && node.context.project_platform_id) {
            return workflow.project_platforms[node.context.project_platform_id];
        }
    }
    static getHookModel(workflow: Workflow, hook: WNodeHook): WorkflowHookModel {
        if (hook && hook.hook_model_id) {
            return workflow.hook_models[hook.hook_model_id];
        }
    }
    static getOutGoingHookModel(workflow: Workflow, hook: WNodeOutgoingHook): WorkflowHookModel {
        if (hook.hook_model_id) {
            return workflow.outgoing_hook_models[hook.hook_model_id];
        }
    }

    ///// MIGRATE

    static updateHook(workflow: Workflow, h: WNodeHook) {
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

    static removeHook(workflow: Workflow, h: WNodeHook) {
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

    static getNodeNameImpact(workflow: Workflow, name: string): WorkflowPipelineNameImpact {
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

    static getMapNodes(data: Workflow): Map<number, WorkflowNode> {
        let nodes = new Map<number, WorkflowNode>();
        nodes = WorkflowNode.getMapNodes(nodes, data.root);

        if (data.joins) {
            data.joins.forEach(j => {
                if (j.triggers) {
                    j.triggers.forEach(t => {
                        nodes = WorkflowNode.getMapNodes(nodes, t.workflow_dest_node);
                    });
                }
            });
        }
        return nodes;
    }

    static getParentNodeIds(workflow: Workflow, currentNodeID: number): number[] {
        // TODO
        /*
        let ancestors = {};

        if (workflow.joins) {
            for (let join of workflow.joins) {
                if (join.triggers) {
                    for (let trigger of join.triggers) {
                        if (trigger.workflow_dest_node) {
                            let parentNodeInfos = this.getParentNode(workflow, trigger.workflow_dest_node, currentNodeID);
                            if (parentNodeInfos.found) {
                                if (parentNodeInfos.node) {
                                    ancestors[parentNodeInfos.node.id] = true;
                                } else {
                                    ancestors[workflow.root.id] = true;
                                    join.source_node_id.forEach((source) => ancestors[source] = true);
                                    return Object.keys(ancestors).map((ancestor) => parseInt(ancestor, 10));
                                }
                            }
                        }
                    }
                }

                for (let sourceNodeId of join.source_node_id) {
                    let nodeFound = Workflow.getNodeByID(sourceNodeId, workflow);
                    if (nodeFound) {
                        let parentNodeInfos = this.getParentNode(workflow, nodeFound, currentNodeID);
                        if (parentNodeInfos.found) {
                            if (parentNodeInfos.node) {
                                ancestors[parentNodeInfos.node.id] = true;
                            }
                        }
                    }
                }
            }
        }


        let parentNodeInfosFromRoot = this.getParentNode(workflow, workflow.root, currentNodeID);
        if (parentNodeInfosFromRoot.found) {
            if (parentNodeInfosFromRoot.node) {
                ancestors[parentNodeInfosFromRoot.node.id] = true;
            } else {
                ancestors[workflow.root.id] = true;
            }
        }

        return Object.keys(ancestors).map((id) => parseInt(id, 10));
        */
        return null;
    }

    static getParentNode(workflow: Workflow, workflowNode: WorkflowNode, currentNodeID: number): { found: boolean, node?: WorkflowNode } {
        if (!workflowNode) {
            return {found: false};
        }
        if (workflowNode.id === currentNodeID) {
            return {found: true};
        }

        if (!Array.isArray(workflowNode.triggers)) {
            return {found: false};
        }

        for (let trigger of workflowNode.triggers) {
            let parentNodeInfos = this.getParentNode(workflow, trigger.workflow_dest_node, currentNodeID);
            if (parentNodeInfos.found) {
                if (parentNodeInfos.node) {
                    return parentNodeInfos;
                } else {
                    return {found: true, node: workflowNode};
                }
            }
        }

        return {found: false};
    }

    static removeNodeInNotifications(workflow: Workflow, node: WorkflowNode): Workflow {
        if (!Array.isArray(workflow.notifications) || !workflow.notifications.length) {
            return workflow;
        }

        workflow.notifications = workflow.notifications.map((notif) => {
            notif.source_node_id = notif.source_node_id.filter((srcId) => srcId !== node.id);
            notif.source_node_ref = notif.source_node_ref.filter((ref) => ref !== node.ref);
            return notif;
        });

        return workflow;
    }

    static getAllHooks(workflow: Workflow): Array<WNodeHook> {
        let res = WNode.getAllHooks(workflow.workflow_data.node);
        if (workflow.workflow_data.joins) {
            workflow.workflow_data.joins.forEach(j => {
                if (j.triggers) {
                    j.triggers.forEach(t => {
                        let hooks = WNode.getAllHooks(t.child_node);
                        if (hooks) {
                            res = res.concat(hooks)
                        }
                    })
                }
            })
        }
        return res;
    }

    constructor() {
        this.root = new WorkflowNode();
    }
}

export class WorkflowNodeJoin {
    id: number;
    ref: string;
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
    pipeline_name: string;
    context: WorkflowNodeContext;
    hooks: Array<WorkflowNodeHook>;
    forks: Array<WorkflowNodeFork>;
    outgoing_hooks: Array<WorkflowNodeOutgoingHook>;
    triggers: Array<WorkflowNodeTrigger>;

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



    static getMapNodes(map: Map<number, WorkflowNode>, n: WorkflowNode): Map<number, WorkflowNode> {
        let smallNode = new WorkflowNode();
        smallNode.id = n.id;
        smallNode.name = n.name;
        map.set(n.id, smallNode);

        if (n.triggers) {
            n.triggers.forEach(t => {
                map = WorkflowNode.getMapNodes(map, t.workflow_dest_node);
            });
        }

        if (n.outgoing_hooks) {
            for (let i = 0; i < n.outgoing_hooks.length; i++) {
                if (n.outgoing_hooks[i].triggers) {
                    for (let j = 0; j < n.outgoing_hooks[i].triggers.length; j++) {
                        map = WorkflowNode.getMapNodes(map, n.outgoing_hooks[i].triggers[j].workflow_dest_node);
                    }
                }
            }
        }

        return map;
    }

    static getAllHooks(n: WorkflowNode): Array<WorkflowNodeHook> {
        let res = n.hooks;
        if (n.triggers) {
            n.triggers.forEach(t => {
                res = res.concat(WorkflowNode.getAllHooks(t.workflow_dest_node));
            });
        }
        if (n.outgoing_hooks) {
            for (let i = 0; i < n.outgoing_hooks.length; i++) {
                if (n.outgoing_hooks[i].triggers) {
                    for (let j = 0; j < n.outgoing_hooks[i].triggers.length; j++) {
                        res = res.concat(WorkflowNode.getAllHooks(n.outgoing_hooks[i].triggers[j].workflow_dest_node));
                    }
                }
            }
        }
        return res;
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
    project_platform: ProjectPlatform;
    project_platform_id: number;
    default_payload: {};
    default_pipeline_parameters: Array<Parameter>;
    conditions: WorkflowNodeConditions;
    mutex: boolean;
}

// WorkflowNodeHook represents a hook which can trigger the workflow from a given node
export class WorkflowNodeHook {
    id: number;
    uuid: string;
    model: WorkflowHookModel;
    config: Map<string, WorkflowNodeHookConfigValue>;
}

export class WorkflowNodeOutgoingHook {
    id: number;
    name: string;
    ref: string;
    model: WorkflowHookModel;
    config: Map<string, WorkflowNodeHookConfigValue>;
    triggers: Array<WorkflowNodeTrigger>;
}

export class WorkflowNodeHookConfigValue {
    value: string;
    configurable: boolean;
    type: string;
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

export class WorkflowNodeFork {
    id: number;
    name: string;
    workflow_node_id: number;
    triggers: Array<WorkflowNodeForkTrigger>;
}

export class WorkflowNodeForkTrigger {
    id: number;
    workflow_node_fork_id: number;
    workflow_dest_node_id: number;
    workflow_dest_node: WorkflowNode;
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

    constructor() {
      this.value = '';
    }
}

export class WorkflowTriggerConditionCache {
    operators: Array<string>;
    names: Array<string>;
}

export class WorkflowNotification {
    id: number;
    source_node_id: Array<number>;
    source_node_ref: Array<string>;
    type: string;
    settings: UserNotificationSettings;

    constructor() {
        this.type = notificationTypes[0];
        this.settings = new UserNotificationSettings();
        this.source_node_ref = new Array<string>();
        this.source_node_id = new Array<number>();
    }
}

export class WorkflowData {
    node: WNode;
    joins: Array<WNode>;
}

export class WNodeType {
    static PIPELINE = 'pipeline';
    static JOIN = 'join';
    static FORK = 'fork';
    static OUTGOINGHOOK = 'outgoinghook';
}

export class WNode {
    id: number;
    workflow_id: number;
    name: string;
    ref: string;
    type: string;
    triggers: Array<WNodeTrigger>;
    context: WNodeContext;
    outgoing_hook: WNodeOutgoingHook;
    parents: Array<WNodeJoin>;
    hooks: Array<WNodeHook>;

    static retroMigrateNode(w: Workflow, node: WNode): WorkflowNode {
        let n = new WorkflowNode();
        n.id = node.id;
        n.ref = node.ref;
        n.name = node.name;
        n.context = new WorkflowNodeContext();
        n.pipeline_id = node.context.pipeline_id;
        if (node.context.application_id) {
            n.context.application = w.applications[node.context.application_id];
        }
        if (node.context.environment_id) {
            n.context.environment = w.environments[node.context.environment_id];
        }
        n.context.conditions = node.context.conditions;
        n.context.default_payload = node.context.default_payload;
        n.context.default_pipeline_parameters = node.context.default_pipeline_parameters;
        n.context.mutex = node.context.mutex;
        if (node.context.project_platform_id) {
            n.context.project_platform = w.project_platforms[node.context.project_platform_id];
        }

        if (node.triggers) {
            node.triggers.forEach(t => {
                let childNode = t.child_node;
                switch (childNode.type) {
                    case 'pipeline':
                        if (!n.triggers) {
                            n.triggers = new Array<WorkflowNodeTrigger>();
                        }
                        let trig = new WorkflowNodeTrigger();
                        trig.workflow_node_id = n.id;
                        trig.workflow_dest_node = WNode.retroMigrateNode(w, childNode);
                        n.triggers.push(trig);
                        break;
                    case 'fork':
                        if (!n.forks) {
                            n.forks = new Array<WorkflowNodeFork>();
                        }
                        n.forks.push(WNode.retroMigrateFork(w, childNode, n.id));
                        break;
                    case 'outgoinghook':
                        if (!n.outgoing_hooks) {
                            n.outgoing_hooks = new Array<WorkflowNodeOutgoingHook>();
                        }
                        n.outgoing_hooks.push(WNode.retroMigrateOutGoingHook(w, childNode));
                        break;
                }
            });
        }
        return n;
    }

    static retroMigrateOutGoingHook(w: Workflow, outgoingHook: WNode): WorkflowNodeOutgoingHook {
        let h = new WorkflowNodeOutgoingHook();
        h.id = outgoingHook.id;
        h.model = w.outgoing_hook_models[outgoingHook.outgoing_hook.hook_model_id];
        h.config = outgoingHook.outgoing_hook.config;
        h.ref = outgoingHook.ref;

        if (outgoingHook.triggers) {
            h.triggers = new Array<WorkflowNodeTrigger>();
            outgoingHook.triggers.forEach(t => {
                let childNode = t.child_node;
                let trig = new WorkflowNodeTrigger();
                trig.workflow_node_id = h.id;
                switch (childNode.type) {
                    case 'pipeline':
                        trig.workflow_dest_node = WNode.retroMigrateNode(w, childNode);
                        break;
                    default: return;
                }
                h.triggers.push(trig);
            });
        }
        return h;
    }

    static retroMigrateFork(w: Workflow, fork: WNode, parentID: number): WorkflowNodeFork {
        let f = new  WorkflowNodeFork();
        f.id = fork.id;
        f.name = fork.name;
        f.workflow_node_id = parentID;
        if (fork.triggers) {
            f.triggers = new Array<WorkflowNodeForkTrigger>();
            fork.triggers.forEach(t => {
                let childNode = t.child_node;
                let trig = new WorkflowNodeForkTrigger();
                trig.workflow_node_fork_id = f.id;
                switch (childNode.type) {
                    case 'pipeline':
                        trig.workflow_dest_node = WNode.retroMigrateNode(w, childNode);
                        break;
                    default: return;
                }
                f.triggers.push(trig);
            });
        }
        return  f;
    }

    static retroMigrateJoin(w: Workflow, join: WNode): WorkflowNodeJoin {
        let j = new WorkflowNodeJoin();
        j.source_node_ref = join.parents.map(pa => pa.parent_name);
        j.source_node_id = join.parents.map(pa => pa.parent_id);
        j.id = join.id;
        j.ref = join.ref;
        if (join.triggers) {
            j.triggers = new Array<WorkflowNodeJoinTrigger>();
            join.triggers.forEach(t => {
                let trig = new  WorkflowNodeJoinTrigger();
                trig.id = t.id;
                trig.join_id = j.id;
                trig.workflow_dest_node_id = t.child_node.id;
                trig.workflow_dest_node = WNode.retroMigrateNode(w, t.child_node);
                j.triggers.push(trig);
            });
        }
        return j;
    }

    static getMapNodesRef(nodes: Map<string, WNode>, node: WNode): Map<string, WNode> {
        nodes.set(node.ref, node);
        if (node.triggers) {
            node.triggers.forEach(t => {
               nodes = WNode.getMapNodesRef(nodes, t.child_node);
            });
        }
        return nodes;
    }

    static getNodeByRef(node: WNode, ref: string): WNode {
        if (node.ref === ref) {
            return node;
        }
        if (node.triggers) {
            for (let i = 0; i < node.triggers.length; i++) {
                let n = WNode.getNodeByRef(node.triggers[i].child_node, ref);
                if (n) {
                    return n;
                }
            }
        }
        return null;
    }

    static getNodeByID(node: WNode, id: number): WNode {
        if (node.id === id) {
            return node;
        }
        if (node.triggers) {
            for (let i = 0; i < node.triggers.length; i++) {
                let n = WNode.getNodeByID(node.triggers[i].child_node, id);
                if (n) {
                    return n;
                }
            }
        }
        return null;
    }

    static removeNodeWithChild(parentNode: WNode, node: WNode, nodeID: number, index: number): boolean {
        if (node.id === nodeID) {
            if (parentNode) {
                parentNode.triggers.splice(index, 1);
                return true;
            }
            return false;
        }
        if (node.triggers) {
            for (let i = 0; i < node.triggers.length; i++) {
                let b = WNode.removeNodeWithChild(node, node.triggers[i].child_node, nodeID, i);
                if (b) {
                    return true;
                }
            }
        }
        return false;
    }

    static removeNodeOnly(parentNode: WNode, node: WNode, nodeID: number): boolean {
        if (node.id === nodeID) {
            if (node.type === WNodeType.JOIN || !parentNode) {
                return false;
            }
            if (!parentNode.triggers) {
                parentNode.triggers = new Array<WNodeTrigger>();
            }
            if (node.triggers) {
                parentNode.triggers.push(...node.triggers);
            }
            return true;
        }
        if (node.triggers) {
            for (let i = 0; i < node.triggers.length; i++) {
                let b = WNode.removeNodeOnly(node, node.triggers[i].child_node, nodeID);
                if (b) {
                    return true;
                }
            }
        }
        return false;
    }

    static getAllNodes(node: WNode): Array<WNode> {
        let nodes = new Array<WNode>();
        nodes.push(node);
        if (node.triggers) {
            node.triggers.forEach(t => {
               nodes.push(...WNode.getAllNodes(t.child_node));
            });
        }
        return nodes;
    }

    static prepareRequestForAPI(node: WNode) {
        node.id = 0;
        if (node.triggers) {
            node.triggers.forEach(t => {
                WNode.prepareRequestForAPI(t.child_node);
            });
        }
    }

    static getAllHooks(n: WNode): Array<WNodeHook> {
        let res = n.hooks;
        if (n.triggers) {
            n.triggers.forEach(t => {
                let hooks = WNode.getAllHooks(t.child_node)
                if (hooks) {
                    res = res.concat(hooks);
                }

            });
        }
        return res;
    }

    static getAllOutgoingHooks(n: WNode): Array<WNode> {
        let res = new Array<WNode>();
        if (n.type === WNodeType.OUTGOINGHOOK) {
            res.push(n);
        }
        if (n.triggers) {
            n.triggers.forEach(t => {
                res.push(...WNode.getAllOutgoingHooks(t.child_node));
            });
        }
        return res;
    }

    constructor() {
        this.context = new WNodeContext();
    }
}

export class WNodeTrigger {
    id: number;
    parent_node_id: number;
    child_node_id: number;
    parent_node_name: string;
    child_node: WNode;
}

export class WNodeContext {
    id: number;
    node_id: number;
    pipeline_id: number;
    application_id: number;
    environment_id: number;
    project_platform_id: number;
    default_payload: {};
    default_pipeline_parameters: Array<Parameter>;
    conditions: WorkflowNodeConditions;
    mutex: boolean;
}
export class WNodeOutgoingHook {
    id: number;
    node_id: number;
    hook_model_id: number;
    uuid: string;
    config: Map<string, WorkflowNodeHookConfigValue>;
}

export class WNodeJoin {
    id: number;
    node_id: number;
    parent_name: string;
    parent_id: number;
}

export class WNodeHook {
    id: number;
    uuid: string;
    ref: string;
    node_id: number;
    hook_model_id: number;
    config: Map<string, WorkflowNodeHookConfigValue>;

    // UI only
    model: WorkflowHookModel;
}
