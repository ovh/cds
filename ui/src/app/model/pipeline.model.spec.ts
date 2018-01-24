/* tslint:disable:no-unused-variable */

import {fakeAsync, TestBed} from '@angular/core/testing';
import {Workflow, WorkflowNode, WorkflowNodeJoin, WorkflowNodeJoinTrigger, WorkflowNodeTrigger} from './workflow.model';
import {Parameter} from './parameter.model';
import {Pipeline} from './pipeline.model';

describe('CDS: Pipeline Model', () => {

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [],
            providers: [
            ],
            imports: []
        });
    });


    it('should add new pipeline params', fakeAsync(() => {
        let pipParams = new Array<Parameter>();
        let p1  = new Parameter();
        p1.name = 'param1';
        pipParams.push(p1);

        let p2 = new Parameter();
        p2.name = 'param2';
        pipParams.push(p2);

        let nodeParams = new Array<Parameter>();
        nodeParams.push(p1);

        let pp = Pipeline.mergeAndKeepOld(pipParams, nodeParams);
        expect(pp.length).toBe(2);

        let mapParam = pp.reduce((m, o) => {
            m[o.name] = o;
            return m;
        }, {});
        expect(mapParam['param1']).toBeTruthy();
        expect(mapParam['param2']).toBeTruthy();
    }));


    it('should keep old value', fakeAsync(() => {
        let pipParams = new Array<Parameter>();
        let p1  = new Parameter();
        p1.name = 'param1';
        pipParams.push(p1);

        let p2 = new Parameter();
        p2.name = 'param2';

        let nodeParams = new Array<Parameter>();
        nodeParams.push(p1, p2);

        let pp = Pipeline.mergeAndKeepOld(pipParams, nodeParams);
        expect(pp.length).toBe(2);

        let mapParam = pp.reduce((m, o) => {
            m[o.name] = o;
            return m;
        }, {});
        expect(mapParam['param1']).toBeTruthy();
        expect(mapParam['param2']).toBeTruthy();
    }));

})
;
