/* tslint:disable:no-unused-variable */
import {async, getTestBed, TestBed} from '@angular/core/testing';
import {APP_BASE_HREF} from '@angular/common';
import {MockBackend} from '@angular/http/testing';
import {Http, RequestOptions, Response, ResponseOptions} from '@angular/http';
import {Injector} from '@angular/core';
import {AppModule} from '../../app.module';
import {AuthentificationStore} from '../auth/authentification.store';
import {HttpService} from '../http-service.service';
import {Router, RouterModule} from '@angular/router';
import {PipelineStore} from './pipeline.store';
import {Pipeline} from '../../model/pipeline.model';
import {ToastService} from '../../shared/toast/ToastService';
import {Stage} from '../../model/stage.model';
import {Action} from '../../model/action.model';
import {Job} from '../../model/job.model';
import {Project} from '../../model/project.model';
import {GroupPermission} from '../../model/group.model';
import {Parameter} from '../../model/parameter.model';

describe('CDS: pipeline Store', () => {

    let injector: Injector;
    let backend: MockBackend;
    let pipelineStore: PipelineStore;


    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [],
            providers: [
                {provide: APP_BASE_HREF, useValue: '/'},
                MockBackend,
                {
                    provide: Http,
                    useFactory: (backendParam: MockBackend,
                                 defaultOptions: RequestOptions,
                                 toast: ToastService,
                                 authStore: AuthentificationStore,
                                 router: Router) =>
                        new HttpService(backendParam, defaultOptions, toast, authStore, router),
                    deps: [MockBackend, RequestOptions, ToastService, AuthentificationStore]
                }
            ],
            imports: [
                AppModule,
                RouterModule
            ]
        });
        injector = getTestBed();
        backend = injector.get(MockBackend);
        pipelineStore = injector.get(PipelineStore);

    });

    afterEach(() => {
        injector = undefined;
        backend = undefined;
        pipelineStore = undefined;
    });

    it('Create and Delete Pipeline', async(() => {
        let call = 0;
        let projectKey = 'key1';
        // Mock Http pipeline request
        backend.connections.subscribe(connection => {
            switch (call) {
                case 0:
                    call++;
                    connection.mockRespond(new Response(new ResponseOptions({body: '{ "name": "myPipeline" }'})));
                    break;
                case 1:
                    call++;
                    connection.mockRespond(new Response(new ResponseOptions({body: '{ "name": "myPipeline2" }'})));
                    break;
                case 2:
                    call++;
                    connection.mockRespond(new Response(new ResponseOptions({body: '{ "name": "myPipeline" }'})));
                    break;
                case 3:
                    call++;
                    connection.mockRespond(new Response(new ResponseOptions({body: '{ "name": "myPipeline" }'})));
                    break;
            }


        });
        // Create 1st pipeline
        let checkPipelineCreated = false;
        pipelineStore.createPipeline(projectKey, createPipeline('myPipeline')).subscribe(res => {
            expect(res.name).toBe('myPipeline', 'Wrong pipeline name');
            checkPipelineCreated = true;
        });
        expect(call).toBe(1, 'Need to have done 1 http call');

        // check get pipeline (get from cache)
        let checkedSinglePipeline = false;
        pipelineStore.getPipelines(projectKey, 'myPipeline').subscribe(pips => {
            expect(pips.get(projectKey + '-' + 'myPipeline')).toBeTruthy();
            expect(pips.get(projectKey + '-' + 'myPipeline').name).toBe('myPipeline', 'Wrong pipeline name. Must be myPipeline');
            checkedSinglePipeline = true;
        }).unsubscribe();
        expect(checkedSinglePipeline).toBeTruthy('Need to get pipeline myPipeline');
        expect(call).toBe(1, 'Need to have done 1 http call');


        // check get pipeline not in cache
        let checkednotCachedPipeline = false;
        pipelineStore.getPipelines(projectKey, 'myPipeline2').subscribe(pips => {
            expect(pips.get(projectKey + '-' + 'myPipeline2')).toBeTruthy();
            expect(pips.get(projectKey + '-' + 'myPipeline2').name).toBe('myPipeline2', 'Wrong pipeline name. Must be myPipeline2');
            checkednotCachedPipeline = true;
        }).unsubscribe();
        expect(checkednotCachedPipeline).toBeTruthy('Need to get pipeline myPipeline2');
        expect(call).toBe(2, 'Need to have done 2 http call');

        // Pipeline deletion

        pipelineStore.deletePipeline(projectKey, 'myPipeline2').subscribe(() => {
        });
        expect(call).toBe(3, 'Need to have done 3 http call');

        let checkedDeletedPipeline = false;
        pipelineStore.getPipelines(projectKey, 'myPipeline2').first().subscribe(pips => {
            checkedDeletedPipeline = true;
        });
        expect(checkedDeletedPipeline).toBeTruthy('Need to get pipeline myPipeline');
        expect(call).toBe(4, 'Need to have done 4 http call');
    }));

    it('Update pipeline', async(() => {
        let call = 0;
        let projectKey = 'key1';
        // Mock Http pipeline request
        backend.connections.subscribe(connection => {
            switch (call) {
                case 0: // create pipeline
                    call++;
                    connection.mockRespond(new Response(new ResponseOptions({body: '{ "name": "myPipeline" }'})));
                    break;
                case 1: // 1st update
                    call++;
                    connection.mockRespond(new Response(new ResponseOptions({body: '{ "name": "myPipelineUpdate1" }'})));
                    break;
            }


        });
        // Create pipeline
        let p = createPipeline('myPipeline');
        pipelineStore.createPipeline(projectKey, p).subscribe(() => {
        });
        expect(call).toBe(1, 'Need to have done 1 http call');

        // Update
        p.name = 'myPipelineUpdate1';
        pipelineStore.updatePipeline(projectKey, 'myPipeline', p).subscribe(() => {
        });
        expect(call).toBe(2, 'Need to have done 2 http call');

        // check get pipeline
        let checkedPipeline = false;
        pipelineStore.getPipelines(projectKey, 'myPipelineUpdate1').subscribe(pips => {
            expect(pips.get(projectKey + '-' + 'myPipelineUpdate1')).toBeTruthy();
            expect(pips.get(projectKey + '-' + 'myPipelineUpdate1').name)
                .toBe('myPipelineUpdate1', 'Wrong pipeline name. Must be myPipelineUpdate1');
            checkedPipeline = true;
        }).unsubscribe();
        expect(checkedPipeline).toBeTruthy('Need to get pipeline myPipelineUpdate1');
        expect(call).toBe(2, 'Need to have done 2 http call');
    }));

    it('should create/update and delete a stage', async(() => {
        let call = 0;
        let projectKey = 'key1';
        // Mock Http pipeline request
        backend.connections.subscribe(connection => {
            switch (call) {
                case 0:
                    call++;
                    connection.mockRespond(new Response(new ResponseOptions({body: '{ "name": "myPipeline" }'})));
                    break;
                case 1:
                    call++;
                    connection.mockRespond(
                        new Response(
                            new ResponseOptions({body: `{ "name": "myPipeline", "stages": [{ "name": "stage1" }] }`})));
                    break;
                case 2:
                    call++;
                    connection.mockRespond(
                        new Response(
                            new ResponseOptions({body: `{ "name": "myPipeline", "stages": [{ "name": "stage1Updated" }] }`})));
                    break;
                case 3:
                    call++;
                    connection.mockRespond(new Response(new ResponseOptions({body: '{ "name": "myPipeline", "stages": [] }'})));
                    break;
            }


        });
        // Create 1st pipeline
        let checkPipelineCreated = false;
        pipelineStore.createPipeline(projectKey, createPipeline('myPipeline')).subscribe(res => {
            expect(res.name).toBe('myPipeline', 'Wrong pipeline name');
            checkPipelineCreated = true;
        }).unsubscribe();
        expect(call).toBe(1, 'Need to have done 1 http call');

        // ADD STAGE

        let s: Stage = new Stage();
        s.name = 'stage1';
        pipelineStore.addStage(projectKey, 'myPipeline', s).subscribe(() => {
        }).unsubscribe();
        expect(call).toBe(2, 'Need to have done 2 http call');

        // check get pipeline (get from cache)
        let checkStageAdd = false;
        pipelineStore.getPipelines(projectKey, 'myPipeline').subscribe(pips => {
            expect(pips.get(projectKey + '-' + 'myPipeline')).toBeTruthy();
            expect(pips.get(projectKey + '-' + 'myPipeline').stages.length).toBe(1, 'Must have 1 stage');
            expect(pips.get(projectKey + '-' + 'myPipeline').stages[0].name).toBe('stage1', 'Wrong stage');
            checkStageAdd = true;
        }).unsubscribe();
        expect(checkStageAdd).toBeTruthy();
        expect(call).toBe(2, 'Need to have done 2 http call');

        // UPDATE STAGE

        pipelineStore.updateStage(projectKey, 'myPipeline', s).subscribe(() => {
        });
        expect(call).toBe(3, 'Need to have done 3 http call');

        let checkStageUpdate = false;
        pipelineStore.getPipelines(projectKey, 'myPipeline').subscribe(pips => {
            expect(pips.get(projectKey + '-' + 'myPipeline')).toBeTruthy();
            expect(pips.get(projectKey + '-' + 'myPipeline').stages.length).toBe(1, 'Must have 1 stage');
            expect(pips.get(projectKey + '-' + 'myPipeline').stages[0].name).toBe('stage1Updated', 'Wrong stage');
            checkStageUpdate = true;
        }).unsubscribe();
        expect(checkStageUpdate).toBeTruthy();
        expect(call).toBe(3, 'Need to have done 3 http call');

        // DELETE STAGE

        pipelineStore.removeStage(projectKey, 'myPipeline', s).subscribe(() => {
        });
        expect(call).toBe(4, 'Need to have done 4 http call');

        let checkStageDelete = false;
        pipelineStore.getPipelines(projectKey, 'myPipeline').subscribe(pips => {
            expect(pips.get(projectKey + '-' + 'myPipeline')).toBeTruthy();
            expect(pips.get(projectKey + '-' + 'myPipeline').stages.length).toBe(0, 'Must have 0 stage');
            checkStageDelete = true;
        }).unsubscribe();
        expect(checkStageDelete).toBeTruthy();
        expect(call).toBe(4, 'Need to have done 4 http call');

    }));

    it('should create/update and delete a job', async(() => {
        let call = 0;
        let projectKey = 'key1';
        // Mock Http pipeline request
        backend.connections.subscribe(connection => {
            switch (call) {
                case 0:
                    call++;
                    connection.mockRespond(new Response(new ResponseOptions({body: '{ "name": "myPipeline", "stages": [{ "id": 1 }] }'})));
                    break;
                case 1:
                    call++;
                    connection.mockRespond(
                        new Response(
                            new ResponseOptions({
                                body: `{
                                "name": "myPipeline",
                                "stages": [{ "id": 1, "jobs": [{"action": { "name": "action1" }}] }]
                            }`
                            })));
                    break;
                case 2:
                    call++;
                    connection.mockRespond(
                        new Response(
                            new ResponseOptions({
                                body: `{
                                "name": "myPipeline",
                                "stages": [{ "id": 1, "jobs": [ {"action": { "name": "action1Updated" }}] }]
                            }`
                            })));
                    break;
                case 3:
                    call++;
                    connection.mockRespond(
                        new Response(
                            new ResponseOptions({
                                body: `{
                                "name": "myPipeline",
                                "stages": [{ "id": 1, "jobs": [] }]
                            }`
                            })
                        )
                    );
                    break;
            }


        });
        // Create 1st pipeline
        let checkPipelineCreated = false;
        pipelineStore.createPipeline(projectKey, createPipeline('myPipeline')).subscribe(res => {
            expect(res.name).toBe('myPipeline', 'Wrong pipeline name');
            checkPipelineCreated = true;
        }).unsubscribe();
        expect(call).toBe(1, 'Need to have done 1 http call');

        // ADD Job
        let j = new Job();
        let a: Action = new Action();
        a.name = 'action1';
        j.action = a;
        pipelineStore.addJob(projectKey, 'myPipeline', 1, j).subscribe(() => {
        }).unsubscribe();
        expect(call).toBe(2, 'Need to have done 2 http call');


        let checkJobAdd = false;
        pipelineStore.getPipelines(projectKey, 'myPipeline').subscribe(pips => {
            expect(pips.get(projectKey + '-' + 'myPipeline')).toBeTruthy();
            expect(pips.get(projectKey + '-' + 'myPipeline').stages.length).toBe(1, 'Must have 1 stage');
            expect(pips.get(projectKey + '-' + 'myPipeline').stages[0].jobs.length).toBe(1, 'Must have 1 action');
            expect(pips.get(projectKey + '-' + 'myPipeline').stages[0].jobs[0].action.name).toBe('action1', 'Wrong action');
            checkJobAdd = true;
        }).unsubscribe();
        expect(checkJobAdd).toBeTruthy();
        expect(call).toBe(2, 'Need to have done 2 http call');

        // UPDATE STAGE

        pipelineStore.updateJob(projectKey, 'myPipeline', 1, j).subscribe(() => {
        });
        expect(call).toBe(3, 'Need to have done 3 http call');

        let checkJobUpdate = false;
        pipelineStore.getPipelines(projectKey, 'myPipeline').subscribe(pips => {
            expect(pips.get(projectKey + '-' + 'myPipeline')).toBeTruthy();
            expect(pips.get(projectKey + '-' + 'myPipeline').stages.length).toBe(1, 'Must have 1 stage');
            expect(pips.get(projectKey + '-' + 'myPipeline').stages[0].jobs.length).toBe(1, 'Must have 1 action');
            expect(pips.get(projectKey + '-' + 'myPipeline').stages[0].jobs[0].action.name).toBe('action1Updated', 'Wrong action');
            checkJobUpdate = true;
        }).unsubscribe();
        expect(checkJobUpdate).toBeTruthy();
        expect(call).toBe(3, 'Need to have done 3 http call');

        // DELETE STAGE

        pipelineStore.removeJob(projectKey, 'myPipeline', 1, j).subscribe(() => {
        });
        expect(call).toBe(4, 'Need to have done 4 http call');

        let checkJobDelete = false;
        pipelineStore.getPipelines(projectKey, 'myPipeline').subscribe(pips => {
            expect(pips.get(projectKey + '-' + 'myPipeline')).toBeTruthy();
            expect(pips.get(projectKey + '-' + 'myPipeline').stages.length).toBe(1, 'Must have10 stage');
            expect(pips.get(projectKey + '-' + 'myPipeline').stages[0].jobs.length).toBe(0);
            checkJobDelete = true;
        }).unsubscribe();
        expect(checkJobDelete).toBeTruthy();
        expect(call).toBe(4, 'Need to have done 4 http call');

    }));

    it('should add/update/delete a permission', async(() => {
        let call = 0;
        // Mock Http application request
        backend.connections.subscribe(connection => {
            switch (call) {
                case 0: // create application
                    call++;
                    connection.mockRespond(new Response(new ResponseOptions({body: '{ "name": "myPipeline", "last_modified": 0 }'})));
                    break;
                case 1: // Add variable
                    call++;
                    connection.mockRespond(new Response(new ResponseOptions({
                        body: `{
                        "name": "myPipeline",
                        "last_modified": 123,
                        "groups": [ {"permission": 7} ] }`
                    })));
                    break;
                case 2: // update variable
                    call++;
                    connection.mockRespond(new Response(new ResponseOptions({
                        body: `{
                         "name": "myPipeline",
                        "last_modified": 456,
                        "groups": [ {"permission": 4} ] }`
                    })));
                    break;
                case 3: // delete variable
                    call++;
                    connection.mockRespond(new Response(new ResponseOptions({
                        body: `{
                         "name": "myPipeline",
                        "last_modified": 789,
                        "groups": [ ] }`
                    })));
                    break;
            }


        });

        let proj: Project = new Project();
        proj.key = 'key1';

        // Create pipeline
        let pip = createPipeline('myPipeline');
        pipelineStore.createPipeline(proj.key, pip).subscribe(() => {
        });
        expect(call).toBe(1, 'Need to have done 1 http call');

        let gp: GroupPermission = new GroupPermission();
        gp.permission = 0;


        pipelineStore.addPermission(proj.key, pip.name, gp).subscribe(() => {
        });
        expect(call).toBe(2, 'Need to have done 2 http call');

        // check get pipeline
        let checkedAddPermission = false;
        pipelineStore.getPipelines(proj.key, 'myPipeline').subscribe(apps => {
            expect(apps.get(proj.key + '-myPipeline').last_modified).toBe(123, 'Pip lastModified date must have been updated');
            expect(apps.get(proj.key + '-myPipeline').groups.length).toBe(1, 'A group must have been added');
            expect(apps.get(proj.key + '-myPipeline').groups[0].permission).toBe(7, 'Permission must be 7');
            checkedAddPermission = true;
        }).unsubscribe();
        expect(checkedAddPermission).toBeTruthy('Need pipeline to be updated');
        expect(call).toBe(2, 'Need to have done 2 http call');


        pipelineStore.updatePermission(proj.key, pip.name, gp).subscribe(() => {
        });
        expect(call).toBe(3, 'Need to have done 3 http call');

        // check get pipeline
        let checkedUpdatePermission = false;
        pipelineStore.getPipelines(proj.key, 'myPipeline').subscribe(apps => {
            expect(apps.get(proj.key + '-myPipeline').last_modified).toBe(456, 'Pip lastModified date must have been updated');
            expect(apps.get(proj.key + '-myPipeline').groups.length).toBe(1, 'Pip must have 1 group');
            expect(apps.get(proj.key + '-myPipeline').groups[0].permission).toBe(4, 'Group permission must be 4');
            checkedUpdatePermission = true;
        }).unsubscribe();
        expect(checkedUpdatePermission).toBeTruthy('Need pipeline to be updated');
        expect(call).toBe(3, 'Need to have done 3 http call');


        pipelineStore.removePermission(proj.key, pip.name, gp).subscribe(() => {
        });
        expect(call).toBe(4, 'Need to have done 4 http call');

        // check get pipeline
        let checkedDeletePermission = false;
        pipelineStore.getPipelines(proj.key, 'myPipeline').subscribe(apps => {
            expect(apps.get(proj.key + '-myPipeline').last_modified).toBe(789, 'Pip lastModified date must have been updated');
            expect(apps.get(proj.key + '-myPipeline').groups.length).toBe(0, 'Ouo must have 0 group');
            checkedDeletePermission = true;
        }).unsubscribe();
        expect(checkedDeletePermission).toBeTruthy('Need pipeline to be updated');
        expect(call).toBe(4, 'Need to have done 3 http call');


    }));

    it('should add/update/delete a parameter', async(() => {
        let call = 0;
        // Mock Http application request
        backend.connections.subscribe(connection => {
            switch (call) {
                case 0: // create application
                    call++;
                    connection.mockRespond(new Response(new ResponseOptions({body: '{ "name": "myPipeline", "last_modified": 0 }'})));
                    break;
                case 1: // Add variable
                    call++;
                    connection.mockRespond(new Response(new ResponseOptions({
                        body: `{
                        "name": "myPipeline",
                        "last_modified": 123,
                        "parameters": [ {"name": "foo"} ] }`
                    })));
                    break;
                case 2: // update variable
                    call++;
                    connection.mockRespond(new Response(new ResponseOptions({
                        body: `{
                         "name": "myPipeline",
                        "last_modified": 456,
                        "parameters": [ {"name": "fooUpdated"} ] }`
                    })));
                    break;
                case 3: // delete variable
                    call++;
                    connection.mockRespond(new Response(new ResponseOptions({
                        body: `{
                         "name": "myPipeline",
                        "last_modified": 789,
                        "parameters": [ ] }`
                    })));
                    break;
            }


        });

        let proj: Project = new Project();
        proj.key = 'key1';

        // Create pipeline
        let pip = createPipeline('myPipeline');
        pipelineStore.createPipeline(proj.key, pip).subscribe(() => {
        });
        expect(call).toBe(1, 'Need to have done 1 http call');

        let param: Parameter = new Parameter();
        param.name = 'foo';
        param.type = 'string';
        param.description = 'my description';
        param.value = 'bar';


        pipelineStore.addParameter(proj.key, pip.name, param).subscribe(() => {
        });
        expect(call).toBe(2, 'Need to have done 2 http call');

        // check get pipeline
        let checkedAddParam = false;
        pipelineStore.getPipelines(proj.key, 'myPipeline').subscribe(apps => {
            expect(apps.get(proj.key + '-myPipeline').last_modified).toBe(123, 'Pip lastModified date must have been updated');
            expect(apps.get(proj.key + '-myPipeline').parameters.length).toBe(1, 'A parameter must have been added');
            expect(apps.get(proj.key + '-myPipeline').parameters[0].name).toBe('foo', 'Name must be foo');
            checkedAddParam = true;
        }).unsubscribe();
        expect(checkedAddParam).toBeTruthy('Need pipeline to be updated');
        expect(call).toBe(2, 'Need to have done 2 http call');


        pipelineStore.updateParameter(proj.key, pip.name, param).subscribe(() => {
        });
        expect(call).toBe(3, 'Need to have done 3 http call');

        // check get pipeline
        let checkedUpdateParam = false;
        pipelineStore.getPipelines(proj.key, 'myPipeline').subscribe(apps => {
            expect(apps.get(proj.key + '-myPipeline').last_modified).toBe(456, 'Pip lastModified date must have been updated');
            expect(apps.get(proj.key + '-myPipeline').parameters.length).toBe(1, 'Pip must have 1 group');
            expect(apps.get(proj.key + '-myPipeline').parameters[0].name).toBe('fooUpdated', 'Name must be fooUpdated');
            checkedUpdateParam = true;
        }).unsubscribe();
        expect(checkedUpdateParam).toBeTruthy('Need pipeline to be updated');
        expect(call).toBe(3, 'Need to have done 3 http call');


        pipelineStore.removeParameter(proj.key, pip.name, param).subscribe(() => {
        });
        expect(call).toBe(4, 'Need to have done 4 http call');

        // check get pipeline
        let checkedDeleteParam = false;
        pipelineStore.getPipelines(proj.key, 'myPipeline').subscribe(apps => {
            expect(apps.get(proj.key + '-myPipeline').last_modified).toBe(789, 'Pip lastModified date must have been updated');
            expect(apps.get(proj.key + '-myPipeline').parameters.length).toBe(0, 'Pip must have 0 parameter');
            checkedDeleteParam = true;
        }).unsubscribe();
        expect(checkedDeleteParam).toBeTruthy('Need pipeline to be updated');
        expect(call).toBe(4, 'Need to have done 3 http call');


    }));

    function createPipeline(name: string): Pipeline {
        let pip: Pipeline = new Pipeline();
        pip.name = name;
        return pip;
    }
});
