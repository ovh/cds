/* tslint:disable:no-unused-variable */
import {TestBed, fakeAsync, getTestBed, tick} from '@angular/core/testing';
import {RouterTestingModule} from '@angular/router/testing';
import {Injector} from '@angular/core';
import {TranslateService, TranslateLoader, TranslateParser} from '@ngx-translate/core';
import {SharedModule} from '../../../shared/shared.module';
import {Observable} from 'rxjs/Observable';
import {CDSWorker} from '../../../shared/worker/worker';
import {PipelineRunWorkflowComponent} from './workflow.component';
import {PipelineBuild, PipelineBuildJob} from '../../../model/pipeline.model';
import {Stage} from '../../../model/stage.model';
import {Job} from '../../../model/job.model';
import {Action} from '../../../model/action.model';
import {ApplicationRunModule} from '../application.run.module';
import {NotificationService} from '../../../service/notification/notification.service';
import {ApplicationPipelineService} from '../../../service/application/pipeline/application.pipeline.service';
import {HttpClientTestingModule} from '@angular/common/http/testing';

describe('CDS: Pipeline Run Workflow', () => {

    let injector: Injector;

    beforeEach(() => {
        TestBed.configureTestingModule({
            declarations: [],
            providers: [
                TranslateService,
                TranslateLoader,
                TranslateParser,
                NotificationService,
                ApplicationPipelineService
            ],
            imports: [
                ApplicationRunModule,
                RouterTestingModule.withRoutes([]),
                SharedModule,
                HttpClientTestingModule
            ]
        });

        injector = getTestBed();
    });

    afterEach(() => {
        injector = undefined;
    });

    it('should load component', fakeAsync(() => {
        // Create component
        let fixture = TestBed.createComponent(PipelineRunWorkflowComponent);
        let component = fixture.debugElement.componentInstance;
        expect(component).toBeTruthy();

        fixture.detectChanges();
        tick(250);

        fixture.componentInstance.build = getPipelineBuild();

        fixture.detectChanges();
        tick(250);

        expect(fixture.componentInstance.currentBuild.version).toBe(123);

        let compiled = fixture.debugElement.nativeElement;
        expect(compiled.querySelectorAll('.ban.grey.icon').length).toBe(2);
        expect(compiled.querySelectorAll('.check.green.icon').length).toBe(1);
        expect(compiled.querySelectorAll('.remove.red.icon').length).toBe(1);
        expect(compiled.querySelectorAll('.wait.blue.icon').length).toBe(1);
        expect(compiled.querySelectorAll('.ui.small.active.inline.blue.loader').length).toBe(1);


        // Select a job
        let job = fixture.componentInstance.currentBuild.stages[0].jobs[0];
        fixture.componentInstance.selectedJob(job, fixture.componentInstance.currentBuild.stages[0]);
        expect(fixture.componentInstance.selectedPipJob.job.pipeline_action_id).toBe(job.pipeline_action_id);

        fixture.detectChanges();
        tick(250);
    }));
});

function getPipelineBuild(): PipelineBuild {
    let pb = new PipelineBuild();
    pb.version = 123;
    pb.stages = new Array<Stage>();
    let s: Stage = new Stage();
    s.name = 'Stage 1';
    s.id = 1;
    s.jobs = new Array<Job>();
    s.jobs.push(createJob(1, 'jobBuilding'));
    s.jobs.push(createJob(2, 'jobSkipped'));
    s.jobs.push(createJob(3, 'jobDisabled'));
    s.jobs.push(createJob(4, 'jobSuccess'));
    s.jobs.push(createJob(5, 'jobFail'));
    s.jobs.push(createJob(6, 'jobWaiting'));
    s.builds = new Array<PipelineBuildJob>();
    s.builds.push(createPipelineBuildJob(1, 'jobBuilding', 'Building'));
    s.builds.push(createPipelineBuildJob(2, 'jobSkipped', 'Skipped'));
    s.builds.push(createPipelineBuildJob(3, 'jobDisabled', 'Disabled'));
    s.builds.push(createPipelineBuildJob(4, 'jobSuccess', 'Success'));
    s.builds.push(createPipelineBuildJob(5, 'jobFail', 'Fail'));
    s.builds.push(createPipelineBuildJob(6, 'jobWaiting', 'Waiting'));
    pb.stages.push(s);
    return pb;
}

function createJob(id: number, name: string): Job {
    let j: Job = new Job();
    j.action = new Action();
    j.pipeline_action_id = id;
    j.action.name = name;
    return j;
}

function createPipelineBuildJob(id: number, name: string, status: string): PipelineBuildJob {
    let pbJob = new PipelineBuildJob();
    let j: Job = new Job();
    j.action = new Action();
    j.pipeline_action_id = id;
    j.action.name = name;
    pbJob.job = j;
    pbJob.status = status;
    return pbJob;
}
