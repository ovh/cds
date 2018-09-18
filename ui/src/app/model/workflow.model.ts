import {notificationTypes, UserNotificationSettings} from 'app/model/notification.model';
import {intersection} from 'lodash';
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
    labels: Label[];

    // UI params
    externalChange: boolean;
    forceRefresh: boolean;
    previewMode: boolean;

    static removeFork(workflow: Workflow, id: number) {
        // Remove from workflow
        let found = WorkflowNode.removeFork(workflow.root, id);
        if (!found && workflow.joins) {
            join: for (let i = 0; i < workflow.joins.length; i++) {
                if (workflow.joins[i].triggers) {
                    for (let j = 0; j < workflow.joins[i].triggers.length; j++) {
                        if (WorkflowNode.removeFork(workflow.joins[i].triggers[j].workflow_dest_node, id)) {
                            break join;
                        }
                    }
                }
            }
        }

        // Remove old ref
        let nodes = Workflow.getAllNodes(workflow);
        Workflow.cleanJoin(workflow, nodes);
        Workflow.cleanNotifications(workflow, nodes);
    }

    static removeForkWithoutChild(workflow: Workflow, id: number) {
        let found = WorkflowNode.removeForkWithoutChild(workflow.root, id);
        if (!found && workflow.joins) {
            join: for (let i = 0; i < workflow.joins.length; i++) {
                if (workflow.joins[i].triggers) {
                    for (let j = 0; j < workflow.joins[i].triggers.length; j++) {
                        if (WorkflowNode.removeForkWithoutChild(workflow.joins[i].triggers[j].workflow_dest_node, id)) {
                            break join;
                        }
                    }
                }
            }
        }
    }

    static cleanNotifications(workflow: Workflow, nodes: Array<WorkflowNode>) {
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

    static cleanJoin(workflow: Workflow, nodes: Array<WorkflowNode>) {
        if (workflow.joins) {
            for (let i = 0; i < workflow.joins.length; i ++) {
                if (workflow.joins[i].source_node_ref && workflow.joins[i].source_node_ref.length > 0) {
                    for (let j = 0; j < workflow.joins[i].source_node_ref.length; j++) {
                        if (-1 === nodes.findIndex(n => n.ref === workflow.joins[i].source_node_ref[j])) {
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

    // Do not remove root node
    static removeNodeWithoutChild(workflow: Workflow, node: WorkflowNode): boolean {
        if (node.id === workflow.root.id) {
            if ((workflow.root.triggers && workflow.root.triggers.length > 1) || (workflow.joins && workflow.joins.length > 1)) {
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

    static updateOutgoingHook(workflow: Workflow, id: number, config: Map<string, WorkflowNodeHookConfigValue>) {
        let oldH = WorkflowNode.findOutgoingHook(workflow.root, id);
        if (!oldH) {
            if (workflow.joins) {
                quit: for (let i = 0; i < workflow.joins.length; i++) {
                    let j = workflow.joins[i];
                    if (j.triggers) {
                        for (let k = 0; k < j.triggers.length; k++) {
                            oldH = WorkflowNode.findOutgoingHook(j.triggers[k].workflow_dest_node, id);
                            if (oldH) {
                                break quit;
                            }
                        }
                    }
                }
            }
        }

        if (oldH) {
            oldH.config = config;
        }
    };

    static removeOutgoingHook(workflow: Workflow, id: number) {
        let done = WorkflowNode.removeOutgoingHook(workflow.root, id);
        if (!done) {
            if (workflow.joins) {
                for (let i = 0; i < workflow.joins.length; i++) {
                    let j = workflow.joins[i];
                    if (j.triggers) {
                        for (let k = 0; k < j.triggers.length; k++) {
                            done = WorkflowNode.removeOutgoingHook(j.triggers[k].workflow_dest_node, id);
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

    static getForkByName(name: string, w: Workflow): WorkflowNodeFork {
        let fork = WorkflowNode.getForkByName(w.root, name);
        if (!fork && w.joins) {
            quit: for (let i = 0; i < w.joins.length; i++) {
                if (w.joins[i].triggers) {
                    for (let j = 0; j < w.joins[i].triggers.length; j++) {
                        fork = WorkflowNode.getForkByName(w.joins[i].triggers[j].workflow_dest_node, name);
                        if (fork) {
                            break quit;
                        }
                    }
                }
            }
        }
        return fork;
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

    static findNode(w: Workflow, compareFunc): WorkflowNode {
        let node = WorkflowNode.findNode(w.root, compareFunc);
        if (!node && w.joins) {
            quit: for (let i = 0; i < w.joins.length; i++) {
                if (w.joins[i].triggers) {
                    for (let j = 0; j < w.joins[i].triggers.length; j++) {
                        node = WorkflowNode.findNode(w.joins[i].triggers[j].workflow_dest_node, compareFunc);
                        if (node) {
                            break quit;
                        }
                    }
                }
            }
        }
        return node;
    }

    static getHookByID(id: number, w: Workflow): WorkflowNodeHook {
        let hook = WorkflowNode.getHookByID(w.root, id);
        if (!hook && w.joins) {
            quit: for (let i = 0; i < w.joins.length; i++) {
                if (w.joins[i].triggers) {
                    for (let j = 0; j < w.joins[i].triggers.length; j++) {
                        hook = WorkflowNode.getHookByID(w.joins[i].triggers[j].workflow_dest_node, id);
                        if (hook) {
                            break quit;
                        }
                    }
                }
            }
        }
        return hook;
    }

    static findOutgoingHook(w: Workflow, id: number): WorkflowNodeOutgoingHook {
        let hook = WorkflowNode.findOutgoingHook(w.root, id);
        if (hook) {
            return hook;
        }
        if (w.joins) {
            for (let i = 0; i < w.joins.length; i++) {
                if (w.joins[i].triggers) {
                    for (let j = 0; j < w.joins[i].triggers.length; j++) {
                        hook = WorkflowNode.findOutgoingHook(w.joins[i].triggers[j].workflow_dest_node, id);
                        if (hook) {
                            return hook;
                        }
                    }
                }
            }
        }
        return null;
    }

    static isChildOfOutgoingHook(w: Workflow, n: WorkflowNode, h: WorkflowNodeOutgoingHook, nodeID: number): boolean {
        if (h) {
            if (h.triggers) {
                for (let i = 0; i < h.triggers.length; i++) {
                    if (h.triggers[i]) {
                        if (h.triggers[i].workflow_dest_node.id === nodeID) {
                            return true;
                        }
                        if (Workflow.isChildOfOutgoingHook(w, h.triggers[i].workflow_dest_node, null, nodeID)) {
                            return true;
                        }
                    }
                }
            }
            return false;
        }

        if (n) {
            if (n.outgoing_hooks) {
                for (let i = 0; i < n.outgoing_hooks.length; i++) {
                    if (Workflow.isChildOfOutgoingHook(w, null, n.outgoing_hooks[i], nodeID)) {
                        return true;
                    }
                }
            }
            if (n.triggers) {
                for (let i = 0; i < n.triggers.length; i++) {
                    if (Workflow.isChildOfOutgoingHook(w, n.triggers[i].workflow_dest_node, null, nodeID)) {
                        return true;
                    }
                }
            }
            if (n.forks) {
                for (let i = 0; i < n.forks.length; i++) {
                    if (n.forks[i].triggers) {
                        for (let j = 0; j < n.forks[i].triggers.length; j++) {
                            if (Workflow.isChildOfOutgoingHook(w, n.forks[i].triggers[j].workflow_dest_node, null, nodeID)) {
                                return true;
                            }
                        }
                    }
                }
            }
            return false
        }

        if (w.joins) {
            for (let i = 0; i < w.joins.length; i++) {
                if (w.joins[i].triggers) {
                    for (let j = 0; j < w.joins[i].triggers.length; j++) {
                        if (Workflow.isChildOfOutgoingHook(w, w.joins[i].triggers[j].workflow_dest_node, null, nodeID)) {
                            return true;
                        }
                    }
                }
            }
            return false
        }

        return Workflow.isChildOfOutgoingHook(w, w.root, null, nodeID);
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

        w.joins = w.joins.filter(j => j.source_node_ref.length > 0);
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

    static getAllNodes(data: Workflow): Array<WorkflowNode> {
        let nodes = new Array<WorkflowNode>();

        nodes.push(...WorkflowNode.getAllNodes(data.root));

        if (data.joins) {
            data.joins.forEach(j => {
                if (j.triggers) {
                    j.triggers.forEach(t => {
                        nodes.push(...WorkflowNode.getAllNodes(t.workflow_dest_node));
                    });
                }
            });
        }
        return nodes;
    }

    static getJoinById(id: number, workflow: Workflow): WorkflowNodeJoin {
        if (!workflow || !Array.isArray(workflow.joins)) {
            return null;
        }
        return workflow.joins.find((join) => join.id === id);
    }

    static prepareRequestForAPI(workflow: Workflow) {
        WorkflowNode.prepareRequestForAPI(workflow.root);
        if (workflow.joins) {
            workflow.joins.forEach(j => {
                j.id = 0;
                j.source_node_id = [];
                if (j.triggers) {
                    j.triggers.forEach(t => {
                        WorkflowNode.prepareRequestForAPI(t.workflow_dest_node);
                    });
                }
            });
        }
        delete workflow.usage;
    }

    static getParentNodeIds(workflow: Workflow, currentNodeID: number): number[] {
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

    static getPipeline(workflow: Workflow, node: WorkflowNode): Pipeline {
        return workflow.pipelines[node.pipeline_id]
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

    static removeNodesInNotifications(workflow: Workflow, currentNode: WorkflowNode, nodeId: number, deleteNode: boolean): Workflow {
        if (!currentNode || !Array.isArray(workflow.notifications) || !workflow.notifications.length) {
            return workflow;
        }
        if (currentNode.id === nodeId) {
            deleteNode = true;
        }

        if (deleteNode) {
            workflow = Workflow.removeNodeInNotifications(workflow, currentNode);
        }

        if (currentNode.id === workflow.root.id && Array.isArray(workflow.joins)) { // Check from joins
            workflow.joins.forEach((join) => {
                join.triggers.forEach((trig) => {
                    workflow = Workflow.removeNodesInNotifications(workflow, trig.workflow_dest_node, nodeId, deleteNode);
                });
            });
        }

        if (Array.isArray(currentNode.triggers)) {
            currentNode.triggers.forEach((trig) => {
                workflow = Workflow.removeNodesInNotifications(workflow, trig.workflow_dest_node, nodeId, deleteNode);
            });
        }

        return workflow;
    }

    static getAllHooks(workflow: Workflow): Array<WorkflowNodeHook> {
        let res = new Array<WorkflowNodeHook>();
        res.push(...WorkflowNode.getAllHooks(workflow.root));
        if (workflow.joins) {
            workflow.joins.forEach(j => {
                if (j.triggers) {
                    j.triggers.forEach(t => {
                        res.push(...WorkflowNode.getAllHooks(t.workflow_dest_node));
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

    static removeForkWithoutChild(node: WorkflowNode, id: number): boolean {
        if (node.forks && node.forks.length > 0) {
            for (let i = 0; i < node.forks.length; i ++) {
                if (node.forks[i].id === id) {
                    // create node trigger for each nodefork trigger
                    if (node.forks[i].triggers) {
                        if (!node.triggers) {
                            node.triggers = new Array<WorkflowNodeTrigger>();
                        }
                        node.forks[i].triggers.forEach(t => {
                            let trig = new WorkflowNodeTrigger();
                            trig.workflow_node_id = node.id;
                            trig.workflow_dest_node_id = t.workflow_dest_node_id;
                            trig.workflow_dest_node = t.workflow_dest_node;
                            node.triggers.push(trig);
                        });
                    }
                    node.forks.splice(i, 1);
                    return true;
                }

                if (node.forks[i].triggers) {
                    for (let j = 0; j < node.forks[i].triggers.length; j++) {
                        if (WorkflowNode.removeForkWithoutChild(node.forks[i].triggers[j].workflow_dest_node, id)) {
                            return true;
                        }
                    }
                }
            }
        }
        if (node.triggers) {
            for (let j = 0; j < node.triggers.length; j++ ) {
                if (WorkflowNode.removeForkWithoutChild(node.triggers[j].workflow_dest_node, id)) {
                    return true;
                }
            }
        }
        if (node.outgoing_hooks) {
            for (let i = 0; i < node.outgoing_hooks.length; i++) {
                if (node.outgoing_hooks[i].triggers) {
                    for (let j = 0; j < node.outgoing_hooks[i].triggers.length; j++ ) {
                        if (WorkflowNode.removeForkWithoutChild(node.outgoing_hooks[i].triggers[j].workflow_dest_node, id)) {
                            return true;
                        }
                    }
                }
            }
        }
        return false;
    }

    static removeFork(node: WorkflowNode, id: number): boolean {
        if (node.forks && node.forks.length > 0) {
            for (let i = 0; i < node.forks.length; i++ ) {
                if (node.forks[i].id === id) {
                    node.forks.splice(i, 1);
                    return true;
                }
                if (node.forks[i].triggers) {
                    for (let j = 0; j < node.forks[i].triggers.length; j++ ) {
                        if (WorkflowNode.removeFork(node.forks[i].triggers[j].workflow_dest_node, id)) {
                            return true;
                        }
                    }
                }
            }
        }
        if (node.triggers) {
            for (let j = 0; j < node.triggers.length; j++ ) {
                if (WorkflowNode.removeFork(node.triggers[j].workflow_dest_node, id)) {
                    return true;
                }
            }
        }
        if (node.outgoing_hooks) {
            for (let i = 0; i < node.outgoing_hooks.length; i++) {
                if (node.outgoing_hooks[i].triggers) {
                    for (let j = 0; j < node.outgoing_hooks[i].triggers.length; j++ ) {
                        if (WorkflowNode.removeFork(node.outgoing_hooks[i].triggers[j].workflow_dest_node, id)) {
                            return true;
                        }
                    }
                }
            }
        }
        return false;
    }

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

    static getForkByName(node: WorkflowNode, name: string): WorkflowNodeFork {
        let fork: WorkflowNodeFork;
        if (node.forks) {
            for (let i = 0; i < node.forks.length; i++) {
                if (node.forks[i].name === name) {
                    return node.forks[i];
                }
                if (node.forks[i].triggers) {
                    for (let j = 0; j < node.forks[i].triggers.length; j++) {
                        fork = WorkflowNode.getForkByName(node.forks[i].triggers[j].workflow_dest_node, name);
                        if (fork) {
                            return fork;
                        }
                    }
                }
            }
        }

        if (node.triggers) {
            for (let i = 0; i < node.triggers.length; i++) {
                fork = WorkflowNode.getForkByName(node.triggers[i].workflow_dest_node, name);
                if (fork) {
                    return fork;
                }
            }
        }

        if (node.outgoing_hooks) {
            for (let i = 0; i < node.outgoing_hooks.length; i++) {
                if (node.outgoing_hooks[i].triggers) {
                    for (let j = 0; j < node.outgoing_hooks[i].triggers.length; j++) {
                        fork = WorkflowNode.getForkByName(node.outgoing_hooks[i].triggers[j].workflow_dest_node, name);
                        if (fork) {
                            return fork;
                        }
                    }
                }
            }
        }

        return null;
    }

    static getNodeByID(node: WorkflowNode, id: number): WorkflowNode {
        if (node.id === id) {
            return node;
        }
        if (node.triggers) {
            for (let i = 0; i < node.triggers.length; i++) {
                let n = WorkflowNode.getNodeByID(node.triggers[i].workflow_dest_node, id);
                if (n) {
                    return n;
                }
            }
        }

        if (node.outgoing_hooks) {
            for (let i = 0; i < node.outgoing_hooks.length; i++) {
                if (node.outgoing_hooks[i].triggers) {
                    for (let j = 0; j < node.outgoing_hooks[i].triggers.length; j++) {
                        let n = WorkflowNode.getNodeByID(node.outgoing_hooks[i].triggers[j].workflow_dest_node, id);
                        if (n) {
                            return n;
                        }
                    }
                }
            }
        }

        return null;
    }

    static findNode(node: WorkflowNode, compareFunc): WorkflowNode {
        if (compareFunc(node)) {
            return node;
        }
        if (node.triggers) {
            for (let i = 0; i < node.triggers.length; i++) {
                let n = WorkflowNode.findNode(node.triggers[i].workflow_dest_node, compareFunc);
                if (n) {
                    return n;
                }
            }
        }
        if (node.outgoing_hooks) {
            for (let i = 0; i < node.outgoing_hooks.length; i++) {
                if (node.outgoing_hooks[i].triggers) {
                    for (let j = 0; j < node.outgoing_hooks[i].triggers.length; j++) {
                        let n = WorkflowNode.findNode(node.outgoing_hooks[i].triggers[j].workflow_dest_node, compareFunc);
                        if (n) {
                            return n;
                        }
                    }
                }
            }
        }
        return null;
    }

    static getHookByID(node: WorkflowNode, id: number): WorkflowNodeHook {
        if (Array.isArray(node.hooks) && node.hooks.length) {
            let hook = node.hooks.find((h) => h.id === id);
            if (hook != null) {
                return hook;
            }
        }
        if (node.triggers) {
            for (let i = 0; i < node.triggers.length; i++) {
                let h = WorkflowNode.getHookByID(node.triggers[i].workflow_dest_node, id);
                if (h) {
                    return h;
                }
            }
        }
        return null;
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

    static removeOutgoingHook(n: WorkflowNode, id: number): boolean {
        if (n.outgoing_hooks) {
            let lengthBefore = n.outgoing_hooks.length;
            n.outgoing_hooks = n.outgoing_hooks.filter(h => h.id !== id);
            if (lengthBefore !== n.outgoing_hooks.length) {
                return true;
            }

            for (let i = 0; i < n.outgoing_hooks.length; i++) {
                if (n.outgoing_hooks[i].triggers) {
                    for (let j = 0; j < n.outgoing_hooks[i].triggers.length; j++) {
                        let done = WorkflowNode.removeOutgoingHook(n.outgoing_hooks[i].triggers[j].workflow_dest_node, id);
                        if (done) {
                            return true;
                        }
                    }
                }
            }
        }

        if (n.triggers) {
            for (let i = 0; i < n.triggers.length; i++) {
                let done = WorkflowNode.removeOutgoingHook(n.triggers[i].workflow_dest_node, id);
                if (done) {
                    return true;
                }
            }
        }

        return false;
    }

    static findOutgoingHook(n: WorkflowNode, id: number): WorkflowNodeOutgoingHook {
        if (n.outgoing_hooks) {
            for (let i = 0; i < n.outgoing_hooks.length; i++) {
                if (n.outgoing_hooks[i].id === id) {
                    return n.outgoing_hooks[i];
                }
                if (n.outgoing_hooks[i].triggers) {
                    for (let j = 0; j < n.outgoing_hooks[i].triggers.length; j++) {
                        let h = WorkflowNode.findOutgoingHook(n.outgoing_hooks[i].triggers[j].workflow_dest_node, id);
                        if (h) {
                            return h;
                        }
                    }
                }
            }
        }
        if (n.triggers) {
            for (let i = 0; i < n.triggers.length; i++) {
                let h = WorkflowNode.findOutgoingHook(n.triggers[i].workflow_dest_node, id);
                if (h) {
                    return h;
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

    static getAllNodes(n: WorkflowNode): Array<WorkflowNode> {
        let nodes = new Array<WorkflowNode>();

        let smallNode = new WorkflowNode();
        smallNode.id = n.id;
        smallNode.name = n.name;
        smallNode.ref = n.ref;
        nodes.push(smallNode);

        if (n.triggers) {
            n.triggers.forEach(t => {
                nodes.push(...WorkflowNode.getAllNodes(t.workflow_dest_node));
            });
        }

        if (n.outgoing_hooks) {
            for (let i = 0; i < n.outgoing_hooks.length; i++) {
                if (n.outgoing_hooks[i].triggers) {
                    for (let j = 0; j < n.outgoing_hooks[i].triggers.length; j++) {
                        nodes.push(...WorkflowNode.getAllNodes(n.outgoing_hooks[i].triggers[j].workflow_dest_node));
                    }
                }
            }
        }
        if (n.forks) {
            for (let i = 0; i < n.forks.length; i++) {
                if (n.forks[i].triggers) {
                    for (let j = 0; j < n.forks[i].triggers.length; j++) {
                        nodes.push(...WorkflowNode.getAllNodes(n.forks[i].triggers[j].workflow_dest_node));
                    }
                }
            }
        }

        return nodes;
    }

    static prepareRequestForAPI(n: WorkflowNode) {
        n.id = 0;
        if (n.context.application && n.context.application.id > 0) {
            n.context.application_id = n.context.application.id;
            delete n.context.application;
        }
        if (n.context.environment && n.context.environment.id > 0) {
            n.context.environment_id = n.context.environment.id;
            delete n.context.environment;
        }
        if (n.triggers) {
            n.triggers.forEach(t => {
                WorkflowNode.prepareRequestForAPI(t.workflow_dest_node);
            });
        }
        if (n.outgoing_hooks) {
            for (let i = 0; i < n.outgoing_hooks.length; i++) {
                if (n.outgoing_hooks[i].triggers) {
                    for (let j = 0; j < n.outgoing_hooks[i].triggers.length; j++) {
                        WorkflowNode.prepareRequestForAPI(n.outgoing_hooks[i].triggers[j].workflow_dest_node);
                    }
                }
            }
        }
    }

    static isLinkedToRepo(node: WorkflowNode): boolean {
      return node.context.application_id !== 0 && node.context.application != null && !!node.context.application.repository_fullname;
    }

    static getAllHooks(n: WorkflowNode): Array<WorkflowNodeHook> {
        let res = new Array<WorkflowNodeHook>();
        res.push(...n.hooks);
        if (n.triggers) {
            n.triggers.forEach(t => {
                res.push(...WorkflowNode.getAllHooks(t.workflow_dest_node));
            });
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
