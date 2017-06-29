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
import {ProjectStore} from './project.store';
import {Project} from '../../model/project.model';
import {ToastService} from '../../shared/toast/ToastService';
import {Variable} from '../../model/variable.model';
import {Group, GroupPermission} from '../../model/group.model';
import {Environment} from '../../model/environment.model';

describe('CDS: project Store', () => {

    let injector: Injector;
    let backend: MockBackend;
    let projectStore: ProjectStore;


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
        projectStore = injector.get(ProjectStore);

    });

    afterEach(() => {
        injector = undefined;
        backend = undefined;
        projectStore = undefined;
    });

    it('should Get Projects', async(() => {
        let call = 0;
        // Mock Http project request
        backend.connections.subscribe(connection => {
            call++;
            connection.mockRespond(new Response(new ResponseOptions({
                body: `
            [
                { "key": "key1", "name": "myProject" },
                { "key": "key2", "name": "myProject2" },
                { "key": "key3", "name": "myProject3" },
                { "key": "key4", "name": "myProject4" }
            ]`
            })));
        });

        // Get projects from HTTP Call
        projectStore.getProjectsList().subscribe(projects => {
            expect(projects.size).toBe(4, 'Wrong number of project. Must be 4.');
        }).unsubscribe();
        expect(call).toBe(1, 'Need to call API to get projects for the first time.');

        projectStore.getProjectsList().subscribe(projects => {
            expect(projects.size).toBe(4, 'Wrong number of project. Must be 4.');
        }).unsubscribe();
        expect(call).toBe(1, 'No need another http call, projects are cached.');
    }));

    it('should create a Project', async(() => {
        let call = 0;
        // Mock Http project request
        backend.connections.subscribe(connection => {
            switch (call) {
                case 0:
                    call++;
                    connection.mockRespond(new Response(new ResponseOptions({body: '{ "key": "key1", "name": "myProject" }'})));
                    break;
                case 1:
                    call++;
                    connection.mockRespond(new Response(new ResponseOptions({body: '{ "key": "key2", "name": "myProject2" }'})));
                    break;
                case 2:
                    call++;
                    connection.mockRespond(new Response(new ResponseOptions({body: '{ "key": "key1", "name": "myProject" }'})));
                    break;
                case 3:
                case 4:
                    call++;
                    connection.mockRespond(new Response(new ResponseOptions({body: '{ }'})));
                    break;
            }


        });
        // Create project
        projectStore.createProject(createProject('key1', 'myProject')).subscribe(() => {
        });
        projectStore.createProject(createProject('key2', 'myProject2')).subscribe(() => {
        });


        // check cache for nav
        let checkedNav = false;
        projectStore.getProjectsList().subscribe(projects => {
            expect(projects.size).toBe(2, 'Wrong number of project. Must be 2.');
            checkedNav = true;
        }).unsubscribe();
        expect(checkedNav).toBeTruthy('Need at least 2 projects in nav');

        // check get project: first time ( get from http call)
        let checkedSingleProject = false;
        projectStore.getProjects('key1').subscribe(projs => {
            expect(projs.get('key1').name).toBe('myProject', 'Wrong project name. Must be myProject');
            checkedSingleProject = true;
        }).unsubscribe();
        expect(checkedSingleProject).toBeTruthy('Need to get project key1');
        expect(call).toBe(3, 'Need to have done 3 http call');

        // check get project: second time (get from cache
        let checkedSingleProjectTwice = false;
        projectStore.getProjects('key1').subscribe(projs => {
            expect(projs.get('key1').name).toBe('myProject', 'Wrong project name. Must be myProject');
            checkedSingleProjectTwice = true;
        }).unsubscribe();
        expect(checkedSingleProjectTwice).toBeTruthy('Need to get project key1 twice');
        expect(call).toBe(3, 'Need to have done 3 http call');

        // Delete project
        projectStore.deleteProject('key1').subscribe(() => {
        });
        expect(call).toBe(4, 'Need to have done 4 http call');

        let checkedDeleteProject = false;
        projectStore.getProjects('key1').subscribe(() => {
            checkedDeleteProject = true;
        }).unsubscribe();
        expect(checkedDeleteProject).toBeTruthy('Need to get project key1 from http');
        expect(call).toBe(5, 'Need to have done 5 http call, because key1 was deleted from cache');

    }));

    it('should update application in Project', async(() => {
        let call = 0;
        // Mock Http project request
        backend.connections.subscribe(connection => {
            connection.mockRespond(new Response(new ResponseOptions({
                body: `{
                "key": "key1",
                "name": "myProject",
                "applications" : [
                    { "name": "app1" },
                    { "name": "app2" },
                    { "name": "app3" }
                ]
            }`
            })));
        });

        // Create project
        let p = createProject('key1', 'myProject');
        projectStore.createProject(p).subscribe(() => {
        });

        let projectChecked = false;
        projectStore.getProjects('key1').subscribe(projs => {
            expect(projs.get('key1').applications.length).toBe(3, 'Wrong number of applications. Must be 3.');
            projectChecked = true;
        }).unsubscribe();
        expect(projectChecked).toBeTruthy();

        projectStore.updateApplicationName('key1', 'app2', 'appupdated');

        let projectAppChecked = false;
        projectStore.getProjects('key1').subscribe(projs => {
            expect(projs.get('key1').applications.length).toBe(3, 'Wrong number of applications. Must be 3.');
            expect(projs.get('key1').applications[1].name).toBe('appupdated');
            projectAppChecked = true;
        }).unsubscribe();
        expect(projectAppChecked).toBeTruthy();
    }));

    it('should update Project', async(() => {
        let call = 0;
        // Mock Http project request
        backend.connections.subscribe(connection => {
            switch (call) {
                case 0: // create project
                    call++;
                    connection.mockRespond(new Response(new ResponseOptions({body: '{ "key": "key1", "name": "myProject" }'})));
                    break;
                case 1: // 1st update
                case 2: // get project
                    call++;
                    connection.mockRespond(new Response(new ResponseOptions({body: '{ "key": "key1", "name": "myProjectUpdate1" }'})));
                    break;
                case 3: // 2nd update
                    call++;
                    connection.mockRespond(new Response(new ResponseOptions({body: '{ "key": "key1", "name": "myProjectUpdate2" }'})));
                    break;
            }


        });
        // Create project
        let p = createProject('key1', 'myProject');
        projectStore.createProject(p).subscribe(() => {
        });
        expect(call).toBe(1, 'Need to have done 1 http call');

        // Update
        p.name = 'myProjectUpdate1';
        projectStore.updateProject(p).subscribe(() => {
        });
        expect(call).toBe(2, 'Need to have done 1 http call');

        // check cache for nav
        let checkedNav = false;
        projectStore.getProjectsList().subscribe(projects => {
            expect(projects.size).toBe(1, 'Wrong number of project. Must be 1.');
            expect(projects.get(0).name).toBe('myProjectUpdate1', 'Wrong name of project. Must be myProjectUpdate1');
            checkedNav = true;
        }).unsubscribe();
        expect(checkedNav).toBeTruthy('Need at least 2 projects in nav');

        // check get project: first time ( get from http call)
        let checkedSingleProject = false;
        projectStore.getProjects('key1').subscribe(projs => {
            expect(projs.get('key1').name).toBe('myProjectUpdate1', 'Wrong project name. Must be myProjectUpdate1');
            checkedSingleProject = true;
        }).unsubscribe();
        expect(checkedSingleProject).toBeTruthy('Need to get project key1');
        expect(call).toBe(3, 'Need to have done 3 http call');

        // Update 2
        p.name = 'myProjectUpdate2';
        projectStore.updateProject(p).subscribe(() => {
        });
        expect(call).toBe(4, 'Need to have done 4 http call');

        // check get project: second time (get from cache
        let checkedSingleProjectTwice = false;
        projectStore.getProjects('key1').subscribe(projs => {
            expect(projs.get('key1').name).toBe('myProjectUpdate2', 'Wrong project name. Must be myProjectUpdate2');
            checkedSingleProjectTwice = true;
        }).unsubscribe();
        expect(checkedSingleProjectTwice).toBeTruthy('Need to get project key1 twice');
        expect(call).toBe(4, 'Still need to have done 4 http call');


    }));

    it('should add a repomananger + validate repo manager + delete repomanager', async(() => {
        let call = 0;
        // Mock Http project request
        backend.connections.subscribe(connection => {
            call++;
            switch (call) {
                case 1: // create project
                    connection.mockRespond(new Response(new ResponseOptions({
                        body: `{
                        "key": "key1",
                        "name": "myProject",
                        "last_modified": 0,
                        "repositories_manager" : []
                        }`
                    })));
                    break;
                case 2: // add repo
                    connection.mockRespond(new Response(new ResponseOptions({
                        body: `{
                        "project_key": "key1",
                        "last_modified" : "123",
                        "repositories_manager" : "repoName",
                        "url" : "ff/.c",
                        "request_token": "mytoken"
                    }`
                    })));
                    break;
                case 3: // Validation
                    connection.mockRespond(new Response(new ResponseOptions({
                        body: `{
                        "key": "key1",
                        "name": "myProject",
                        "last_modified": 456,
                        "repositories_manager" : [{ "name": "repoName"}]
                        }`
                    })));
                    break;
                case 4: // Delete
                    connection.mockRespond(new Response(new ResponseOptions({
                        body: `{
                            "key": "key1",
                            "name": "myProject",
                            "last_modified": 789,
                            "repositories_manager" : [{ "name": "repoName"}]
                            }`
                    })));
                    break;
            }


        });

        // Get project in cache
        projectStore.getProjects('key1').first().subscribe(() => {
        });
        expect(call).toBe(1, 'Need to have done 1 http call');


        // Add repo
        projectStore.connectRepoManager('key1', 'repoName').subscribe(() => {
        }).unsubscribe();
        expect(call).toBe(2, 'Need to have done 2 http call');

        let addRepoCheck = false;
        projectStore.getProjects('key1').first().subscribe(projs => {
            addRepoCheck = true;
            expect(projs.get('key1').last_modified).toBe('123', 'Project must have been updated');
        });
        expect(addRepoCheck).toBeTruthy('Must check project update');

        // validate repo
        projectStore.verificationCallBackRepoManager('key1', 'repoName', 'token', 'code').subscribe(() => {
        }).unsubscribe();
        expect(call).toBe(3, 'Need to have done 3 http call');

        let validationRepoCheck = false;
        projectStore.getProjects('key1').first().subscribe(projs => {
            validationRepoCheck = true;
            expect(projs.get('key1').last_modified).toBe(456);
            expect(projs.get('key1').repositories_manager.length).toBe(1);
        });
        expect(validationRepoCheck).toBeTruthy('Must check project update');

        projectStore.disconnectRepoManager('key1', 'repoName').subscribe(() => {
        });
        expect(call).toBe(4, 'Need to have done 4 http call');

        let deleteRepoCheck = false;
        projectStore.getProjects('key1').first().subscribe(projs => {
            deleteRepoCheck = true;
            expect(projs.get('key1').last_modified).toBe(789);
            expect(projs.get('key1').repositories_manager.length).toBe(0);
        });
        expect(deleteRepoCheck).toBeTruthy('Must check project update');
    }));

    it('should add/update/delete a variable', async(() => {
        let call = 0;
        // Mock Http project request
        backend.connections.subscribe(connection => {
            call++;
            switch (call) {
                case 1: // create project
                    connection.mockRespond(new Response(new ResponseOptions({
                        body: `{
                        "key": "key1",
                        "name": "myProject",
                        "last_modified": 0,
                        "variables" : []
                        }`
                    })));
                    break;
                case 2: // add variable
                    connection.mockRespond(new Response(new ResponseOptions({
                        body: `{
                       "key": "key1",
                        "name": "myProject",
                        "last_modified": 123,
                        "variables" : [ { \"name\" : \"myvar\"}]
                    }`
                    })));
                    break;
                case 3: // Update variable
                    connection.mockRespond(new Response(new ResponseOptions({body: `{ \"name\" : \"myvarUpdate\"}`})));
                    break;
                case 4: // Delete
                    connection.mockRespond(new Response(new ResponseOptions({})));
                    break;
            }


        });

        // Get project in cache
        expect(call).toBe(0, 'Need to have done 0 http call');
        projectStore.getProjects('key1').first().subscribe(() => {
        });
        expect(call).toBe(1, 'Need to have done 1 http call');


        // Add variable
        let v: Variable = new Variable();
        v.name = 'var1';
        v.type = 'string';
        v.value = 'value1';
        v.description = 'myDescription';
        projectStore.addProjectVariable('key1', v).subscribe(() => {
        }).unsubscribe();
        expect(call).toBe(2, 'Need to have done 2 http call');

        let addVariableCheck = false;
        projectStore.getProjects('key1').first().subscribe(projs => {
            addVariableCheck = true;
            expect(projs.get('key1').last_modified).toBe(123, 'Project must have been updated');
            expect(projs.get('key1').variables.length).toBe(1, 'Project must have 1 variable');
            expect(projs.get('key1').variables[0].name).toBe('myvar');
        });
        expect(addVariableCheck).toBeTruthy('Must check project update');


        // update variable
        expect(call).toBe(2, 'Need to have done 2 http call');
        projectStore.updateProjectVariable('key1', v).subscribe(() => {
        }).unsubscribe();
        expect(call).toBe(3, 'Need to have done 3 http call');

        let updateVariableCheck = false;
        projectStore.getProjects('key1').first().subscribe(projs => {
            updateVariableCheck = true;
            expect(projs.get('key1').variables.length).toBe(1);
            expect(projs.get('key1').variables[0].name).toBe('myvarUpdate');
        });
        expect(updateVariableCheck).toBeTruthy('Must check project update');

        // Delete variable
        expect(call).toBe(3, 'Need to have done 4 http call');
        projectStore.deleteProjectVariable('key1', v).subscribe(() => {
        });
        expect(call).toBe(4, 'Need to have done 4 http call');

        let deleteVariableCheck = false;
        projectStore.getProjects('key1').first().subscribe(projs => {
            deleteVariableCheck = true;
            expect(projs.get('key1').variables.length).toBe(0);
        });
        expect(deleteVariableCheck).toBeTruthy('Must check project update');
    }));

    it('should add/update/delete a permission', async(() => {
        let call = 0;
        // Mock Http project request
        backend.connections.subscribe(connection => {
            call++;
            switch (call) {
                case 1: // create project
                    connection.mockRespond(new Response(new ResponseOptions({
                        body: `{
                        "key": "key1",
                        "name": "myProject",
                        "last_modified": 0,
                        "groups" : []
                        }`
                    })));
                    break;
                case 2: // add group
                    connection.mockRespond(new Response(new ResponseOptions({
                        body: `[ { \"permission\" : 7, \"group\" : { \"id\" : 1}}]`
                    })));
                    break;
                case 3: // Update group
                    connection.mockRespond(new Response(new ResponseOptions({body: `{ \"permission\" : 4, \"group\" : { \"id\" : 1}}`})));
                    break;
                case 4: // Delete
                    connection.mockRespond(new Response(new ResponseOptions({})));
                    break;
            }


        });

        // Get project in cache
        expect(call).toBe(0, 'Need to have done 0 http call');
        projectStore.getProjects('key1').first().subscribe(() => {
        });
        expect(call).toBe(1, 'Need to have done 1 http call');


        // Add groupPermission
        let gp: GroupPermission = new GroupPermission();
        gp.group = new Group();
        gp.group.name = 'mygroup';
        gp.permission = 7;
        projectStore.addProjectPermission('key1', gp).subscribe(() => {
        }).unsubscribe();
        expect(call).toBe(2, 'Need to have done 2 http call');

        let addPermissionCheck = false;
        projectStore.getProjects('key1').first().subscribe(projs => {
            addPermissionCheck = true;
            expect(projs.get('key1').groups.length).toBe(1, 'Project must have 1 variable');
            expect(projs.get('key1').groups[0].permission).toBe(7);
        });
        expect(addPermissionCheck).toBeTruthy('Must check project update');


        // update permission
        expect(call).toBe(2, 'Need to have done 2 http call');
        projectStore.updateProjectPermission('key1', gp).subscribe(() => {
        }).unsubscribe();
        expect(call).toBe(3, 'Need to have done 3 http call');

        let updatePermissionCheck = false;
        projectStore.getProjects('key1').first().subscribe(projs => {
            updatePermissionCheck = true;
            expect(projs.get('key1').groups.length).toBe(1);
            expect(projs.get('key1').groups[0].permission).toBe(4);
        });
        expect(updatePermissionCheck).toBeTruthy('Must check project update');

        // Delete group permission
        expect(call).toBe(3, 'Need to have done 4 http call');
        gp.group.id = 1;
        projectStore.removeProjectPermission('key1', gp).subscribe(() => {
        });
        expect(call).toBe(4, 'Need to have done 4 http call');

        let deletePermissionCheck = false;
        projectStore.getProjects('key1').first().subscribe(projs => {
            deletePermissionCheck = true;
            expect(projs.get('key1').groups.length).toBe(0);
        });
        expect(deletePermissionCheck).toBeTruthy('Must check project update');
    }));

    it('should add/update/delete an environment', async(() => {
        let call = 0;
        // Mock Http project request
        backend.connections.subscribe(connection => {
            call++;
            switch (call) {
                case 1: // create project
                    connection.mockRespond(new Response(new ResponseOptions({
                        body: `{
                        "key": "key1",
                        "name": "myProject",
                        "last_modified": 0,
                        "environments" : []
                        }`
                    })));
                    break;
                case 2: // add variable
                    connection.mockRespond(new Response(new ResponseOptions({
                        body: `{
                       "key": "key1",
                        "name": "myProject",
                        "last_modified": 123,
                        "environments" : [ { \"name\" : \"Production\"}]
                    }`
                    })));
                    break;
                case 3: // Update variable
                    connection.mockRespond(new Response(new ResponseOptions({
                        body: `{
                        "key": "key1",
                        "name": "myProject",
                        "last_modified": 456,
                        "environments" : [ { \"name\" : \"PreProduction\"}]
                        }`
                    })));
                    break;
                case 4: // Delete
                    connection.mockRespond(new Response(new ResponseOptions({
                        body: `{
                            "key": "key1",
                            "name": "myProject",
                            "last_modified": 789,
                             "environments" : []
                            }`
                    })));
                    break;
            }


        });

        // Get project in cache
        expect(call).toBe(0, 'Need to have done 0 http call');
        projectStore.getProjects('key1').first().subscribe(() => {
        });
        expect(call).toBe(1, 'Need to have done 1 http call');


        // Add evn
        let env: Environment = new Environment();
        env.name = 'Production';
        projectStore.addProjectEnvironment('key1', env).subscribe(() => {
        }).unsubscribe();
        expect(call).toBe(2, 'Need to have done 2 http call');

        let addEnvCheck = false;
        projectStore.getProjects('key1').first().subscribe(projs => {
            addEnvCheck = true;
            expect(projs.get('key1').last_modified).toBe(123, 'Project must have been updated');
            expect(projs.get('key1').environments.length).toBe(1, 'Project must have 1 variable');
            expect(projs.get('key1').environments[0].name).toBe('Production');
        });
        expect(addEnvCheck).toBeTruthy('Must check project update');


        // update env
        expect(call).toBe(2, 'Need to have done 2 http call');
        projectStore.renameProjectEnvironment('key1', env.name, env).subscribe(() => {
        }).unsubscribe();
        expect(call).toBe(3, 'Need to have done 3 http call');

        let renameEnvCheck = false;
        projectStore.getProjects('key1').first().subscribe(projs => {
            renameEnvCheck = true;
            expect(projs.get('key1').last_modified).toBe(456);
            expect(projs.get('key1').environments.length).toBe(1);
            expect(projs.get('key1').environments[0].name).toBe('PreProduction');
        });
        expect(renameEnvCheck).toBeTruthy('Must check project update');

        // Delete env
        expect(call).toBe(3, 'Need to have done 4 http call');
        projectStore.deleteProjectEnvironment('key1', env).subscribe(() => {
        });
        expect(call).toBe(4, 'Need to have done 4 http call');

        let deleteEnvCheck = false;
        projectStore.getProjects('key1').first().subscribe(projs => {
            deleteEnvCheck = true;
            expect(projs.get('key1').last_modified).toBe(789);
            expect(projs.get('key1').environments.length).toBe(0);
        });
        expect(deleteEnvCheck).toBeTruthy('Must check project delete env');
    }));

    it('should add/update/delete an environment variable', async(() => {
        let call = 0;
        // Mock Http project request
        backend.connections.subscribe(connection => {
            call++;
            switch (call) {
                case 1: // create project
                    connection.mockRespond(new Response(new ResponseOptions({
                        body: `{
                        "key": "key1",
                        "name": "myProject",
                        "last_modified": 0,
                        "environments" : [ { "name" : "prod", "variables": []}]
                        }`
                    })));
                    break;
                case 2: // add variable
                    connection.mockRespond(new Response(new ResponseOptions({
                        body: `{
                       "key": "key1",
                        "name": "myProject",
                        "last_modified": 123,
                        "environments" : [ { "name" : "prod", "variables": [{ "name": "foo" }]}]
                    }`
                    })));
                    break;
                case 3: // Update variable
                    connection.mockRespond(new Response(new ResponseOptions({
                        body: `{
                        "key": "key1",
                        "name": "myProject",
                        "last_modified": 456,
                        "environments" :[ { "name" : "prod", "variables": [{ "name": "fooUpdated" }]}]
                        }`
                    })));
                    break;
                case 4: // Delete
                    connection.mockRespond(new Response(new ResponseOptions({
                        body: `{
                            "key": "key1",
                            "name": "myProject",
                            "last_modified": 789,
                             "environments" : [ { "name" : "prod", "variables": []}]
                            }`
                    })));
                    break;
            }


        });

        // Get project in cache
        expect(call).toBe(0, 'Need to have done 0 http call');
        projectStore.getProjects('key1').first().subscribe(() => {
        });
        expect(call).toBe(1, 'Need to have done 1 http call');


        // Add env variable
        let v: Variable = new Variable();
        v.name = 'prod';
        projectStore.addEnvironmentVariable('key1', 'prod', v).subscribe(() => {
        }).unsubscribe();
        expect(call).toBe(2, 'Need to have done 2 http call');

        let addEnvCheck = false;
        projectStore.getProjects('key1').first().subscribe(projs => {
            addEnvCheck = true;
            expect(projs.get('key1').last_modified).toBe(123, 'Project must have been updated');
            expect(projs.get('key1').environments.length).toBe(1, 'Project must have 1 variable');
            expect(projs.get('key1').environments[0].variables.length).toBe(1);
            expect(projs.get('key1').environments[0].variables[0].name).toBe('foo');
        });
        expect(addEnvCheck).toBeTruthy('Must check project update');


        // update var
        expect(call).toBe(2, 'Need to have done 2 http call');
        projectStore.updateEnvironmentVariable('key1', 'prod', v).subscribe(() => {
        }).unsubscribe();
        expect(call).toBe(3, 'Need to have done 3 http call');

        let renameVarEnvCheck = false;
        projectStore.getProjects('key1').first().subscribe(projs => {
            renameVarEnvCheck = true;
            expect(projs.get('key1').last_modified).toBe(456);
            expect(projs.get('key1').environments.length).toBe(1);
            expect(projs.get('key1').environments[0].variables.length).toBe(1);
            expect(projs.get('key1').environments[0].variables[0].name).toBe('fooUpdated');
        });
        expect(renameVarEnvCheck).toBeTruthy('Must check project update');

        // Delete var
        expect(call).toBe(3, 'Need to have done 4 http call');
        projectStore.removeEnvironmentVariable('key1', 'prod', v).subscribe(() => {
        });
        expect(call).toBe(4, 'Need to have done 4 http call');

        let deleteVarCheck = false;
        projectStore.getProjects('key1').first().subscribe(projs => {
            deleteVarCheck = true;
            expect(projs.get('key1').last_modified).toBe(789);
            expect(projs.get('key1').environments.length).toBe(1);
            expect(projs.get('key1').environments[0].variables.length).toBe(0);
        });
        expect(deleteVarCheck).toBeTruthy('Must check project delete var');
    }));

    it('should add/update/delete an environment permission', async(() => {
        let call = 0;
        // Mock Http project request
        backend.connections.subscribe(connection => {
            call++;
            switch (call) {
                case 1: // create project
                    connection.mockRespond(new Response(new ResponseOptions({
                        body: `{
                        "key": "key1",
                        "name": "myProject",
                        "last_modified": 0,
                        "environments" : [ { "name" : "prod", "groups": []}]
                        }`
                    })));
                    break;
                case 2: // add permission
                    connection.mockRespond(new Response(new ResponseOptions({
                        body: `{ "name" : "prod", "groups": [{ "permission": 7, "group": { "name": "grp1" } }]}`
                    })));
                    break;
                case 3: // Update permission
                    connection.mockRespond(new Response(new ResponseOptions({
                        body: `{ "name" : "prod", "groups": [{ "permission": 4, "group": { "name": "grp1" } }]}`
                    })));
                    break;
                case 4: // Delete
                    connection.mockRespond(new Response(new ResponseOptions({
                        body: `{ "name" : "prod", "groups": []}`
                    })));
                    break;
            }


        });

        // Get project in cache
        expect(call).toBe(0, 'Need to have done 0 http call');
        projectStore.getProjects('key1').first().subscribe(() => {
        });
        expect(call).toBe(1, 'Need to have done 1 http call');


        // Add env permission
        let gpA: Array<GroupPermission> = new Array<GroupPermission>();
        let gp = new GroupPermission();
        gp.permission = 7;
        gp.group = new Group();
        gp.group.name = 'grp1';
        gpA.push(gp);
        projectStore.addEnvironmentPermission('key1', 'prod', gpA).subscribe(() => {
        }).unsubscribe();
        expect(call).toBe(2, 'Need to have done 2 http call');

        let addEnvCheck = false;
        projectStore.getProjects('key1').first().subscribe(projs => {
            addEnvCheck = true;
            expect(projs.get('key1').environments.length).toBe(1, 'Project must have 1 env');
            expect(projs.get('key1').environments[0].groups.length).toBe(1);
            expect(projs.get('key1').environments[0].groups[0].permission).toBe(7);
            expect(projs.get('key1').environments[0].groups[0].group.name).toBe('grp1');
        });
        expect(addEnvCheck).toBeTruthy('Must check env update');

        // update gp
        expect(call).toBe(2, 'Need to have done 2 http call');
        gp.permission = 4;
        projectStore.updateEnvironmentPermission('key1', 'prod', gp).subscribe(() => {
        }).unsubscribe();
        expect(call).toBe(3, 'Need to have done 3 http call');

        let renameVarEnvCheck = false;
        projectStore.getProjects('key1').first().subscribe(projs => {
            renameVarEnvCheck = true;
            expect(projs.get('key1').environments.length).toBe(1);
            expect(projs.get('key1').environments[0].groups.length).toBe(1);
            expect(projs.get('key1').environments[0].groups[0].permission).toBe(4);
        });
        expect(renameVarEnvCheck).toBeTruthy('Must check env update');

        // Delete gp
        expect(call).toBe(3, 'Need to have done 4 http call');
        projectStore.removeEnvironmentPermission('key1', 'prod', gp).subscribe(() => {
        });
        expect(call).toBe(4, 'Need to have done 4 http call');

        let deletePermCheck = false;
        projectStore.getProjects('key1').first().subscribe(projs => {
            deletePermCheck = true;
            expect(projs.get('key1').environments.length).toBe(1);
            expect(projs.get('key1').environments[0].groups.length).toBe(0);
        });
        expect(deletePermCheck).toBeTruthy('Must check env delete perm');
    }));


    function createProject(key: string, name: string): Project {
        let project: Project = new Project();
        project.name = name;
        project.key = key;
        return project;
    }
});
