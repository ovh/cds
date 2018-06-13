/* tslint:disable:no-unused-variable */
import {async, TestBed} from '@angular/core/testing';
import {APP_BASE_HREF} from '@angular/common';
import {AppModule} from '../../app.module';
import {RouterModule} from '@angular/router';
import {PipelineStore} from './pipeline.store';
import {Pipeline} from '../../model/pipeline.model';
import {Stage} from '../../model/stage.model';
import {Action} from '../../model/action.model';
import {Job} from '../../model/job.model';
import {Project} from '../../model/project.model';
import {Group, GroupPermission} from '../../model/group.model';
import {Parameter} from '../../model/parameter.model';
import {first} from 'rxjs/operators';
import {PipelineService} from './pipeline.service';
import {Observable} from 'rxjs/Observable';

describe('CDS: pipeline Store', () => {

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [],
            providers: [
                {provide: APP_BASE_HREF, useValue: '/'},
                {provide: PipelineService, useClass: MockPipelineService}
            ],
            imports: [
                AppModule,
                RouterModule
            ]
        });

    });

    it('Create and Delete Pipeline', async(() => {
        const pipelineStore = TestBed.get(PipelineStore);

        let pipeline1 = new Pipeline();
        pipeline1.name = 'myPipeline';

        let pipeline2 = new Pipeline();
        pipeline2.name = 'myPipeline2';

        let projectKey = 'key1';

        // Create 1st pipeline
        let checkPipelineCreated = false;
        pipelineStore.createPipeline(projectKey, createPipeline('myPipeline')).subscribe(res => {
            expect(res.name).toBe('myPipeline', 'Wrong pipeline name');
            checkPipelineCreated = true;
        });

        // check get pipeline (get from cache)
        let checkedSinglePipeline = false;
        pipelineStore.getPipelines(projectKey, 'myPipeline').pipe(first()).subscribe(pips => {
            expect(pips.get(projectKey + '-' + 'myPipeline')).toBeTruthy();
            expect(pips.get(projectKey + '-' + 'myPipeline').name).toBe('myPipeline', 'Wrong pipeline name. Must be myPipeline');
            checkedSinglePipeline = true;
        });
        expect(checkedSinglePipeline).toBeTruthy('Need to get pipeline myPipeline');


        let checkedInCachedPipeline = false;
        pipelineStore.getPipelines(projectKey, 'myPipeline2').pipe(first()).subscribe(pips => {
            expect(pips.get(projectKey + '-' + 'myPipeline2')).toBeTruthy();
            checkedInCachedPipeline = true;
        });
        expect(checkedInCachedPipeline).toBeTruthy();

        // Pipeline deletion
        pipelineStore.deletePipeline(projectKey, 'myPipeline2').subscribe(() => {
        });

        let checkedDeletedPipeline = false;
        pipelineStore.getPipelines(projectKey, 'myPipeline2').pipe(first()).subscribe(pips => {
            checkedDeletedPipeline = true;
        });
        expect(checkedDeletedPipeline).toBeTruthy('Need to get pipeline myPipeline');
    }));

    it('Update pipeline', async(() => {
        const pipelineStore = TestBed.get(PipelineStore);

        let pip1 = new Pipeline();
        pip1.name = 'myPipeline';

        let pipUp = new Pipeline();
        pipUp.name = 'myPipelineUpdate1';

        let projectKey = 'key1';

        // Create pipeline
        let p = createPipeline('myPipeline');
        pipelineStore.createPipeline(projectKey, p).subscribe(() => {
        });

        // Update
        p.name = 'myPipelineUpdate1';
        pipelineStore.updatePipeline(projectKey, 'myPipeline', p).subscribe(() => {
        });

        // check get pipeline
        let checkedPipeline = false;
        pipelineStore.getPipelines(projectKey, 'myPipelineUpdate1').subscribe(pips => {
            expect(pips.get(projectKey + '-' + 'myPipelineUpdate1')).toBeTruthy();
            expect(pips.get(projectKey + '-' + 'myPipelineUpdate1').name)
                .toBe('myPipelineUpdate1', 'Wrong pipeline name. Must be myPipelineUpdate1');
            checkedPipeline = true;
        }).unsubscribe();
        expect(checkedPipeline).toBeTruthy('Need to get pipeline myPipelineUpdate1');
    }));

    it('should create/update and delete a stage', async(() => {
        const pipelineStore = TestBed.get(PipelineStore);

        let pip1 = new Pipeline();
        pip1.name = 'myPipeline';

        let pipAddStage = new Pipeline();
        pipAddStage.stages = new Array<Stage>();
        let s1 = new Stage();
        s1.name = 'stage1';
        pipAddStage.stages.push(s1);

        let pipUpStage = new Pipeline();
        pipUpStage.stages = new Array<Stage>();
        let s2 = new Stage();
        s2.name = 'stage1Updated';
        pipUpStage.stages.push(s2);

        let pipDelStage = new Pipeline();
        pipDelStage.stages = new Array<Stage>();

        let projectKey = 'key1';

        // Create 1st pipeline
        let checkPipelineCreated = false;
        pipelineStore.createPipeline(projectKey, createPipeline('myPipeline')).subscribe(res => {
            expect(res.name).toBe('myPipeline', 'Wrong pipeline name');
            checkPipelineCreated = true;
        });

        // ADD STAGE
        let s: Stage = new Stage();
        s.name = 'stage1';
        s.id = 1;
        pipelineStore.addStage(projectKey, 'myPipeline', s).subscribe(() => {
        });

        // check get pipeline (get from cache)
        let checkStageAdd = false;
        pipelineStore.getPipelines(projectKey, 'myPipeline').pipe(first()).subscribe(pips => {
            expect(pips.get(projectKey + '-' + 'myPipeline')).toBeTruthy();
            expect(pips.get(projectKey + '-' + 'myPipeline').stages.length).toBe(1, 'Must have 1 stage');
            expect(pips.get(projectKey + '-' + 'myPipeline').stages[0].name).toBe('stage1', 'Wrong stage');
            checkStageAdd = true;
        });
        expect(checkStageAdd).toBeTruthy();

        // UPDATE STAGE
        s.name = 'stage1Updated';
        pipelineStore.updateStage(projectKey, 'myPipeline', s).subscribe(() => {
        });

        let checkStageUpdate = false;
        pipelineStore.getPipelines(projectKey, 'myPipeline').pipe(first()).subscribe(pips => {
            expect(pips.get(projectKey + '-' + 'myPipeline')).toBeTruthy();
            expect(pips.get(projectKey + '-' + 'myPipeline').stages.length).toBe(1, 'Must have 1 stage');
            expect(pips.get(projectKey + '-' + 'myPipeline').stages[0].name).toBe('stage1Updated', 'Wrong stage');
            checkStageUpdate = true;
        });
        expect(checkStageUpdate).toBeTruthy();

        // DELETE STAGE
        pipelineStore.removeStage(projectKey, 'myPipeline', s).subscribe(() => {
        });

        let checkStageDelete = false;
        pipelineStore.getPipelines(projectKey, 'myPipeline').subscribe(pips => {
            expect(pips.get(projectKey + '-' + 'myPipeline')).toBeTruthy();
            expect(pips.get(projectKey + '-' + 'myPipeline').stages.length).toBe(0, 'Must have 0 stage');
            checkStageDelete = true;
        }).unsubscribe();
        expect(checkStageDelete).toBeTruthy();
    }));

    it('should create/update and delete a job', async(() => {
        const pipelineStore = TestBed.get(PipelineStore);

        let pip = new Pipeline();
        pip.name = 'myPipeline';
        pip.stages = new Array<Stage>();
        let s = new Stage();
        s.id = 1;
        pip.stages.push(s);

        let pipAddJob = new Pipeline();
        pipAddJob.name = 'myPipeline';
        pipAddJob.stages = new Array<Stage>();
        let sAdd = new Stage();
        sAdd.id = 1;
        sAdd.jobs = new Array<Job>();
        let jAdd = new Job();
        jAdd.action = new Action();
        jAdd.action.name = 'action1';
        sAdd.jobs.push(jAdd);
        pipAddJob.stages.push(sAdd);

        let pipUpJob = new Pipeline();
        pipUpJob.name = 'myPipeline';
        pipUpJob.stages = new Array<Stage>();
        let sUp = new Stage();
        sUp.id = 1;
        sUp.jobs = new Array<Job>();
        let jUp = new Job();
        jUp.action = new Action();
        jUp.action.name = 'action1Updated';
        sUp.jobs.push(jUp);
        pipUpJob.stages.push(sUp);

        let pipDelJob = new Pipeline();
        pipDelJob.name = 'myPipeline';
        pipDelJob.stages = new Array<Stage>();
        let sDel = new Stage();
        sDel.id = 1;
        sDel.jobs = new Array<Job>();
        pipDelJob.stages.push(sDel);

        let projectKey = 'key1';

        // Create 1st pipeline
        let checkPipelineCreated = false;
        pipelineStore.createPipeline(projectKey, createPipeline('myPipeline')).subscribe(res => {
            expect(res.name).toBe('myPipeline', 'Wrong pipeline name');
            checkPipelineCreated = true;
        });

        // ADD Job
        let j = new Job();
        let a: Action = new Action();
        a.name = 'action1';
        j.action = a;
        j.pipeline_action_id = 0;
        pipelineStore.addJob(projectKey, 'myPipeline', 1, j).subscribe(() => {
        });


        let checkJobAdd = false;
        pipelineStore.getPipelines(projectKey, 'myPipeline').pipe(first()).subscribe(pips => {
            expect(pips.get(projectKey + '-' + 'myPipeline')).toBeTruthy();
            expect(pips.get(projectKey + '-' + 'myPipeline').stages.length).toBe(1, 'Must have 1 stage');
            expect(pips.get(projectKey + '-' + 'myPipeline').stages[0].jobs.length).toBe(1, 'Must have 1 action');
            expect(pips.get(projectKey + '-' + 'myPipeline').stages[0].jobs[0].action.name).toBe('action1', 'Wrong action');
            checkJobAdd = true;
        });
        expect(checkJobAdd).toBeTruthy();

        // UPDATE JOB

        j.action.name = 'action1Updated';
        pipelineStore.updateJob(projectKey, 'myPipeline', 1, j).subscribe(() => {
        });

        let checkJobUpdate = false;
        pipelineStore.getPipelines(projectKey, 'myPipeline').pipe(first()).subscribe(pips => {
            expect(pips.get(projectKey + '-' + 'myPipeline')).toBeTruthy();
            expect(pips.get(projectKey + '-' + 'myPipeline').stages.length).toBe(1, 'Must have 1 stage');
            expect(pips.get(projectKey + '-' + 'myPipeline').stages[0].jobs.length).toBe(1, 'Must have 1 action');
            expect(pips.get(projectKey + '-' + 'myPipeline').stages[0].jobs[0].action.name).toBe('action1Updated', 'Wrong action');
            checkJobUpdate = true;
        });
        expect(checkJobUpdate).toBeTruthy();

        // DELETE JOB
        pipelineStore.removeJob(projectKey, 'myPipeline', 1, j).subscribe(() => {
        });

        let checkJobDelete = false;
        pipelineStore.getPipelines(projectKey, 'myPipeline').pipe(first()).subscribe(pips => {
            expect(pips.get(projectKey + '-' + 'myPipeline')).toBeTruthy();
            expect(pips.get(projectKey + '-' + 'myPipeline').stages.length).toBe(1, 'Must have10 stage');
            expect(pips.get(projectKey + '-' + 'myPipeline').stages[0].jobs.length).toBe(0);
            checkJobDelete = true;
        });
        expect(checkJobDelete).toBeTruthy();
    }));

    it('should add/update/delete a permission', async(() => {
        const pipelineStore = TestBed.get(PipelineStore);

        let grp1 = new Group();
        grp1.name = 'grp';

        let pip = new Pipeline();
        pip.name = 'myPipeline';
        pip.last_modified = 0;

        let pipAddGroup = new Pipeline();
        pipAddGroup.name = 'myPipeline';
        pipAddGroup.last_modified = 123;
        pipAddGroup.groups = new Array<GroupPermission>();
        let gpAdd = new GroupPermission();
        gpAdd.group = grp1;
        gpAdd.permission = 7;
        pipAddGroup.groups.push(gpAdd);

        let pipUpGroup = new Pipeline();
        pipUpGroup.name = 'myPipeline';
        pipUpGroup.last_modified = 456;
        pipUpGroup.groups = new Array<GroupPermission>();
        let gpUp = new GroupPermission();
        gpUp.group = grp1;
        gpUp.permission = 4;
        pipUpGroup.groups.push(gpUp);

        let pipDelGroup = new Pipeline();
        pipDelGroup.name = 'myPipeline';
        pipDelGroup.last_modified = 789;
        pipDelGroup.groups = new Array<GroupPermission>();

        let proj: Project = new Project();
        proj.key = 'key1';

        // Create pipeline
        let pipeline = createPipeline('myPipeline');
        pipelineStore.createPipeline(proj.key, pipeline).subscribe(() => {
        });

        let gp: GroupPermission = new GroupPermission();
        gp.permission = 7;
        gp.group = new Group();
        gp.group.name = 'grp';

        pipelineStore.addPermission(proj.key, pip.name, gp).subscribe(() => {
        });

        // check get pipeline
        let checkedAddPermission = false;
        pipelineStore.getPipelines(proj.key, 'myPipeline').pipe(first()).subscribe(apps => {
            expect(apps.get(proj.key + '-myPipeline').groups.length).toBe(1, 'A group must have been added');
            expect(apps.get(proj.key + '-myPipeline').groups[0].permission).toBe(7, 'Permission must be 7');
            checkedAddPermission = true;
        });
        expect(checkedAddPermission).toBeTruthy('Need pipeline to be updated');

        gp.permission = 4;
        pipelineStore.updatePermission(proj.key, pip.name, gp).subscribe(() => {
        });

        // check get pipeline
        let checkedUpdatePermission = false;
        pipelineStore.getPipelines(proj.key, 'myPipeline').pipe(first()).subscribe(apps => {
            expect(apps.get(proj.key + '-myPipeline').groups.length).toBe(1, 'Pip must have 1 group');
            expect(apps.get(proj.key + '-myPipeline').groups[0].permission).toBe(4, 'Group permission must be 4');
            checkedUpdatePermission = true;
        });
        expect(checkedUpdatePermission).toBeTruthy('Need pipeline to be updated');

        pipelineStore.removePermission(proj.key, pip.name, gp).subscribe(() => {
        });

        // check get pipeline
        let checkedDeletePermission = false;
        pipelineStore.getPipelines(proj.key, 'myPipeline').pipe(first()).subscribe(apps => {
            expect(apps.get(proj.key + '-myPipeline').groups.length).toBe(0, 'Ouo must have 0 group');
            checkedDeletePermission = true;
        });
        expect(checkedDeletePermission).toBeTruthy('Need pipeline to be updated');
    }));

    it('should add/update/delete a parameter', async(() => {
        const pipelineStore = TestBed.get(PipelineStore);

        let pip = new Pipeline();
        pip.name = 'myPipeline';
        pip.last_modified = 0;

        let pipAddParam = new Pipeline();
        pipAddParam.name = 'myPipeline';
        pipAddParam.last_modified = 123;
        pipAddParam.parameters = new Array<Parameter>();
        let pa = new Parameter();
        pa.name = 'foo';
        pipAddParam.parameters.push(pa);

        let pipUpParam = new Pipeline();
        pipUpParam.name = 'myPipeline';
        pipUpParam.last_modified = 456;
        pipUpParam.parameters = new Array<Parameter>();
        let pa2 = new Parameter();
        pa2.name = 'fooUpdated';
        pipUpParam.parameters.push(pa2);

        let pipDelParam = new Pipeline();
        pipDelParam.name = 'myPipeline';
        pipDelParam.last_modified = 789;
        pipDelParam.parameters = new Array<Parameter>();

        let proj: Project = new Project();
        proj.key = 'key1';

        // Create pipeline
        let pipeline = createPipeline('myPipeline');
        pipelineStore.createPipeline(proj.key, pipeline).subscribe(() => {
        });

        let param: Parameter = new Parameter();
        param.name = 'foo';
        param.type = 'string';
        param.description = 'my description';
        param.value = 'bar';


        pipelineStore.addParameter(proj.key, pip.name, param).subscribe(() => {
        });

        // check get pipeline
        let checkedAddParam = false;
        pipelineStore.getPipelines(proj.key, 'myPipeline').pipe(first()).subscribe(apps => {
            expect(apps.get(proj.key + '-myPipeline').parameters.length).toBe(1, 'A parameter must have been added');
            expect(apps.get(proj.key + '-myPipeline').parameters[0].name).toBe('foo', 'Name must be foo');
            checkedAddParam = true;
        });
        expect(checkedAddParam).toBeTruthy('Need pipeline to be updated');


        param.name = 'fooUpdated';
        pipelineStore.updateParameter(proj.key, pip.name, param).subscribe(() => {
        });

        // check get pipeline
        let checkedUpdateParam = false;
        pipelineStore.getPipelines(proj.key, 'myPipeline').pipe(first()).subscribe(apps => {
            expect(apps.get(proj.key + '-myPipeline').parameters.length).toBe(1, 'Pip must have 1 group');
            expect(apps.get(proj.key + '-myPipeline').parameters[0].name).toBe('fooUpdated', 'Name must be fooUpdated');
            checkedUpdateParam = true;
        });
        expect(checkedUpdateParam).toBeTruthy('Need pipeline to be updated');


        pipelineStore.removeParameter(proj.key, pip.name, param).subscribe(() => {
        });

        // check get pipeline
        let checkedDeleteParam = false;
        pipelineStore.getPipelines(proj.key, 'myPipeline').pipe(first()).subscribe(apps => {
            expect(apps.get(proj.key + '-myPipeline').parameters.length).toBe(0, 'Pip must have 0 parameter');
            checkedDeleteParam = true;
        }).unsubscribe();
        expect(checkedDeleteParam).toBeTruthy('Need pipeline to be updated');
    }));

    function createPipeline(name: string): Pipeline {
        let pip: Pipeline = new Pipeline();
        pip.name = name;
        return pip;
    }

    class MockPipelineService {

        getPipeline(key: string, pipName: string): Observable<Pipeline> {
            let pip = new Pipeline();
            pip.name = pipName;
            return Observable.of(pip);
        }
        createPipeline(key: string, pipeline: Pipeline): Observable<Pipeline> {
            return Observable.of(pipeline);
        };

        updatePipeline(key: string, oldName: string, pipeline: Pipeline): Observable<Pipeline> {
            return Observable.of(pipeline);
        }

        deletePipeline(key: string, pipName: string): Observable<boolean> {
            return Observable.of(true);
        }

        addJob(key: string, pipName: string, stageID: number, job: Job): Observable<Pipeline> {
            let pip = new Pipeline();
            pip.name = pipName;
            pip.stages = new Array<Stage>();
            let s = new Stage();
            s.id = stageID;
            s.jobs = new Array<Job>();
            s.jobs.push(job);
            pip.stages.push(s);
            return Observable.of(pip);
        }

        updateJob(key: string, pipName: string, stageID: number, job: Job): Observable<Pipeline> {
            let pip = new Pipeline();
            pip.name = pipName;
            pip.stages = new Array<Stage>();
            let s = new Stage();
            s.id = stageID;
            s.jobs = new Array<Job>();
            s.jobs.push(job);
            pip.stages.push(s);
            return Observable.of(pip);
        }

        removeJob(key: string, pipName: string, stageID: number, job: Job): Observable<Pipeline> {
            let pip = new Pipeline();
            pip.name = pipName;
            pip.stages = new Array<Stage>();
            let s = new Stage();
            s.id = stageID;
            s.jobs = new Array<Job>();
            pip.stages.push(s);
            return Observable.of(pip);
        }

        addPermission(key: string, pipName: string, gp: GroupPermission): Observable<Pipeline> {
            let pip = new Pipeline();
            pip.name = pipName;
            pip.groups = new Array<GroupPermission>();
            pip.groups.push(gp);
            return Observable.of(pip);
        }

        updatePermission(key: string, pipName: string, gp: GroupPermission): Observable<Pipeline> {
            let pip = new Pipeline();
            pip.name = pipName;
            pip.groups = new Array<GroupPermission>();
            pip.groups.push(gp);
            return Observable.of(pip);
        }

        removePermission(key: string, pipName: string, gp: GroupPermission): Observable<Pipeline> {
            let pip = new Pipeline();
            pip.name = pipName;
            pip.groups = new Array<GroupPermission>();
            return Observable.of(pip);
        }

        addParameter(key: string, pipName: string, param: Parameter): Observable<Pipeline> {
            let pip = new Pipeline();
            pip.name = pipName;
            pip.parameters = new Array<Parameter>();
            pip.parameters.push(param);
            return Observable.of(pip);
        }

        updateParameter(key: string, pipName: string, param: Parameter): Observable<Pipeline> {
            let pip = new Pipeline();
            pip.name = pipName;
            pip.parameters = new Array<Parameter>();
            pip.parameters.push(param);
            return Observable.of(pip);
        }

        removeParameter(key: string, pipName: string, param: Parameter): Observable<Pipeline> {
            let pip = new Pipeline();
            pip.name = pipName;
            pip.parameters = new Array<Parameter>();
            return Observable.of(pip);
        }

        insertStage(key: string, pipName: string, stage: Stage): Observable<Pipeline> {
            let pip = new Pipeline();
            pip.name = pipName;
            pip.stages = new Array<Stage>();
            pip.stages.push(stage);
            return Observable.of(pip);
        }

        updateStage(key: string, pipName: string, stage: Stage): Observable<Pipeline> {
            let pip = new Pipeline();
            pip.name = pipName;
            pip.stages = new Array<Stage>();
            pip.stages.push(stage);
            return Observable.of(pip);
        }

        deleteStage(key: string, pipName: string, stage: Stage): Observable<Pipeline> {
            let pip = new Pipeline();
            pip.name = pipName;
            pip.stages = new Array<Stage>();
            return Observable.of(pip);
        }
    }
});
