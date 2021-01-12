import { HttpRequest } from '@angular/common/http';
import { HttpClientTestingModule, HttpTestingController } from '@angular/common/http/testing';
import { TestBed, waitForAsync } from '@angular/core/testing';
import { NgxsModule, Store } from '@ngxs/store';
import { Environment } from 'app/model/environment.model';
import { Project } from 'app/model/project.model';
import { Variable } from 'app/model/variable.model';
import { NavbarService } from 'app/service/navbar/navbar.service';
import { ProjectService } from 'app/service/project/project.service';
import { ProjectStore } from 'app/service/project/project.store';
import { WorkflowRunService } from 'app/service/workflow/run/workflow.run.service';
import { WorkflowService } from 'app/service/workflow/workflow.service';
import {
    AddEnvironment,
    FetchEnvironment,
    UpdateEnvironment,
    AddEnvironmentVariable,
    LoadEnvironment, UpdateEnvironmentVariable, DeleteEnvironmentVariable
} from 'app/store/environment.action';
import { EnvironmentState, EnvironmentStateModel } from 'app/store/environment.state';
import { cloneDeep } from 'lodash-es';
import { PipelineService } from 'app/service/pipeline/pipeline.service';
import { EnvironmentService } from 'app/service/environment/environment.service';
import { ApplicationService } from 'app/service/application/application.service';
import { RouterService } from 'app/service/router/router.service';
import { RouterTestingModule } from '@angular/router/testing';
import { ApplicationsState } from './applications.state';
import { PipelinesState } from './pipelines.state';
import * as ProjectAction from './project.action';
import { ProjectState, ProjectStateModel } from './project.state';
import { WorkflowState } from './workflow.state';

describe('Environment', () => {
    let store: Store;
    let http: HttpTestingController;

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            providers: [NavbarService, WorkflowService, WorkflowRunService, ProjectStore, RouterService,
                ProjectService, PipelineService, EnvironmentService, ApplicationService, EnvironmentService],
            imports: [
                HttpClientTestingModule, RouterTestingModule.withRoutes([]),
                NgxsModule.forRoot([ProjectState, ApplicationsState, PipelinesState, WorkflowState, EnvironmentState])
            ],
        }).compileComponents();

        store = TestBed.inject(Store);
        http = TestBed.inject(HttpTestingController);
    }));

    //  ------- Environment --------- //
    it('add environment in project', waitForAsync(() => {
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project')).flush(<Project>{
            name: 'proj1',
            key: 'test1',
        });

        let env = new Environment();
        env.name = 'prod';
        store.dispatch(new AddEnvironment({ projectKey: project.key, environment: env }));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/test1/environment')).flush(<Project>{
            ...project,
            environments: [env]
        });

        store.selectOnce(EnvironmentState).subscribe((state: EnvironmentStateModel) => {
            expect(state.currentProjectKey).toEqual('test1');
            expect(state.environment).toBeTruthy();
            expect(state.environment.name).toEqual('prod');
        });
    }));

    it('fetch environment in project', waitForAsync(() => {
        const http = TestBed.inject(HttpTestingController);
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project')).flush(<Project>{
            name: 'proj1',
            key: 'test1',
        });

        let env = new Environment();
        env.name = 'prod';
        store.dispatch(new FetchEnvironment({ projectKey: project.key, envName: env.name }));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/test1/environment/prod')).flush(env);

        store.selectOnce(EnvironmentState).subscribe((state: EnvironmentStateModel) => {
            expect(state.currentProjectKey).toEqual('test1');
            expect(state.environment).toBeTruthy();
            expect(state.environment.name).toEqual('prod');
        });
    }));

    it('update environment in project', waitForAsync(() => {
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        let env = new Environment();
        env.name = 'prod';

        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project')).flush(<Project>{
            name: 'proj1',
            key: 'test1',
            environments: [env]
        });

        store.dispatch(new LoadEnvironment({projectKey: project.key, env}))

        env.name = 'dev';
        store.dispatch(new UpdateEnvironment({
            projectKey: project.key,
            environmentName: 'prod',
            changes: env
        }));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/test1/environment/prod')).flush(<Project>{
            ...project,
            environments: [env]
        });

        store.selectOnce(EnvironmentState).subscribe((state: EnvironmentStateModel) => {
            expect(state.currentProjectKey).toEqual('test1');
            expect(state.environment).toBeTruthy();
            expect(state.environment.name).toEqual('dev');
        });
    }));

    it('add environment variable in project', waitForAsync(() => {
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project')).flush(<Project>{
            name: 'proj1',
            key: 'test1',
            environments: [{ name: 'prod' }]
        });

        let env = new Environment();
        env.name = 'prod';

        store.dispatch(new LoadEnvironment({projectKey: project.key, env: cloneDeep(env)}));

        let variable = new Variable();
        variable.name = 'testvar';
        variable.type = 'string';
        variable.value = 'myvalue';
        store.dispatch(new AddEnvironmentVariable({
            projectKey: project.key,
            environmentName: env.name,
            variable
        }));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/test1/environment/prod/variable/testvar')).flush(variable);

        store.selectOnce(EnvironmentState).subscribe((state: EnvironmentStateModel) => {
            expect(state.currentProjectKey).toEqual('test1');
            expect(state.environment).toBeTruthy();
            expect(state.environment.name).toEqual('prod');
            expect(state.environment.variables).toBeTruthy();
            expect(state.environment.variables.length).toEqual(1);
            expect(state.environment.variables[0].name).toEqual('testvar');
        });
    }));

    it('update environment variable in project', waitForAsync(() => {
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        let env = new Environment();
        env.name = 'prod';
        let variable = new Variable();
        variable.name = 'testvar';
        variable.type = 'string';
        variable.value = 'myvalue';
        env.variables = [variable];

        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project')).flush(<Project>{
            name: 'proj1',
            key: 'test1',
            environments: [env]
        });

        store.dispatch(new LoadEnvironment({projectKey: project.key, env}));

        variable.name = 'testvarbis';
        store.dispatch(new UpdateEnvironmentVariable({
            projectKey: project.key,
            environmentName: env.name,
            variableName: 'testvar',
            changes: variable
        }));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/test1/environment/prod/variable/testvar')).flush(<Project>{
            ...project,
            environments: [Object.assign({}, env, { variables: [variable] })]
        });

        store.selectOnce(EnvironmentState).subscribe((state: EnvironmentStateModel) => {
            expect(state.currentProjectKey).toEqual('test1');
            expect(state.environment).toBeTruthy();
            expect(state.environment.variables).toBeTruthy();
            expect(state.environment.variables.length).toEqual(1);
            expect(state.environment.variables[0].name).toEqual('testvarbis');
        });
    }));

    it('delete environment variable in project', waitForAsync(() => {
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        let env = new Environment();
        env.name = 'prod';
        let variable = new Variable();
        variable.name = 'testvar';
        variable.type = 'string';
        variable.value = 'myvalue';
        env.variables = [variable];

        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project')).flush(<Project>{
            name: 'proj1',
            key: 'test1',
            environments: [env]
        });

        store.dispatch(new LoadEnvironment({projectKey: project.key, env}));

        store.dispatch(new DeleteEnvironmentVariable({
            projectKey: project.key,
            environmentName: env.name,
            variable
        }));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/test1/environment/prod/variable/testvar')).flush(<Project>{
            ...project,
            environments: [Object.assign({}, env, { variables: [] })]
        });

        store.selectOnce(EnvironmentState).subscribe((state: EnvironmentStateModel) => {
            expect(state.currentProjectKey).toEqual('test1');
            expect(state.environment).toBeTruthy();
            expect(state.environment.variables).toBeTruthy();
            expect(state.environment.variables.length).toEqual(0);
        });
    }));

});
