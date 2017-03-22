/* tslint:disable:no-unused-variable */

import {TestBed, async, getTestBed, fakeAsync} from '@angular/core/testing';
import {APP_BASE_HREF} from '@angular/common';
import {MockBackend} from '@angular/http/testing';
import {Http, RequestOptions, Response, ResponseOptions} from '@angular/http';
import {Injector} from '@angular/core';
import {AppModule} from '../../app.module';
import {AuthentificationStore} from '../auth/authentification.store';
import {HttpService} from '../http-service.service';
import {RouterModule, Router} from '@angular/router';
import {ApplicationStore} from './application.store';
import {Application} from '../../model/application.model';
import {ToastService} from '../../shared/toast/ToastService';
import {RepositoryPoller} from '../../model/polling.model';
import {Pipeline} from '../../model/pipeline.model';
import {Hook} from '../../model/hook.model';
import {Project} from '../../model/project.model';
import {Variable} from '../../model/variable.model';
import {GroupPermission} from '../../model/group.model';
import {Trigger} from '../../model/trigger.model';
import {ApplyTemplateRequest} from '../../model/template.model';
import {ProjectStore} from '../project/project.store';

describe('CDS: application Store', () => {

    let injector: Injector;
    let backend: MockBackend;
    let applicationStore: ApplicationStore;


    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [],
            providers: [
                { provide: APP_BASE_HREF, useValue: '/' },
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
            imports : [
                AppModule,
                RouterModule
            ]
        });
        injector = getTestBed();
        backend = injector.get(MockBackend);
        applicationStore = injector.get(ApplicationStore);

    });

    afterEach(() => {
        injector = undefined;
        backend = undefined;
        applicationStore = undefined;
    });

    it('should create and delete an Application', fakeAsync( () => {
        let call = 0;
        let projectKey = 'key1';
        // Mock Http application request
        backend.connections.subscribe(connection => {
            switch (call) {
                case 0:
                    call ++;
                    connection.mockRespond(new Response(new ResponseOptions({body: '{ "name": "myApplication" }'})));
                    break;
                case 1:
                    call ++;
                    connection.mockRespond(new Response(new ResponseOptions({body: '{ "name": "myApplication2" }'})));
                    break;
                case 2:
                case 3:
                    call ++;
                    connection.mockRespond(new Response(new ResponseOptions({body: '{}'})));
                    break;
            }


        });
        // Create Get application
        let checkApplicationCreated = false;
        applicationStore.getApplicationResolver(projectKey, 'myApplication').subscribe( res => {
            expect(res.name).toBe('myApplication', 'Wrong application name');
            checkApplicationCreated = true;
        }).unsubscribe();
        expect(call).toBe(1, 'Need to have done 1 http call');

        // check get application (get from cache)
        let checkedSingleApplication = false;
        applicationStore.getApplications(projectKey, 'myApplication').subscribe( apps => {
            expect(apps.get(projectKey + '-myApplication').name).toBe('myApplication', 'Wrong application name. Must be myApplication');
            checkedSingleApplication = true;
        }).unsubscribe();
        expect(checkedSingleApplication).toBeTruthy('Need to get application myApplication');
        expect(call).toBe(1, 'Need to have done 1 http call');


        let checkedDeleteApp = false;
        applicationStore.deleteApplication(projectKey, 'myApplication2').subscribe( () => {});
        applicationStore.getApplications(projectKey, 'myApplication2').subscribe( () => {
            checkedDeleteApp = true;
        }).unsubscribe();
        // 1 call for update + 1 for get because app was deleted from cache
        expect(call).toBe(3, 'Need to have done 3 http call');

    }));

    it('should update the application', async( () => {
        let call = 0;
        let projectKey = 'key1';
        // Mock Http application request
        backend.connections.subscribe(connection => {
            switch (call) {
                case 0: // create application
                    call ++;
                    connection.mockRespond(new Response(new ResponseOptions({body: '{ "name": "myApplication" }'})));
                    break;
                case 1: // 1st update
                    call++;
                    connection.mockRespond(new Response(new ResponseOptions({body: '{ "name": "myApplicationUpdate1" }'})));
                    break;
            }


        });
        // Create application
        let p = createApplication('myApplication');
        applicationStore.getApplicationResolver(projectKey, p.name).subscribe( () => {});
        expect(call).toBe(1, 'Need to have done 1 http call');

        // Update
        p.name = 'myApplicationUpdate1';
        applicationStore.renameApplication(projectKey, 'myApplication', p.name).subscribe( () => {});
        expect(call).toBe(2, 'Need to have done 2 http call');

        // check get application
        let checkedApplication = false;
        applicationStore.getApplications(projectKey, 'myApplicationUpdate1').subscribe( apps => {
            expect(apps.get(projectKey + '-myApplicationUpdate1').name)
                .toBe('myApplicationUpdate1', 'Wrong application name. Must be myApplicationUpdate1');
            checkedApplication = true;
        }).unsubscribe();
        expect(checkedApplication).toBeTruthy('Need to get application myApplicationUpdate1');
        expect(call).toBe(2, 'Need to have done 2 http call');
    }));

    it('should attach then Detach a repository', async( () => {
        let call = 0;
        let projectKey = 'key1';
        // Mock Http application request
        backend.connections.subscribe(connection => {
            switch (call) {
                case 0: // create application
                    call ++;
                    connection.mockRespond(new Response(new ResponseOptions({body: '{ "name": "myApplication", "last_modified": 123 }'})));
                    break;
                case 1: // Connect repo
                    call++;
                    connection.mockRespond(new Response(new ResponseOptions({body: `{ 
                        "name": "myApplicationUpdate1", 
                        "last_modified": 456,
                        "repositories_manager": { "name": "repoman" }, 
                        "repository_fullname": "myrepo"  }`
                    })));
                    break;
                case 2: // Connect repo
                    call++;
                    connection.mockRespond(new Response(new ResponseOptions({body: `{ 
                        "name": "myApplicationUpdate1", 
                        "last_modified": 789,
                        "repositories_manager": { "name": "repoman" }, 
                        "repository_fullname": "myrepo"  }`
                    })));
                    break;
            }


        });
        // Create application
        let p = createApplication('myApplication');
        applicationStore.getApplicationResolver(projectKey, p.name).subscribe( () => {});
        expect(call).toBe(1, 'Need to have done 1 http call');

        applicationStore.connectRepository(projectKey, 'myApplication', 'repoman', 'myrepo').subscribe(() => {});
        expect(call).toBe(2, 'Need to have done 2 http call');

        // check get application
        let checkedAttached = false;
        applicationStore.getApplications(projectKey, 'myApplication').subscribe( apps => {
            expect(apps.get(projectKey + '-myApplication').repository_fullname).toBe('myrepo', 'Repo fullname must be set to "myrepo"');
            expect(apps.get(projectKey + '-myApplication').repositories_manager.name)
                .toBe('repoman', 'Repo manager must be set to "repoman"');
            checkedAttached = true;
        }).unsubscribe();
        expect(checkedAttached).toBeTruthy('Need application to be updated');
        expect(call).toBe(2, 'Need to have done 2 http call');

        applicationStore.removeRepository(projectKey, 'myApplication', 'repoman').subscribe(() => {});
        expect(call).toBe(3, 'Need to have done 3 http call');

        // check get application
        let checkedDettach = false;
        applicationStore.getApplications(projectKey, 'myApplication').subscribe( apps => {
            expect(apps.get(projectKey + '-myApplication').repository_fullname).toBeFalsy('Repo fullname must not be set');
            expect(apps.get(projectKey + '-myApplication').repositories_manager).toBeFalsy('Repo manager must not be set');
            checkedDettach = true;
        }).unsubscribe();
        expect(checkedDettach).toBeTruthy('Need application to be updated');
        expect(call).toBe(3, 'Need to have done 3 http call');


    }));

    it('should add then delete a poller', async( () => {
        let call = 0;
        let projectKey = 'key1';
        // Mock Http application request
        backend.connections.subscribe(connection => {
            switch (call) {
                case 0: // create application
                    call ++;
                    connection.mockRespond(new Response(new ResponseOptions({body: '{ "name": "myApplication", "last_modified": 123 }'})));
                    break;
                case 1: // Add repo
                    call++;
                    connection.mockRespond(new Response(new ResponseOptions({body: `{ 
                        "name": "myApplicationUpdate1", 
                        "last_modified": 456,
                        "pollers": [ {"name": "repoman"} ] }`
                    })));
                    break;
                case 2: // Delete repo
                    call++;
                    connection.mockRespond(new Response(new ResponseOptions({body: `{ 
                        "name": "myApplicationUpdate1", 
                        "last_modified": 789 
                        }`
                    })));
                    break;
            }


        });
        // Create application
        let a = createApplication('myApplication');
        applicationStore.getApplicationResolver(projectKey, a.name).subscribe( () => {});
        expect(call).toBe(1, 'Need to have done 1 http call');

        let p: RepositoryPoller = new RepositoryPoller();
        p.name = 'repoman';
        p.pipeline = new Pipeline();
        p.pipeline.name = 'pipName';

        applicationStore.addPoller(projectKey, 'myApplication', 'pipName', p).subscribe(() => {});
        expect(call).toBe(2, 'Need to have done 2 http call');

        // check get application
        let checkedAddPoller = false;
        applicationStore.getApplications(projectKey, 'myApplication').subscribe( apps => {
            expect(apps.get(projectKey + '-myApplication').last_modified).toBe(456, 'App lastModified date must have been updated');
            checkedAddPoller = true;
        }).unsubscribe();
        expect(checkedAddPoller).toBeTruthy('Need application to be updated');
        expect(call).toBe(2, 'Need to have done 2 http call');

        applicationStore.deletePoller(projectKey, 'myApplication', p).subscribe(() => {});
        expect(call).toBe(3, 'Need to have done 3 http call');

        // check get application
        let checkedDeletePoller = false;
        applicationStore.getApplications(projectKey, 'myApplication').subscribe( apps => {
            expect(apps.get(projectKey + '-myApplication').last_modified).toBe(789, 'App lastModified date must have been updated');
            expect(apps.get(projectKey + '-myApplication').pollers).toBeFalsy('App must not have pollers');
            checkedDeletePoller = true;
        }).unsubscribe();
        expect(checkedDeletePoller).toBeTruthy('Need application to be updated');
        expect(call).toBe(3, 'Need to have done 3 http call');


    }));

    it('should add/update/delete a variable', async( () => {
        let call = 0;
        // Mock Http application request
        backend.connections.subscribe(connection => {
            switch (call) {
                case 0: // create application
                    call ++;
                    connection.mockRespond(new Response(new ResponseOptions({body: '{ "name": "myApplication", "last_modified": 0 }'})));
                    break;
                case 1: // Add variable
                    call++;
                    connection.mockRespond(new Response(new ResponseOptions({body: `{ 
                        "name": "myApplication", 
                        "last_modified": 123,
                        "variables": [ {"name": "foo"} ] }`
                    })));
                    break;
                case 2: // update variable
                    call++;
                    connection.mockRespond(new Response(new ResponseOptions({body: `{ 
                         "name": "myApplication", 
                        "last_modified": 456,
                        "variables": [ {"name": "bar"} ] }`
                    })));
                    break;
                case 3: // delete variable
                    call++;
                    connection.mockRespond(new Response(new ResponseOptions({body: `{ 
                         "name": "myApplication", 
                        "last_modified": 789,
                        "variables": [ ] }`
                    })));
                    break;
            }


        });

        let proj: Project = new Project();
        proj.key = 'key1';

        // Create application
        let a = createApplication('myApplication');
        applicationStore.getApplicationResolver(proj.key, a.name).subscribe( () => {});
        expect(call).toBe(1, 'Need to have done 1 http call');

        let v: Variable = new Variable();
        v.name = 'toto';


        applicationStore.addVariable(proj.key, a.name, v).subscribe(() => {});
        expect(call).toBe(2, 'Need to have done 2 http call');

        // check get application
        let checkedAddVariable = false;
        applicationStore.getApplications(proj.key, 'myApplication').subscribe( apps => {
            expect(apps.get(proj.key + '-myApplication').last_modified).toBe(123, 'App lastModified date must have been updated');
            expect(apps.get(proj.key + '-myApplication').variables.length).toBe(1, 'A variable must have been added');
            expect(apps.get(proj.key + '-myApplication').variables[0].name).toBe('foo', 'Variable name must be foo');
            checkedAddVariable = true;
        }).unsubscribe();
        expect(checkedAddVariable).toBeTruthy('Need application to be updated');
        expect(call).toBe(2, 'Need to have done 2 http call');


        applicationStore.updateVariable(proj.key, a.name, v).subscribe(() => {});
        expect(call).toBe(3, 'Need to have done 3 http call');

        // check get application
        let checkedUpdateVariable = false;
        applicationStore.getApplications(proj.key, 'myApplication').subscribe( apps => {
            expect(apps.get(proj.key + '-myApplication').last_modified).toBe(456, 'App lastModified date must have been updated');
            expect(apps.get(proj.key + '-myApplication').variables.length).toBe(1, 'App must have 1 variables');
            expect(apps.get(proj.key + '-myApplication').variables[0].name).toBe('bar', 'Variable name must be bar');
            checkedUpdateVariable = true;
        }).unsubscribe();
        expect(checkedUpdateVariable).toBeTruthy('Need application to be updated');
        expect(call).toBe(3, 'Need to have done 3 http call');


        applicationStore.removeVariable(proj.key, a.name, v).subscribe(() => {});
        expect(call).toBe(4, 'Need to have done 4 http call');

        // check get application
        let checkedDeleteVariable = false;
        applicationStore.getApplications(proj.key, 'myApplication').subscribe( apps => {
            expect(apps.get(proj.key + '-myApplication').last_modified).toBe(789, 'App lastModified date must have been updated');
            expect(apps.get(proj.key + '-myApplication').variables.length).toBe(0, 'App must have 0 variable');
            checkedDeleteVariable = true;
        }).unsubscribe();
        expect(checkedDeleteVariable).toBeTruthy('Need application to be updated');
        expect(call).toBe(4, 'Need to have done 3 http call');


    }));

    it('should add/update/delete a permission', async( () => {
        let call = 0;
        // Mock Http application request
        backend.connections.subscribe(connection => {
            switch (call) {
                case 0: // create application
                    call ++;
                    connection.mockRespond(new Response(new ResponseOptions({body: '{ "name": "myApplication", "last_modified": 0 }'})));
                    break;
                case 1: // Add variable
                    call++;
                    connection.mockRespond(new Response(new ResponseOptions({body: `{ 
                        "name": "myApplication", 
                        "last_modified": 123,
                        "groups": [ {"permission": 7} ] }`
                    })));
                    break;
                case 2: // update variable
                    call++;
                    connection.mockRespond(new Response(new ResponseOptions({body: `{ 
                         "name": "myApplication", 
                        "last_modified": 456,
                        "groups": [ {"permission": 4} ] }`
                    })));
                    break;
                case 3: // delete variable
                    call++;
                    connection.mockRespond(new Response(new ResponseOptions({body: `{ 
                         "name": "myApplication", 
                        "last_modified": 789,
                        "groups": [ ] }`
                    })));
                    break;
            }


        });

        let proj: Project = new Project();
        proj.key = 'key1';

        // Create application
        let a = createApplication('myApplication');
        applicationStore.getApplicationResolver(proj.key, a.name).subscribe( () => {});
        expect(call).toBe(1, 'Need to have done 1 http call');

        let gp: GroupPermission = new GroupPermission();
        gp.permission = 0;


        applicationStore.addPermission(proj.key, a.name, gp).subscribe(() => {});
        expect(call).toBe(2, 'Need to have done 2 http call');

        // check get application
        let checkedAddPermission = false;
        applicationStore.getApplications(proj.key, 'myApplication').subscribe( apps => {
            expect(apps.get(proj.key + '-myApplication').last_modified).toBe(123, 'App lastModified date must have been updated');
            expect(apps.get(proj.key + '-myApplication').groups.length).toBe(1, 'A group must have been added');
            expect(apps.get(proj.key + '-myApplication').groups[0].permission).toBe(7, 'Permission must be 7');
            checkedAddPermission = true;
        }).unsubscribe();
        expect(checkedAddPermission).toBeTruthy('Need application to be updated');
        expect(call).toBe(2, 'Need to have done 2 http call');


        applicationStore.updatePermission(proj.key, a.name, gp).subscribe(() => {});
        expect(call).toBe(3, 'Need to have done 3 http call');

        // check get application
        let checkedUpdatePermission = false;
        applicationStore.getApplications(proj.key, 'myApplication').subscribe( apps => {
            expect(apps.get(proj.key + '-myApplication').last_modified).toBe(456, 'App lastModified date must have been updated');
            expect(apps.get(proj.key + '-myApplication').groups.length).toBe(1, 'App must have 1 group');
            expect(apps.get(proj.key + '-myApplication').groups[0].permission).toBe(4, 'Group permission must be 4');
            checkedUpdatePermission = true;
        }).unsubscribe();
        expect(checkedUpdatePermission).toBeTruthy('Need application to be updated');
        expect(call).toBe(3, 'Need to have done 3 http call');


        applicationStore.removePermission(proj.key, a.name, gp).subscribe(() => {});
        expect(call).toBe(4, 'Need to have done 4 http call');

        // check get application
        let checkedDeletePermission = false;
        applicationStore.getApplications(proj.key, 'myApplication').subscribe( apps => {
            expect(apps.get(proj.key + '-myApplication').last_modified).toBe(789, 'App lastModified date must have been updated');
            expect(apps.get(proj.key + '-myApplication').groups.length).toBe(0, 'App must have 0 group');
            checkedDeletePermission = true;
        }).unsubscribe();
        expect(checkedDeletePermission).toBeTruthy('Need application to be updated');
        expect(call).toBe(4, 'Need to have done 3 http call');


    }));

    it('should add/update/delete a trigger', async( () => {
        let call = 0;
        // Mock Http application request
        backend.connections.subscribe(connection => {
            switch (call) {
                case 0: // create application
                    call ++;
                    connection.mockRespond(new Response(new ResponseOptions({body: '{ "name": "myApplication", "last_modified": 0 }'})));
                    break;
                case 1: // Add trigger
                    call++;
                    connection.mockRespond(new Response(new ResponseOptions({body: `{ 
                        "name": "myApplication", 
                        "last_modified": 123,
                        "workflows": [ 
                            {
                                "subPipelines": [
                                    {
                                        "trigger": {"manual": true}
                                    }
                                ]
                            } 
                        ] 
                    }`
                    })));
                    break;
                case 2: // update trigger
                    call++;
                    connection.mockRespond(new Response(new ResponseOptions({body: `{ 
                         "name": "myApplication", 
                        "last_modified": 456,
                         "workflows": [ 
                            {
                                "subPipelines": [
                                    {
                                        "trigger": {"manual": false}
                                    }
                                ]
                            } 
                         ] 
                    }`
                    })));
                    break;
                case 3: // delete trigger
                    call++;
                    connection.mockRespond(new Response(new ResponseOptions({body: `{ 
                         "name": "myApplication", 
                        "last_modified": 789,
                        "workflows": [ 
                            {
                                "subPipelines": []
                            }, 
                            {
                                "subPipelines": []
                            } 
                         ]
                    }`
                    })));
                    break;
            }


        });


        let proj: Project = new Project();
        proj.key = 'key1';

        let pip: Pipeline = new Pipeline();
        pip.name = 'pip1';

        // Create application
        let a = createApplication('myApplication');
        applicationStore.getApplicationResolver(proj.key, a.name).subscribe( () => {});
        expect(call).toBe(1, 'Need to have done 1 http call');

        let t: Trigger = new Trigger();
        t.src_project = proj;
        t.src_application = a;
        t.src_pipeline = pip;
        t.manual = true;


        applicationStore.addTrigger(proj.key, a.name, pip.name, t).subscribe(() => {});
        expect(call).toBe(2, 'Need to have done 2 http call');

        // check get application
        let checkedAddTrigger = false;
        applicationStore.getApplications(proj.key, 'myApplication').subscribe( apps => {
            expect(apps.get(proj.key + '-myApplication').last_modified).toBe(123, 'App lastModified date must have been updated');
            expect(apps.get(proj.key + '-myApplication').workflows.length).toBe(1, 'Must have 1 workflow');
            expect(apps.get(proj.key + '-myApplication').workflows[0].subPipelines.length).toBe(1, 'Root item must have 1 child');
            expect(apps.get(proj.key + '-myApplication').workflows[0].subPipelines[0].trigger.manual)
                .toBeTruthy('Child must have 1 manual trigger');
            checkedAddTrigger = true;
        }).unsubscribe();
        expect(checkedAddTrigger).toBeTruthy('Need application to be updated');
        expect(call).toBe(2, 'Need to have done 2 http call');


        applicationStore.updateTrigger(proj.key, a.name, pip.name, t).subscribe(() => {});
        expect(call).toBe(3, 'Need to have done 3 http call');

        // check get application
        let checkedUpdateTrigger = false;
        applicationStore.getApplications(proj.key, 'myApplication').subscribe( apps => {
            expect(apps.get(proj.key + '-myApplication').last_modified).toBe(456, 'App lastModified date must have been updated');
            expect(apps.get(proj.key + '-myApplication').workflows.length).toBe(1, 'Must have 1 workflow');
            expect(apps.get(proj.key + '-myApplication').workflows[0].subPipelines.length).toBe(1, 'Root item must have 1 child');
            expect(apps.get(proj.key + '-myApplication').workflows[0].subPipelines[0].trigger.manual)
                .toBeFalsy('Child must have 1 auto trigger');
            checkedUpdateTrigger = true;
        }).unsubscribe();
        expect(checkedUpdateTrigger).toBeTruthy('Need application to be updated');
        expect(call).toBe(3, 'Need to have done 3 http call');


        applicationStore.removeTrigger(proj.key, a.name, pip.name, t).subscribe(() => {});
        expect(call).toBe(4, 'Need to have done 4 http call');

        // check get application
        let checkedDeleteTrigger = false;
        applicationStore.getApplications(proj.key, 'myApplication').subscribe( apps => {
            expect(apps.get(proj.key + '-myApplication').last_modified).toBe(789, 'App lastModified date must have been updated');
            expect(apps.get(proj.key + '-myApplication').workflows.length).toBe(2, 'Must have 1 workflow');
            checkedDeleteTrigger = true;
        }).unsubscribe();
        expect(checkedDeleteTrigger).toBeTruthy('Need application to be updated');
        expect(call).toBe(4, 'Need to have done 3 http call');


    }));

    it('should create application from template', async( () => {
        let call = 0;
        // Mock Http application request
        backend.connections.subscribe(connection => {
            switch (call) {
                case 0: // create project
                    call ++;
                    connection.mockRespond(new Response(new ResponseOptions({body: '{ "name": "proj", "key": "proj1" }'})));
                    break;
                case 1: // Create application
                    call++;
                    connection.mockRespond(new Response(new ResponseOptions({body: `{ 
                        "name": "proj", 
                        "key": "proj1",
                        "last_modified": 123,
                        "applications": [ 
                            {
                                "name": "app1"
                            }
                        ] 
                    }`
                    })));
                    break;
            }


        });


        let project: Project = new Project();
        let projectStore: ProjectStore = injector.get(ProjectStore);
        projectStore.getProjects('proj1').subscribe().unsubscribe();
        expect(call).toBe(1, 'Need to have done 1 http call');
        applicationStore.applyTemplate('proj1', new ApplyTemplateRequest()).subscribe().unsubscribe();
        expect(call).toBe(2, 'Need to have done 2 http call');


        let checked = false;
        projectStore.getProjectResolver('proj1').subscribe(p => {
            checked = true;
            expect(p.last_modified).toBe(123);
            expect(p.applications.length).toBe(1);
            expect(p.applications[0].name).toBe('app1');
        });
        expect(checked).toBeTruthy();

    }));

    function createApplication(name: string): Application {
        let app: Application = new Application();
        app.name = name;
        return app;
    }
});
