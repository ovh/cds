/* tslint:disable:no-unused-variable */

import {fakeAsync, TestBed} from '@angular/core/testing';
import {Workflow, WorkflowNode, WorkflowNodeJoin, WorkflowNodeJoinTrigger, WorkflowNodeTrigger} from './workflow.model';

describe('CDS: Workflow Model', () => {

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [],
            providers: [
            ],
            imports: []
        });
    });


    /**
     * Test deletion of X in this workflow
     *                    O----
     *                    |    |
     *                J---X----J
     *                |   |    |
     *                O   O    O
     *
     */
    it('should delete a node in the middle', fakeAsync(() => {
        let workflow = new Workflow();
        workflow.joins = new Array<WorkflowNodeJoin>();

        // Add root node
        let nRoot = new WorkflowNode();
        nRoot.id = 1;
        workflow.root = nRoot;
        workflow.root.triggers = new Array<WorkflowNodeTrigger>();

        // Add root child
        let nToDelete = new WorkflowNode();
        nToDelete.id = 2;
        let rootTrigger = new WorkflowNodeTrigger();
        rootTrigger.id = 1;
        rootTrigger.workflow_dest_node = nToDelete;
        workflow.root.triggers.push(rootTrigger);


        let nTriggerChild = new WorkflowNode();
        nTriggerChild.id = 3;
        nToDelete.triggers = new Array<WorkflowNodeTrigger>();
        let nDeleteTrigger = new WorkflowNodeTrigger();
        nDeleteTrigger.workflow_dest_node = nTriggerChild;
        nToDelete.triggers.push(nDeleteTrigger);

        let nJ1Chlid = new WorkflowNode();
        nJ1Chlid.id = 4;
        let nJ2Children = new WorkflowNode();
        nJ2Children.id = 5;

        // Create Join with 1 parent and 1 child
        let j1Child = new WorkflowNodeJoin();
        j1Child.source_node_id = new Array<number>();
        j1Child.source_node_id.push(nToDelete.id);
        j1Child.triggers = new Array<WorkflowNodeJoinTrigger>();
        let jt1 = new WorkflowNodeJoinTrigger();
        jt1.workflow_dest_node = nJ1Chlid;
        jt1.id = 1;
        j1Child.triggers.push(jt1);
        workflow.joins.push(j1Child);

        // Create Join with 2 parent and 1 child
        let j2Child = new WorkflowNodeJoin();
        j2Child.source_node_id = new Array<number>();
        j2Child.source_node_id.push(nToDelete.id, nRoot.id);
        j2Child.triggers = new Array<WorkflowNodeJoinTrigger>();
        let jt2 = new WorkflowNodeJoinTrigger();
        jt2.workflow_dest_node = nJ2Children;
        jt2.id = 2;
        j2Child.triggers.push(jt2);
        workflow.joins.push(j2Child);

        let ok = Workflow.removeNodeWithoutChild(workflow, nToDelete);

        expect(ok).toBeTruthy();

        // Assert join are attached to the root node
        expect(workflow.joins.length).toBe(2, 'root node must have 2 joins');
        expect(workflow.joins[0].source_node_id.length).toBe(1);
        expect(workflow.joins[1].source_node_id.length).toBe(1, 'source node id for joins 1: ' + workflow.joins[1].source_node_id);
        expect(workflow.joins[0].source_node_id[0]).toBe(1);
        expect(workflow.joins[1].source_node_id[0]).toBe(1);

        // Assert child of deleted node is now on the root node
        expect(workflow.root.triggers).toBeTruthy();
        expect(workflow.root.triggers.length).toBe(1, 'root node must have 1 trigger');
        expect(workflow.root.triggers[0].workflow_dest_node.id).toBe(nTriggerChild.id);
    }));


    /**
     * X --> o --> o
     */
    it('should delete root node: simple', fakeAsync(() => {
        let workflow = new Workflow();
        workflow.joins = new Array<WorkflowNodeJoin>();

        // Add root node
        let nRoot = new WorkflowNode();
        nRoot.id = 1;
        workflow.root = nRoot;
        workflow.root.triggers = new Array<WorkflowNodeTrigger>();

        // Add root child
        let child1 = new WorkflowNode();
        child1.id = 2;
        let rootTrigger = new WorkflowNodeTrigger();
        rootTrigger.id = 1;
        rootTrigger.workflow_dest_node = child1;
        workflow.root.triggers.push(rootTrigger);

        let child2 = new WorkflowNode();
        child2.id = 3;
        child1.triggers = new Array<WorkflowNodeTrigger>();
        let childTrigger = new WorkflowNodeTrigger();
        childTrigger.workflow_dest_node = child2;


        let ok = Workflow.removeNodeWithoutChild(workflow, nRoot);

        expect(ok).toBeTruthy();
        expect(workflow.root.id).toBe(2);
    }));

    /**
     *     X --> o
     *     |
     *     v
     *     o
     */
    it('should not delete the root node because it has 2 triggers', fakeAsync(() => {
        let workflow = new Workflow();
        workflow.joins = new Array<WorkflowNodeJoin>();

        // Add root node
        let nRoot = new WorkflowNode();
        nRoot.id = 1;
        workflow.root = nRoot;
        workflow.root.triggers = new Array<WorkflowNodeTrigger>();

        // Add root child
        let child1 = new WorkflowNode();
        child1.id = 2;
        let rootTrigger1 = new WorkflowNodeTrigger();
        rootTrigger1.id = 1;
        rootTrigger1.workflow_dest_node = child1;
        workflow.root.triggers.push(rootTrigger1);

        let child2 = new WorkflowNode();
        child2.id = 2;
        let rootTrigger2 = new WorkflowNodeTrigger();
        rootTrigger2.id = 1;
        rootTrigger2.workflow_dest_node = child2;
        workflow.root.triggers.push(rootTrigger2);


        let ok = Workflow.removeNodeWithoutChild(workflow, nRoot);

        expect(ok).toBeFalsy();
    }));

    /**
     *     X --> J --> O
     */
    it('should delete the root node. O become the new root', fakeAsync(() => {
        let workflow = new Workflow();
        workflow.joins = new Array<WorkflowNodeJoin>();

        // Add root node
        let nRoot = new WorkflowNode();
        nRoot.id = 1;
        workflow.root = nRoot;
        workflow.joins = new Array<WorkflowNodeJoin>();


        let j1 = new WorkflowNodeJoin();
        j1.source_node_id = new Array<number>();
        j1.source_node_id.push(1);
        j1.triggers = new Array<WorkflowNodeJoinTrigger>();

        let t1 = new WorkflowNodeJoinTrigger();
        let child = new WorkflowNode();
        child.id = 2;
        t1.workflow_dest_node = child;
        j1.triggers.push(t1);
        workflow.joins.push(j1);

        let ok = Workflow.removeNodeWithoutChild(workflow, nRoot);

        expect(ok).toBeTruthy();
        expect(workflow.root.id).toBe(2);
    }));

    /**
     *     X --> -J --> n2
     *     |   ^  |
     *     T   |  n1
     *     |   |
     *     c --
     */
    it('should delete root node, c because the new root', fakeAsync(() => {
        let workflow = new Workflow();
        workflow.joins = new Array<WorkflowNodeJoin>();

        // Add root node
        let nRoot = new WorkflowNode();
        nRoot.id = 1;
        workflow.root = nRoot;
        nRoot.triggers = new Array<WorkflowNodeTrigger>();
        workflow.joins = new Array<WorkflowNodeJoin>();

        // Add trigger T
        let c = new WorkflowNode();
        c.id = 2;
        let nRootTrigger = new WorkflowNodeTrigger();
        nRootTrigger.workflow_dest_node = c;
        nRoot.triggers.push(nRootTrigger);


        let j1 = new WorkflowNodeJoin();
        j1.source_node_id = new Array<number>();
        j1.source_node_id.push(1, 2);
        j1.triggers = new Array<WorkflowNodeJoinTrigger>();

        let jt1 = new WorkflowNodeJoinTrigger();
        let childjt1 = new WorkflowNode();
        childjt1.id = 3;
        jt1.workflow_dest_node = childjt1;
        j1.triggers.push(jt1);
        workflow.joins.push(j1);

        let jt2 = new WorkflowNodeJoinTrigger();
        let childjt2 = new WorkflowNode();
        childjt2.id = 4;
        jt2.workflow_dest_node = childjt2;
        j1.triggers.push(jt2);

        let ok = Workflow.removeNodeWithoutChild(workflow, nRoot);

        expect(ok).toBeTruthy();
        expect(workflow.root.id).toBe(2);
        expect(workflow.joins.length).toBe(1);
        expect(workflow.joins[0].source_node_id.length).toBe(1);
        expect(workflow.joins[0].source_node_id[0]).toBe(2);
        expect(workflow.joins[0].triggers.length).toBe(2);
    }));

    /**
     *     X -----> - J --> n2
     *     |   |   ^  |
     *     T   J   |  n1
     *     |       |
     *     c ------
     */
    it('should not delete the root node because it has 2 Joins', fakeAsync(() => {
        let workflow = new Workflow();
        workflow.joins = new Array<WorkflowNodeJoin>();

        // Add root node
        let nRoot = new WorkflowNode();
        nRoot.id = 1;
        workflow.root = nRoot;
        nRoot.triggers = new Array<WorkflowNodeTrigger>();
        workflow.joins = new Array<WorkflowNodeJoin>();

        // Add trigger T
        let c = new WorkflowNode();
        c.id = 2;
        let nRootTrigger = new WorkflowNodeTrigger();
        nRootTrigger.workflow_dest_node = c;
        nRoot.triggers.push(nRootTrigger);


        let j1 = new WorkflowNodeJoin();
        j1.source_node_id = new Array<number>();
        j1.source_node_id.push(1, 2);
        j1.triggers = new Array<WorkflowNodeJoinTrigger>();

        let jt1 = new WorkflowNodeJoinTrigger();
        let childjt1 = new WorkflowNode();
        childjt1.id = 3;
        jt1.workflow_dest_node = childjt1;
        j1.triggers.push(jt1);
        workflow.joins.push(j1);

        let jt2 = new WorkflowNodeJoinTrigger();
        let childjt2 = new WorkflowNode();
        childjt2.id = 4;
        jt2.workflow_dest_node = childjt2;
        j1.triggers.push(jt2);

        let j2 = new WorkflowNodeJoin();
        j2.source_node_id = new Array<number>();
        j2.source_node_id.push(1, 2);
        workflow.joins.push(j2);

        let ok = Workflow.removeNodeWithoutChild(workflow, nRoot);

        expect(ok).toBeFalsy();
    }));


    /**
     *     W --> J --> X ----> n2
     *     |            |
     *     v            v
     *     root         n3
     */
    it('should delete the node after join', fakeAsync(() => {
        let workflow = new Workflow();
        workflow.joins = new Array<WorkflowNodeJoin>();

        // Add root node
        let nRoot = new WorkflowNode();
        nRoot.id = 10;
        workflow.root = nRoot;
        workflow.joins = new Array<WorkflowNodeJoin>();

        let j1 = new WorkflowNodeJoin();
        j1.source_node_id = new Array<number>();
        j1.source_node_id.push(10);
        j1.triggers = new Array<WorkflowNodeJoinTrigger>();

        let jt1 = new WorkflowNodeJoinTrigger();
        let childjt1 = new WorkflowNode();
        childjt1.id = 1;
        childjt1.triggers = new Array<WorkflowNodeTrigger>();
        jt1.workflow_dest_node = childjt1;
        j1.triggers.push(jt1);
        workflow.joins.push(j1);

        // 2 triggers
        let triggern2 = new WorkflowNodeTrigger();
        triggern2.workflow_dest_node = new WorkflowNode();
        triggern2.workflow_dest_node.id = 2;

        let triggern3 = new WorkflowNodeTrigger();
        triggern3.workflow_dest_node = new WorkflowNode();
        triggern3.workflow_dest_node.id = 3;

        childjt1.triggers.push(triggern2, triggern3);


        let ok = Workflow.removeNodeWithoutChild(workflow, childjt1);

        expect(ok).toBeTruthy();
        expect(workflow.joins[0].triggers.length).toBe(2)
    }));

    /**
     *     R --> n1
     *     |     |
     *     v     v
     *     X--->J
     */
    it('should delete the node after join', fakeAsync(() => {
        let workflow = new Workflow();
        workflow.joins = new Array<WorkflowNodeJoin>();

        // Add root node
        let nRoot = new WorkflowNode();
        nRoot.id = 1;
        nRoot.triggers = new Array<WorkflowNodeTrigger>();
        workflow.root = nRoot;
        workflow.joins = new Array<WorkflowNodeJoin>();

        let triggern1 = new WorkflowNodeTrigger();
        triggern1.workflow_dest_node = new WorkflowNode();
        triggern1.workflow_dest_node.id = 2;

        let triggernX = new WorkflowNodeTrigger();
        triggernX.workflow_dest_node = new WorkflowNode();
        triggernX.workflow_dest_node.id = 3;

        nRoot.triggers.push(triggern1, triggernX);

        let j1 = new WorkflowNodeJoin();
        j1.source_node_id = new Array<number>();
        j1.source_node_id.push(2, 3);
        workflow.joins.push(j1);

        let ok = Workflow.removeNodeWithoutChild(workflow, triggernX.workflow_dest_node);

        expect(ok).toBeTruthy();
        expect(workflow.joins[0].source_node_id.length).toBe(2);
        expect(workflow.joins[0].source_node_id[0] + workflow.joins[0].source_node_id[1]).toBe(3)
    }));

    /**
     *     R --> n1 -> n2 -> X -> J
     */
    it('should delete the node after join', fakeAsync(() => {
        let workflow = new Workflow();
        workflow.joins = new Array<WorkflowNodeJoin>();

        // Add root node
        let nRoot = new WorkflowNode();
        nRoot.id = 1;
        nRoot.triggers = new Array<WorkflowNodeTrigger>();
        workflow.root = nRoot;
        workflow.joins = new Array<WorkflowNodeJoin>();

        let triggern1 = new WorkflowNodeTrigger();
        triggern1.workflow_dest_node = new WorkflowNode();
        triggern1.workflow_dest_node.id = 2;
        nRoot.triggers.push(triggern1);


        triggern1.workflow_dest_node.triggers = new Array<WorkflowNodeTrigger>();
        let triggern2 = new WorkflowNodeTrigger();
        triggern2.workflow_dest_node = new WorkflowNode();
        triggern2.workflow_dest_node.id = 3;
        triggern1.workflow_dest_node.triggers.push(triggern2);

        triggern2.workflow_dest_node.triggers = new Array<WorkflowNodeTrigger>();
        let triggern3 = new WorkflowNodeTrigger();
        triggern3.workflow_dest_node = new WorkflowNode();
        triggern3.workflow_dest_node.id = 4;
        triggern2.workflow_dest_node.triggers.push(triggern3);

        let j1 = new WorkflowNodeJoin();
        j1.source_node_id = new Array<number>();
        j1.source_node_id.push(4);
        workflow.joins.push(j1);

        let ok = Workflow.removeNodeWithoutChild(workflow, triggern3.workflow_dest_node);

        expect(ok).toBeTruthy();
        expect(workflow.joins[0].source_node_id.length).toBe(1);
        expect(workflow.joins[0].source_node_id[0]).toBe(3)
    }));
})
;
