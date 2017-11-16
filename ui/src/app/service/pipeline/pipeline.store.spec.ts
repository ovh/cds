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
import {HttpClientTestingModule, HttpTestingController} from '@angular/common/http/testing';
import {HttpRequest} from '@angular/common/http';
import {first} from 'rxjs/operators';

describe('CDS: pipeline Store', () => {

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

    it('Create and Delete Pipeline', async(() => {
        const pipelineStore = TestBed.get(PipelineStore);
        const http = TestBed.get(HttpTestingController);

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
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/pipeline'
        })).flush(pipeline1);

        // check get pipeline (get from cache)
        let checkedSinglePipeline = false;
        pipelineStore.getPipelines(projectKey, 'myPipeline').pipe(first()).subscribe(pips => {
            expect(pips.get(projectKey + '-' + 'myPipeline')).toBeTruthy();
            expect(pips.get(projectKey + '-' + 'myPipeline').name).toBe('myPipeline', 'Wrong pipeline name. Must be myPipeline');
            checkedSinglePipeline = true;
        });
        expect(checkedSinglePipeline).toBeTruthy('Need to get pipeline myPipeline');

        // check get pipeline not in cache
        let checkednotCachedPipeline = false;
        pipelineStore.getPipelines(projectKey, 'myPipeline2').pipe(first()).subscribe(pips => {
            expect(pips.get(projectKey + '-' + 'myPipeline2')).toBeFalsy();
            checkednotCachedPipeline = true;
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/pipeline/myPipeline2'
        })).flush(pipeline2);
        expect(checkednotCachedPipeline).toBeTruthy('Need to get pipeline myPipeline2');

        // Now in cache
        let checkedInCachedPipeline = false;
        pipelineStore.getPipelines(projectKey, 'myPipeline2').pipe(first()).subscribe(pips => {
            expect(pips.get(projectKey + '-' + 'myPipeline2')).toBeTruthy();
            checkedInCachedPipeline = true;
        });
        expect(checkedInCachedPipeline).toBeTruthy();

        // Pipeline deletion
        pipelineStore.deletePipeline(projectKey, 'myPipeline2').subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/pipeline/myPipeline2'
        })).flush(null);

        let checkedDeletedPipeline = false;
        pipelineStore.getPipelines(projectKey, 'myPipeline2').pipe(first()).subscribe(pips => {
            checkedDeletedPipeline = true;
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/pipeline/myPipeline2'
        })).flush(pipeline2);
        expect(checkedDeletedPipeline).toBeTruthy('Need to get pipeline myPipeline');

        http.verify();
    }));

    it('Update pipeline', async(() => {
        const pipelineStore = TestBed.get(PipelineStore);
        const http = TestBed.get(HttpTestingController);

        let pip1 = new Pipeline();
        pip1.name = 'myPipeline';

        let pipUp = new Pipeline();
        pipUp.name = 'myPipelineUpdate1';

        let projectKey = 'key1';

        // Create pipeline
        let p = createPipeline('myPipeline');
        pipelineStore.createPipeline(projectKey, p).subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/pipeline'
        })).flush(pip1);

        // Update
        p.name = 'myPipelineUpdate1';
        pipelineStore.updatePipeline(projectKey, 'myPipeline', p).subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/pipeline/myPipeline'
        })).flush(pipUp);

        // check get pipeline
        let checkedPipeline = false;
        pipelineStore.getPipelines(projectKey, 'myPipelineUpdate1').subscribe(pips => {
            expect(pips.get(projectKey + '-' + 'myPipelineUpdate1')).toBeTruthy();
            expect(pips.get(projectKey + '-' + 'myPipelineUpdate1').name)
                .toBe('myPipelineUpdate1', 'Wrong pipeline name. Must be myPipelineUpdate1');
            checkedPipeline = true;
        }).unsubscribe();
        expect(checkedPipeline).toBeTruthy('Need to get pipeline myPipelineUpdate1');

        http.verify();
    }));

    it('should create/update and delete a stage', async(() => {
        const pipelineStore = TestBed.get(PipelineStore);
        const http = TestBed.get(HttpTestingController);

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
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/pipeline'
        })).flush(pip1);

        // ADD STAGE
        let s: Stage = new Stage();
        s.name = 'stage1';
        s.id = 1;
        pipelineStore.addStage(projectKey, 'myPipeline', s).subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/pipeline/myPipeline/stage'
        })).flush(pipAddStage);

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

        pipelineStore.updateStage(projectKey, 'myPipeline', s).subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/pipeline/myPipeline/stage/1'
        })).flush(pipUpStage);

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
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/pipeline/myPipeline/stage/1'
        })).flush(pipDelStage);

        let checkStageDelete = false;
        pipelineStore.getPipelines(projectKey, 'myPipeline').subscribe(pips => {
            expect(pips.get(projectKey + '-' + 'myPipeline')).toBeTruthy();
            expect(pips.get(projectKey + '-' + 'myPipeline').stages.length).toBe(0, 'Must have 0 stage');
            checkStageDelete = true;
        }).unsubscribe();
        expect(checkStageDelete).toBeTruthy();

        http.verify();
    }));

    it('should create/update and delete a job', async(() => {
        const pipelineStore = TestBed.get(PipelineStore);
        const http = TestBed.get(HttpTestingController);

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
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/pipeline'
        })).flush(pip);

        // ADD Job
        let j = new Job();
        let a: Action = new Action();
        a.name = 'action1';
        j.action = a;
        j.pipeline_action_id = 0;
        pipelineStore.addJob(projectKey, 'myPipeline', 1, j).subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/pipeline/myPipeline/stage/1/job'
        })).flush(pipAddJob);


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

        pipelineStore.updateJob(projectKey, 'myPipeline', 1, j).subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/pipeline/myPipeline/stage/1/job/0'
        })).flush(pipUpJob);

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
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/pipeline/myPipeline/stage/1/job/0'
        })).flush(pipDelJob);

        let checkJobDelete = false;
        pipelineStore.getPipelines(projectKey, 'myPipeline').pipe(first()).subscribe(pips => {
            expect(pips.get(projectKey + '-' + 'myPipeline')).toBeTruthy();
            expect(pips.get(projectKey + '-' + 'myPipeline').stages.length).toBe(1, 'Must have10 stage');
            expect(pips.get(projectKey + '-' + 'myPipeline').stages[0].jobs.length).toBe(0);
            checkJobDelete = true;
        });
        expect(checkJobDelete).toBeTruthy();

        http.verify();
    }));

    it('should add/update/delete a permission', async(() => {
        const pipelineStore = TestBed.get(PipelineStore);
        const http = TestBed.get(HttpTestingController);

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
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/pipeline'
        })).flush(pip);

        let gp: GroupPermission = new GroupPermission();
        gp.permission = 0;
        gp.group = new Group();
        gp.group.name = 'grp';

        pipelineStore.addPermission(proj.key, pip.name, gp).subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/pipeline/myPipeline/group'
        })).flush(pipAddGroup);

        // check get pipeline
        let checkedAddPermission = false;
        pipelineStore.getPipelines(proj.key, 'myPipeline').pipe(first()).subscribe(apps => {
            expect(apps.get(proj.key + '-myPipeline').last_modified).toBe(123, 'Pip lastModified date must have been updated');
            expect(apps.get(proj.key + '-myPipeline').groups.length).toBe(1, 'A group must have been added');
            expect(apps.get(proj.key + '-myPipeline').groups[0].permission).toBe(7, 'Permission must be 7');
            checkedAddPermission = true;
        });
        expect(checkedAddPermission).toBeTruthy('Need pipeline to be updated');

        pipelineStore.updatePermission(proj.key, pip.name, gp).subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/pipeline/myPipeline/group/grp'
        })).flush(pipUpGroup);

        // check get pipeline
        let checkedUpdatePermission = false;
        pipelineStore.getPipelines(proj.key, 'myPipeline').pipe(first()).subscribe(apps => {
            expect(apps.get(proj.key + '-myPipeline').last_modified).toBe(456, 'Pip lastModified date must have been updated');
            expect(apps.get(proj.key + '-myPipeline').groups.length).toBe(1, 'Pip must have 1 group');
            expect(apps.get(proj.key + '-myPipeline').groups[0].permission).toBe(4, 'Group permission must be 4');
            checkedUpdatePermission = true;
        });
        expect(checkedUpdatePermission).toBeTruthy('Need pipeline to be updated');

        pipelineStore.removePermission(proj.key, pip.name, gp).subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/pipeline/myPipeline/group/grp'
        })).flush(pipDelGroup);

        // check get pipeline
        let checkedDeletePermission = false;
        pipelineStore.getPipelines(proj.key, 'myPipeline').pipe(first()).subscribe(apps => {
            expect(apps.get(proj.key + '-myPipeline').last_modified).toBe(789, 'Pip lastModified date must have been updated');
            expect(apps.get(proj.key + '-myPipeline').groups.length).toBe(0, 'Ouo must have 0 group');
            checkedDeletePermission = true;
        });
        expect(checkedDeletePermission).toBeTruthy('Need pipeline to be updated');

        http.verify();
    }));

    it('should add/update/delete a parameter', async(() => {
        const pipelineStore = TestBed.get(PipelineStore);
        const http = TestBed.get(HttpTestingController);

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
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/pipeline'
        })).flush(pip);

        let param: Parameter = new Parameter();
        param.name = 'foo';
        param.type = 'string';
        param.description = 'my description';
        param.value = 'bar';


        pipelineStore.addParameter(proj.key, pip.name, param).subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/pipeline/myPipeline/parameter/foo'
        })).flush(pipAddParam);

        // check get pipeline
        let checkedAddParam = false;
        pipelineStore.getPipelines(proj.key, 'myPipeline').pipe(first()).subscribe(apps => {
            expect(apps.get(proj.key + '-myPipeline').last_modified).toBe(123, 'Pip lastModified date must have been updated');
            expect(apps.get(proj.key + '-myPipeline').parameters.length).toBe(1, 'A parameter must have been added');
            expect(apps.get(proj.key + '-myPipeline').parameters[0].name).toBe('foo', 'Name must be foo');
            checkedAddParam = true;
        });
        expect(checkedAddParam).toBeTruthy('Need pipeline to be updated');


        pipelineStore.updateParameter(proj.key, pip.name, param).subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/pipeline/myPipeline/parameter/foo'
        })).flush(pipUpParam);

        // check get pipeline
        let checkedUpdateParam = false;
        pipelineStore.getPipelines(proj.key, 'myPipeline').pipe(first()).subscribe(apps => {
            expect(apps.get(proj.key + '-myPipeline').last_modified).toBe(456, 'Pip lastModified date must have been updated');
            expect(apps.get(proj.key + '-myPipeline').parameters.length).toBe(1, 'Pip must have 1 group');
            expect(apps.get(proj.key + '-myPipeline').parameters[0].name).toBe('fooUpdated', 'Name must be fooUpdated');
            checkedUpdateParam = true;
        });
        expect(checkedUpdateParam).toBeTruthy('Need pipeline to be updated');


        pipelineStore.removeParameter(proj.key, pip.name, param).subscribe(() => {
        });
        http.expectOne(((req: HttpRequest<any>) => {
            return req.url === 'foo.bar/project/key1/pipeline/myPipeline/parameter/foo'
        })).flush(pipDelParam);

        // check get pipeline
        let checkedDeleteParam = false;
        pipelineStore.getPipelines(proj.key, 'myPipeline').pipe(first()).subscribe(apps => {
            expect(apps.get(proj.key + '-myPipeline').last_modified).toBe(789, 'Pip lastModified date must have been updated');
            expect(apps.get(proj.key + '-myPipeline').parameters.length).toBe(0, 'Pip must have 0 parameter');
            checkedDeleteParam = true;
        }).unsubscribe();
        expect(checkedDeleteParam).toBeTruthy('Need pipeline to be updated');

        http.verify();
    }));

    function createPipeline(name: string): Pipeline {
        let pip: Pipeline = new Pipeline();
        pip.name = name;
        return pip;
    }
});
