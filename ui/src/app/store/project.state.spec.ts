import { HttpRequest } from '@angular/common/http';
import { HttpClientTestingModule, HttpTestingController } from '@angular/common/http/testing';
import { async, TestBed } from '@angular/core/testing';
import { NgxsModule, Store } from '@ngxs/store';
import { Application } from 'app/model/application.model';
import { Group, GroupPermission } from 'app/model/group.model';
import { Key } from 'app/model/keys.model';
import { Pipeline } from 'app/model/pipeline.model';
import { Label, LoadOpts, Project } from 'app/model/project.model';
import { Variable } from 'app/model/variable.model';
import { Workflow } from 'app/model/workflow.model';
import * as ProjectAction from './project.action';
import { ProjectState, ProjectStateModel } from './project.state';

describe('Project', () => {
    let store: Store;

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            imports: [
                NgxsModule.forRoot([ProjectState]),
                HttpClientTestingModule
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

    it('rename application in project', async(() => {
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

        store.dispatch(new ProjectAction.RenameApplicationInProject({ previousAppName: 'myApp', newAppName: 'myAppRenamed' }));
        store.selectOnce(ProjectState).subscribe((state: ProjectStateModel) => {
            expect(state.project).toBeTruthy();
            expect(state.project.name).toEqual('proj1');
            expect(state.project.key).toEqual('test1');
            expect(state.project.application_names).toBeTruthy();
            expect(state.project.application_names.length).toEqual(1);
            expect(state.project.application_names[0].name).toEqual('myAppRenamed');
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
        })).flush({
            ...project,
            variables: [variable]
        });

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
            return req.url === '/project/test1/keys';
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
});
