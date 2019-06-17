import { HttpRequest } from '@angular/common/http';
import { HttpClientTestingModule, HttpTestingController } from '@angular/common/http/testing';
import { async, TestBed } from '@angular/core/testing';
import { NgxsModule, Store } from '@ngxs/store';
import { Action } from 'app/model/action.model';
import { Job } from 'app/model/job.model';
import { Parameter } from 'app/model/parameter.model';
import { Pipeline, PipelineAudit } from 'app/model/pipeline.model';
import { Project } from 'app/model/project.model';
import { Stage } from 'app/model/stage.model';
import { NavbarService } from 'app/service/navbar/navbar.service';
import { ApplicationsState } from './applications.state';
import * as pipelinesActions from './pipelines.action';
import { PipelinesState, PipelinesStateModel } from './pipelines.state';
import { AddProject } from './project.action';
import { ProjectState, ProjectStateModel } from './project.state';
import { WorkflowState } from './workflow.state';
import { WorkflowRunService } from 'app/service/workflow/run/workflow.run.service';
import { WorkflowService } from 'app/service/workflow/workflow.service';

describe('Pipelines', () => {
    let store: Store;
    let testProjectKey = 'test1';

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            providers: [NavbarService, WorkflowRunService, WorkflowService],
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

    it('fetch pipeline', async(() => {
        const http = TestBed.get(HttpTestingController);
        store.dispatch(new pipelinesActions.FetchPipeline({
            projectKey: testProjectKey,
            pipelineName: 'pip1'
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/pipeline/pip1';
        })).flush(<Pipeline>{
            name: 'pip1',
            projectKey: testProjectKey
        });
        store.selectOnce(PipelinesState).subscribe((state: PipelinesStateModel) => {
            expect(Object.keys(state.pipelines).length).toEqual(1);
        });
        store.selectOnce(PipelinesState.selectPipeline(testProjectKey, 'pip1')).subscribe((pip: Pipeline) => {
            expect(pip).toBeTruthy();
            expect(pip.name).toEqual('pip1');
            expect(pip.projectKey).toEqual(testProjectKey);
        });
    }));

    it('add pipeline', async(() => {
        const http = TestBed.get(HttpTestingController);
        let pipeline = new Pipeline();
        pipeline.name = 'pip1';
        pipeline.projectKey = testProjectKey;
        store.dispatch(new pipelinesActions.AddPipeline({
            projectKey: testProjectKey,
            pipeline
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/pipeline';
        })).flush(pipeline);
        store.selectOnce(PipelinesState).subscribe(state => {
            expect(Object.keys(state.pipelines).length).toEqual(1);
        });
        store.selectOnce(PipelinesState.selectPipeline(testProjectKey, 'pip1')).subscribe((pip: Pipeline) => {
            expect(pip).toBeTruthy();
            expect(pip.name).toEqual('pip1');
            expect(pip.projectKey).toEqual(testProjectKey);
        });

        store.dispatch(new pipelinesActions.FetchPipeline({
            projectKey: testProjectKey,
            pipelineName: 'pip1'
        }));
        store.selectOnce(PipelinesState).subscribe(state => {
            expect(Object.keys(state.pipelines).length).toEqual(1);
        });
        store.selectOnce(PipelinesState.selectPipeline(testProjectKey, 'pip1')).subscribe((pip: Pipeline) => {
            expect(pip).toBeTruthy();
            expect(pip.name).toEqual('pip1');
            expect(pip.projectKey).toEqual(testProjectKey);
        });

        store.selectOnce(ProjectState).subscribe((projState: ProjectStateModel) => {
            expect(projState.project).toBeTruthy();
            expect(projState.project.pipeline_names).toBeTruthy();
            expect(projState.project.pipeline_names.length).toEqual(1);
            expect(projState.project.pipeline_names[0].name).toEqual('pip1');
        });
    }));

    it('update a pipeline', async(() => {
        const http = TestBed.get(HttpTestingController);
        let pipeline = new Pipeline();
        pipeline.name = 'pip1';
        pipeline.projectKey = testProjectKey;
        store.dispatch(new pipelinesActions.AddPipeline({
            projectKey: testProjectKey,
            pipeline
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/pipeline';
        })).flush(pipeline);
        store.selectOnce(PipelinesState).subscribe(state => {
            expect(Object.keys(state.pipelines).length).toEqual(1);
        });
        store.selectOnce(PipelinesState.selectPipeline(testProjectKey, 'pip1')).subscribe((pip: Pipeline) => {
            expect(pip).toBeTruthy();
            expect(pip.name).toEqual('pip1');
            expect(pip.projectKey).toEqual(testProjectKey);
        });

        pipeline.name = 'pip1bis';
        store.dispatch(new pipelinesActions.UpdatePipeline({
            projectKey: testProjectKey,
            pipelineName: 'pip1',
            changes: pipeline
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/pipeline/pip1';
        })).flush(pipeline);
        store.selectOnce(PipelinesState).subscribe(state => {
            expect(Object.keys(state.pipelines).length).toEqual(1);
        });
        store.selectOnce(PipelinesState.selectPipeline(testProjectKey, 'pip1bis')).subscribe((pip: Pipeline) => {
            expect(pip).toBeTruthy();
            expect(pip.name).toEqual('pip1bis');
            expect(pip.projectKey).toEqual(testProjectKey);
        });

        store.selectOnce(ProjectState).subscribe((projState: ProjectStateModel) => {
            expect(projState.project).toBeTruthy();
            expect(projState.project.pipeline_names).toBeTruthy();
            expect(projState.project.pipeline_names.length).toEqual(1);
            expect(projState.project.pipeline_names[0].name).toEqual('pip1bis');
        });
    }));

    it('delete a pipeline', async(() => {
        const http = TestBed.get(HttpTestingController);
        let pipeline = new Pipeline();
        pipeline.name = 'pip1';
        pipeline.projectKey = testProjectKey;
        store.dispatch(new pipelinesActions.AddPipeline({
            projectKey: testProjectKey,
            pipeline
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/pipeline';
        })).flush(pipeline);
        store.selectOnce(PipelinesState).subscribe(state => {
            expect(Object.keys(state.pipelines).length).toEqual(1);
        });
        store.selectOnce(PipelinesState.selectPipeline(testProjectKey, 'pip1')).subscribe(pip => {
            expect(pip).toBeTruthy();
            expect(pip.name).toEqual('pip1');
            expect(pip.projectKey).toEqual(testProjectKey);
        });

        store.dispatch(new pipelinesActions.DeletePipeline({
            projectKey: testProjectKey,
            pipelineName: 'pip1'
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/pipeline/pip1';
        })).flush(null);
        store.selectOnce(PipelinesState).subscribe(state => {
            expect(Object.keys(state.pipelines).length).toEqual(0);
        });

        store.selectOnce(ProjectState).subscribe((projState: ProjectStateModel) => {
            expect(projState.project).toBeTruthy();
            expect(projState.project.pipeline_names).toBeTruthy();
            expect(projState.project.pipeline_names.length).toEqual(0);
        });
    }));

    it('fetch audits pipeline', async(() => {
        const http = TestBed.get(HttpTestingController);
        let pipeline = new Pipeline();
        pipeline.name = 'pip1';
        pipeline.projectKey = testProjectKey;
        store.dispatch(new pipelinesActions.AddPipeline({
            projectKey: testProjectKey,
            pipeline
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/pipeline';
        })).flush(pipeline);
        store.selectOnce(PipelinesState).subscribe(state => {
            expect(Object.keys(state.pipelines).length).toEqual(1);
        });
        store.selectOnce(PipelinesState.selectPipeline(testProjectKey, 'pip1')).subscribe(pip => {
            expect(pip).toBeTruthy();
            expect(pip.overview).toBeFalsy();
            expect(pip.name).toEqual('pip1');
            expect(pip.projectKey).toEqual(testProjectKey);
        });

        store.dispatch(new pipelinesActions.FetchPipelineAudits({
            projectKey: testProjectKey,
            pipelineName: 'pip1'
        }));
        let audit = new PipelineAudit();
        audit.action = 'update';
        audit.pipeline = new Pipeline();
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/pipeline/pip1/audits';
        })).flush([audit]);
        store.selectOnce(PipelinesState).subscribe(state => {
            expect(Object.keys(state.pipelines).length).toEqual(1);
        });
        store.selectOnce(PipelinesState.selectPipeline(testProjectKey, 'pip1')).subscribe((pip: Pipeline) => {
            expect(pip).toBeTruthy();
            expect(pip.audits).toBeTruthy();
            expect(pip.name).toEqual('pip1');
            expect(pip.projectKey).toEqual(testProjectKey);
            expect(pip.audits.length).toEqual(1);
            expect(pip.audits[0].action).toEqual('update');
        });
    }));

    //  ------- Parameters --------- //
    it('add a parameter on pipeline', async(() => {
        const http = TestBed.get(HttpTestingController);
        let pipeline = new Pipeline();
        pipeline.name = 'pip1';
        pipeline.projectKey = testProjectKey;
        store.dispatch(new pipelinesActions.AddPipeline({
            projectKey: testProjectKey,
            pipeline
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/pipeline';
        })).flush(pipeline);
        store.selectOnce(PipelinesState).subscribe(state => {
            expect(Object.keys(state.pipelines).length).toEqual(1);
        });
        store.selectOnce(PipelinesState.selectPipeline(testProjectKey, 'pip1')).subscribe(pip => {
            expect(pip).toBeTruthy();
            expect(pip.name).toEqual('pip1');
            expect(pip.projectKey).toEqual(testProjectKey);
        });

        let parameter = new Parameter();
        parameter.name = 'testvar';
        parameter.type = 'string';
        parameter.value = 'myvalue';

        store.dispatch(new pipelinesActions.AddPipelineParameter({
            projectKey: testProjectKey,
            pipelineName: 'pip1',
            parameter
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/pipeline/pip1/parameter/testvar';
        })).flush(parameter);
        store.selectOnce(PipelinesState).subscribe(state => {
            expect(Object.keys(state.pipelines).length).toEqual(1);
        });
        store.selectOnce(PipelinesState.selectPipeline(testProjectKey, 'pip1')).subscribe((pip: Pipeline) => {
            expect(pip).toBeTruthy();
            expect(pip.name).toEqual('pip1');
            expect(pip.projectKey).toEqual(testProjectKey);
            expect(pip.parameters).toBeTruthy();
            expect(pip.parameters.length).toEqual(1);
            expect(pip.parameters[0].name).toEqual('testvar');
        });
    }));

    it('update a parameter on pipeline', async(() => {
        const http = TestBed.get(HttpTestingController);
        let pipeline = new Pipeline();
        pipeline.name = 'pip1';
        pipeline.projectKey = testProjectKey;
        store.dispatch(new pipelinesActions.AddPipeline({
            projectKey: testProjectKey,
            pipeline
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/pipeline';
        })).flush(pipeline);

        let parameter = new Parameter();
        parameter.name = 'testvar';
        parameter.type = 'string';
        parameter.value = 'myvalue';

        store.dispatch(new pipelinesActions.AddPipelineParameter({
            projectKey: testProjectKey,
            pipelineName: 'pip1',
            parameter
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/pipeline/pip1/parameter/testvar';
        })).flush(parameter);

        parameter.name = 'testvarrenamed';
        store.dispatch(new pipelinesActions.UpdatePipelineParameter({
            projectKey: testProjectKey,
            pipelineName: 'pip1',
            parameterName: 'testvar',
            parameter
        }));
        store.selectOnce(PipelinesState).subscribe(state => {
            expect(Object.keys(state.pipelines).length).toEqual(1);
        });
        store.selectOnce(PipelinesState.selectPipeline(testProjectKey, 'pip1')).subscribe((pip: Pipeline) => {
            expect(pip).toBeTruthy();
            expect(pip.name).toEqual('pip1');
            expect(pip.projectKey).toEqual(testProjectKey);
            expect(pip.parameters).toBeTruthy();
            expect(pip.parameters.length).toEqual(1);
            expect(pip.parameters[0].name).toEqual('testvarrenamed');
        });
    }));

    it('delete a parameter on pipeline', async(() => {
        const http = TestBed.get(HttpTestingController);
        let pipeline = new Pipeline();
        pipeline.name = 'pip1';
        pipeline.projectKey = testProjectKey;
        store.dispatch(new pipelinesActions.AddPipeline({
            projectKey: testProjectKey,
            pipeline
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/pipeline';
        })).flush(pipeline);

        let parameter = new Parameter();
        parameter.name = 'testvar';
        parameter.type = 'string';
        parameter.value = 'myvalue';

        store.dispatch(new pipelinesActions.AddPipelineParameter({
            projectKey: testProjectKey,
            pipelineName: 'pip1',
            parameter
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/pipeline/pip1/parameter/testvar';
        })).flush(parameter);

        store.dispatch(new pipelinesActions.DeletePipelineParameter({
            projectKey: testProjectKey,
            pipelineName: 'pip1',
            parameter
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/pipeline/pip1/parameter/testvar';
        })).flush(parameter);
        store.selectOnce(PipelinesState).subscribe(state => {
            expect(Object.keys(state.pipelines).length).toEqual(1);
        });
        store.selectOnce(PipelinesState.selectPipeline(testProjectKey, 'pip1')).subscribe((pip: Pipeline) => {
            expect(pip).toBeTruthy();
            expect(pip.name).toEqual('pip1');
            expect(pip.projectKey).toEqual(testProjectKey);
            expect(pip.parameters).toBeTruthy();
            expect(pip.parameters.length).toEqual(0);
        });
    }));


    //  ------- Workflow --------- //
    it('add a stage on pipeline', async(() => {
        const http = TestBed.get(HttpTestingController);
        let pipeline = new Pipeline();
        pipeline.name = 'pip1';
        pipeline.projectKey = testProjectKey;
        store.dispatch(new pipelinesActions.AddPipeline({
            projectKey: testProjectKey,
            pipeline
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/pipeline';
        })).flush(pipeline);
        store.selectOnce(PipelinesState).subscribe(state => {
            expect(Object.keys(state.pipelines).length).toEqual(1);
        });
        store.selectOnce(PipelinesState.selectPipeline(testProjectKey, 'pip1')).subscribe(pip => {
            expect(pip).toBeTruthy();
            expect(pip.name).toEqual('pip1');
            expect(pip.projectKey).toEqual(testProjectKey);
        });

        let stage = new Stage();
        stage.id = 1;
        stage.name = 'testStage';

        store.dispatch(new pipelinesActions.AddPipelineStage({
            projectKey: testProjectKey,
            pipelineName: 'pip1',
            stage
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/pipeline/pip1/stage';
        })).flush(<Pipeline>{
            name: 'pip1',
            projectKey: testProjectKey,
            stages: [stage],
        });
        store.selectOnce(PipelinesState).subscribe(state => {
            expect(Object.keys(state.pipelines).length).toEqual(1);
        });
        store.selectOnce(PipelinesState.selectPipeline(testProjectKey, 'pip1')).subscribe((pip: Pipeline) => {
            expect(pip).toBeTruthy();
            expect(pip.name).toEqual('pip1');
            expect(pip.projectKey).toEqual(testProjectKey);
            expect(pip.stages).toBeTruthy();
            expect(pip.stages.length).toEqual(1);
            expect(pip.stages[0].name).toEqual('testStage');
        });
    }));

    it('update a stage on pipeline', async(() => {
        const http = TestBed.get(HttpTestingController);
        let pipeline = new Pipeline();
        pipeline.name = 'pip1';
        pipeline.projectKey = testProjectKey;
        store.dispatch(new pipelinesActions.AddPipeline({
            projectKey: testProjectKey,
            pipeline
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/pipeline';
        })).flush(pipeline);

        let stage = new Stage();
        stage.id = 1;
        stage.name = 'testStage';

        store.dispatch(new pipelinesActions.AddPipelineStage({
            projectKey: testProjectKey,
            pipelineName: 'pip1',
            stage
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/pipeline/pip1/stage';
        })).flush(<Pipeline>{
            name: 'pip1',
            projectKey: testProjectKey,
            stages: [stage],
        });

        stage.name = 'testStageRenamed';
        store.dispatch(new pipelinesActions.UpdatePipelineStage({
            projectKey: testProjectKey,
            pipelineName: 'pip1',
            changes: stage
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/pipeline/pip1/stage/1';
        })).flush(<Pipeline>{
            name: 'pip1',
            projectKey: testProjectKey,
            stages: [stage],
        });
        store.selectOnce(PipelinesState).subscribe(state => {
            expect(Object.keys(state.pipelines).length).toEqual(1);
        });
        store.selectOnce(PipelinesState.selectPipeline(testProjectKey, 'pip1')).subscribe((pip: Pipeline) => {
            expect(pip).toBeTruthy();
            expect(pip.name).toEqual('pip1');
            expect(pip.projectKey).toEqual(testProjectKey);
            expect(pip.stages).toBeTruthy();
            expect(pip.stages.length).toEqual(1);
            expect(pip.stages[0].name).toEqual('testStageRenamed');
        });
    }));

    it('delete a stage on pipeline', async(() => {
        const http = TestBed.get(HttpTestingController);
        let pipeline = new Pipeline();
        pipeline.name = 'pip1';
        pipeline.projectKey = testProjectKey;
        store.dispatch(new pipelinesActions.AddPipeline({
            projectKey: testProjectKey,
            pipeline
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/pipeline';
        })).flush(pipeline);
        store.selectOnce(PipelinesState).subscribe(state => {
            expect(Object.keys(state.pipelines).length).toEqual(1);
        });
        store.selectOnce(PipelinesState.selectPipeline(testProjectKey, 'pip1')).subscribe(pip => {
            expect(pip).toBeTruthy();
            expect(pip.name).toEqual('pip1');
            expect(pip.projectKey).toEqual(testProjectKey);
        });

        let stage = new Stage();
        stage.id = 1;
        stage.name = 'testStage';

        store.dispatch(new pipelinesActions.AddPipelineStage({
            projectKey: testProjectKey,
            pipelineName: 'pip1',
            stage
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/pipeline/pip1/stage';
        })).flush(<Pipeline>{
            name: 'pip1',
            projectKey: testProjectKey,
            stages: [stage],
        });
        store.selectOnce(PipelinesState).subscribe(state => {
            expect(Object.keys(state.pipelines).length).toEqual(1);
        });
        store.selectOnce(PipelinesState.selectPipeline(testProjectKey, 'pip1')).subscribe((pip: Pipeline) => {
            expect(pip).toBeTruthy();
            expect(pip.name).toEqual('pip1');
            expect(pip.projectKey).toEqual(testProjectKey);
            expect(pip.stages).toBeTruthy();
            expect(pip.stages.length).toEqual(1);
            expect(pip.stages[0].name).toEqual('testStage');
        });

        store.dispatch(new pipelinesActions.DeletePipelineStage({
            projectKey: testProjectKey,
            pipelineName: 'pip1',
            stage
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/pipeline/pip1/stage/1';
        })).flush(<Pipeline>{
            name: 'pip1',
            projectKey: testProjectKey,
            stages: [],
        });
        store.selectOnce(PipelinesState.selectPipeline(testProjectKey, 'pip1')).subscribe((pip: Pipeline) => {
            expect(pip).toBeTruthy();
            expect(pip.name).toEqual('pip1');
            expect(pip.projectKey).toEqual(testProjectKey);
            expect(pip.stages).toBeTruthy();
            expect(pip.stages.length).toEqual(0);
        });
    }));



    it('add a job on pipeline', async(() => {
        const http = TestBed.get(HttpTestingController);
        let pipeline = new Pipeline();
        pipeline.name = 'pip1';
        pipeline.projectKey = testProjectKey;
        store.dispatch(new pipelinesActions.AddPipeline({
            projectKey: testProjectKey,
            pipeline
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/pipeline';
        })).flush(pipeline);
        store.selectOnce(PipelinesState).subscribe(state => {
            expect(Object.keys(state.pipelines).length).toEqual(1);
        });
        store.selectOnce(PipelinesState.selectPipeline(testProjectKey, 'pip1')).subscribe(pip => {
            expect(pip).toBeTruthy();
            expect(pip.name).toEqual('pip1');
            expect(pip.projectKey).toEqual(testProjectKey);
        });

        let stage = new Stage();
        stage.id = 1;
        stage.name = 'testStage';
        let job = new Job();
        job.pipeline_action_id = 2;
        job.action = new Action();
        job.action.name = 'testjob';

        store.dispatch(new pipelinesActions.AddPipelineStage({
            projectKey: testProjectKey,
            pipelineName: 'pip1',
            stage
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/pipeline/pip1/stage';
        })).flush(<Pipeline>{
            name: 'pip1',
            projectKey: testProjectKey,
            stages: [stage],
        });
        store.dispatch(new pipelinesActions.AddPipelineJob({
            projectKey: testProjectKey,
            pipelineName: 'pip1',
            stageId: stage.id,
            job
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/pipeline/pip1/stage/1/job';
        })).flush(<Pipeline>{
            name: 'pip1',
            projectKey: testProjectKey,
            stages: [Object.assign({}, stage, <Stage>{ jobs: [job] })],
        });


        store.selectOnce(PipelinesState).subscribe(state => {
            expect(Object.keys(state.pipelines).length).toEqual(1);
        });
        store.selectOnce(PipelinesState.selectPipeline(testProjectKey, 'pip1')).subscribe((pip: Pipeline) => {
            expect(pip).toBeTruthy();
            expect(pip.name).toEqual('pip1');
            expect(pip.projectKey).toEqual(testProjectKey);
            expect(pip.stages).toBeTruthy();
            expect(pip.stages.length).toEqual(1);
            expect(pip.stages[0].name).toEqual('testStage');
            expect(pip.stages[0].jobs).toBeTruthy();
            expect(pip.stages[0].jobs.length).toEqual(1);
            expect(pip.stages[0].jobs[0].pipeline_action_id).toEqual(2);
            expect(pip.stages[0].jobs[0].action.name).toEqual('testjob');
        });
    }));

    it('update a job on pipeline', async(() => {
        const http = TestBed.get(HttpTestingController);
        let pipeline = new Pipeline();
        pipeline.name = 'pip1';
        pipeline.projectKey = testProjectKey;
        store.dispatch(new pipelinesActions.AddPipeline({
            projectKey: testProjectKey,
            pipeline
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/pipeline';
        })).flush(pipeline);
        store.selectOnce(PipelinesState).subscribe(state => {
            expect(Object.keys(state.pipelines).length).toEqual(1);
        });
        store.selectOnce(PipelinesState.selectPipeline(testProjectKey, 'pip1')).subscribe(pip => {
            expect(pip).toBeTruthy();
            expect(pip.name).toEqual('pip1');
            expect(pip.projectKey).toEqual(testProjectKey);
        });

        let stage = new Stage();
        stage.id = 1;
        stage.name = 'testStage';
        let job = new Job();
        job.pipeline_action_id = 2;
        job.action = new Action();
        job.action.name = 'testjob';

        store.dispatch(new pipelinesActions.AddPipelineStage({
            projectKey: testProjectKey,
            pipelineName: 'pip1',
            stage
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/pipeline/pip1/stage';
        })).flush(<Pipeline>{
            name: 'pip1',
            projectKey: testProjectKey,
            stages: [stage],
        });
        store.dispatch(new pipelinesActions.AddPipelineJob({
            projectKey: testProjectKey,
            pipelineName: 'pip1',
            stageId: stage.id,
            job
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/pipeline/pip1/stage/1/job';
        })).flush(<Pipeline>{
            name: 'pip1',
            projectKey: testProjectKey,
            stages: [Object.assign({}, stage, <Stage>{ jobs: [job] })],
        });

        job.action.name = 'jobupdated';
        store.dispatch(new pipelinesActions.UpdatePipelineJob({
            projectKey: testProjectKey,
            pipelineName: 'pip1',
            stageId: stage.id,
            changes: job
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/pipeline/pip1/stage/1/job/2';
        })).flush(<Pipeline>{
            name: 'pip1',
            projectKey: testProjectKey,
            stages: [Object.assign({}, stage, <Stage>{ jobs: [job] })],
        });

        store.selectOnce(PipelinesState).subscribe(state => {
            expect(Object.keys(state.pipelines).length).toEqual(1);
        });
        store.selectOnce(PipelinesState.selectPipeline(testProjectKey, 'pip1')).subscribe((pip: Pipeline) => {
            expect(pip).toBeTruthy();
            expect(pip.name).toEqual('pip1');
            expect(pip.projectKey).toEqual(testProjectKey);
            expect(pip.stages).toBeTruthy();
            expect(pip.stages.length).toEqual(1);
            expect(pip.stages[0].name).toEqual('testStage');
            expect(pip.stages[0].jobs).toBeTruthy();
            expect(pip.stages[0].jobs.length).toEqual(1);
            expect(pip.stages[0].jobs[0].pipeline_action_id).toEqual(2);
            expect(pip.stages[0].jobs[0].action.name).toEqual('jobupdated');
        });
    }));

    it('delete a job on pipeline', async(() => {
        const http = TestBed.get(HttpTestingController);
        let pipeline = new Pipeline();
        pipeline.name = 'pip1';
        pipeline.projectKey = testProjectKey;
        store.dispatch(new pipelinesActions.AddPipeline({
            projectKey: testProjectKey,
            pipeline
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/pipeline';
        })).flush(pipeline);
        store.selectOnce(PipelinesState).subscribe(state => {
            expect(Object.keys(state.pipelines).length).toEqual(1);
        });
        store.selectOnce(PipelinesState.selectPipeline(testProjectKey, 'pip1')).subscribe(pip => {
            expect(pip).toBeTruthy();
            expect(pip.name).toEqual('pip1');
            expect(pip.projectKey).toEqual(testProjectKey);
        });

        let stage = new Stage();
        stage.id = 1;
        stage.name = 'testStage';
        let job = new Job();
        job.pipeline_action_id = 2;
        job.action = new Action();
        job.action.name = 'testjob';

        store.dispatch(new pipelinesActions.AddPipelineStage({
            projectKey: testProjectKey,
            pipelineName: 'pip1',
            stage
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/pipeline/pip1/stage';
        })).flush(<Pipeline>{
            name: 'pip1',
            projectKey: testProjectKey,
            stages: [stage],
        });
        store.dispatch(new pipelinesActions.AddPipelineJob({
            projectKey: testProjectKey,
            pipelineName: 'pip1',
            stageId: stage.id,
            job
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/pipeline/pip1/stage/1/job';
        })).flush(<Pipeline>{
            name: 'pip1',
            projectKey: testProjectKey,
            stages: [Object.assign({}, stage, <Stage>{ jobs: [job] })],
        });

        store.dispatch(new pipelinesActions.DeletePipelineJob({
            projectKey: testProjectKey,
            pipelineName: 'pip1',
            stageId: stage.id,
            job
        }));
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === '/project/test1/pipeline/pip1/stage/1/job/2';
        })).flush(<Pipeline>{
            name: 'pip1',
            projectKey: testProjectKey,
            stages: [Object.assign({}, stage, <Stage>{ jobs: [] })],
        });

        store.selectOnce(PipelinesState).subscribe(state => {
            expect(Object.keys(state.pipelines).length).toEqual(1);
        });
        store.selectOnce(PipelinesState.selectPipeline(testProjectKey, 'pip1')).subscribe((pip: Pipeline) => {
            expect(pip).toBeTruthy();
            expect(pip.name).toEqual('pip1');
            expect(pip.projectKey).toEqual(testProjectKey);
            expect(pip.stages).toBeTruthy();
            expect(pip.stages.length).toEqual(1);
            expect(pip.stages[0].name).toEqual('testStage');
            expect(pip.stages[0].jobs).toBeTruthy();
            expect(pip.stages[0].jobs.length).toEqual(0);
        });
    }));
});
