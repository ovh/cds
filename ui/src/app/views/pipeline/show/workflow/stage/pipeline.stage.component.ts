import {Component, Input, OnInit, DoCheck} from '@angular/core';
import {Pipeline} from '../../../../../model/pipeline.model';
import {Stage} from '../../../../../model/stage.model';
import {Prerequisite} from '../../../../../model/prerequisite.model';
import {ActionEvent} from '../../../../../shared/action/action.event.model';
import {PipelineStore} from '../../../../../service/pipeline/pipeline.store';
import {Project} from '../../../../../model/project.model';
import {Job} from '../../../../../model/job.model';
import {ToastService} from '../../../../../shared/toast/ToastService';
import {TranslateService} from 'ng2-translate';
import {cloneDeep} from 'lodash';

@Component({
    selector: 'app-pipeline-stage',
    templateUrl: './pipeline.stage.html',
    styleUrls: ['./pipeline.stage.scss']
})
export class PipelineStageComponent implements OnInit, DoCheck {

    selectedJob: Job;
    editableStage: Stage;
    currentStageID: number;

    @Input() pipeline: Pipeline;
    @Input() project: Project;
    @Input() suggest: Array<string>;

    @Input()
    set stage(data: Stage) {
        this.editableStage = cloneDeep(data);
        if (!this.editableStage.prerequisites) {
            this.editableStage.prerequisites = new Array<Prerequisite>();
        }
    }

    constructor(private _pipelineStore: PipelineStore, private _toast: ToastService, private _translate: TranslateService) {
    }

    ngOnInit(): void {
        if (this.editableStage && this.editableStage.jobs && this.editableStage.jobs.length > 0) {
            this.selectJob(this.editableStage.jobs[0]);
        }
        this.currentStageID = this.editableStage.id;
    }

    /**
     * Update selected Stage On pipeline update.
     * Do not work with ngOnChange.
     */
    ngDoCheck() {
        if (this.currentStageID !== this.editableStage.id) {
            if (this.selectedJob && this.editableStage.jobs && this.editableStage.jobs.length > 0) {
                this.selectJob(this.editableStage.jobs[0]);
            } else {
                this.selectedJob = undefined;
            }
            this.currentStageID = this.editableStage.id;
        }
    }



    selectJob(j: Job) {
        this.selectedJob = j;
        this.selectedJob.action.enabled = j.enabled;
    }

    addJob(): void {
        let jobToAdd = new Job();
        jobToAdd.action.name = 'New Job';
        jobToAdd.enabled = true;
        this._pipelineStore.addJob(this.project.key, this.pipeline.name, this.editableStage.id, jobToAdd).subscribe(() => {
            this._toast.success('', this._translate.instant('stage_job_added'));
        });
    }

    /**
     * Manage action from jobs
     * @param event
     */
    jobEvent(event: ActionEvent): void {
        let job: Job = cloneDeep(this.selectedJob);
        job.action = event.action;
        job.action.loading = true;
        job.enabled = event.action.enabled;
        if (job.action.actions) {
            job.action.actions.forEach(a => {
               if (a.parameters) {
                   a.parameters.forEach(p => {
                      p.value = p.value.toString();
                   });
               }
            });
        }
        if (job.action.parameters) {
            job.action.parameters.forEach(p => {
                p.value = p.value.toString();
            });
        }

        switch (event.type) {
            case 'update':
                this._pipelineStore.updateJob(this.project.key, this.pipeline.name, this.editableStage.id, job).subscribe(() => {
                    this._toast.success('', this._translate.instant('stage_job_updated'));
                    job.action.loading = false;
                    job.action.hasChanged = false;
                });
                break;
            case 'delete':
                this._pipelineStore.removeJob(this.project.key, this.pipeline.name, this.editableStage.id, job).subscribe(() => {
                    this._toast.success('', this._translate.instant('stage_job_deleted'));
                    this.selectedJob = undefined;
                });
                break;
        }
    }
}
