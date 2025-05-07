import { HttpRequest, provideHttpClient, withInterceptorsFromDi } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed, waitForAsync } from '@angular/core/testing';
import { NgxsModule, Store } from '@ngxs/store';
import { Application } from 'app/model/application.model';
import { Group, GroupPermission } from 'app/model/group.model';
import { Pipeline } from 'app/model/pipeline.model';
import { Label, LoadOpts, Project } from 'app/model/project.model';
import { Variable } from 'app/model/variable.model';
import { Workflow } from 'app/model/workflow.model';
import { ProjectService } from 'app/service/project/project.service';
import { ProjectStore } from 'app/service/project/project.store';
import { WorkflowRunService } from 'app/service/workflow/run/workflow.run.service';
import { WorkflowService } from 'app/service/workflow/workflow.service';
import { PipelineService } from 'app/service/pipeline/pipeline.service';
import { EnvironmentService } from 'app/service/environment/environment.service';
import { ApplicationService } from 'app/service/application/application.service';
import { RouterService } from 'app/service/router/router.service';
import { RouterTestingModule } from '@angular/router/testing';
import { ApplicationsState } from './applications.state';
import { PipelinesState } from './pipelines.state';
import * as ProjectAction from './project.action';
import { ProjectState } from './project.state';
import { WorkflowState } from './workflow.state';

describe('Project', () => {
    let store: Store;
    let http: HttpTestingController;

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            providers: [
                WorkflowService,
                WorkflowRunService,
                ProjectStore,
                RouterService,
                ProjectService,
                PipelineService,
                EnvironmentService,
                ApplicationService,
                provideHttpClient(withInterceptorsFromDi()),
                provideHttpClientTesting()
            ],
            imports: [
                RouterTestingModule.withRoutes([]),
                NgxsModule.forRoot([ProjectState, ApplicationsState, PipelinesState, WorkflowState], { developmentMode: true })
            ]
        }).compileComponents();
        store = TestBed.inject(Store);
        http = TestBed.inject(HttpTestingController);
    }));

    it('fetch project', waitForAsync(() => {
        store.dispatch(new ProjectAction.FetchProject({
            projectKey: 'test1',
            opts: []
        }));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/test1')).flush(<Project>{
            name: 'test1',
            key: 'test1'
        });
        store.selectOnce(ProjectState.projectSnapshot).subscribe((p: Project) => {
            expect(p).toBeTruthy();
            expect(p.name).toEqual('test1');
            expect(p.key).toEqual('test1');
        });
    }));

    it('fetch project with options', waitForAsync(() => {
        store.dispatch(new ProjectAction.FetchProject({
            projectKey: 'test1',
            opts: []
        }));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/test1')).flush(<Project>{
            name: 'test1',
            key: 'test1'
        });
        store.selectOnce(ProjectState.projectSnapshot).subscribe((p: Project) => {
            expect(p).toBeTruthy();
            expect(p.name).toEqual('test1');
            expect(p.key).toEqual('test1');
            expect(p.workflow_names).toBeFalsy();
        });

        store.dispatch(new ProjectAction.FetchProject({
            projectKey: 'test1',
            opts: [new LoadOpts('withWorkflowNames', 'workflow_names')]
        }));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/test1')).flush(<Project>{
            name: 'test1',
            key: 'test1',
            workflow_names: [{ id: 0, name: 'testworkflow', mute: false }]
        });
        store.selectOnce(ProjectState.projectSnapshot).subscribe((p: Project) => {
            expect(p).toBeTruthy();
            expect(p.name).toEqual('test1');
            expect(p.key).toEqual('test1');
            expect(p.workflow_names).toBeTruthy();
            expect(p.workflow_names.length).toEqual(1);
            expect(p.workflow_names[0].name).toEqual('testworkflow');
        });

        // Fetch from cache
        store.dispatch(new ProjectAction.FetchProject({
            projectKey: 'test1',
            opts: [new LoadOpts('withWorkflowNames', 'workflow_names')]
        }));
        store.selectOnce(ProjectState.projectSnapshot).subscribe((p: Project) => {
            expect(p).toBeTruthy();
            expect(p.name).toEqual('test1');
            expect(p.key).toEqual('test1');
            expect(p.workflow_names).toBeTruthy();
            expect(p.workflow_names.length).toEqual(1);
            expect(p.workflow_names[0].name).toEqual('testworkflow');
        });
    }));

    it('add project', waitForAsync(() => {
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project')).flush(<Project>{
            name: 'proj1',
            key: 'test1',
        });
        store.selectOnce(ProjectState.projectSnapshot).subscribe(p => {
            expect(p).toBeTruthy();
            expect(p.name).toEqual('proj1');
            expect(p.key).toEqual('test1');
        });
    }));

    //  ------- Application --------- //

    it('add application in project', waitForAsync(() => {
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project')).flush(<Project>{
            name: 'proj1',
            key: 'test1',
        });

        let application = new Application();
        application.name = 'myApp';
        application.project_key = 'test1';
        store.dispatch(new ProjectAction.AddApplicationInProject(application));

        store.selectOnce(ProjectState.projectSnapshot).subscribe((p: Project) => {
            expect(p).toBeTruthy();
            expect(p.name).toEqual('proj1');
            expect(p.key).toEqual('test1');
            expect(p.application_names).toBeTruthy();
            expect(p.application_names.length).toEqual(1);
            expect(p.application_names[0].name).toEqual('myApp');
        });
    }));

    it('update application in project', waitForAsync(() => {
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project')).flush(<Project>{
            name: 'proj1',
            key: 'test1',
        });

        let application = new Application();
        application.name = 'myApp';
        application.project_key = 'test1';
        store.dispatch(new ProjectAction.AddApplicationInProject(application));

        store.selectOnce(ProjectState.projectSnapshot).subscribe((p: Project) => {
            expect(p).toBeTruthy();
            expect(p.name).toEqual('proj1');
            expect(p.key).toEqual('test1');
            expect(p.application_names).toBeTruthy();
            expect(p.application_names.length).toEqual(1);
            expect(p.application_names[0].name).toEqual('myApp');
        });

        let appUpdated = new Application();
        appUpdated.name = 'myAppRenamed';
        appUpdated.description = 'my desc';
        store.dispatch(new ProjectAction.UpdateApplicationInProject({ previousAppName: 'myApp', changes: appUpdated }));
        store.selectOnce(ProjectState.projectSnapshot).subscribe((p: Project) => {
            expect(p).toBeTruthy();
            expect(p.name).toEqual('proj1');
            expect(p.key).toEqual('test1');
            expect(p.application_names).toBeTruthy();
            expect(p.application_names.length).toEqual(1);
            expect(p.application_names[0].name).toEqual('myAppRenamed');
            expect(p.application_names[0].description).toEqual('my desc');
        });
    }));

    it('delete application in project', waitForAsync(() => {
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project')).flush(<Project>{
            name: 'proj1',
            key: 'test1',
            application_names: [{ id: 1, name: 'myApp' }]
        });

        store.dispatch(new ProjectAction.DeleteApplicationInProject({ applicationName: 'myApp' }));

        store.selectOnce(ProjectState.projectSnapshot).subscribe((p: Project) => {
            expect(p).toBeTruthy();
            expect(p.name).toEqual('proj1');
            expect(p.key).toEqual('test1');
            expect(p.application_names).toBeTruthy();
            expect(p.application_names.length).toEqual(0);
        });
    }));

    //  ------- Workflow --------- //

    it('add workflow in project', waitForAsync(() => {
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project')).flush(<Project>{
            name: 'proj1',
            key: 'test1',
        });

        let workflow = new Workflow();
        workflow.name = 'myWorkflow';
        workflow.project_key = 'test1';
        store.dispatch(new ProjectAction.AddWorkflowInProject(workflow));

        store.selectOnce(ProjectState.projectSnapshot).subscribe((p: Project) => {
            expect(p).toBeTruthy();
            expect(p.name).toEqual('proj1');
            expect(p.key).toEqual('test1');
            expect(p.workflow_names).toBeTruthy();
            expect(p.workflow_names.length).toEqual(1);
            expect(p.workflow_names[0].name).toEqual('myWorkflow');
        });
    }));

    it('update workflow in project', waitForAsync(() => {
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        let workflow = new Workflow();
        workflow.name = 'myWorkflow';
        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project')).flush(<Project>{
            name: 'proj1',
            key: 'test1',
            workflow_names: [{ id: 1, name: workflow.name }]
        });

        workflow.name = 'myNewName';
        workflow.description = 'myDesc';
        store.dispatch(new ProjectAction.UpdateWorkflowInProject({
            previousWorkflowName: 'myWorkflow',
            changes: workflow
        }));

        store.selectOnce(ProjectState.projectSnapshot).subscribe((p: Project) => {
            expect(p).toBeTruthy();
            expect(p.name).toEqual('proj1');
            expect(p.key).toEqual('test1');
            expect(p.workflow_names).toBeTruthy();
            expect(p.workflow_names.length).toEqual(1);
            expect(p.workflow_names[0].name).toEqual('myNewName');
            expect(p.workflow_names[0].description).toEqual('myDesc');
        });
    }));

    it('delete workflow in project', waitForAsync(() => {
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project')).flush(<Project>{
            name: 'proj1',
            key: 'test1',
            workflow_names: [{ id: 1, name: 'myWorkflow' }]
        });

        store.dispatch(new ProjectAction.DeleteWorkflowInProject({ workflowName: 'myWorkflow' }));

        store.selectOnce(ProjectState.projectSnapshot).subscribe((p: Project) => {
            expect(p).toBeTruthy();
            expect(p.name).toEqual('proj1');
            expect(p.key).toEqual('test1');
            expect(p.workflow_names).toBeTruthy();
            expect(p.workflow_names.length).toEqual(0);
        });
    }));

    //  ------- Pipeline --------- //

    it('add pipeline in project', waitForAsync(() => {
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project')).flush(<Project>{
            name: 'proj1',
            key: 'test1',
        });

        let pipeline = new Pipeline();
        pipeline.name = 'myPipeline';
        store.dispatch(new ProjectAction.AddPipelineInProject(pipeline));

        store.selectOnce(ProjectState.projectSnapshot).subscribe((p: Project) => {
            expect(p).toBeTruthy();
            expect(p.name).toEqual('proj1');
            expect(p.key).toEqual('test1');
            expect(p.pipeline_names).toBeTruthy();
            expect(p.pipeline_names.length).toEqual(1);
            expect(p.pipeline_names[0].name).toEqual('myPipeline');
        });
    }));

    it('update pipeline in project', waitForAsync(() => {
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        store.dispatch(new ProjectAction.AddProject(project));
        let pip = new Pipeline();
        pip.name = 'myPipeline';
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project')).flush(<Project>{
            name: 'proj1',
            key: 'test1',
            pipeline_names: [{ id: 1, name: pip.name }]
        });

        pip.name = 'otherName';
        pip.description = 'my description';
        store.dispatch(new ProjectAction.UpdatePipelineInProject({
            previousPipName: 'myPipeline',
            changes: pip
        }));

        store.selectOnce(ProjectState.projectSnapshot).subscribe((p: Project) => {
            expect(p).toBeTruthy();
            expect(p.name).toEqual('proj1');
            expect(p.key).toEqual('test1');
            expect(p.pipeline_names).toBeTruthy();
            expect(p.pipeline_names.length).toEqual(1);
            expect(p.pipeline_names[0].name).toEqual('otherName');
            expect(p.pipeline_names[0].description).toEqual('my description');
        });
    }));

    it('delete pipeline in project', waitForAsync(() => {
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project')).flush(<Project>{
            name: 'proj1',
            key: 'test1',
            pipeline_names: [{ id: 1, name: 'myPipeline' }]
        });

        store.dispatch(new ProjectAction.DeletePipelineInProject({ pipelineName: 'myPipeline' }));

        store.selectOnce(ProjectState.projectSnapshot).subscribe((p: Project) => {
            expect(p).toBeTruthy();
            expect(p.name).toEqual('proj1');
            expect(p.key).toEqual('test1');
            expect(p.pipeline_names).toBeTruthy();
            expect(p.pipeline_names.length).toEqual(0);
        });
    }));

    //  ------- Label --------- //
    it('add label to workflow in project', waitForAsync(() => {
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project')).flush(<Project>{
            name: 'proj1',
            key: 'test1',
        });

        let workflow = new Workflow();
        workflow.name = 'myWorkflow';
        workflow.project_key = 'test1';
        store.dispatch(new ProjectAction.AddWorkflowInProject(workflow));

        let label = new Label();
        label.name = 'testLabel';
        label.color = 'red';
        store.dispatch(new ProjectAction.AddLabelWorkflowInProject({ workflowName: 'myWorkflow', label }));

        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/test1/workflows/myWorkflow/label')).flush(<Label>{
            name: 'testLabel'
        });
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/test1')).flush(<Project>{
            name: 'proj1',
            key: 'test1',
            workflow_names: [
                {
                    name: 'myWorkflow',
                    labels: [
                        {
                            name: 'testLabel'
                        }
                    ]
                }
            ]
        });

        store.selectOnce(ProjectState.projectSnapshot).subscribe((p: Project) => {
            expect(p).toBeTruthy();
            expect(p.name).toEqual('proj1');
            expect(p.key).toEqual('test1');
            expect(p.workflow_names).toBeTruthy();
            expect(p.workflow_names.length).toEqual(1);
            expect(p.workflow_names[0].name).toEqual('myWorkflow');
            expect(p.workflow_names[0].labels).toBeTruthy();
            expect(p.workflow_names[0].labels.length).toEqual(1);
            expect(p.workflow_names[0].labels[0].name).toEqual('testLabel');
        });
    }));

    it('delete label to workflow in project', waitForAsync(() => {
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project')).flush(<Project>{
            name: 'proj1',
            key: 'test1',
        });

        let label = new Label();
        label.name = 'testLabel';
        label.color = 'red';
        label.id = 25;
        let workflow = new Workflow();
        workflow.name = 'myWorkflow';
        workflow.project_key = 'test1';
        workflow.labels = [label];
        store.dispatch(new ProjectAction.AddWorkflowInProject(workflow));

        store.dispatch(new ProjectAction.DeleteLabelWorkflowInProject({ workflowName: 'myWorkflow', labelId: label.id }));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/test1/workflows/myWorkflow/label/25')).flush(<any>{
        });

        store.selectOnce(ProjectState.projectSnapshot).subscribe((p: Project) => {
            expect(p).toBeTruthy();
            expect(p.name).toEqual('proj1');
            expect(p.key).toEqual('test1');
            expect(p.workflow_names).toBeTruthy();
            expect(p.workflow_names.length).toEqual(1);
            expect(p.workflow_names[0].name).toEqual('myWorkflow');
            expect(p.workflow_names[0].labels.length).toEqual(0);
        });
    }));

    //  ------- Variable --------- //
    it('add variable in project', waitForAsync(() => {
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project')).flush(<Project>{
            name: 'proj1',
            key: 'test1',
        });

        let variable = new Variable();
        variable.name = 'myVar';
        variable.value = 'myValue';
        store.dispatch(new ProjectAction.AddVariableInProject(variable));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/test1/variable/myVar')).flush(variable);

        store.selectOnce(ProjectState.projectSnapshot).subscribe((p: Project) => {
            expect(p).toBeTruthy();
            expect(p.name).toEqual('proj1');
            expect(p.key).toEqual('test1');
            expect(p.variables).toBeTruthy();
            expect(p.variables.length).toEqual(1);
            expect(p.variables[0].name).toEqual('myVar');
            expect(p.variables[0].value).toEqual('myValue');
        });
    }));

    it('update variable in project', waitForAsync(() => {
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        let variable = new Variable();
        variable.name = 'myVar';
        variable.value = 'myValue';

        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project')).flush(<Project>{
            name: 'proj1',
            key: 'test1',
            variables: [variable]
        });

        let variableUpdated = new Variable();
        variableUpdated.name = 'myTestVar';
        variableUpdated.value = 'myTestValue';
        store.dispatch(new ProjectAction.UpdateVariableInProject({
            variableName: 'myVar',
            changes: variableUpdated
        }));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/test1/variable/myVar')).flush(variableUpdated);

        store.selectOnce(ProjectState.projectSnapshot).subscribe((p: Project) => {
            expect(p).toBeTruthy();
            expect(p.name).toEqual('proj1');
            expect(p.key).toEqual('test1');
            expect(p.variables).toBeTruthy();
            expect(p.variables.length).toEqual(1);
            expect(p.variables[0].name).toEqual('myTestVar');
            expect(p.variables[0].value).toEqual('myTestValue');
        });
    }));

    it('delete variable in project', waitForAsync(() => {
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';

        let variable = new Variable();
        variable.name = 'myVar';
        variable.value = 'myValue';
        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project')).flush(<Project>{
            name: 'proj1',
            key: 'test1',
            variables: [variable]
        });

        store.dispatch(new ProjectAction.DeleteVariableInProject(variable));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/test1/variable/myVar')).flush(null);

        store.selectOnce(ProjectState.projectSnapshot).subscribe((p: Project) => {
            expect(p).toBeTruthy();
            expect(p.name).toEqual('proj1');
            expect(p.key).toEqual('test1');
            expect(p.variables).toBeTruthy();
            expect(p.variables.length).toEqual(0);
        });
    }));

    it('fetch variable in project', waitForAsync(() => {
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';

        let variable = new Variable();
        variable.name = 'myVar';
        variable.value = 'myValue';
        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project')).flush(<Project>{
            name: 'proj1',
            key: 'test1'
        });

        store.dispatch(new ProjectAction.FetchVariablesInProject({ projectKey: project.key }));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/test1/variable')).flush([variable]);

        store.selectOnce(ProjectState.projectSnapshot).subscribe((p: Project) => {
            expect(p).toBeTruthy();
            expect(p.name).toEqual('proj1');
            expect(p.key).toEqual('test1');
            expect(p.variables).toBeTruthy();
            expect(p.variables.length).toEqual(1);
            expect(p.variables[0].name).toEqual('myVar');
            expect(p.variables[0].value).toEqual('myValue');
        });
    }));

    //  ------- Group --------- //
    it('add group permission in project', waitForAsync(() => {
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project')).flush(<Project>{
            name: 'proj1',
            key: 'test1',
        });

        let group = new GroupPermission();
        group.group = new Group();
        group.group.id = 1;
        group.group.name = 'admin';
        group.permission = 7;
        store.dispatch(new ProjectAction.AddGroupInProject({ projectKey: project.key, group }));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/test1/group')).flush([group]);

        store.selectOnce(ProjectState.projectSnapshot).subscribe((p: Project) => {
            expect(p).toBeTruthy();
            expect(p.name).toEqual('proj1');
            expect(p.key).toEqual('test1');
            expect(p.groups).toBeTruthy();
            expect(p.groups.length).toEqual(1);
            expect(p.groups[0].permission).toEqual(7);
            expect(p.groups[0].group.id).toEqual(1);
            expect(p.groups[0].group.name).toEqual('admin');
        });
    }));

    it('delete group permission in project', waitForAsync(() => {
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        let group = new GroupPermission();
        group.group = new Group();
        group.group.id = 1;
        group.group.name = 'admin';
        group.permission = 7;

        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project')).flush(<Project>{
            name: 'proj1',
            key: 'test1',
            groups: [group]
        });

        store.dispatch(new ProjectAction.DeleteGroupInProject({ projectKey: project.key, group }));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/test1/group/admin')).flush(null);

        store.selectOnce(ProjectState.projectSnapshot).subscribe((p: Project) => {
            expect(p).toBeTruthy();
            expect(p.name).toEqual('proj1');
            expect(p.key).toEqual('test1');
            expect(p.groups).toBeTruthy();
            expect(p.groups.length).toEqual(0);
        });
    }));

    it('update group permission in project', waitForAsync(() => {
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        let group = new GroupPermission();
        group.group = new Group();
        group.group.id = 1;
        group.group.name = 'admin';
        group.permission = 7;

        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project')).flush(<Project>{
            name: 'proj1',
            key: 'test1',
            groups: [group]
        });

        let groupUpdated = new GroupPermission();
        groupUpdated.group = group.group;
        groupUpdated.permission = 4;
        store.dispatch(new ProjectAction.UpdateGroupInProject({ projectKey: project.key, group: groupUpdated }));
        http.expectOne(((req: HttpRequest<any>) => req.url === '/project/test1/group/admin')).flush(groupUpdated);

        store.selectOnce(ProjectState.projectSnapshot).subscribe((p: Project) => {
            expect(p).toBeTruthy();
            expect(p.name).toEqual('proj1');
            expect(p.key).toEqual('test1');
            expect(p.groups).toBeTruthy();
            expect(p.groups.length).toEqual(1);
            expect(p.groups[0].group.name).toEqual('admin');
            expect(p.groups[0].permission).toEqual(4);
        });
    }));
});
