import {Component, Input, OnInit, DoCheck} from '@angular/core';
import {Pipeline} from '../../../../../../model/pipeline.model';
import {Stage} from '../../../../../../model/stage.model';
import {Prerequisite} from '../../../../../../model/prerequisite.model';
import {PrerequisiteEvent} from '../../../../../../shared/prerequisites/prerequisite.event.model';
import {ActionEvent} from '../../../../../../shared/action/action.event.model';
import {PipelineStore} from '../../../../../../service/pipeline/pipeline.store';
import {Project} from '../../../../../../model/project.model';
import {Job} from '../../../../../../model/job.model';
import {ToastService} from '../../../../../../shared/toast/ToastService';
import {TranslateService} from 'ng2-translate';

declare var _: any;

@Component({
    selector: 'app-pipeline-stage',
    templateUrl: './pipeline.stage.html',
    styleUrls: ['./pipeline.stage.scss']
})
export class PipelineStageComponent implements OnInit, DoCheck {

    selectedJob: Job;
    editableStage: Stage;
    availablePrerequisites: Array<Prerequisite>;
    currentStageID: number;

    @Input() pipeline: Pipeline;
    @Input() project: Project;

    @Input()
    set stage(data: Stage) {
        this.editableStage = _.cloneDeep(data);
        if (!this.editableStage.prerequisites) {
            this.editableStage.prerequisites = new Array<Prerequisite>();
        }
    }

    constructor(private _pipelineStore: PipelineStore, private _toast: ToastService, private _translate: TranslateService) {
    }

    ngOnInit(): void {
        this.initPrerequisites();
        if (this.editableStage && this.editableStage.jobs && this.editableStage.jobs.length > 0) {
            this.selectedJob = this.editableStage.jobs[0];
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
                this.selectedJob = this.editableStage.jobs[0];
            } else {
                this.selectedJob = undefined;
            }
            this.currentStageID = this.editableStage.id;
        }
    }

    private initPrerequisites() {
        if (!this.availablePrerequisites) {
            this.availablePrerequisites = new Array<Prerequisite>();
        }
        this.availablePrerequisites.push({
            parameter: 'git.branch',
            expected_value: ''
        });

        if (this.pipeline.parameters) {
            this.pipeline.parameters.forEach(p => {
                this.availablePrerequisites.push({
                    parameter: p.name,
                    expected_value: ''
                });
            });
        }
    }


    prerequisiteEvent(event: PrerequisiteEvent): void {
        this.editableStage.hasChanged = true;
        switch (event.type) {
            case 'add':
                if (!this.editableStage.prerequisites) {
                    this.editableStage.prerequisites = new Array<Prerequisite>();
                }

                let indexAdd = this.editableStage.prerequisites.findIndex(p => p.parameter === event.prerequisite.parameter);
                if (indexAdd === -1) {
                    this.editableStage.prerequisites.push(_.cloneDeep(event.prerequisite));
                }
                break;
            case 'delete':
                let indexDelete = this.editableStage.prerequisites.findIndex(p => p.parameter === event.prerequisite.parameter);
                if (indexDelete > -1) {
                    this.editableStage.prerequisites.splice(indexDelete, 1);
                }
                break;
        }
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
        let job: Job = _.cloneDeep(this.selectedJob);
        job.action = event.action;
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

    /**
     * Event on stage
     * @param type Type of event (update/delete)
     */
    stageEvent(type: string): void {
        switch (type) {
            case 'update':
                this._pipelineStore.updateStage(this.project.key, this.pipeline.name, this.editableStage).subscribe(() => {
                    this._toast.success('', this._translate.instant('stage_updated'));
                });
                break;
            case 'delete':
                this._pipelineStore.removeStage(this.project.key, this.pipeline.name, this.editableStage).subscribe(() => {
                    this._toast.success('', this._translate.instant('stage_deleted'));
                });
                break;
        }
    }
}
