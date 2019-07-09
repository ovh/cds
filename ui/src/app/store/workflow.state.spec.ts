import { HttpRequest } from '@angular/common/http';
import { HttpClientTestingModule, HttpTestingController } from '@angular/common/http/testing';
import { async, TestBed } from '@angular/core/testing';
import { NgxsModule, Store } from '@ngxs/store';
import { AuditWorkflow } from 'app/model/audit.model';
import { Label, Project } from 'app/model/project.model';
import { WNode, Workflow } from 'app/model/workflow.model';
import { NavbarService } from 'app/service/navbar/navbar.service';
import { ApplicationsState } from './applications.state';
import { PipelinesState } from './pipelines.state';
import { AddProject } from './project.action';
import { ProjectState, ProjectStateModel } from './project.state';
import * as workflowsActions from './workflow.action';
import {WorkflowState, WorkflowStateModel} from './workflow.state';
import { WorkflowService } from 'app/service/workflow/workflow.service';
import { WorkflowRunService } from 'app/service/workflow/run/workflow.run.service';

describe('Workflows', () => {
    let store: Store;
    let testProjectKey = 'test1';

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            providers: [NavbarService, WorkflowService, WorkflowRunService],
            imports: [
                NgxsModule.forRoot([ApplicationsState, ProjectState, PipelinesState, WorkflowState]),
                HttpClientTestingModule
            ],
        }).compileComponents();

        store = TestBed.get(Store);
        let project = new Project();
        project.key = testProjectKey;
        project.name = testProjectKey;
        store.dispatch(new AddProject(project));
        const http = TestBed.get(HttpTestingController);
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project';
        })).flush(<Project>{
            name: testProjectKey,
            key: testProjectKey,
        });
        store.selectOnce(ProjectState).subscribe((projState) => {
            expect(projState.project).toBeTruthy();
            expect(projState.project.key).toBeTruthy();
        });
    }));

    it('fetch workflow', async(() => {
        const http = TestBed.get(HttpTestingController);
        store.dispatch(new workflowsActions.GetWorkflow({
            projectKey: testProjectKey,
            workflowName: 'wf1'
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/workflows/wf1';
        })).flush(<Workflow>{
            name: 'wf1',
            project_key: testProjectKey
        });
        store.selectOnce(WorkflowState.getCurrent()).subscribe((wsf: WorkflowStateModel) => {
            expect(wsf.workflow).toBeTruthy();
            expect(wsf.workflow.name).toEqual('wf1');
            expect(wsf.workflow.project_key).toEqual(testProjectKey);
        });
    }));

    it('add workflow', async(() => {
        const http = TestBed.get(HttpTestingController);
        let workflow = new Workflow();
        workflow.name = 'wf1';
        workflow.project_key = testProjectKey;
        store.dispatch(new workflowsActions.CreateWorkflow({
            projectKey: testProjectKey,
            workflow
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/workflows';
        })).flush(workflow);
        store.select(WorkflowState.getCurrent()).subscribe((wfs: WorkflowStateModel) => {
            expect(wfs.workflow).toBeTruthy();
            expect(wfs.workflow.name).toEqual('wf1');
            expect(wfs.workflow.project_key).toEqual(testProjectKey);
        });

        store.selectOnce(ProjectState).subscribe((projState: ProjectStateModel) => {
            expect(projState.project).toBeTruthy();
            expect(projState.project.workflow_names).toBeTruthy();
            expect(projState.project.workflow_names.length).toEqual(1);
            expect(projState.project.workflow_names[0].name).toEqual('wf1');
        });
    }));

    it('update a workflow', async(() => {
        const http = TestBed.get(HttpTestingController);
        let workflow = new Workflow();
        workflow.name = 'wf1';
        workflow.project_key = testProjectKey;
        store.dispatch(new workflowsActions.CreateWorkflow({
            projectKey: testProjectKey,
            workflow
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/workflows';
        })).flush(workflow);
        store.selectOnce(WorkflowState.getCurrent()).subscribe((wfs: WorkflowStateModel) => {
            expect(wfs.workflow).toBeTruthy();
            expect(wfs.workflow.name).toEqual('wf1');
            expect(wfs.workflow.project_key).toEqual(testProjectKey);
        });

        workflow.name = 'wf1bis';
        store.dispatch(new workflowsActions.UpdateWorkflow({
            projectKey: testProjectKey,
            workflowName: 'wf1',
            changes: workflow
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/workflows/wf1';
        })).flush(workflow);
        store.selectOnce(WorkflowState.getCurrent()).subscribe((wfs: WorkflowStateModel) => {
            expect(wfs.workflow).toBeTruthy();
            expect(wfs.workflow.name).toEqual('wf1bis');
            expect(wfs.workflow.project_key).toEqual(testProjectKey);
        });

        store.selectOnce(ProjectState).subscribe((projState: ProjectStateModel) => {
            expect(projState.project).toBeTruthy();
            expect(projState.project.workflow_names).toBeTruthy();
            expect(projState.project.workflow_names.length).toEqual(1);
            expect(projState.project.workflow_names[0].name).toEqual('wf1bis');
        });
    }));

    it('delete a workflow', async(() => {
        const http = TestBed.get(HttpTestingController);
        let workflow = new Workflow();
        workflow.name = 'wf1';
        workflow.project_key = testProjectKey;
        store.dispatch(new workflowsActions.CreateWorkflow({
            projectKey: testProjectKey,
            workflow
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/workflows';
        })).flush(workflow);
        store.selectOnce(WorkflowState.getCurrent()).subscribe((wfs: WorkflowStateModel) => {
            expect(wfs.workflow).toBeTruthy();
            expect(wfs.workflow.name).toEqual('wf1');
            expect(wfs.workflow.project_key).toEqual(testProjectKey);
        });

        store.dispatch(new workflowsActions.DeleteWorkflow({
            projectKey: testProjectKey,
            workflowName: 'wf1'
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/workflows/wf1';
        })).flush(null);

        store.selectOnce(WorkflowState.getCurrent()).subscribe((wfs: WorkflowStateModel) => {
            expect(wfs.workflow).toBeFalsy();
        });

        store.selectOnce(ProjectState).subscribe((projState: ProjectStateModel) => {
            expect(projState.project).toBeTruthy();
            expect(projState.project.workflow_names).toBeTruthy();
            expect(projState.project.workflow_names.length).toEqual(0);
        });
    }));

    it('update a workflow icon', async(() => {
        const http = TestBed.get(HttpTestingController);
        let workflow = new Workflow();
        workflow.name = 'wf1';
        workflow.project_key = testProjectKey;
        store.dispatch(new workflowsActions.CreateWorkflow({
            projectKey: testProjectKey,
            workflow
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/workflows';
        })).flush(workflow);
        store.selectOnce(WorkflowState.getCurrent()).subscribe((wfs: WorkflowStateModel) => {
            expect(wfs).toBeTruthy();
            expect(wfs.workflow.name).toEqual('wf1');
            expect(wfs.workflow.project_key).toEqual(testProjectKey);
            expect(wfs.workflow.icon).toBeFalsy();
        });

        store.dispatch(new workflowsActions.UpdateWorkflowIcon({
            projectKey: testProjectKey,
            workflowName: 'wf1',
            icon: 'testicon'
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/workflows/wf1/icon';
        })).flush(null);

        store.selectOnce(WorkflowState.getCurrent()).subscribe((wfs: WorkflowStateModel) => {
            expect(wfs).toBeTruthy();
            expect(wfs.workflow.name).toEqual('wf1');
            expect(wfs.workflow.project_key).toEqual(testProjectKey);
            expect(wfs.workflow.icon).toEqual('testicon');
        });

        store.selectOnce(ProjectState).subscribe((projState: ProjectStateModel) => {
            expect(projState.project).toBeTruthy();
            expect(projState.project.workflow_names).toBeTruthy();
            expect(projState.project.workflow_names.length).toEqual(1);
            expect(projState.project.workflow_names[0].name).toEqual('wf1');
            expect(projState.project.workflow_names[0].icon).toEqual('testicon');
        });
    }));

    it('delete a workflow icon', async(() => {
        const http = TestBed.get(HttpTestingController);
        let workflow = new Workflow();
        workflow.name = 'wf1';
        workflow.project_key = testProjectKey;
        workflow.icon = 'testicon';
        store.dispatch(new workflowsActions.CreateWorkflow({
            projectKey: testProjectKey,
            workflow
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/workflows';
        })).flush(workflow);
        store.selectOnce(WorkflowState.getCurrent()).subscribe((wfs: WorkflowStateModel) => {
            expect(wfs.workflow).toBeTruthy();
            expect(wfs.workflow.name).toEqual('wf1');
            expect(wfs.workflow.project_key).toEqual(testProjectKey);
            expect(wfs.workflow.icon).toEqual('testicon');
        });

        store.dispatch(new workflowsActions.DeleteWorkflowIcon({
            projectKey: testProjectKey,
            workflowName: 'wf1'
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/workflows/wf1/icon';
        })).flush(null);
        store.selectOnce(WorkflowState.getCurrent()).subscribe((wfs: WorkflowStateModel) => {
            expect(wfs.workflow).toBeTruthy();
            expect(wfs.workflow.name).toEqual('wf1');
            expect(wfs.workflow.project_key).toEqual(testProjectKey);
            expect(wfs.workflow.icon).toBeFalsy();
        });

        store.selectOnce(ProjectState).subscribe((projState: ProjectStateModel) => {
            expect(projState.project).toBeTruthy();
            expect(projState.project.workflow_names).toBeTruthy();
            expect(projState.project.workflow_names.length).toEqual(1);
            expect(projState.project.workflow_names[0].name).toEqual('wf1');
            expect(projState.project.workflow_names[0].icon).toBeFalsy();
        });
    }));

    it('fetch audits', async(() => {
        const http = TestBed.get(HttpTestingController);
        let workflow = new Workflow();
        workflow.name = 'wf1';
        workflow.project_key = testProjectKey;
        store.dispatch(new workflowsActions.CreateWorkflow({
            projectKey: testProjectKey,
            workflow
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/workflows';
        })).flush(workflow);
        store.selectOnce(WorkflowState.getCurrent()).subscribe((wfs: WorkflowStateModel) => {
            expect(wfs.workflow).toBeTruthy();
            expect(wfs.workflow.name).toEqual('wf1');
            expect(wfs.workflow.project_key).toEqual(testProjectKey);
        });

        store.dispatch(new workflowsActions.FetchWorkflowAudits({
            projectKey: testProjectKey,
            workflowName: 'wf1'
        }));
        let audit = new AuditWorkflow();
        audit.event_type = 'update';
        audit.data_before = 'before';
        audit.triggered_by = 'test_user';
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/workflows/wf1';
        })).flush(<Workflow>{
            project_key: testProjectKey,
            name: 'wf1',
            audits: [audit]
        });
        store.selectOnce(WorkflowState.getCurrent()).subscribe((wfs: WorkflowStateModel) => {
            expect(wfs.workflow).toBeTruthy();
            expect(wfs.workflow.name).toEqual('wf1');
            expect(wfs.workflow.project_key).toEqual(testProjectKey);
            expect(wfs.workflow.audits).toBeTruthy();
            expect(wfs.workflow.audits.length).toEqual(1);
            expect(wfs.workflow.audits[0].event_type).toEqual('update');
            expect(wfs.workflow.audits[0].triggered_by).toEqual('test_user');
        });
    }));

    it('rollback', async(() => {
        const http = TestBed.get(HttpTestingController);
        let workflow = new Workflow();
        workflow.name = 'wf1';
        workflow.project_key = testProjectKey;
        store.dispatch(new workflowsActions.CreateWorkflow({
            projectKey: testProjectKey,
            workflow
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/workflows';
        })).flush(workflow);
        store.selectOnce(WorkflowState.getCurrent()).subscribe((wfs: WorkflowStateModel) => {
            expect(wfs.workflow).toBeTruthy();
            expect(wfs.workflow.name).toEqual('wf1');
            expect(wfs.workflow.project_key).toEqual(testProjectKey);
        });

        store.dispatch(new workflowsActions.RollbackWorkflow({
            projectKey: testProjectKey,
            workflowName: 'wf1',
            auditId: 1
        }));
        let node = new WNode;
        node.name = 'root';
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/workflows/wf1/rollback/1';
        })).flush(<Workflow>{
            project_key: testProjectKey,
            name: 'wf1',
            workflow_data: {
                node,
                joins: []
            },
        });
        store.selectOnce(WorkflowState.getCurrent()).subscribe((wfs: WorkflowStateModel) => {
            expect(wfs.workflow).toBeTruthy();
            expect(wfs.workflow.name).toEqual('wf1');
            expect(wfs.workflow.project_key).toEqual(testProjectKey);
            expect(wfs.workflow.workflow_data.node.name).toEqual('root');
        });
    }));

    it('fetch as code', async(() => {
        const http = TestBed.get(HttpTestingController);
        let workflow = new Workflow();
        workflow.name = 'wf1';
        workflow.project_key = testProjectKey;
        let label = new Label();
        label.name = 'my label';
        label.id = 2;
        workflow.labels = [label];
        store.dispatch(new workflowsActions.CreateWorkflow({
            projectKey: testProjectKey,
            workflow
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/workflows';
        })).flush(workflow);

        store.selectOnce(WorkflowState.getCurrent()).subscribe((wfs: WorkflowStateModel) => {
            expect(wfs.workflow).toBeTruthy();
            expect(wfs.workflow.name).toEqual('wf1');
            expect(wfs.workflow.project_key).toEqual(testProjectKey);
        });
        store.dispatch(new workflowsActions.FetchAsCodeWorkflow({
            projectKey: testProjectKey,
            workflowName: 'wf1'
        }));
        const asCode = `name: wf1
description: some description`;
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/export/workflows/wf1';
        })).flush(asCode);
        store.selectOnce(WorkflowState.getCurrent()).subscribe((wfs: WorkflowStateModel) => {
            expect(wfs.workflow).toBeTruthy();
            expect(wfs.workflow.name).toEqual('wf1');
            expect(wfs.workflow.project_key).toEqual(testProjectKey);
            expect(wfs.workflow.labels).toBeTruthy();
            expect(wfs.workflow.asCode).toEqual(asCode);
        });
    }));

    it('preview', async(() => {
        const http = TestBed.get(HttpTestingController);
        let workflow = new Workflow();
        workflow.name = 'wf1';
        workflow.project_key = testProjectKey;
        let label = new Label();
        label.name = 'my label';
        label.id = 2;
        workflow.labels = [label];
        store.dispatch(new workflowsActions.CreateWorkflow({
            projectKey: testProjectKey,
            workflow
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/workflows';
        })).flush(workflow);

        store.selectOnce(WorkflowState.getCurrent()).subscribe((wfs: WorkflowStateModel) => {
            expect(wfs.workflow).toBeTruthy();
            expect(wfs.workflow.name).toEqual('wf1');
            expect(wfs.workflow.project_key).toEqual(testProjectKey);
        });
        const asCode = `name: wf1
description: some description`;
        store.dispatch(new workflowsActions.PreviewWorkflow({
            projectKey: testProjectKey,
            workflowName: 'wf1',
            wfCode: asCode
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/preview/workflows';
        })).flush({ ...workflow, name: 'wf1preview' });
        store.selectOnce(WorkflowState.getCurrent()).subscribe((wfs: WorkflowStateModel) => {
            expect(wfs.workflow).toBeTruthy();
            expect(wfs.workflow.name).toEqual('wf1');
            expect(wfs.workflow.project_key).toEqual(testProjectKey);
            expect(wfs.workflow.labels).toBeTruthy();
            expect(wfs.workflow.preview.name).toEqual('wf1preview');
        });
    }));

    it('update favorite', async(() => {
        const http = TestBed.get(HttpTestingController);
        let workflow = new Workflow();
        workflow.name = 'wf1';
        workflow.project_key = testProjectKey;
        let label = new Label();
        label.name = 'my label';
        label.id = 2;
        workflow.labels = [label];
        store.dispatch(new workflowsActions.CreateWorkflow({
            projectKey: testProjectKey,
            workflow
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/workflows';
        })).flush(workflow);
        store.selectOnce(WorkflowState.getCurrent()).subscribe((wfs: WorkflowStateModel) => {
            expect(wfs.workflow).toBeTruthy();
            expect(wfs.workflow.name).toEqual('wf1');
            expect(wfs.workflow.project_key).toEqual(testProjectKey);
        });
        store.dispatch(new workflowsActions.UpdateFavoriteWorkflow({
            projectKey: testProjectKey,
            workflowName: 'wf1'
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/user/favorite';
        })).flush(null);
        store.selectOnce(WorkflowState.getCurrent()).subscribe((wfs: WorkflowStateModel) => {
            expect(wfs.workflow).toBeTruthy();
            expect(wfs.workflow.name).toEqual('wf1');
            expect(wfs.workflow.project_key).toEqual(testProjectKey);
            expect(wfs.workflow.favorite).toEqual(true);
        });
        store.dispatch(new workflowsActions.UpdateFavoriteWorkflow({
            projectKey: testProjectKey,
            workflowName: 'wf1'
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/user/favorite';
        })).flush(null);
        store.selectOnce(WorkflowState.getCurrent()).subscribe((wfs: WorkflowStateModel) => {
            expect(wfs.workflow).toBeTruthy();
            expect(wfs.workflow.name).toEqual('wf1');
            expect(wfs.workflow.project_key).toEqual(testProjectKey);
            expect(wfs.workflow.favorite).toEqual(false);
        });
    }));
});
