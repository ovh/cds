import { HttpRequest } from '@angular/common/http';
import { HttpClientTestingModule, HttpTestingController } from '@angular/common/http/testing';
import { TestBed, waitForAsync } from '@angular/core/testing';
import { NgxsModule, Store } from '@ngxs/store';
import { Application, Overview } from 'app/model/application.model';
import { IntegrationModel, ProjectIntegration } from 'app/model/integration.model';
import { Key } from 'app/model/keys.model';
import { Project } from 'app/model/project.model';
import { Variable } from 'app/model/variable.model';
import { NavbarService } from 'app/service/navbar/navbar.service';
import { ProjectService } from 'app/service/project/project.service';
import { ProjectStore } from 'app/service/project/project.store';
import { WorkflowRunService } from 'app/service/workflow/run/workflow.run.service';
import { WorkflowService } from 'app/service/workflow/workflow.service';
import { PipelineService } from 'app/service/pipeline/pipeline.service';
import { EnvironmentService } from 'app/service/environment/environment.service';
import { ApplicationService } from 'app/service/application/application.service';
import { RouterService } from 'app/service/router/router.service';
import { RouterTestingModule } from '@angular/router/testing';
import * as ActionApplication from './applications.action';
import { ApplicationsState, ApplicationStateModel } from './applications.state';
import { PipelinesState } from './pipelines.state';
import { AddProject } from './project.action';
import { ProjectState, ProjectStateModel } from './project.state';
import { WorkflowState } from './workflow.state';

describe('Applications', () => {
    let store: Store;
    let testProjectKey = 'test1';

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            providers: [
                RouterService,
                NavbarService,
                WorkflowService,
                WorkflowRunService,
                ProjectService,
                ProjectStore,
                PipelineService,
                EnvironmentService,
                ApplicationService
            ],
            imports: [
                NgxsModule.forRoot([ApplicationsState, ProjectState, PipelinesState, WorkflowState]),
                HttpClientTestingModule, RouterTestingModule.withRoutes([]),
            ],
        }).compileComponents();

        store = TestBed.get(Store);
        let project = new Project();
        project.key = testProjectKey;
        project.name = testProjectKey;
        store.dispatch(new AddProject(project));
        const http = TestBed.get(HttpTestingController);
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project')).flush(<Project>{
            name: testProjectKey,
            key: testProjectKey,
        });
        store.selectOnce(ProjectState).subscribe((projState) => {
            expect(projState.project).toBeTruthy();
            expect(projState.project.key).toBeTruthy();
        });
    }));

    it('fetch application', waitForAsync(() => {
        const http = TestBed.get(HttpTestingController);
        store.dispatch(new ActionApplication.FetchApplication({
            projectKey: testProjectKey,
            applicationName: 'app1'
        }));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/test1/application/app1')).flush({
            name: 'app1',
            project_key: testProjectKey,
            vcs_strategy: {}
        });
        store.selectOnce(ApplicationsState.currentState()).subscribe((s: ApplicationStateModel) => {
            expect(s.application).toBeTruthy();
            expect(s.application.name).toEqual('app1');
            expect(s.application.project_key).toEqual(testProjectKey);
        });
    }));

    it('add application', waitForAsync(() => {
        const http = TestBed.get(HttpTestingController);
        let application = new Application();
        application.name = 'app1';
        application.project_key = testProjectKey;
        store.dispatch(new ActionApplication.AddApplication({
            projectKey: testProjectKey,
            application
        }));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/test1/applications')).flush({
            name: 'app1',
            project_key: testProjectKey,
            vcs_strategy: {}
        });
        store.selectOnce(ApplicationsState.currentState()).subscribe(s => {
            expect(s.application).toBeTruthy();
            expect(s.application.name).toEqual('app1');
            expect(s.application.project_key).toEqual(testProjectKey);
        });

        store.dispatch(new ActionApplication.FetchApplication({
            projectKey: testProjectKey,
            applicationName: 'app1'
        }));
        store.selectOnce(ApplicationsState.currentState()).subscribe((s) => {
            expect(s.application).toBeTruthy();
            expect(s.application.name).toEqual('app1');
            expect(s.application.project_key).toEqual(testProjectKey);
        });

        store.selectOnce(ProjectState).subscribe((projState: ProjectStateModel) => {
            expect(projState.project).toBeTruthy();
            expect(projState.project.application_names).toBeTruthy();
            expect(projState.project.application_names.length).toEqual(1);
            expect(projState.project.application_names[0].name).toEqual('app1');
        });
    }));

    it('update an application', waitForAsync(() => {
        const http = TestBed.get(HttpTestingController);
        let application = new Application();
        application.name = 'app1';
        application.project_key = testProjectKey;
        store.dispatch(new ActionApplication.AddApplication({
            projectKey: testProjectKey,
            application
        }));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/test1/applications')).flush({
            name: 'app1',
            project_key: testProjectKey,
            vcs_strategy: {}
        });
        store.selectOnce(ApplicationsState.currentState()).subscribe(s => {
            expect(s.application).toBeTruthy();
            expect(s.application.name).toEqual('app1');
            expect(s.application.project_key).toEqual(testProjectKey);
        });

        application.name = 'app1bis';
        store.dispatch(new ActionApplication.UpdateApplication({
            projectKey: testProjectKey,
            applicationName: 'app1',
            changes: application
        }));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/test1/application/app1')).flush({
            name: 'app1bis',
            project_key: testProjectKey,
            vcs_strategy: {}
        });
        store.selectOnce(ApplicationsState.currentState()).subscribe(s => {
            expect(s.application).toBeTruthy();
            expect(s.application.name).toEqual('app1bis');
            expect(s.application.project_key).toEqual(testProjectKey);
        });

        store.selectOnce(ProjectState).subscribe((projState: ProjectStateModel) => {
            expect(projState.project).toBeTruthy();
            expect(projState.project.application_names).toBeTruthy();
            expect(projState.project.application_names.length).toEqual(1);
            expect(projState.project.application_names[0].name).toEqual('app1bis');
        });
    }));

    it('clone an application', waitForAsync(() => {
        const http = TestBed.get(HttpTestingController);
        let application = new Application();
        application.name = 'app1';
        application.project_key = testProjectKey;
        store.dispatch(new ActionApplication.AddApplication({
            projectKey: testProjectKey,
            application
        }));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/test1/applications')).flush({
            name: 'app1',
            project_key: testProjectKey,
            vcs_strategy: {}
        });
        store.selectOnce(ApplicationsState.currentState()).subscribe(s => {
            expect(s.application).toBeTruthy();
            expect(s.application.name).toEqual('app1');
            expect(s.application.project_key).toEqual(testProjectKey);
        });

        application.name = 'app1cloned';
        store.dispatch(new ActionApplication.CloneApplication({
            projectKey: testProjectKey,
            clonedAppName: 'app1',
            newApplication: application
        }));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/test1/application/app1/clone')).flush({
            name: 'app1cloned',
            project_key: testProjectKey,
            vcs_strategy: {}
        });
        store.selectOnce(ApplicationsState.currentState()).subscribe(s => {
            expect(s.application).toBeTruthy();
            expect(s.application.name).toEqual('app1cloned');
            expect(s.application.project_key).toEqual(testProjectKey);
        });

        store.selectOnce(ProjectState).subscribe((projState: ProjectStateModel) => {
            expect(projState.project).toBeTruthy();
            expect(projState.project.application_names).toBeTruthy();
            expect(projState.project.application_names.length).toEqual(2);
        });
    }));

    it('delete an application', waitForAsync(() => {
        const http = TestBed.get(HttpTestingController);
        let application = new Application();
        application.name = 'app1';
        application.project_key = testProjectKey;
        store.dispatch(new ActionApplication.AddApplication({
            projectKey: testProjectKey,
            application
        }));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/test1/applications')).flush({
            name: 'app1',
            project_key: testProjectKey,
            vcs_strategy: {}
        });
        store.selectOnce(ApplicationsState.currentState()).subscribe(s => {
            expect(s.application).toBeTruthy();
            expect(s.application.name).toEqual('app1');
            expect(s.application.project_key).toEqual(testProjectKey);
        });

        store.dispatch(new ActionApplication.DeleteApplication({
            projectKey: testProjectKey,
            applicationName: 'app1'
        }));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/test1/application/app1')).flush(null);

        store.selectOnce(ProjectState).subscribe((projState: ProjectStateModel) => {
            expect(projState.project).toBeTruthy();
            expect(projState.project.application_names).toBeTruthy();
            expect(projState.project.application_names.length).toEqual(0);
        });
    }));

    it('fetch an overview application', waitForAsync(() => {
        const http = TestBed.get(HttpTestingController);
        store.dispatch(new ActionApplication.FetchApplicationOverview({
            projectKey: testProjectKey,
            applicationName: 'app1'
        }));
        let overview = new Overview();
        overview.git_url = 'git+ssh://thisisatest';
        http.expectOne(((req: HttpRequest<any>) => req.url === '/ui/project/test1/application/app1/overview')).flush(overview);
        store.selectOnce(ApplicationsState.selectOverview()).subscribe(o => {
            expect(o).toBeTruthy();
            expect(o.git_url).toEqual('git+ssh://thisisatest');
        });
    }));

    //  ------- Variables --------- //
    it('add a variable on application', waitForAsync(() => {
        const http = TestBed.get(HttpTestingController);
        let application = new Application();
        application.name = 'app1';
        application.project_key = testProjectKey;
        store.dispatch(new ActionApplication.AddApplication({
            projectKey: testProjectKey,
            application
        }));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/test1/applications')).flush({
            name: 'app1',
            project_key: testProjectKey,
            vcs_strategy: {}
        });
        store.selectOnce(ApplicationsState.currentState()).subscribe(s => {
            expect(s.application).toBeTruthy();
            expect(s.application.overview).toBeFalsy();
            expect(s.application.name).toEqual('app1');
            expect(s.application.project_key).toEqual(testProjectKey);
        });

        let variable = new Variable();
        variable.name = 'testvar';
        variable.type = 'string';
        variable.value = 'myvalue';

        store.dispatch(new ActionApplication.AddApplicationVariable({
            projectKey: testProjectKey,
            applicationName: 'app1',
            variable
        }));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/test1/application/app1/variable/testvar')).flush(variable);
        store.selectOnce(ApplicationsState.currentState()).subscribe((s) => {
            expect(s.application).toBeTruthy();
            expect(s.application.name).toEqual('app1');
            expect(s.application.project_key).toEqual(testProjectKey);
            expect(s.application.variables).toBeTruthy();
            expect(s.application.variables.length).toEqual(1);
            expect(s.application.variables[0].name).toEqual('testvar');
        });
    }));

    it('update a variable on application', waitForAsync(() => {
        const http = TestBed.get(HttpTestingController);
        let application = new Application();
        application.name = 'app1';
        application.project_key = testProjectKey;
        store.dispatch(new ActionApplication.AddApplication({
            projectKey: testProjectKey,
            application
        }));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/test1/applications')).flush({
            name: 'app1',
            project_key: testProjectKey,
            vcs_strategy: {}
        });

        let variable = new Variable();
        variable.name = 'testvar';
        variable.type = 'string';
        variable.value = 'myvalue';

        store.dispatch(new ActionApplication.AddApplicationVariable({
            projectKey: testProjectKey,
            applicationName: 'app1',
            variable
        }));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/test1/application/app1/variable/testvar')).flush(variable);

        variable.name = 'testvarrenamed';
        store.dispatch(new ActionApplication.UpdateApplicationVariable({
            projectKey: testProjectKey,
            applicationName: 'app1',
            variableName: 'testvar',
            variable
        }));
        store.selectOnce(ApplicationsState.currentState()).subscribe((s) => {
            expect(s.application).toBeTruthy();
            expect(s.application.name).toEqual('app1');
            expect(s.application.project_key).toEqual(testProjectKey);
            expect(s.application.variables).toBeTruthy();
            expect(s.application.variables.length).toEqual(1);
            expect(s.application.variables[0].name).toEqual('testvarrenamed');
        });
    }));

    it('delete a variable on application', waitForAsync(() => {
        const http = TestBed.get(HttpTestingController);
        let application = new Application();
        application.name = 'app1';
        application.project_key = testProjectKey;
        store.dispatch(new ActionApplication.AddApplication({
            projectKey: testProjectKey,
            application
        }));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/test1/applications')).flush({
            name: 'app1',
            project_key: testProjectKey,
            vcs_strategy: {}
        });

        let variable = new Variable();
        variable.name = 'testvar';
        variable.type = 'string';
        variable.value = 'myvalue';

        store.dispatch(new ActionApplication.AddApplicationVariable({
            projectKey: testProjectKey,
            applicationName: 'app1',
            variable
        }));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/test1/application/app1/variable/testvar')).flush(variable);

        store.dispatch(new ActionApplication.DeleteApplicationVariable({
            projectKey: testProjectKey,
            applicationName: 'app1',
            variable
        }));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/test1/application/app1/variable/testvar')).flush(<Application>{
            name: 'app1',
            project_key: testProjectKey,
            variables: [],
        });
        store.selectOnce(ApplicationsState.currentState()).subscribe((s) => {
            expect(s.application).toBeTruthy();
            expect(s.application.name).toEqual('app1');
            expect(s.application.project_key).toEqual(testProjectKey);
            expect(s.application.variables).toBeTruthy();
            expect(s.application.variables.length).toEqual(0);
        });
    }));

    //  ------- Keys --------- //
    it('add a key on application', waitForAsync(() => {
        const http = TestBed.get(HttpTestingController);
        let application = new Application();
        application.name = 'app1';
        application.project_key = testProjectKey;
        store.dispatch(new ActionApplication.AddApplication({
            projectKey: testProjectKey,
            application
        }));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/test1/applications')).flush({
            name: 'app1',
            project_key: testProjectKey,
            vcs_strategy: {}
        });

        let key = new Key();
        key.name = 'app-mykey';
        key.type = 'ssh';

        store.dispatch(new ActionApplication.AddApplicationKey({
            projectKey: testProjectKey,
            applicationName: 'app1',
            key
        }));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/test1/application/app1/keys')).flush(key);
        store.selectOnce(ApplicationsState.currentState()).subscribe((s) => {
            expect(s.application).toBeTruthy();
            expect(s.application.name).toEqual('app1');
            expect(s.application.project_key).toEqual(testProjectKey);
            expect(s.application.keys).toBeTruthy();
            expect(s.application.keys.length).toEqual(1);
            expect(s.application.keys[0].name).toEqual('app-mykey');
        });
    }));

    it('delete a key on application', waitForAsync(() => {
        const http = TestBed.get(HttpTestingController);
        let application = new Application();
        application.name = 'app1';
        application.project_key = testProjectKey;
        store.dispatch(new ActionApplication.AddApplication({
            projectKey: testProjectKey,
            application
        }));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/test1/applications')).flush({
            name: 'app1',
            project_key: testProjectKey,
            vcs_strategy: {}
        });

        let key = new Key();
        key.name = 'app-mykey';
        key.type = 'ssh';

        store.dispatch(new ActionApplication.AddApplicationKey({
            projectKey: testProjectKey,
            applicationName: 'app1',
            key
        }));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/test1/application/app1/keys')).flush(key);
        store.selectOnce(ApplicationsState.currentState()).subscribe((s) => {
            expect(s.application).toBeTruthy();
            expect(s.application.name).toEqual('app1');
            expect(s.application.project_key).toEqual(testProjectKey);
            expect(s.application.keys).toBeTruthy();
            expect(s.application.keys.length).toEqual(1);
            expect(s.application.keys[0].name).toEqual('app-mykey');
        });

        store.dispatch(new ActionApplication.DeleteApplicationKey({
            projectKey: testProjectKey,
            applicationName: 'app1',
            key
        }));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/test1/application/app1/keys/app-mykey')).flush(null);
        store.selectOnce(ApplicationsState.currentState()).subscribe((s) => {
            expect(s.application).toBeTruthy();
            expect(s.application.name).toEqual('app1');
            expect(s.application.project_key).toEqual(testProjectKey);
            expect(s.application.keys).toBeTruthy();
            expect(s.application.keys.length).toEqual(0);
        });
    }));

    //  ------- Deployment strategies --------- //
    it('add a deployment on application', waitForAsync(() => {
        const http = TestBed.get(HttpTestingController);
        let application = new Application();
        application.name = 'app1';
        application.project_key = testProjectKey;
        store.dispatch(new ActionApplication.AddApplication({
            projectKey: testProjectKey,
            application
        }));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/test1/applications')).flush({
            name: 'app1',
            project_key: testProjectKey,
            vcs_strategy: {}
        });

        let key = new Key();
        key.name = 'app-mykey';
        key.type = 'ssh';

        let integration = new ProjectIntegration();
        integration.name = 'testintegration';
        integration.model = new IntegrationModel();
        integration.model.deployment_default_config = {
            key1: 'value'
        };

        store.dispatch(new ActionApplication.AddApplicationDeployment({
            projectKey: testProjectKey,
            applicationName: 'app1',
            integration
        }));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/test1/application/app1/deployment/config/testintegration')).flush(<Application>{
            name: 'app1',
            project_key: testProjectKey,
            deployment_strategies: {},
        });
        store.selectOnce(ApplicationsState.currentState()).subscribe((s) => {
            expect(s.application).toBeTruthy();
            expect(s.application.name).toEqual('app1');
            expect(s.application.project_key).toEqual(testProjectKey);
            expect(s.application.deployment_strategies).toBeTruthy();
        });
    }));

    //  ------- VCS strategies --------- //
    it('connect a repository on application', waitForAsync(() => {
        const http = TestBed.get(HttpTestingController);
        let application = new Application();
        application.name = 'app1';
        application.project_key = testProjectKey;
        store.dispatch(new ActionApplication.AddApplication({
            projectKey: testProjectKey,
            application
        }));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/test1/applications')).flush({
            name: 'app1',
            project_key: testProjectKey,
            vcs_strategy: {}
        });

        store.dispatch(new ActionApplication.ConnectVcsRepoOnApplication({
            projectKey: testProjectKey,
            applicationName: 'app1',
            repoManager: 'github',
            repoFullName: 'cds'
        }));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/test1/repositories_manager/github/application/app1/attach')).flush(<Application>{
            name: 'app1',
            project_key: testProjectKey,
            vcs_server: 'github',
            repository_fullname: 'cds'
        });
        store.selectOnce(ApplicationsState.currentState()).subscribe((s) => {
            expect(s.application).toBeTruthy();
            expect(s.application.name).toEqual('app1');
            expect(s.application.project_key).toEqual(testProjectKey);
            expect(s.application.vcs_server).toEqual('github');
            expect(s.application.repository_fullname).toEqual('cds');
        });
    }));

    it('delete a repository on application', waitForAsync(() => {
        const http = TestBed.get(HttpTestingController);
        let application = new Application();
        application.name = 'app1';
        application.project_key = testProjectKey;
        store.dispatch(new ActionApplication.AddApplication({
            projectKey: testProjectKey,
            application
        }));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/test1/applications')).flush({
            name: 'app1',
            project_key: testProjectKey,
            vcs_strategy: {}
        });

        store.dispatch(new ActionApplication.DeleteVcsRepoOnApplication({
            projectKey: testProjectKey,
            applicationName: 'app1',
            repoManager: 'github'
        }));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/test1/repositories_manager/github/application/app1/detach')).flush(<Application>{
            name: 'app1',
            project_key: testProjectKey
        });

        store.selectOnce(ApplicationsState.currentState()).subscribe((s) => {
            expect(s.application).toBeTruthy();
            expect(s.application.name).toEqual('app1');
            expect(s.application.project_key).toEqual(testProjectKey);
            expect(s.application.vcs_server).toBeFalsy();
            expect(s.application.repository_fullname).toBeFalsy();
        });
    }));

    //  ------- Misc --------- //
    it('mark an external change on application', waitForAsync(() => {
        const http = TestBed.get(HttpTestingController);
        let application = new Application();
        application.name = 'app1';
        application.project_key = testProjectKey;
        store.dispatch(new ActionApplication.AddApplication({
            projectKey: testProjectKey,
            application
        }));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/test1/applications')).flush({
            name: 'app1',
            project_key: testProjectKey,
            vcs_strategy: {}
        });

        store.dispatch(new ActionApplication.ExternalChangeApplication({
            projectKey: testProjectKey,
            applicationName: 'app1'
        }));
        store.selectOnce(ApplicationsState.currentState()).subscribe((s) => {
            expect(s.application).toBeTruthy();
            expect(s.application.name).toEqual('app1');
            expect(s.application.project_key).toEqual(testProjectKey);
            expect(s.application.externalChange).toEqual(true);
        });
    }));

    it('delete application from cache', waitForAsync(() => {
        const http = TestBed.get(HttpTestingController);
        let application = new Application();
        application.name = 'app1';
        application.project_key = testProjectKey;
        store.dispatch(new ActionApplication.AddApplication({
            projectKey: testProjectKey,
            application
        }));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/test1/applications')).flush({
            name: 'app1',
            project_key: testProjectKey,
            vcs_strategy: {}
        });
        store.selectOnce(ApplicationsState.currentState()).subscribe((s) => {
            expect(s.application).toBeTruthy();
            expect(s.application.name).toEqual('app1');
            expect(s.application.project_key).toEqual(testProjectKey);
        });

        store.dispatch(new ActionApplication.ClearCacheApplication());
        store.selectOnce(ApplicationsState.currentState()).subscribe((s) => {
            expect(s.application).toBeFalsy();
        });
    }));

    it('resync application from cache', waitForAsync(() => {
        const http = TestBed.get(HttpTestingController);
        let application = new Application();
        application.name = 'app1';
        application.project_key = testProjectKey;
        store.dispatch(new ActionApplication.AddApplication({
            projectKey: testProjectKey,
            application
        }));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/test1/applications')).flush({
            name: 'app1',
            project_key: testProjectKey,
            vcs_strategy: {}
        });
        store.selectOnce(ApplicationsState.currentState()).subscribe((s) => {
            expect(s.application).toBeTruthy();
            expect(s.application.name).toEqual('app1');
            expect(s.application.project_key).toEqual(testProjectKey);
            expect(s.application.vcs_server).toBeFalsy();
        });

        store.dispatch(new ActionApplication.ResyncApplication({
            projectKey: testProjectKey,
            applicationName: 'app1'
        }));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/test1/application/app1')).flush({
            name: 'app1',
            project_key: testProjectKey,
            vcs_server: 'github'
        });
        store.selectOnce(ApplicationsState.currentState()).subscribe((s) => {
            expect(s.application).toBeTruthy();
            expect(s.application.name).toEqual('app1');
            expect(s.application.project_key).toEqual(testProjectKey);
            expect(s.application.vcs_server).toEqual('github');
        });
    }));
});
