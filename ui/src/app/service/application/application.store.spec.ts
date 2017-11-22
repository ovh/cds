/* tslint:disable:no-unused-variable */

import {async, fakeAsync, TestBed} from '@angular/core/testing';
import {APP_BASE_HREF} from '@angular/common';
import {AppModule} from '../../app.module';
import {RouterModule} from '@angular/router';
import {ApplicationStore} from './application.store';
import {Application} from '../../model/application.model';
import {RepositoryPoller} from '../../model/polling.model';
import {Pipeline} from '../../model/pipeline.model';
import {Project} from '../../model/project.model';
import {Variable} from '../../model/variable.model';
import {Group, GroupPermission} from '../../model/group.model';
import {HttpClientTestingModule, HttpTestingController} from '@angular/common/http/testing';
import {RepositoriesManager} from '../../model/repositories.model';
import {HttpRequest} from '@angular/common/http';
import {WorkflowItem} from '../../model/application.workflow.model';
import {Trigger} from '../../model/trigger.model';
import {ProjectStore} from '../project/project.store';
import {ApplyTemplateRequest} from '../../model/template.model';
import {first} from 'rxjs/operators';

describe('CDS: application Store', () => {

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [],
            providers: [
                {provide: APP_BASE_HREF, useValue: '/'},
            ],
            imports: [
                AppModule,
                RouterModule,
                HttpClientTestingModule
            ]
        });

    });

    it('should create and delete an Application', fakeAsync(() => {
        const applicationStore = TestBed.get(ApplicationStore);
        const http = TestBed.get(HttpTestingController);

        let app1 = new Application();
        app1.name = 'myApplication';

        let app2 = new Application();
        app2.name = 'myApplication2';


        // Create Get application
        let checkApplicationCreated = false;
        applicationStore.getApplicationResolver('key1', 'myApplication').subscribe(res => {
            expect(res.name).toBe('myApplication', 'Wrong application name');
            checkApplicationCreated = true;
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/application/myApplication'
        })).flush(app1);
        expect(checkApplicationCreated).toBeTruthy();

        // check get application (get from cache)
        let checkedSingleApplication = false;
        applicationStore.getApplications('key1', 'myApplication').subscribe(apps => {
            expect(apps.get('key1' + '-myApplication').name).toBe('myApplication', 'Wrong application name. Must be myApplication');
            checkedSingleApplication = true;
        }).unsubscribe();
        expect(checkedSingleApplication).toBeTruthy('Need to get application myApplication');


        let checkedDeleteApp = false;
        applicationStore.deleteApplication('key1', 'myApplication2').subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/application/myApplication2'
        })).flush(null);
        applicationStore.getApplications('key1', 'myApplication2').subscribe(() => {
            checkedDeleteApp = true;
        }).unsubscribe();
        expect(checkedDeleteApp).toBeTruthy();
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/application/myApplication2'
        })).flush(app2);

        http.verify();
    }));

    it('should update the application', async(() => {
        const applicationStore = TestBed.get(ApplicationStore);
        const http = TestBed.get(HttpTestingController);

        let application = new Application();
        application.name = 'myApplication';

        let applicationU = new Application();
        applicationU.name = 'myApplicationUpdate1';

        let projectKey = 'key1';


        // Create application
        let p = createApplication('myApplication');
        applicationStore.getApplicationResolver(projectKey, p.name).subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/application/myApplication'
        })).flush(application);

        // Update
        p.name = 'myApplicationUpdate1';
        applicationStore.renameApplication(projectKey, 'myApplication', p.name).subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/application/myApplication'
        })).flush(applicationU);
        // check get application
        let checkedApplication = false;
        applicationStore.getApplications(projectKey, 'myApplicationUpdate1').subscribe(apps => {
            expect(apps.get(projectKey + '-myApplicationUpdate1').name)
                .toBe('myApplicationUpdate1', 'Wrong application name. Must be myApplicationUpdate1');
            checkedApplication = true;
        }).unsubscribe();
        expect(checkedApplication).toBeTruthy('Need to get application myApplicationUpdate1');

        http.verify();
    }));

    it('should attach then Detach a repository', async(() => {
        const applicationStore = TestBed.get(ApplicationStore);
        const http = TestBed.get(HttpTestingController);

        let application = new Application();
        application.name = 'myApplication';
        application.last_modified = '123';

        let applicationC = new Application();
        applicationC.name = 'myApplicationUpdate1';
        applicationC.last_modified = '456';
        applicationC.vcs_server = 'repoman';
        applicationC.repository_fullname = 'myrepo';

        let applicationLast = new Application();
        applicationLast.name = 'myApplicationUpdate1';
        applicationLast.last_modified = '789';

        let projectKey = 'key1';

        // Create application
        let p = createApplication('myApplication');
        applicationStore.getApplicationResolver(projectKey, p.name).subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/application/myApplication'
        })).flush(application);

        applicationStore.connectRepository(projectKey, 'myApplication', 'repoman', 'myrepo').subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/repositories_manager/repoman/application/myApplication/attach'
        })).flush(applicationC);

        // check get application
        let checkedAttached = false;
        applicationStore.getApplications(projectKey, 'myApplication').subscribe(apps => {
            expect(apps.get(projectKey + '-myApplication').repository_fullname).toBe('myrepo', 'Repo fullname must be set to "myrepo"');
            expect(apps.get(projectKey + '-myApplication').vcs_server)
                .toBe('repoman', 'Repo manager must be set to "repoman"');
            checkedAttached = true;
        }).unsubscribe();
        expect(checkedAttached).toBeTruthy('Need application to be updated');

        applicationStore.removeRepository(projectKey, 'myApplication', 'repoman').subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/repositories_manager/repoman/application/myApplication/detach'
        })).flush(applicationLast);

        // check get application
        let checkedDettach = false;
        applicationStore.getApplications(projectKey, 'myApplication').subscribe(apps => {
            expect(apps.get(projectKey + '-myApplication').repository_fullname).toBeFalsy('Repo fullname must not be set');
            expect(apps.get(projectKey + '-myApplication').repositories_manager).toBeFalsy('Repo manager must not be set');
            checkedDettach = true;
        }).unsubscribe();
        expect(checkedDettach).toBeTruthy('Need application to be updated');

        http.verify();
    }));

    it('should add then delete a poller', async(() => {
        const applicationStore = TestBed.get(ApplicationStore);
        const http = TestBed.get(HttpTestingController);

        let app1 = new Application();
        app1.name = 'myApplication';
        app1.last_modified = '123';

        let appWithPoller = new Application();
        appWithPoller.name = 'myApplicationUpdate1';
        appWithPoller.last_modified = '456';
        appWithPoller.pollers = new Array<RepositoryPoller>();
        let poller = new RepositoryPoller();
        poller.name = 'repoman';
        appWithPoller.pollers.push(poller);

        let appFinal = new Application();
        appFinal.name = 'myApplicationUpdate1';
        appFinal.last_modified = '789';

        let projectKey = 'key1';

        // Create application
        let a = createApplication('myApplication');
        applicationStore.getApplicationResolver(projectKey, a.name).subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/application/myApplication'
        })).flush(app1);

        let p: RepositoryPoller = new RepositoryPoller();
        p.name = 'repoman';
        p.pipeline = new Pipeline();
        p.pipeline.name = 'pipName';

        applicationStore.addPoller(projectKey, 'myApplication', 'pipName', p).subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/application/myApplication/pipeline/pipName/polling'
        })).flush(appWithPoller);

        // check get application
        let checkedAddPoller = false;
        applicationStore.getApplications(projectKey, 'myApplication').subscribe(apps => {
            expect(apps.get(projectKey + '-myApplication').last_modified).toBe('456', 'App lastModified date must have been updated');
            checkedAddPoller = true;
        }).unsubscribe();
        expect(checkedAddPoller).toBeTruthy('Need application to be updated');

        applicationStore.deletePoller(projectKey, 'myApplication', p).subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/application/myApplication/pipeline/pipName/polling'
        })).flush(appFinal);

        // check get application
        let checkedDeletePoller = false;
        applicationStore.getApplications(projectKey, 'myApplication').subscribe(apps => {
            expect(apps.get(projectKey + '-myApplication').last_modified).toBe('789', 'App lastModified date must have been updated');
            expect(apps.get(projectKey + '-myApplication').pollers).toBeFalsy('App must not have pollers');
            checkedDeletePoller = true;
        }).unsubscribe();
        expect(checkedDeletePoller).toBeTruthy('Need application to be updated');

        http.verify();
    }));


    it('should add/update/delete a variable', async(() => {
        const applicationStore = TestBed.get(ApplicationStore);
        const http = TestBed.get(HttpTestingController);

        let app = new Application();
        app.name = 'myApplication';
        app.last_modified = '0';

        let appAddVar = new Application();
        appAddVar.name = 'myApplication';
        appAddVar.last_modified = '123';
        appAddVar.variables = new Array<Variable>();
        let v1 = new Variable();
        v1.name = 'foo';
        appAddVar.variables.push(v1);

        let appUpVar = new Application();
        appUpVar.name = 'myApplication';
        appUpVar.last_modified = '456';
        appUpVar.variables = new Array<Variable>();
        let v2 = new Variable();
        v2.name = 'bar';
        appUpVar.variables.push(v2);

        let appDelVar = new Application();
        appDelVar.name = 'myApplication';
        appDelVar.last_modified = '789';
        appDelVar.variables = new Array<Variable>();


        let proj: Project = new Project();
        proj.key = 'key1';

        // Create application
        let a = createApplication('myApplication');
        applicationStore.getApplicationResolver(proj.key, a.name).subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/application/myApplication'
        })).flush(app);

        let v: Variable = new Variable();
        v.name = 'foo';


        applicationStore.addVariable(proj.key, a.name, v).subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/application/myApplication/variable/foo'
        })).flush(appAddVar);

        // check get application
        let checkedAddVariable = false;
        applicationStore.getApplications(proj.key, 'myApplication').subscribe(apps => {
            expect(apps.get(proj.key + '-myApplication').last_modified).toBe('123', 'App lastModified date must have been updated');
            expect(apps.get(proj.key + '-myApplication').variables.length).toBe(1, 'A variable must have been added');
            expect(apps.get(proj.key + '-myApplication').variables[0].name).toBe('foo', 'Variable name must be foo');
            checkedAddVariable = true;
        }).unsubscribe();
        expect(checkedAddVariable).toBeTruthy('Need application to be updated');


        applicationStore.updateVariable(proj.key, a.name, v).subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/application/myApplication/variable/foo'
        })).flush(appUpVar);

        // check get application
        let checkedUpdateVariable = false;
        applicationStore.getApplications(proj.key, 'myApplication').subscribe(apps => {
            expect(apps.get(proj.key + '-myApplication').last_modified).toBe('456', 'App lastModified date must have been updated');
            expect(apps.get(proj.key + '-myApplication').variables.length).toBe(1, 'App must have 1 variables');
            expect(apps.get(proj.key + '-myApplication').variables[0].name).toBe('bar', 'Variable name must be bar');
            checkedUpdateVariable = true;
        }).unsubscribe();
        expect(checkedUpdateVariable).toBeTruthy('Need application to be updated');


        applicationStore.removeVariable(proj.key, a.name, v).subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/application/myApplication/variable/foo'
        })).flush(appDelVar);

        // check get application
        let checkedDeleteVariable = false;
        applicationStore.getApplications(proj.key, 'myApplication').subscribe(apps => {
            expect(apps.get(proj.key + '-myApplication').last_modified).toBe('789', 'App lastModified date must have been updated');
            expect(apps.get(proj.key + '-myApplication').variables.length).toBe(0, 'App must have 0 variable');
            checkedDeleteVariable = true;
        }).unsubscribe();
        expect(checkedDeleteVariable).toBeTruthy('Need application to be updated');

        http.verify();
    }));

    it('should add/update/delete a permission', async(() => {
        const applicationStore = TestBed.get(ApplicationStore);
        const http = TestBed.get(HttpTestingController);

        let app = new Application();
        app.name = 'myApplication';
        app.last_modified = '0';

        let appAddPerm = new Application();
        appAddPerm.name = 'myApplication';
        appAddPerm.last_modified = '123';
        appAddPerm.groups = new Array<GroupPermission>();
        let gp1 = new GroupPermission();
        gp1.permission = 7;
        gp1.group = new Group();
        gp1.group.name = 'grp';
        appAddPerm.groups.push(gp1);

        let appUpPerm = new Application();
        appUpPerm.name = 'myApplication';
        appUpPerm.last_modified = '456';
        appUpPerm.groups = new Array<GroupPermission>();
        let gp2 = new GroupPermission();
        gp2.permission = 4;
        gp2.group = new Group();
        gp2.group.name = 'grp';
        appUpPerm.groups.push(gp2);

        let appDelPerm = new Application();
        appDelPerm.name = 'myApplication';
        appDelPerm.last_modified = '789';
        appDelPerm.groups = new Array<GroupPermission>();

        let proj: Project = new Project();
        proj.key = 'key1';

        // Create application
        let a = createApplication('myApplication');
        applicationStore.getApplicationResolver(proj.key, a.name).subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/application/myApplication'
        })).flush(app);

        let gp: GroupPermission = new GroupPermission();
        gp.group = new Group();
        gp.group.name = 'grp';
        gp.permission = 0;

        applicationStore.addPermission(proj.key, a.name, gp).subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/application/myApplication/group'
        })).flush(appAddPerm);

        // check get application
        let checkedAddPermission = false;
        applicationStore.getApplications(proj.key, 'myApplication').subscribe(apps => {
            expect(apps.get(proj.key + '-myApplication').last_modified).toBe('123', 'App lastModified date must have been updated');
            expect(apps.get(proj.key + '-myApplication').groups.length).toBe(1, 'A group must have been added');
            expect(apps.get(proj.key + '-myApplication').groups[0].permission).toBe(7, 'Permission must be 7');
            checkedAddPermission = true;
        }).unsubscribe();
        expect(checkedAddPermission).toBeTruthy('Need application to be updated');

        applicationStore.updatePermission(proj.key, a.name, gp).subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/application/myApplication/group/grp'
        })).flush(appUpPerm);

        // check get application
        let checkedUpdatePermission = false;
        applicationStore.getApplications(proj.key, 'myApplication').subscribe(apps => {
            expect(apps.get(proj.key + '-myApplication').last_modified).toBe('456', 'App lastModified date must have been updated');
            expect(apps.get(proj.key + '-myApplication').groups.length).toBe(1, 'App must have 1 group');
            expect(apps.get(proj.key + '-myApplication').groups[0].permission).toBe(4, 'Group permission must be 4');
            checkedUpdatePermission = true;
        }).unsubscribe();
        expect(checkedUpdatePermission).toBeTruthy('Need application to be updated');

        applicationStore.removePermission(proj.key, a.name, gp).subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/application/myApplication/group/grp'
        })).flush(appDelPerm);

        // check get application
        let checkedDeletePermission = false;
        applicationStore.getApplications(proj.key, 'myApplication').subscribe(apps => {
            expect(apps.get(proj.key + '-myApplication').last_modified).toBe('789', 'App lastModified date must have been updated');
            expect(apps.get(proj.key + '-myApplication').groups.length).toBe(0, 'App must have 0 group');
            checkedDeletePermission = true;
        }).unsubscribe();
        expect(checkedDeletePermission).toBeTruthy('Need application to be updated');

        http.verify();
    }));

    it('should add/update/delete a trigger', async(() => {
        const applicationStore = TestBed.get(ApplicationStore);
        const http = TestBed.get(HttpTestingController);

        let app = new Application();
        app.name = 'myApplication';
        app.last_modified = '0';

        let appAddTrig = new Application();
        appAddTrig.name = 'myApplication';
        appAddTrig.last_modified = '123';
        appAddTrig.workflows = new Array<WorkflowItem>();
        let wi = new WorkflowItem();
        wi.subPipelines = new Array<WorkflowItem>();
        let sub1 = new WorkflowItem();
        sub1.trigger = new Trigger();
        sub1.trigger.manual = true;
        wi.subPipelines.push(sub1);
        appAddTrig.workflows.push(wi);

        let appUpTrig = new Application();
        appUpTrig.name = 'myApplication';
        appUpTrig.last_modified = '456';
        appUpTrig.workflows = new Array<WorkflowItem>();
        let wi2 = new WorkflowItem();
        wi2.subPipelines = new Array<WorkflowItem>();
        let sub2 = new WorkflowItem();
        sub2.trigger = new Trigger();
        sub2.trigger.manual = false;
        wi2.subPipelines.push(sub2);
        appUpTrig.workflows.push(wi2);

        let appDelTrig = new Application();
        appDelTrig.name = 'myApplication';
        appDelTrig.last_modified = '789';
        appDelTrig.workflows = new Array<WorkflowItem>();

        let proj: Project = new Project();
        proj.key = 'key1';

        let pip: Pipeline = new Pipeline();
        pip.name = 'pip1';

        // Create application
        let a = createApplication('myApplication');
        applicationStore.getApplicationResolver(proj.key, a.name).subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/application/myApplication'
        })).flush(app);

        let t: Trigger = new Trigger();
        t.src_project = proj;
        t.src_application = a;
        t.src_pipeline = pip;
        t.manual = true;


        applicationStore.addTrigger(proj.key, a.name, pip.name, t).subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/application/myApplication/pipeline/pip1/trigger'
        })).flush(appAddTrig);

        // check get application
        let checkedAddTrigger = false;
        applicationStore.getApplications(proj.key, 'myApplication').pipe(first()).subscribe(apps => {
            expect(apps.get(proj.key + '-myApplication').last_modified).toBe('123', 'App lastModified date must have been updated');
            expect(apps.get(proj.key + '-myApplication').workflows.length).toBe(1, 'Must have 1 workflow');
            expect(apps.get(proj.key + '-myApplication').workflows[0].subPipelines.length).toBe(1, 'Root item must have 1 child');
            expect(apps.get(proj.key + '-myApplication').workflows[0].subPipelines[0].trigger.manual)
                .toBeTruthy('Child must have 1 manual trigger');
            checkedAddTrigger = true;
        });
        expect(checkedAddTrigger).toBeTruthy('Need trigger to be added');


        applicationStore.updateTrigger(proj.key, a.name, pip.name, t).subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/application/myApplication/pipeline/pip1/trigger/0';
        })).flush(appUpTrig);

        // check get application
        let checkedUpdateTrigger = false;
        applicationStore.getApplications(proj.key, 'myApplication').pipe(first()).subscribe(apps => {
            expect(apps.get(proj.key + '-myApplication').last_modified).toBe('456', 'App lastModified date must have been updated');
            expect(apps.get(proj.key + '-myApplication').workflows.length).toBe(1, 'Must have 1 workflow');
            expect(apps.get(proj.key + '-myApplication').workflows[0].subPipelines.length).toBe(1, 'Root item must have 1 child');
            expect(apps.get(proj.key + '-myApplication').workflows[0].subPipelines[0].trigger.manual)
                .toBeFalsy('Child must have 1 auto trigger');
            checkedUpdateTrigger = true;
        });
        expect(checkedUpdateTrigger).toBeTruthy('Need application to be updated');

        applicationStore.removeTrigger(proj.key, a.name, pip.name, t).subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/application/myApplication/pipeline/pip1/trigger/0';
        })).flush(appDelTrig);

        // check get application
        let checkedDeleteTrigger = false;
        applicationStore.getApplications(proj.key, 'myApplication').pipe(first()).subscribe(apps => {
            expect(apps.get(proj.key + '-myApplication').last_modified).toBe('789', 'App lastModified date must have been updated');
            expect(apps.get(proj.key + '-myApplication').workflows.length).toBe(0, 'Must have 0 workflow');
            checkedDeleteTrigger = true;
        });
        expect(checkedDeleteTrigger).toBeTruthy('Need application to be updated');

        http.verify();
    }));

    it('should create application from template', async(() => {
        const projectStore = TestBed.get(ProjectStore);
        const applicationStore = TestBed.get(ApplicationStore);
        const http = TestBed.get(HttpTestingController);

        let project = new Project();
        project.name = 'proj';
        project.key = 'proj1';

        let projectUp = new Project();
        projectUp.name = 'proj';
        projectUp.key = 'proj1';
        projectUp.last_modified = '123';
        projectUp.applications = new Array<Application>();
        let app = new Application();
        app.name = 'app1';
        projectUp.applications.push(app);

        projectStore.getProjects('proj1').subscribe(() => {});
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/proj1';
        })).flush(project);


        applicationStore.applyTemplate('proj1', new ApplyTemplateRequest()).subscribe(() => {});
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/proj1/template';
        })).flush(projectUp);

        let checked = false;
        projectStore.getProjectResolver('proj1').subscribe(p => {
            checked = true;
            expect(p.last_modified).toBe('123');
            expect(p.applications.length).toBe(1);
            expect(p.applications[0].name).toBe('app1');
        });
        expect(checked).toBeTruthy();

        http.verify();
    }));

    function createApplication(name: string): Application {
        let app: Application = new Application();
        app.name = name;
        return app;
    }
});
