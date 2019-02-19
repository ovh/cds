import { HttpRequest } from '@angular/common/http';
import { HttpClientTestingModule, HttpTestingController } from '@angular/common/http/testing';
import { async, TestBed } from '@angular/core/testing';
import { NgxsModule, Store } from '@ngxs/store';
import { Application, Overview } from 'app/model/application.model';
import { IntegrationModel, ProjectIntegration } from 'app/model/integration.model';
import { Key } from 'app/model/keys.model';
import { Variable } from 'app/model/variable.model';
import * as ActionApplication from './applications.action';
import { ApplicationsState } from './applications.state';

describe('Applications', () => {
    let store: Store;

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            imports: [
                NgxsModule.forRoot([ApplicationsState]),
                HttpClientTestingModule
            ],
        }).compileComponents();

        store = TestBed.get(Store);
        // store.reset(getInitialApplicationsState());
    }));

    it('fetch application', async(() => {
        const http = TestBed.get(HttpTestingController);
        store.dispatch(new ActionApplication.FetchApplication({
            projectKey: 'test1',
            applicationName: 'app1'
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/application/app1';
        })).flush({
            name: 'app1',
            project_key: 'test1',
            vcs_strategy: {}
        });
        store.selectOnce(ApplicationsState).subscribe(state => {
            expect(Object.keys(state.applications).length).toEqual(1);
        });
        store.selectOnce(ApplicationsState.selectApplication('test1', 'app1')).subscribe(app => {
            expect(app).toBeTruthy();
            expect(app.name).toEqual('app1');
            expect(app.project_key).toEqual('test1');
        });
    }));

    it('add application', async(() => {
        const http = TestBed.get(HttpTestingController);
        let application = new Application();
        application.name = 'app1';
        application.project_key = 'test1';
        store.dispatch(new ActionApplication.AddApplication({
            projectKey: 'test1',
            application
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/applications';
        })).flush({
            name: 'app1',
            project_key: 'test1',
            vcs_strategy: {}
        });
        store.selectOnce(ApplicationsState).subscribe(state => {
            expect(Object.keys(state.applications).length).toEqual(1);
        });
        store.selectOnce(ApplicationsState.selectApplication('test1', 'app1')).subscribe(app => {
            expect(app).toBeTruthy();
            expect(app.name).toEqual('app1');
            expect(app.project_key).toEqual('test1');
        });

        store.dispatch(new ActionApplication.FetchApplication({
            projectKey: 'test1',
            applicationName: 'app1'
        }));
        store.selectOnce(ApplicationsState).subscribe(state => {
            expect(Object.keys(state.applications).length).toEqual(1);
        });
        store.selectOnce(ApplicationsState.selectApplication('test1', 'app1')).subscribe((app: Application) => {
            expect(app).toBeTruthy();
            expect(app.name).toEqual('app1');
            expect(app.project_key).toEqual('test1');
        });
    }));

    it('update an application', async(() => {
        const http = TestBed.get(HttpTestingController);
        let application = new Application();
        application.name = 'app1';
        application.project_key = 'test1';
        store.dispatch(new ActionApplication.AddApplication({
            projectKey: 'test1',
            application
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/applications';
        })).flush({
            name: 'app1',
            project_key: 'test1',
            vcs_strategy: {}
        });
        store.selectOnce(ApplicationsState).subscribe(state => {
            expect(Object.keys(state.applications).length).toEqual(1);
        });
        store.selectOnce(ApplicationsState.selectApplication('test1', 'app1')).subscribe(app => {
            expect(app).toBeTruthy();
            expect(app.name).toEqual('app1');
            expect(app.project_key).toEqual('test1');
        });

        application.name = 'app1bis';
        store.dispatch(new ActionApplication.UpdateApplication({
            projectKey: 'test1',
            applicationName: 'app1',
            changes: application
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/application/app1';
        })).flush({
            name: 'app1bis',
            project_key: 'test1',
            vcs_strategy: {}
        });
        store.selectOnce(ApplicationsState).subscribe(state => {
            expect(Object.keys(state.applications).length).toEqual(1);
        });
        store.selectOnce(ApplicationsState.selectApplication('test1', 'app1bis')).subscribe(app => {
            expect(app).toBeTruthy();
            expect(app.name).toEqual('app1bis');
            expect(app.project_key).toEqual('test1');
        });
    }));

    it('clone an application', async(() => {
        const http = TestBed.get(HttpTestingController);
        let application = new Application();
        application.name = 'app1';
        application.project_key = 'test1';
        store.dispatch(new ActionApplication.AddApplication({
            projectKey: 'test1',
            application
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/applications';
        })).flush({
            name: 'app1',
            project_key: 'test1',
            vcs_strategy: {}
        });
        store.selectOnce(ApplicationsState).subscribe(state => {
            expect(Object.keys(state.applications).length).toEqual(1);
        });
        store.selectOnce(ApplicationsState.selectApplication('test1', 'app1')).subscribe(app => {
            expect(app).toBeTruthy();
            expect(app.name).toEqual('app1');
            expect(app.project_key).toEqual('test1');
        });

        application.name = 'app1cloned';
        store.dispatch(new ActionApplication.CloneApplication({
            projectKey: 'test1',
            clonedAppName: 'app1',
            newApplication: application
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/application/app1/clone';
        })).flush({
            name: 'app1cloned',
            project_key: 'test1',
            vcs_strategy: {}
        });
        store.selectOnce(ApplicationsState).subscribe(state => {
            expect(Object.keys(state.applications).length).toEqual(2);
        });
        store.selectOnce(ApplicationsState.selectApplication('test1', 'app1')).subscribe(app => {
            expect(app).toBeTruthy();
            expect(app.name).toEqual('app1');
            expect(app.project_key).toEqual('test1');
        });
        store.selectOnce(ApplicationsState.selectApplication('test1', 'app1cloned')).subscribe(app => {
            expect(app).toBeTruthy();
            expect(app.name).toEqual('app1cloned');
            expect(app.project_key).toEqual('test1');
        });
    }));

    it('delete an application', async(() => {
        const http = TestBed.get(HttpTestingController);
        let application = new Application();
        application.name = 'app1';
        application.project_key = 'test1';
        store.dispatch(new ActionApplication.AddApplication({
            projectKey: 'test1',
            application
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/applications';
        })).flush({
            name: 'app1',
            project_key: 'test1',
            vcs_strategy: {}
        });
        store.selectOnce(ApplicationsState).subscribe(state => {
            expect(Object.keys(state.applications).length).toEqual(1);
        });
        store.selectOnce(ApplicationsState.selectApplication('test1', 'app1')).subscribe(app => {
            expect(app).toBeTruthy();
            expect(app.name).toEqual('app1');
            expect(app.project_key).toEqual('test1');
        });

        store.dispatch(new ActionApplication.DeleteApplication({
            projectKey: 'test1',
            applicationName: 'app1'
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/application/app1';
        })).flush(null);
        store.selectOnce(ApplicationsState).subscribe(state => {
            expect(Object.keys(state.applications).length).toEqual(0);
        });
    }));

    it('fetch an overview application', async(() => {
        const http = TestBed.get(HttpTestingController);
        let application = new Application();
        application.name = 'app1';
        application.project_key = 'test1';
        store.dispatch(new ActionApplication.AddApplication({
            projectKey: 'test1',
            application
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/applications';
        })).flush({
            name: 'app1',
            project_key: 'test1',
            vcs_strategy: {}
        });
        store.selectOnce(ApplicationsState).subscribe(state => {
            expect(Object.keys(state.applications).length).toEqual(1);
        });
        store.selectOnce(ApplicationsState.selectApplication('test1', 'app1')).subscribe(app => {
            expect(app).toBeTruthy();
            expect(app.overview).toBeFalsy();
            expect(app.name).toEqual('app1');
            expect(app.project_key).toEqual('test1');
        });

        store.dispatch(new ActionApplication.FetchApplicationOverview({
            projectKey: 'test1',
            applicationName: 'app1'
        }));
        let overview = new Overview();
        overview.git_url = 'git+ssh://thisisatest';
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/ui/project/test1/application/app1/overview';
        })).flush(overview);
        store.selectOnce(ApplicationsState).subscribe(state => {
            expect(Object.keys(state.applications).length).toEqual(1);
        });
        store.selectOnce(ApplicationsState.selectApplication('test1', 'app1')).subscribe(app => {
            expect(app).toBeTruthy();
            expect(app.overview).toBeTruthy();
            expect(app.name).toEqual('app1');
            expect(app.project_key).toEqual('test1');
            expect(app.overview.git_url).toEqual('git+ssh://thisisatest');
        });
    }));

    //  ------- Variables --------- //
    it('add a variable on application', async(() => {
        const http = TestBed.get(HttpTestingController);
        let application = new Application();
        application.name = 'app1';
        application.project_key = 'test1';
        store.dispatch(new ActionApplication.AddApplication({
            projectKey: 'test1',
            application
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/applications';
        })).flush({
            name: 'app1',
            project_key: 'test1',
            vcs_strategy: {}
        });
        store.selectOnce(ApplicationsState).subscribe(state => {
            expect(Object.keys(state.applications).length).toEqual(1);
        });
        store.selectOnce(ApplicationsState.selectApplication('test1', 'app1')).subscribe(app => {
            expect(app).toBeTruthy();
            expect(app.overview).toBeFalsy();
            expect(app.name).toEqual('app1');
            expect(app.project_key).toEqual('test1');
        });

        let variable = new Variable();
        variable.name = 'testvar';
        variable.type = 'string';
        variable.value = 'myvalue';

        store.dispatch(new ActionApplication.AddApplicationVariable({
            projectKey: 'test1',
            applicationName: 'app1',
            variable
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/application/app1/variable/testvar';
        })).flush(<Application>{
            name: 'app1',
            project_key: 'test1',
            variables: [variable],
        });
        store.selectOnce(ApplicationsState).subscribe(state => {
            expect(Object.keys(state.applications).length).toEqual(1);
        });
        store.selectOnce(ApplicationsState.selectApplication('test1', 'app1')).subscribe((app: Application) => {
            expect(app).toBeTruthy();
            expect(app.name).toEqual('app1');
            expect(app.project_key).toEqual('test1');
            expect(app.variables).toBeTruthy();
            expect(app.variables.length).toEqual(1);
            expect(app.variables[0].name).toEqual('testvar');
        });
    }));

    it('update a variable on application', async(() => {
        const http = TestBed.get(HttpTestingController);
        let application = new Application();
        application.name = 'app1';
        application.project_key = 'test1';
        store.dispatch(new ActionApplication.AddApplication({
            projectKey: 'test1',
            application
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/applications';
        })).flush({
            name: 'app1',
            project_key: 'test1',
            vcs_strategy: {}
        });

        let variable = new Variable();
        variable.name = 'testvar';
        variable.type = 'string';
        variable.value = 'myvalue';

        store.dispatch(new ActionApplication.AddApplicationVariable({
            projectKey: 'test1',
            applicationName: 'app1',
            variable
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/application/app1/variable/testvar';
        })).flush(<Application>{
            name: 'app1',
            project_key: 'test1',
            variables: [variable],
        });

        variable.name = 'testvarrenamed';
        store.dispatch(new ActionApplication.UpdateApplicationVariable({
            projectKey: 'test1',
            applicationName: 'app1',
            variableName: 'testvar',
            variable
        }));
        store.selectOnce(ApplicationsState).subscribe(state => {
            expect(Object.keys(state.applications).length).toEqual(1);
        });
        store.selectOnce(ApplicationsState.selectApplication('test1', 'app1')).subscribe((app: Application) => {
            expect(app).toBeTruthy();
            expect(app.name).toEqual('app1');
            expect(app.project_key).toEqual('test1');
            expect(app.variables).toBeTruthy();
            expect(app.variables.length).toEqual(1);
            expect(app.variables[0].name).toEqual('testvarrenamed');
        });
    }));

    it('delete a variable on application', async(() => {
        const http = TestBed.get(HttpTestingController);
        let application = new Application();
        application.name = 'app1';
        application.project_key = 'test1';
        store.dispatch(new ActionApplication.AddApplication({
            projectKey: 'test1',
            application
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/applications';
        })).flush({
            name: 'app1',
            project_key: 'test1',
            vcs_strategy: {}
        });

        let variable = new Variable();
        variable.name = 'testvar';
        variable.type = 'string';
        variable.value = 'myvalue';

        store.dispatch(new ActionApplication.AddApplicationVariable({
            projectKey: 'test1',
            applicationName: 'app1',
            variable
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/application/app1/variable/testvar';
        })).flush(<Application>{
            name: 'app1',
            project_key: 'test1',
            variables: [variable],
        });

        store.dispatch(new ActionApplication.DeleteApplicationVariable({
            projectKey: 'test1',
            applicationName: 'app1',
            variable
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/application/app1/variable/testvar';
        })).flush(<Application>{
            name: 'app1',
            project_key: 'test1',
            variables: [],
        });
        store.selectOnce(ApplicationsState).subscribe(state => {
            expect(Object.keys(state.applications).length).toEqual(1);
        });
        store.selectOnce(ApplicationsState.selectApplication('test1', 'app1')).subscribe((app: Application) => {
            expect(app).toBeTruthy();
            expect(app.name).toEqual('app1');
            expect(app.project_key).toEqual('test1');
            expect(app.variables).toBeTruthy();
            expect(app.variables.length).toEqual(0);
        });
    }));

    //  ------- Keys --------- //
    it('add a key on application', async(() => {
        const http = TestBed.get(HttpTestingController);
        let application = new Application();
        application.name = 'app1';
        application.project_key = 'test1';
        store.dispatch(new ActionApplication.AddApplication({
            projectKey: 'test1',
            application
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/applications';
        })).flush({
            name: 'app1',
            project_key: 'test1',
            vcs_strategy: {}
        });

        let key = new Key();
        key.name = 'app-mykey';
        key.type = 'ssh';

        store.dispatch(new ActionApplication.AddApplicationKey({
            projectKey: 'test1',
            applicationName: 'app1',
            key
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/application/app1/keys';
        })).flush(key);
        store.selectOnce(ApplicationsState).subscribe(state => {
            expect(Object.keys(state.applications).length).toEqual(1);
        });
        store.selectOnce(ApplicationsState.selectApplication('test1', 'app1')).subscribe((app: Application) => {
            expect(app).toBeTruthy();
            expect(app.name).toEqual('app1');
            expect(app.project_key).toEqual('test1');
            expect(app.keys).toBeTruthy();
            expect(app.keys.length).toEqual(1);
            expect(app.keys[0].name).toEqual('app-mykey');
        });
    }));

    it('delete a key on application', async(() => {
        const http = TestBed.get(HttpTestingController);
        let application = new Application();
        application.name = 'app1';
        application.project_key = 'test1';
        store.dispatch(new ActionApplication.AddApplication({
            projectKey: 'test1',
            application
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/applications';
        })).flush({
            name: 'app1',
            project_key: 'test1',
            vcs_strategy: {}
        });

        let key = new Key();
        key.name = 'app-mykey';
        key.type = 'ssh';

        store.dispatch(new ActionApplication.AddApplicationKey({
            projectKey: 'test1',
            applicationName: 'app1',
            key
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/application/app1/keys';
        })).flush(key);
        store.selectOnce(ApplicationsState).subscribe(state => {
            expect(Object.keys(state.applications).length).toEqual(1);
        });
        store.selectOnce(ApplicationsState.selectApplication('test1', 'app1')).subscribe((app: Application) => {
            expect(app).toBeTruthy();
            expect(app.name).toEqual('app1');
            expect(app.project_key).toEqual('test1');
            expect(app.keys).toBeTruthy();
            expect(app.keys.length).toEqual(1);
            expect(app.keys[0].name).toEqual('app-mykey');
        });

        store.dispatch(new ActionApplication.DeleteApplicationKey({
            projectKey: 'test1',
            applicationName: 'app1',
            key
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/application/app1/keys/app-mykey';
        })).flush(null);
        store.selectOnce(ApplicationsState).subscribe(state => {
            expect(Object.keys(state.applications).length).toEqual(1);
        });
        store.selectOnce(ApplicationsState.selectApplication('test1', 'app1')).subscribe((app: Application) => {
            expect(app).toBeTruthy();
            expect(app.name).toEqual('app1');
            expect(app.project_key).toEqual('test1');
            expect(app.keys).toBeTruthy();
            expect(app.keys.length).toEqual(0);
        });
    }));

    //  ------- Deployment strategies --------- //
    it('add a deployment on application', async(() => {
        const http = TestBed.get(HttpTestingController);
        let application = new Application();
        application.name = 'app1';
        application.project_key = 'test1';
        store.dispatch(new ActionApplication.AddApplication({
            projectKey: 'test1',
            application
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/applications';
        })).flush({
            name: 'app1',
            project_key: 'test1',
            vcs_strategy: {}
        });

        let key = new Key();
        key.name = 'app-mykey';
        key.type = 'ssh';

        let integration = new ProjectIntegration();
        integration.name = 'testintegration';
        integration.model = new IntegrationModel();
        integration.model.deployment_default_config = {
            'key1': 'value'
        };

        store.dispatch(new ActionApplication.AddApplicationDeployment({
            projectKey: 'test1',
            applicationName: 'app1',
            integration
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/application/app1/deployment/config/testintegration';
        })).flush(<Application>{
            name: 'app1',
            project_key: 'test1',
            deployment_strategies: {},
        });
        store.selectOnce(ApplicationsState).subscribe(state => {
            expect(Object.keys(state.applications).length).toEqual(1);
        });
        store.selectOnce(ApplicationsState.selectApplication('test1', 'app1')).subscribe((app: Application) => {
            expect(app).toBeTruthy();
            expect(app.name).toEqual('app1');
            expect(app.project_key).toEqual('test1');
            expect(app.deployment_strategies).toBeTruthy();
        });
    }));

    //  ------- VCS strategies --------- //
    it('connect a repository on application', async(() => {
        const http = TestBed.get(HttpTestingController);
        let application = new Application();
        application.name = 'app1';
        application.project_key = 'test1';
        store.dispatch(new ActionApplication.AddApplication({
            projectKey: 'test1',
            application
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/applications';
        })).flush({
            name: 'app1',
            project_key: 'test1',
            vcs_strategy: {}
        });

        store.dispatch(new ActionApplication.ConnectVcsRepoOnApplication({
            projectKey: 'test1',
            applicationName: 'app1',
            repoManager: 'github',
            repoFullName: 'cds'
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/repositories_manager/github/application/app1/attach';
        })).flush(<Application>{
            name: 'app1',
            project_key: 'test1',
            vcs_server: 'github',
            repository_fullname: 'cds'
        });
        store.selectOnce(ApplicationsState).subscribe(state => {
            expect(Object.keys(state.applications).length).toEqual(1);
        });
        store.selectOnce(ApplicationsState.selectApplication('test1', 'app1')).subscribe((app: Application) => {
            expect(app).toBeTruthy();
            expect(app.name).toEqual('app1');
            expect(app.project_key).toEqual('test1');
            expect(app.vcs_server).toEqual('github');
            expect(app.repository_fullname).toEqual('cds');
        });
    }));

    it('delete a repository on application', async(() => {
        const http = TestBed.get(HttpTestingController);
        let application = new Application();
        application.name = 'app1';
        application.project_key = 'test1';
        store.dispatch(new ActionApplication.AddApplication({
            projectKey: 'test1',
            application
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/applications';
        })).flush({
            name: 'app1',
            project_key: 'test1',
            vcs_strategy: {}
        });

        store.dispatch(new ActionApplication.DeleteVcsRepoOnApplication({
            projectKey: 'test1',
            applicationName: 'app1',
            repoManager: 'github'
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/repositories_manager/github/application/app1/detach';
        })).flush(<Application>{
            name: 'app1',
            project_key: 'test1'
        });
        store.selectOnce(ApplicationsState).subscribe(state => {
            expect(Object.keys(state.applications).length).toEqual(1);
        });
        store.selectOnce(ApplicationsState.selectApplication('test1', 'app1')).subscribe((app: Application) => {
            expect(app).toBeTruthy();
            expect(app.name).toEqual('app1');
            expect(app.project_key).toEqual('test1');
            expect(app.vcs_server).toBeFalsy();
            expect(app.repository_fullname).toBeFalsy();
        });
    }));

    //  ------- Misc --------- //
    it('mark an external change on application', async(() => {
        const http = TestBed.get(HttpTestingController);
        let application = new Application();
        application.name = 'app1';
        application.project_key = 'test1';
        store.dispatch(new ActionApplication.AddApplication({
            projectKey: 'test1',
            application
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/applications';
        })).flush({
            name: 'app1',
            project_key: 'test1',
            vcs_strategy: {}
        });

        store.dispatch(new ActionApplication.ExternalChangeApplication({
            projectKey: 'test1',
            applicationName: 'app1'
        }));
        store.selectOnce(ApplicationsState).subscribe(state => {
            expect(Object.keys(state.applications).length).toEqual(1);
        });
        store.selectOnce(ApplicationsState.selectApplication('test1', 'app1')).subscribe((app: Application) => {
            expect(app).toBeTruthy();
            expect(app.name).toEqual('app1');
            expect(app.project_key).toEqual('test1');
            expect(app.externalChange).toEqual(true);
        });
    }));

    it('delete application from cache', async(() => {
        const http = TestBed.get(HttpTestingController);
        let application = new Application();
        application.name = 'app1';
        application.project_key = 'test1';
        store.dispatch(new ActionApplication.AddApplication({
            projectKey: 'test1',
            application
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/applications';
        })).flush({
            name: 'app1',
            project_key: 'test1',
            vcs_strategy: {}
        });
        store.selectOnce(ApplicationsState).subscribe(state => {
            expect(Object.keys(state.applications).length).toEqual(1);
        });
        store.selectOnce(ApplicationsState.selectApplication('test1', 'app1')).subscribe((app: Application) => {
            expect(app).toBeTruthy();
            expect(app.name).toEqual('app1');
            expect(app.project_key).toEqual('test1');
        });

        store.dispatch(new ActionApplication.DeleteFromCacheApplication({
            projectKey: 'test1',
            applicationName: 'app1'
        }));
        store.selectOnce(ApplicationsState).subscribe(state => {
            expect(Object.keys(state.applications).length).toEqual(0);
        });
    }));

    it('resync application from cache', async(() => {
        const http = TestBed.get(HttpTestingController);
        let application = new Application();
        application.name = 'app1';
        application.project_key = 'test1';
        store.dispatch(new ActionApplication.AddApplication({
            projectKey: 'test1',
            application
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/applications';
        })).flush({
            name: 'app1',
            project_key: 'test1',
            vcs_strategy: {}
        });
        store.selectOnce(ApplicationsState).subscribe(state => {
            expect(Object.keys(state.applications).length).toEqual(1);
        });
        store.selectOnce(ApplicationsState.selectApplication('test1', 'app1')).subscribe((app: Application) => {
            expect(app).toBeTruthy();
            expect(app.name).toEqual('app1');
            expect(app.project_key).toEqual('test1');
            expect(app.vcs_server).toBeFalsy();
        });

        store.dispatch(new ActionApplication.ResyncApplication({
            projectKey: 'test1',
            applicationName: 'app1'
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/application/app1';
        })).flush({
            name: 'app1',
            project_key: 'test1',
            vcs_server: 'github'
        });
        store.selectOnce(ApplicationsState).subscribe(state => {
            expect(Object.keys(state.applications).length).toEqual(1);
        });
        store.selectOnce(ApplicationsState.selectApplication('test1', 'app1')).subscribe((app: Application) => {
            expect(app).toBeTruthy();
            expect(app.name).toEqual('app1');
            expect(app.project_key).toEqual('test1');
            expect(app.vcs_server).toEqual('github');
        });
    }));
});
