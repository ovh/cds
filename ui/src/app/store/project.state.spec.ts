import { HttpRequest } from '@angular/common/http';
import { HttpClientTestingModule, HttpTestingController } from '@angular/common/http/testing';
import { async, TestBed } from '@angular/core/testing';
import { NgxsModule, Store } from '@ngxs/store';
import { Application } from 'app/model/application.model';
import { Environment } from 'app/model/environment.model';
import { Group, GroupPermission } from 'app/model/group.model';
import { ProjectIntegration } from 'app/model/integration.model';
import { Key } from 'app/model/keys.model';
import { Pipeline } from 'app/model/pipeline.model';
import { Label, LoadOpts, Project } from 'app/model/project.model';
import { RepositoriesManager } from 'app/model/repositories.model';
import { Variable } from 'app/model/variable.model';
import { Workflow } from 'app/model/workflow.model';
import { NavbarService } from 'app/service/navbar/navbar.service';
import { WorkflowRunService } from 'app/service/workflow/run/workflow.run.service';
import { WorkflowService } from 'app/service/workflow/workflow.service';
import { ApplicationsState } from './applications.state';
import { PipelinesState } from './pipelines.state';
import * as ProjectAction from './project.action';
import { ProjectState, ProjectStateModel } from './project.state';
import { WorkflowState } from './workflow.state';

describe('Project', () => {
    let store: Store;

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            providers: [NavbarService, WorkflowService, WorkflowRunService],
            imports: [
                HttpClientTestingModule,
                NgxsModule.forRoot([ProjectState, ApplicationsState, PipelinesState, WorkflowState])
            ],
        }).compileComponents();

        store = TestBed.get(Store);
        // store.reset(getInitialProjectState());
    }));

    it('fetch project', async(() => {
        const http = TestBed.get(HttpTestingController);
        store.dispatch(new ProjectAction.FetchProject({
            projectKey: 'test1',
            opts: []
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1';
        })).flush(<Project>{
            name: 'test1',
            key: 'test1'
        });
        store.selectOnce(ProjectState).subscribe((proj: ProjectStateModel) => {
            expect(proj).toBeTruthy();
            expect(proj.project.name).toEqual('test1');
            expect(proj.project.key).toEqual('test1');
        });
    }));

    it('fetch project with options', async(() => {
        const http = TestBed.get(HttpTestingController);
        store.dispatch(new ProjectAction.FetchProject({
            projectKey: 'test1',
            opts: []
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1';
        })).flush(<Project>{
            name: 'test1',
            key: 'test1'
        });
        store.selectOnce(ProjectState).subscribe((proj: ProjectStateModel) => {
            expect(proj).toBeTruthy();
            expect(proj.project.name).toEqual('test1');
            expect(proj.project.key).toEqual('test1');
            expect(proj.project.workflow_names).toBeFalsy();
        });

        store.dispatch(new ProjectAction.FetchProject({
            projectKey: 'test1',
            opts: [new LoadOpts('withWorkflowNames', 'workflow_names')]
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1';
        })).flush(<Project>{
            name: 'test1',
            key: 'test1',
            workflow_names: [{ id: 0, name: 'testworkflow', mute: false }]
        });
        store.selectOnce(ProjectState).subscribe((proj: ProjectStateModel) => {
            expect(proj).toBeTruthy();
            expect(proj.project.name).toEqual('test1');
            expect(proj.project.key).toEqual('test1');
            expect(proj.project.workflow_names).toBeTruthy();
            expect(proj.project.workflow_names.length).toEqual(1);
            expect(proj.project.workflow_names[0].name).toEqual('testworkflow');
        });

        // Fetch from cache
        store.dispatch(new ProjectAction.FetchProject({
            projectKey: 'test1',
            opts: [new LoadOpts('withWorkflowNames', 'workflow_names')]
        }));
        store.selectOnce(ProjectState).subscribe((proj: ProjectStateModel) => {
            expect(proj).toBeTruthy();
            expect(proj.project.name).toEqual('test1');
            expect(proj.project.key).toEqual('test1');
            expect(proj.project.workflow_names).toBeTruthy();
            expect(proj.project.workflow_names.length).toEqual(1);
            expect(proj.project.workflow_names[0].name).toEqual('testworkflow');
        });
    }));

    it('add project', async(() => {
        const http = TestBed.get(HttpTestingController);
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project';
        })).flush(<Project>{
            name: 'proj1',
            key: 'test1',
        });
        store.selectOnce(ProjectState).subscribe(state => {
            expect(state.project).toBeTruthy();
            expect(state.project.name).toEqual('proj1');
            expect(state.project.key).toEqual('test1');
        });
    }));

    it('update project', async(() => {
        const http = TestBed.get(HttpTestingController);
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project';
        })).flush(<Project>{
            name: 'proj1',
            key: 'test1',
        });

        project.name = 'proj1updated';
        store.dispatch(new ProjectAction.UpdateProject(project));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1';
        })).flush(<Project>{
            name: 'proj1updated',
            key: 'test1',
        });

        store.selectOnce(ProjectState).subscribe(state => {
            expect(state.project).toBeTruthy();
            expect(state.project.name).toEqual('proj1updated');
            expect(state.project.key).toEqual('test1');
        });
    }));

    it('delete project', async(() => {
        const http = TestBed.get(HttpTestingController);
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project';
        })).flush(<Project>{
            name: 'proj1',
            key: 'test1',
        });

        store.dispatch(new ProjectAction.DeleteProject({ projectKey: 'test1' }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1';
        })).flush(null);

        store.selectOnce(ProjectState).subscribe(state => {
            expect(state.project).toBeFalsy();
        });
    }));

    //  ------- Application --------- //

    it('add application in project', async(() => {
        const http = TestBed.get(HttpTestingController);
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project';
        })).flush(<Project>{
            name: 'proj1',
            key: 'test1',
        });

        let application = new Application();
        application.name = 'myApp';
        application.project_key = 'test1';
        store.dispatch(new ProjectAction.AddApplicationInProject(application));

        store.selectOnce(ProjectState).subscribe((state: ProjectStateModel) => {
            expect(state.project).toBeTruthy();
            expect(state.project.name).toEqual('proj1');
            expect(state.project.key).toEqual('test1');
            expect(state.project.application_names).toBeTruthy();
            expect(state.project.application_names.length).toEqual(1);
            expect(state.project.application_names[0].name).toEqual('myApp');
        });
    }));

    it('update application in project', async(() => {
        const http = TestBed.get(HttpTestingController);
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project';
        })).flush(<Project>{
            name: 'proj1',
            key: 'test1',
        });

        let application = new Application();
        application.name = 'myApp';
        application.project_key = 'test1';
        store.dispatch(new ProjectAction.AddApplicationInProject(application));

        store.selectOnce(ProjectState).subscribe((state: ProjectStateModel) => {
            expect(state.project).toBeTruthy();
            expect(state.project.name).toEqual('proj1');
            expect(state.project.key).toEqual('test1');
            expect(state.project.application_names).toBeTruthy();
            expect(state.project.application_names.length).toEqual(1);
            expect(state.project.application_names[0].name).toEqual('myApp');
        });

        application.name = 'myAppRenamed';
        application.description = 'my desc';
        store.dispatch(new ProjectAction.UpdateApplicationInProject({ previousAppName: 'myApp', changes: application }));
        store.selectOnce(ProjectState).subscribe((state: ProjectStateModel) => {
            expect(state.project).toBeTruthy();
            expect(state.project.name).toEqual('proj1');
            expect(state.project.key).toEqual('test1');
            expect(state.project.application_names).toBeTruthy();
            expect(state.project.application_names.length).toEqual(1);
            expect(state.project.application_names[0].name).toEqual('myAppRenamed');
            expect(state.project.application_names[0].description).toEqual('my desc');
        });
    }));

    it('delete application in project', async(() => {
        const http = TestBed.get(HttpTestingController);
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project';
        })).flush(<Project>{
            name: 'proj1',
            key: 'test1',
            application_names: [{ id: 1, name: 'myApp' }]
        });

        store.dispatch(new ProjectAction.DeleteApplicationInProject({ applicationName: 'myApp' }));

        store.selectOnce(ProjectState).subscribe((state: ProjectStateModel) => {
            expect(state.project).toBeTruthy();
            expect(state.project.name).toEqual('proj1');
            expect(state.project.key).toEqual('test1');
            expect(state.project.application_names).toBeTruthy();
            expect(state.project.application_names.length).toEqual(0);
        });
    }));

    //  ------- Workflow --------- //

    it('add workflow in project', async(() => {
        const http = TestBed.get(HttpTestingController);
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project';
        })).flush(<Project>{
            name: 'proj1',
            key: 'test1',
        });

        let workflow = new Workflow();
        workflow.name = 'myWorkflow';
        workflow.project_key = 'test1';
        store.dispatch(new ProjectAction.AddWorkflowInProject(workflow));

        store.selectOnce(ProjectState).subscribe((state: ProjectStateModel) => {
            expect(state.project).toBeTruthy();
            expect(state.project.name).toEqual('proj1');
            expect(state.project.key).toEqual('test1');
            expect(state.project.workflow_names).toBeTruthy();
            expect(state.project.workflow_names.length).toEqual(1);
            expect(state.project.workflow_names[0].name).toEqual('myWorkflow');
        });
    }));

    it('update workflow in project', async(() => {
        const http = TestBed.get(HttpTestingController);
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        let workflow = new Workflow();
        workflow.name = 'myWorkflow';
        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project';
        })).flush(<Project>{
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

        store.selectOnce(ProjectState).subscribe((state: ProjectStateModel) => {
            expect(state.project).toBeTruthy();
            expect(state.project.name).toEqual('proj1');
            expect(state.project.key).toEqual('test1');
            expect(state.project.workflow_names).toBeTruthy();
            expect(state.project.workflow_names.length).toEqual(1);
            expect(state.project.workflow_names[0].name).toEqual('myNewName');
            expect(state.project.workflow_names[0].description).toEqual('myDesc');
        });
    }));

    it('delete workflow in project', async(() => {
        const http = TestBed.get(HttpTestingController);
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project';
        })).flush(<Project>{
            name: 'proj1',
            key: 'test1',
            workflow_names: [{ id: 1, name: 'myWorkflow' }]
        });

        store.dispatch(new ProjectAction.DeleteWorkflowInProject({ workflowName: 'myWorkflow' }));

        store.selectOnce(ProjectState).subscribe((state: ProjectStateModel) => {
            expect(state.project).toBeTruthy();
            expect(state.project.name).toEqual('proj1');
            expect(state.project.key).toEqual('test1');
            expect(state.project.workflow_names).toBeTruthy();
            expect(state.project.workflow_names.length).toEqual(0);
        });
    }));

    //  ------- Pipeline --------- //

    it('add pipeline in project', async(() => {
        const http = TestBed.get(HttpTestingController);
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project';
        })).flush(<Project>{
            name: 'proj1',
            key: 'test1',
        });

        let pipeline = new Pipeline();
        pipeline.name = 'myPipeline';
        store.dispatch(new ProjectAction.AddPipelineInProject(pipeline));

        store.selectOnce(ProjectState).subscribe((state: ProjectStateModel) => {
            expect(state.project).toBeTruthy();
            expect(state.project.name).toEqual('proj1');
            expect(state.project.key).toEqual('test1');
            expect(state.project.pipeline_names).toBeTruthy();
            expect(state.project.pipeline_names.length).toEqual(1);
            expect(state.project.pipeline_names[0].name).toEqual('myPipeline');
        });
    }));

    it('update pipeline in project', async(() => {
        const http = TestBed.get(HttpTestingController);
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        store.dispatch(new ProjectAction.AddProject(project));
        let pip = new Pipeline();
        pip.name = 'myPipeline';
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project';
        })).flush(<Project>{
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

        store.selectOnce(ProjectState).subscribe((state: ProjectStateModel) => {
            expect(state.project).toBeTruthy();
            expect(state.project.name).toEqual('proj1');
            expect(state.project.key).toEqual('test1');
            expect(state.project.pipeline_names).toBeTruthy();
            expect(state.project.pipeline_names.length).toEqual(1);
            expect(state.project.pipeline_names[0].name).toEqual('otherName');
            expect(state.project.pipeline_names[0].description).toEqual('my description');
        });
    }));

    it('delete pipeline in project', async(() => {
        const http = TestBed.get(HttpTestingController);
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project';
        })).flush(<Project>{
            name: 'proj1',
            key: 'test1',
            pipeline_names: [{ id: 1, name: 'myPipeline' }]
        });

        store.dispatch(new ProjectAction.DeletePipelineInProject({ pipelineName: 'myPipeline' }));

        store.selectOnce(ProjectState).subscribe((state: ProjectStateModel) => {
            expect(state.project).toBeTruthy();
            expect(state.project.name).toEqual('proj1');
            expect(state.project.key).toEqual('test1');
            expect(state.project.pipeline_names).toBeTruthy();
            expect(state.project.pipeline_names.length).toEqual(0);
        });
    }));

    //  ------- Label --------- //
    it('add label to workflow in project', async(() => {
        const http = TestBed.get(HttpTestingController);
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project';
        })).flush(<Project>{
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

        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/workflows/myWorkflow/label';
        })).flush(<Label>{
            name: 'testLabel'
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1';
        })).flush(<Project>{
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

        store.selectOnce(ProjectState).subscribe((state: ProjectStateModel) => {
            expect(state.project).toBeTruthy();
            expect(state.project.name).toEqual('proj1');
            expect(state.project.key).toEqual('test1');
            expect(state.project.workflow_names).toBeTruthy();
            expect(state.project.workflow_names.length).toEqual(1);
            expect(state.project.workflow_names[0].name).toEqual('myWorkflow');
            expect(state.project.workflow_names[0].labels).toBeTruthy();
            expect(state.project.workflow_names[0].labels.length).toEqual(1);
            expect(state.project.workflow_names[0].labels[0].name).toEqual('testLabel');
        });
    }));

    it('delete label to workflow in project', async(() => {
        const http = TestBed.get(HttpTestingController);
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project';
        })).flush(<Project>{
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
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/workflows/myWorkflow/label/25';
        })).flush(<any>{
        });

        store.selectOnce(ProjectState).subscribe((state: ProjectStateModel) => {
            expect(state.project).toBeTruthy();
            expect(state.project.name).toEqual('proj1');
            expect(state.project.key).toEqual('test1');
            expect(state.project.workflow_names).toBeTruthy();
            expect(state.project.workflow_names.length).toEqual(1);
            expect(state.project.workflow_names[0].name).toEqual('myWorkflow');
            expect(state.project.workflow_names[0].labels.length).toEqual(0);
        });
    }));

    //  ------- Variable --------- //
    it('add variable in project', async(() => {
        const http = TestBed.get(HttpTestingController);
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project';
        })).flush(<Project>{
            name: 'proj1',
            key: 'test1',
        });

        let variable = new Variable();
        variable.name = 'myVar';
        variable.value = 'myValue';
        store.dispatch(new ProjectAction.AddVariableInProject(variable));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/variable/myVar';
        })).flush(variable);

        store.selectOnce(ProjectState).subscribe((state: ProjectStateModel) => {
            expect(state.project).toBeTruthy();
            expect(state.project.name).toEqual('proj1');
            expect(state.project.key).toEqual('test1');
            expect(state.project.variables).toBeTruthy();
            expect(state.project.variables.length).toEqual(1);
            expect(state.project.variables[0].name).toEqual('myVar');
            expect(state.project.variables[0].value).toEqual('myValue');
        });
    }));

    it('update variable in project', async(() => {
        const http = TestBed.get(HttpTestingController);
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        let variable = new Variable();
        variable.name = 'myVar';
        variable.value = 'myValue';

        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project';
        })).flush(<Project>{
            name: 'proj1',
            key: 'test1',
            variables: [variable]
        });

        variable.name = 'myTestVar';
        variable.value = 'myTestValue';
        store.dispatch(new ProjectAction.UpdateVariableInProject({
            variableName: 'myVar',
            changes: variable
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/variable/myVar';
        })).flush(variable);

        store.selectOnce(ProjectState).subscribe((state: ProjectStateModel) => {
            expect(state.project).toBeTruthy();
            expect(state.project.name).toEqual('proj1');
            expect(state.project.key).toEqual('test1');
            expect(state.project.variables).toBeTruthy();
            expect(state.project.variables.length).toEqual(1);
            expect(state.project.variables[0].name).toEqual('myTestVar');
            expect(state.project.variables[0].value).toEqual('myTestValue');
        });
    }));

    it('delete variable in project', async(() => {
        const http = TestBed.get(HttpTestingController);
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';

        let variable = new Variable();
        variable.name = 'myVar';
        variable.value = 'myValue';
        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project';
        })).flush(<Project>{
            name: 'proj1',
            key: 'test1',
            variables: [variable]
        });

        store.dispatch(new ProjectAction.DeleteVariableInProject(variable));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/variable/myVar';
        })).flush(null);

        store.selectOnce(ProjectState).subscribe((state: ProjectStateModel) => {
            expect(state.project).toBeTruthy();
            expect(state.project.name).toEqual('proj1');
            expect(state.project.key).toEqual('test1');
            expect(state.project.variables).toBeTruthy();
            expect(state.project.variables.length).toEqual(0);
        });
    }));

    it('fetch variable in project', async(() => {
        const http = TestBed.get(HttpTestingController);
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';

        let variable = new Variable();
        variable.name = 'myVar';
        variable.value = 'myValue';
        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project';
        })).flush(<Project>{
            name: 'proj1',
            key: 'test1'
        });

        store.dispatch(new ProjectAction.FetchVariablesInProject({ projectKey: project.key }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/variable';
        })).flush([variable]);

        store.selectOnce(ProjectState).subscribe((state: ProjectStateModel) => {
            expect(state.project).toBeTruthy();
            expect(state.project.name).toEqual('proj1');
            expect(state.project.key).toEqual('test1');
            expect(state.project.variables).toBeTruthy();
            expect(state.project.variables.length).toEqual(1);
            expect(state.project.variables[0].name).toEqual('myVar');
            expect(state.project.variables[0].value).toEqual('myValue');
        });
    }));

    //  ------- Group --------- //
    it('add group permission in project', async(() => {
        const http = TestBed.get(HttpTestingController);
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project';
        })).flush(<Project>{
            name: 'proj1',
            key: 'test1',
        });

        let group = new GroupPermission();
        group.group = new Group();
        group.group.id = 1;
        group.group.name = 'admin';
        group.permission = 7;
        store.dispatch(new ProjectAction.AddGroupInProject({ projectKey: project.key, group }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/group';
        })).flush([group]);

        store.selectOnce(ProjectState).subscribe((state: ProjectStateModel) => {
            expect(state.project).toBeTruthy();
            expect(state.project.name).toEqual('proj1');
            expect(state.project.key).toEqual('test1');
            expect(state.project.groups).toBeTruthy();
            expect(state.project.groups.length).toEqual(1);
            expect(state.project.groups[0].permission).toEqual(7);
            expect(state.project.groups[0].group.id).toEqual(1);
            expect(state.project.groups[0].group.name).toEqual('admin');
        });
    }));

    it('delete group permission in project', async(() => {
        const http = TestBed.get(HttpTestingController);
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        let group = new GroupPermission();
        group.group = new Group();
        group.group.id = 1;
        group.group.name = 'admin';
        group.permission = 7;

        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project';
        })).flush(<Project>{
            name: 'proj1',
            key: 'test1',
            groups: [group]
        });

        store.dispatch(new ProjectAction.DeleteGroupInProject({ projectKey: project.key, group }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/group/admin';
        })).flush(null);

        store.selectOnce(ProjectState).subscribe((state: ProjectStateModel) => {
            expect(state.project).toBeTruthy();
            expect(state.project.name).toEqual('proj1');
            expect(state.project.key).toEqual('test1');
            expect(state.project.groups).toBeTruthy();
            expect(state.project.groups.length).toEqual(0);
        });
    }));

    it('update group permission in project', async(() => {
        const http = TestBed.get(HttpTestingController);
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        let group = new GroupPermission();
        group.group = new Group();
        group.group.id = 1;
        group.group.name = 'admin';
        group.permission = 7;

        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project';
        })).flush(<Project>{
            name: 'proj1',
            key: 'test1',
            groups: [group]
        });

        group.permission = 4;
        store.dispatch(new ProjectAction.UpdateGroupInProject({ projectKey: project.key, group }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/group/admin';
        })).flush(group);

        store.selectOnce(ProjectState).subscribe((state: ProjectStateModel) => {
            expect(state.project).toBeTruthy();
            expect(state.project.name).toEqual('proj1');
            expect(state.project.key).toEqual('test1');
            expect(state.project.groups).toBeTruthy();
            expect(state.project.groups.length).toEqual(1);
            expect(state.project.groups[0].group.name).toEqual('admin');
            expect(state.project.groups[0].permission).toEqual(4);
        });
    }));

    //  ------- Key --------- //
    it('add key in project', async(() => {
        const http = TestBed.get(HttpTestingController);
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project';
        })).flush(<Project>{
            name: 'proj1',
            key: 'test1',
        });

        let key = new Key();
        key.type = 'ssh';
        key.name = 'proj-test';
        store.dispatch(new ProjectAction.AddKeyInProject({ projectKey: project.key, key }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/keys' && req.body === key;
        })).flush(key);

        store.selectOnce(ProjectState).subscribe((state: ProjectStateModel) => {
            expect(state.project).toBeTruthy();
            expect(state.project.name).toEqual('proj1');
            expect(state.project.key).toEqual('test1');
            expect(state.project.keys).toBeTruthy();
            expect(state.project.keys.length).toEqual(1);
            expect(state.project.keys[0].name).toEqual('proj-test');
            expect(state.project.keys[0].type).toEqual('ssh');
        });
    }));

    it('delete key in project', async(() => {
        const http = TestBed.get(HttpTestingController);
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        let key = new Key();
        key.type = 'ssh';
        key.name = 'proj-test';

        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project';
        })).flush(<Project>{
            name: 'proj1',
            key: 'test1',
            keys: [key]
        });

        store.dispatch(new ProjectAction.DeleteKeyInProject({ projectKey: project.key, key }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/keys/proj-test';
        })).flush(null);

        store.selectOnce(ProjectState).subscribe((state: ProjectStateModel) => {
            expect(state.project).toBeTruthy();
            expect(state.project.name).toEqual('proj1');
            expect(state.project.key).toEqual('test1');
            expect(state.project.keys).toBeTruthy();
            expect(state.project.keys.length).toEqual(0);
        });
    }));

    //  ------- Integration --------- //
    it('add integration in project', async(() => {
        const http = TestBed.get(HttpTestingController);
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project';
        })).flush(<Project>{
            name: 'proj1',
            key: 'test1',
        });

        let integration = new ProjectIntegration();
        integration.name = 'myIntegration';
        store.dispatch(new ProjectAction.AddIntegrationInProject({ projectKey: project.key, integration }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/integrations';
        })).flush(integration);

        store.selectOnce(ProjectState).subscribe((state: ProjectStateModel) => {
            expect(state.project).toBeTruthy();
            expect(state.project.name).toEqual('proj1');
            expect(state.project.key).toEqual('test1');
            expect(state.project.integrations).toBeTruthy();
            expect(state.project.integrations.length).toEqual(1);
            expect(state.project.integrations[0].name).toEqual('myIntegration');
        });
    }));

    it('update integration in project', async(() => {
        const http = TestBed.get(HttpTestingController);
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        let integration = new ProjectIntegration();
        integration.name = 'myIntegration';

        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project';
        })).flush(<Project>{
            name: 'proj1',
            key: 'test1',
            integrations: [integration]
        });

        integration.name = 'myInteBis';
        store.dispatch(new ProjectAction.UpdateIntegrationInProject({
            projectKey: project.key,
            integrationName: 'myIntegration',
            changes: integration
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/integrations/myIntegration';
        })).flush(integration);

        store.selectOnce(ProjectState).subscribe((state: ProjectStateModel) => {
            expect(state.project).toBeTruthy();
            expect(state.project.name).toEqual('proj1');
            expect(state.project.key).toEqual('test1');
            expect(state.project.integrations).toBeTruthy();
            expect(state.project.integrations.length).toEqual(1);
            expect(state.project.integrations[0].name).toEqual('myInteBis');
        });
    }));

    it('delete integration in project', async(() => {
        const http = TestBed.get(HttpTestingController);
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        let integration = new ProjectIntegration();
        integration.name = 'myIntegration';

        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project';
        })).flush(<Project>{
            name: 'proj1',
            key: 'test1',
            integrations: [integration]
        });

        store.dispatch(new ProjectAction.DeleteIntegrationInProject({ projectKey: project.key, integration }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/integrations/myIntegration';
        })).flush(null);

        store.selectOnce(ProjectState).subscribe((state: ProjectStateModel) => {
            expect(state.project).toBeTruthy();
            expect(state.project.name).toEqual('proj1');
            expect(state.project.key).toEqual('test1');
            expect(state.project.integrations).toBeTruthy();
            expect(state.project.integrations.length).toEqual(0);
        });
    }));

    //  ------- Environment --------- //
    it('add environment in project', async(() => {
        const http = TestBed.get(HttpTestingController);
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project';
        })).flush(<Project>{
            name: 'proj1',
            key: 'test1',
        });

        let env = new Environment();
        env.name = 'prod';
        store.dispatch(new ProjectAction.AddEnvironmentInProject({ projectKey: project.key, environment: env }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/environment';
        })).flush(<Project>{
            ...project,
            environments: [env]
        });

        store.selectOnce(ProjectState).subscribe((state: ProjectStateModel) => {
            expect(state.project).toBeTruthy();
            expect(state.project.name).toEqual('proj1');
            expect(state.project.key).toEqual('test1');
            expect(state.project.environments).toBeTruthy();
            expect(state.project.environments.length).toEqual(1);
            expect(state.project.environments[0].name).toEqual('prod');
        });
    }));

    it('fetch environment in project', async(() => {
        const http = TestBed.get(HttpTestingController);
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project';
        })).flush(<Project>{
            name: 'proj1',
            key: 'test1',
        });

        let env = new Environment();
        env.name = 'prod';
        store.dispatch(new ProjectAction.FetchEnvironmentInProject({ projectKey: project.key, envName: env.name }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/environment/prod';
        })).flush(env);

        store.selectOnce(ProjectState).subscribe((state: ProjectStateModel) => {
            expect(state.project).toBeTruthy();
            expect(state.project.name).toEqual('proj1');
            expect(state.project.key).toEqual('test1');
            expect(state.project.environments).toBeTruthy();
            expect(state.project.environments.length).toEqual(1);
            expect(state.project.environments[0].name).toEqual('prod');
        });
    }));

    it('update environment in project', async(() => {
        const http = TestBed.get(HttpTestingController);
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        let env = new Environment();
        env.name = 'prod';

        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project';
        })).flush(<Project>{
            name: 'proj1',
            key: 'test1',
            environments: [env]
        });

        env.name = 'dev';
        store.dispatch(new ProjectAction.UpdateEnvironmentInProject({
            projectKey: project.key,
            environmentName: 'prod',
            changes: env
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/environment/prod';
        })).flush(<Project>{
            ...project,
            environments: [env]
        });

        store.selectOnce(ProjectState).subscribe((state: ProjectStateModel) => {
            expect(state.project).toBeTruthy();
            expect(state.project.name).toEqual('proj1');
            expect(state.project.key).toEqual('test1');
            expect(state.project.environments).toBeTruthy();
            expect(state.project.environments.length).toEqual(1);
            expect(state.project.environments[0].name).toEqual('dev');
        });
    }));

    it('delete environment in project', async(() => {
        const http = TestBed.get(HttpTestingController);
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        let env = new Environment();
        env.name = 'prod';

        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project';
        })).flush(<Project>{
            name: 'proj1',
            key: 'test1',
            environments: [env]
        });

        store.dispatch(new ProjectAction.DeleteEnvironmentInProject({ projectKey: project.key, environment: env }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/environment/prod';
        })).flush(<Project>{
            ...project,
            environments: []
        });

        store.selectOnce(ProjectState).subscribe((state: ProjectStateModel) => {
            expect(state.project).toBeTruthy();
            expect(state.project.name).toEqual('proj1');
            expect(state.project.key).toEqual('test1');
            expect(state.project.environments).toBeTruthy();
            expect(state.project.environments.length).toEqual(0);
        });
    }));

    it('add environment variable in project', async(() => {
        const http = TestBed.get(HttpTestingController);
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project';
        })).flush(<Project>{
            name: 'proj1',
            key: 'test1',
            environments: [{name: 'prod'}]
        });

        let env = new Environment();
        env.name = 'prod';
        let variable = new Variable();
        variable.name = 'testvar';
        variable.type = 'string';
        variable.value = 'myvalue';
        env.variables = [variable];
        store.dispatch(new ProjectAction.AddEnvironmentVariableInProject({
            projectKey: project.key,
            environmentName: env.name,
            variable
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/environment/prod/variable/testvar';
        })).flush(variable);

        store.selectOnce(ProjectState).subscribe((state: ProjectStateModel) => {
            expect(state.project).toBeTruthy();
            expect(state.project.name).toEqual('proj1');
            expect(state.project.key).toEqual('test1');
            expect(state.project.environments).toBeTruthy();
            expect(state.project.environments.length).toEqual(1);
            expect(state.project.environments[0].name).toEqual('prod');
            expect(state.project.environments[0].variables).toBeTruthy();
            expect(state.project.environments[0].variables.length).toEqual(1);
            expect(state.project.environments[0].variables[0].name).toEqual('testvar');
        });
    }));

    it('update environment variable in project', async(() => {
        const http = TestBed.get(HttpTestingController);
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
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project';
        })).flush(<Project>{
            name: 'proj1',
            key: 'test1',
            environments: [env]
        });

        variable.name = 'testvarbis';
        store.dispatch(new ProjectAction.UpdateEnvironmentVariableInProject({
            projectKey: project.key,
            environmentName: env.name,
            variableName: 'testvar',
            changes: variable
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/environment/prod/variable/testvar';
        })).flush(<Project>{
            ...project,
            environments: [Object.assign({}, env, { variables: [variable] })]
        });

        store.selectOnce(ProjectState).subscribe((state: ProjectStateModel) => {
            expect(state.project).toBeTruthy();
            expect(state.project.name).toEqual('proj1');
            expect(state.project.key).toEqual('test1');
            expect(state.project.environments).toBeTruthy();
            expect(state.project.environments.length).toEqual(1);
            expect(state.project.environments[0].variables).toBeTruthy();
            expect(state.project.environments[0].variables.length).toEqual(1);
            expect(state.project.environments[0].variables[0].name).toEqual('testvarbis');
        });
    }));

    it('delete environment variable in project', async(() => {
        const http = TestBed.get(HttpTestingController);
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
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project';
        })).flush(<Project>{
            name: 'proj1',
            key: 'test1',
            environments: [env]
        });

        store.dispatch(new ProjectAction.DeleteEnvironmentVariableInProject({
            projectKey: project.key,
            environmentName: env.name,
            variable
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/environment/prod/variable/testvar';
        })).flush(<Project>{
            ...project,
            environments: [Object.assign({}, env, { variables: [] })]
        });

        store.selectOnce(ProjectState).subscribe((state: ProjectStateModel) => {
            expect(state.project).toBeTruthy();
            expect(state.project.name).toEqual('proj1');
            expect(state.project.key).toEqual('test1');
            expect(state.project.environments).toBeTruthy();
            expect(state.project.environments.length).toEqual(1);
            expect(state.project.environments[0].variables).toBeTruthy();
            expect(state.project.environments[0].variables.length).toEqual(0);
        });
    }));

    //  ------- Repository Manager --------- //
    it('connect repository manager variable in project', async(() => {
        const http = TestBed.get(HttpTestingController);
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project';
        })).flush(<Project>{
            name: 'proj1',
            key: 'test1',
        });

        store.dispatch(new ProjectAction.ConnectRepositoryManagerInProject({
            projectKey: project.key,
            repoManager: 'github'
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/repositories_manager/github/authorize';
        })).flush({
            url: 'https://github.com',
            request_token: 'XXX'
        });

        store.selectOnce(ProjectState).subscribe((state: ProjectStateModel) => {
            expect(state.project).toBeTruthy();
            expect(state.project.name).toEqual('proj1');
            expect(state.project.key).toEqual('test1');
            expect(state.repoManager).toBeTruthy();
            expect(state.repoManager.request_token).toEqual('XXX');
            expect(state.repoManager.url).toEqual('https://github.com');
        });
    }));

    it('callback repository manager basic auth in project', async(() => {
        const http = TestBed.get(HttpTestingController);
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project';
        })).flush(<Project>{
            name: 'proj1',
            key: 'test1',
        });

        store.dispatch(new ProjectAction.CallbackRepositoryManagerBasicAuthInProject({
            projectKey: project.key,
            basicUser: 'user',
            repoManager: 'gerrit',
            basicPassword: 'password'
        }));
        let repoMan = new RepositoriesManager();
        repoMan.name = 'gerrit';
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/repositories_manager/gerrit/authorize/basicauth';
        })).flush(<Project>{
            ...project,
            vcs_servers: [repoMan]
        });

        store.selectOnce(ProjectState).subscribe((state: ProjectStateModel) => {
            expect(state.project).toBeTruthy();
            expect(state.project.name).toEqual('proj1');
            expect(state.project.key).toEqual('test1');
            expect(state.project.vcs_servers).toBeTruthy();
            expect(state.project.vcs_servers.length).toEqual(1);
            expect(state.project.vcs_servers[0].name).toEqual('gerrit');
        });
    }));


    it('callback repository manager variable in project', async(() => {
        const http = TestBed.get(HttpTestingController);
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project';
        })).flush(<Project>{
            name: 'proj1',
            key: 'test1',
        });

        store.dispatch(new ProjectAction.CallbackRepositoryManagerInProject({
            projectKey: project.key,
            code: 'XXX',
            repoManager: 'github',
            requestToken: 'XXXXX'
        }));
        let repoMan = new RepositoriesManager();
        repoMan.name = 'github';
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/repositories_manager/github/authorize/callback';
        })).flush(<Project>{
            ...project,
            vcs_servers: [repoMan]
        });

        store.selectOnce(ProjectState).subscribe((state: ProjectStateModel) => {
            expect(state.project).toBeTruthy();
            expect(state.project.name).toEqual('proj1');
            expect(state.project.key).toEqual('test1');
            expect(state.project.vcs_servers).toBeTruthy();
            expect(state.project.vcs_servers.length).toEqual(1);
            expect(state.project.vcs_servers[0].name).toEqual('github');
        });
    }));

    it('disconnect repository manager variable in project', async(() => {
        const http = TestBed.get(HttpTestingController);
        let project = new Project();
        project.name = 'proj1';
        project.key = 'test1';
        let repoMan = new RepositoriesManager();
        repoMan.name = 'github';

        store.dispatch(new ProjectAction.AddProject(project));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project';
        })).flush(<Project>{
            name: 'proj1',
            key: 'test1',
            vcs_servers: [repoMan]
        });

        store.dispatch(new ProjectAction.DisconnectRepositoryManagerInProject({
            projectKey: project.key,
            repoManager: 'github'
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/repositories_manager/github';
        })).flush(<Project>{
            ...project,
            vcs_servers: []
        });

        store.selectOnce(ProjectState).subscribe((state: ProjectStateModel) => {
            expect(state.project).toBeTruthy();
            expect(state.project.name).toEqual('proj1');
            expect(state.project.key).toEqual('test1');
            expect(state.project.vcs_servers).toBeTruthy();
            expect(state.project.vcs_servers.length).toEqual(0);
        });
    }));
});
