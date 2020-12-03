/* eslint-disable @typescript-eslint/no-unused-vars */

import {fakeAsync, TestBed} from '@angular/core/testing';
import {WNode, WNodeJoin, WNodeTrigger, Workflow} from './workflow.model';

describe('CDS: Workflow Model', () => {

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            declarations: [],
            providers: [
            ],
            imports: []
        }).compileComponents();
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
        workflow.workflow_data.joins = new Array<WNode>();

        // Add root node
        let nRoot = new WNode();
        nRoot.id = 1;
        workflow.workflow_data.node = nRoot;
        workflow.workflow_data.node.triggers = new Array<WNodeTrigger>();

        // Add root child
        let nToDelete = new WNode();
        nToDelete.id = 2;
        let rootTrigger = new WNodeTrigger();
        rootTrigger.id = 1;
        rootTrigger.child_node = nToDelete;
        workflow.workflow_data.node.triggers.push(rootTrigger);


        let nTriggerChild = new WNode();
        nTriggerChild.id = 3;
        nToDelete.triggers = new Array<WNodeTrigger>();
        let nDeleteTrigger = new WNodeTrigger();
        nDeleteTrigger.child_node = nTriggerChild;
        nToDelete.triggers.push(nDeleteTrigger);

        let nJ1Chlid = new WNode();
        nJ1Chlid.id = 4;
        let nJ2Children = new WNode();
        nJ2Children.id = 5;

        // Create Join with 1 parent and 1 child
        let join1 = new WNode();
        join1.parents = new Array<WNodeJoin>();

        let c = new WNodeJoin();
        c.parent_id = nToDelete.id;
        join1.parents.push(c);

        join1.triggers = new Array<WNodeTrigger>();
        let j1ChildT = new WNodeTrigger();
        j1ChildT.child_node = nJ1Chlid;
        join1.triggers.push(j1ChildT);
        workflow.workflow_data.joins.push(join1);

        // Create Join with 2 parent and 1 child
        let join2 = new WNode();
        join2.parents = new Array<WNodeJoin>();

        let j2parent1 = new WNodeJoin();
        j2parent1.parent_id = nToDelete.id;
        let j2parent2 = new WNodeJoin();
        j2parent2.parent_id = nRoot.id;
        join2.parents.push(j2parent1, j2parent2);
        join2.triggers = new Array<WNodeTrigger>();
        let jt2 = new WNodeTrigger();
        jt2.child_node = nJ2Children;
        jt2.id = 22;
        join2.triggers.push(jt2);
        workflow.workflow_data.joins.push(join2);

        let ok = Workflow.removeNodeOnly(workflow, nToDelete.id);

        expect(ok).toBeTruthy();

        // Assert join are attached to the root node
        expect(workflow.workflow_data.joins.length).toBe(2, 'root node must have 2 joins');
        expect(workflow.workflow_data.joins[0].parents.length).toBe(1, 'join 0 must have 1 parent');
        expect(workflow.workflow_data.joins[1].parents.length).toBe(1, 'join 1 must have 1 parent');
        expect(workflow.workflow_data.joins[0].parents[0].parent_id).toBe(1);
        expect(workflow.workflow_data.joins[1].parents[0].parent_id).toBe(1);

        // Assert child of deleted node is now on the root node
        expect(workflow.workflow_data.node.triggers).toBeTruthy();
        expect(workflow.workflow_data.node.triggers.length).toBe(1, 'root node must have 1 trigger');
        expect(workflow.workflow_data.node.triggers[0].child_node.id).toBe(nTriggerChild.id);
    }));


    /**
     * X --> o --> o
     */
    it('should delete root node: simple', fakeAsync(() => {
        let workflow = new Workflow();
        workflow.workflow_data.joins = new Array<WNode>();

        // Add root node
        let nRoot = new WNode();
        nRoot.id = 1;
        workflow.workflow_data.node = nRoot;
        workflow.workflow_data.node.triggers = new Array<WNodeTrigger>();

        // Add root child
        let child1 = new WNode();
        child1.id = 2;
        let rootTrigger = new WNodeTrigger();
        rootTrigger.id = 1;
        rootTrigger.child_node = child1;
        workflow.workflow_data.node.triggers.push(rootTrigger);

        let child2 = new WNode();
        child2.id = 3;
        child1.triggers = new Array<WNodeTrigger>();
        let childTrigger = new WNodeTrigger();
        childTrigger.child_node = child2;
        child1.triggers.push(childTrigger);


        let ok = Workflow.removeNodeOnly(workflow, nRoot.id);

        expect(ok).toBeTruthy();
        expect(workflow.workflow_data.node.type).toBe('fork');
        expect(workflow.workflow_data.node.triggers.length).toBe(1);
        expect(workflow.workflow_data.node.triggers[0].child_node.id).toBe(2);
        expect(workflow.workflow_data.node.triggers[0].child_node.triggers.length).toBe(1);
        expect(workflow.workflow_data.node.triggers[0].child_node.triggers[0].child_node.id).toBe(3);
    }));


    /**
     *     W --> J --> X ----> n2
     *     |            |
     *     v            v
     *     root         n3
     */
    it('should delete the node after join', fakeAsync(() => {
        let workflow = new Workflow();
        workflow.workflow_data.joins = new Array<WNode>();

        // Add root node
        let nRoot = new WNode();
        nRoot.id = 10;
        workflow.workflow_data.node = nRoot;

        let j1 = new WNode();
        j1.parents = new Array<WNodeJoin>();

        let j11 = new WNodeJoin();
        j11.parent_id = 10;
        j1.parents.push(j11);
        j1.triggers = new Array<WNodeTrigger>();

        let jt1 = new WNodeTrigger();
        let childjt1 = new WNode();
        childjt1.id = 1;
        childjt1.triggers = new Array<WNodeTrigger>();
        jt1.child_node = childjt1;
        j1.triggers.push(jt1);
        workflow.workflow_data.joins.push(j1);

        // 2 triggers
        let triggern2 = new WNodeTrigger();
        triggern2.child_node = new WNode();
        triggern2.child_node.id = 2;

        let triggern3 = new WNodeTrigger();
        triggern3.child_node = new WNode();
        triggern3.child_node.id = 3;

        childjt1.triggers.push(triggern2, triggern3);


        let ok = Workflow.removeNodeOnly(workflow, childjt1.id);

        expect(ok).toBeTruthy();
        expect(workflow.workflow_data.joins[0].triggers.length).toBe(2)
    }));

    /**
     *     R --> n1
     *     |     |
     *     v     v
     *     X--->J
     */
    it('should delete the parent node join', fakeAsync(() => {
        let workflow = new Workflow();
        workflow.workflow_data.joins = new Array<WNode>();

        // Add root node
        let nRoot = new WNode();
        nRoot.id = 1;
        nRoot.triggers = new Array<WNodeTrigger>();
        workflow.workflow_data.node = nRoot;

        let triggern1 = new WNodeTrigger();
        triggern1.child_node = new WNode();
        triggern1.child_node.id = 2;

        let triggernX = new WNodeTrigger();
        triggernX.child_node = new WNode();
        triggernX.child_node.id = 3;

        nRoot.triggers.push(triggern1, triggernX);

        let j1 = new WNode();
        j1.parents = new Array<WNodeJoin>();

        let j11 = new WNodeJoin();
        j11.parent_id = 2;
        let j12 = new WNodeJoin();
        j12.parent_id = 3;
        j1.parents.push(j11, j12);
        workflow.workflow_data.joins.push(j1);

        let ok = Workflow.removeNodeOnly(workflow, triggernX.child_node.id);

        expect(ok).toBeTruthy();
        expect(workflow.workflow_data.joins[0].parents.length).toBe(2);
        expect(workflow.workflow_data.joins[0].parents[0].parent_id + workflow.workflow_data.joins[0].parents[1].parent_id).toBe(3, 'Parent must be id 1 and 2, got '+ JSON.stringify(workflow.workflow_data.joins[0].parents));
    }));

    /**
     *     R --> n1 -> n2 -> X -> J
     */
    it('should delete the node after join', fakeAsync(() => {
        let workflow = new Workflow();
        workflow.workflow_data.joins = new Array<WNode>();

        // Add root node
        let nRoot = new WNode();
        nRoot.id = 1;
        nRoot.triggers = new Array<WNodeTrigger>();
        workflow.workflow_data.node = nRoot;

        let triggern1 = new WNodeTrigger();
        triggern1.child_node = new WNode();
        triggern1.child_node.id = 2;
        nRoot.triggers.push(triggern1);


        triggern1.child_node.triggers = new Array<WNodeTrigger>();
        let triggern2 = new WNodeTrigger();
        triggern2.child_node = new WNode();
        triggern2.child_node.id = 3;
        triggern1.child_node.triggers.push(triggern2);

        triggern2.child_node.triggers = new Array<WNodeTrigger>();
        let triggern3 = new WNodeTrigger();
        triggern3.child_node = new WNode();
        triggern3.child_node.id = 4;
        triggern2.child_node.triggers.push(triggern3);

        let j1 = new WNode();
        j1.parents = new Array<WNodeJoin>();

        let j11 = new WNodeJoin();
        j11.parent_id = 4;

        j1.parents.push(j11);
        workflow.workflow_data.joins.push(j1);

        let ok = Workflow.removeNodeOnly(workflow, triggern3.child_node.id);

        expect(ok).toBeTruthy();
        expect(workflow.workflow_data.joins[0].parents.length).toBe(1);
        expect(workflow.workflow_data.joins[0].parents[0].parent_id).toBe(3)
    }));
});
