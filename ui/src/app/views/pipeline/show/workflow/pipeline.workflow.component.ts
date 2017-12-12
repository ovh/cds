import {Component, DoCheck, Input, OnInit, OnDestroy, ViewChild} from '@angular/core';
import {Pipeline} from '../../../../model/pipeline.model';
import {Project} from '../../../../model/project.model';
import {PipelineStore} from '../../../../service/pipeline/pipeline.store';
import {Stage} from '../../../../model/stage.model';
import {ToastService} from '../../../../shared/toast/ToastService';
import {TranslateService} from '@ngx-translate/core';
import {SemanticModalComponent} from 'ng-semantic/ng-semantic';
import {VariableService} from '../../../../service/variable/variable.service';
import {cloneDeep} from 'lodash';
import {Job} from '../../../../model/job.model';
import {ActionEvent} from '../../../../shared/action/action.event.model';
import {DragulaService} from 'ng2-dragula';
import {PermissionValue} from '../../../../model/permission.model';
import {first} from 'rxjs/operators';

@Component({
    selector: 'app-pipeline-workflow',
    templateUrl: './pipeline.workflow.html',
    styleUrls: ['./pipeline.workflow.scss']
})
export class PipelineWorkflowComponent implements OnInit, OnDestroy {

    @Input() project: Project;
    @Input('currentPipeline')
    set currentPipeline (data: Pipeline) {
        this.pipeline = data;

        if (!this.pipeline) {
            return;
        }

        if (this.pipeline.stages && this.pipeline.stages.length > 0 && !this.selectedStage) {
            this.selectStage(this.pipeline.stages[0]);
        }
        if (this.selectedStage && Array.isArray(this.selectedStage.jobs) &&
            this.selectedStage.jobs.length > 0 && !this.selectedJob) {

            this.selectJob(this.selectedStage.jobs[0], this.selectedStage);
        }
        if (!this.pipeline.stages) {
            this.pipeline.stages = new Array<Stage>();
        }
    }
    @Input() queryParams: {};

    @ViewChild('editStageModal')
    editStageModal: SemanticModalComponent;

    pipeline: Pipeline;
    selectedStage: Stage;
    selectedJob: Job;
    suggest: Array<string>;
    permissionValue = PermissionValue;

    loadingStage = false;

    constructor(private _pipelineStore: PipelineStore, private _toast: ToastService, private _dragularService: DragulaService,
                private _translate: TranslateService, private _varService: VariableService) {
        this._dragularService.setOptions('bag-stage', {
            moves: function (el, source, handle) {
                return handle.classList.contains('move');
            },
            accepts: function (el, target, source, sibling) {
                if (sibling === null) {
                    return false;
                }
                return true;
            }
        });
        this._dragularService.drop.subscribe(v => {
            setTimeout(() => {
                if (v[0] !== 'bag-stage') {
                    return;
                }
                let stageMovedBuildOrder = Number(v[1].id.replace('step', ''));
                let stageMoved: Stage;
                for (let i = 0; i < this.pipeline.stages.length; i++) {
                    if (this.pipeline.stages[i].build_order === stageMovedBuildOrder) {
                        stageMoved = this.pipeline.stages[i];
                        stageMoved.build_order = i + 1;
                        break;
                    }
                }
                this._pipelineStore.moveStage(this.project.key, this.pipeline.name, stageMoved).subscribe(() => {
                    this._toast.success('', this._translate.instant('pipeline_stage_moved'));
                });
            });
        });
    }

    ngOnDestroy() {
        this._dragularService.destroy('bag-stage');
    }

    /**
     * Init selected stage + pipeline date
     */
    ngOnInit() {
        if (this.queryParams && this.queryParams['stage']) {
            this.selectedStage = cloneDeep(this.pipeline.stages.find(s => s.name === this.queryParams['stage']));
        }
        if (this.pipeline.stages && this.pipeline.stages.length > 0 && !this.selectedStage) {
            this.selectedStage = cloneDeep(this.pipeline.stages[0]);
        }

        this._varService.getContextVariable(this.project.key, this.pipeline.id).pipe(first()).subscribe(s => this.suggest = s);

    }

    addStageAndJob(): void {
        let s = new Stage();
        s.enabled = true;
        if (!this.pipeline.stages) {
            this.pipeline.stages = new Array<Stage>();
        }

        s.name = 'Stage ' + (this.pipeline.stages.length + 1);
        this._pipelineStore.addStage(this.project.key, this.pipeline.name, s).subscribe(p => {
            this._toast.success('', this._translate.instant('stage_added'));
            this.selectedStage = p.stages[p.stages.length - 1];
            this.addJob(this.selectedStage);
        });
    }

    addJob(s: Stage): void {
        let jobToAdd = new Job();
        jobToAdd.action.name = 'New Job';
        jobToAdd.enabled = true;
        this._pipelineStore.addJob(this.project.key, this.pipeline.name, s.id, jobToAdd).subscribe((pip) => {
            this._toast.success('', this._translate.instant('stage_job_added'));

            let currentStage = pip.stages.find((stage) => this.selectedStage.id === stage.id);
            if (currentStage) {
                this.selectJob(currentStage.jobs[currentStage.jobs.length - 1], this.selectedStage);
            }
        });
    }

    /**
     * Event on stage
     * @param type Type of event (update/delete)
     */
    stageEvent(type: string): void {
        this.loadingStage = true;
        switch (type) {
            case 'update':
                this._pipelineStore.updateStage(this.project.key, this.pipeline.name, this.selectedStage).subscribe(() => {
                    this._toast.success('', this._translate.instant('stage_updated'));
                    this.loadingStage = false;
                    this.editStageModal.hide();
                }, () => {
                this.loadingStage = false;
            });
                break;
            case 'delete':
                this._pipelineStore.removeStage(this.project.key, this.pipeline.name, this.selectedStage).subscribe(() => {
                    this._toast.success('', this._translate.instant('stage_deleted'));
                    this.loadingStage = false;
                    this.editStageModal.hide();
                    delete this.selectedStage;
                    delete this.selectedJob;
                }, () => {
                    this.loadingStage = false;
                });
                break;
        }
    }

    openEditModal(s: Stage): void {
        this.selectedStage = cloneDeep(s);
        if (this.editStageModal) {
            this.editStageModal.show({autofocus: false, closable: false, observeChanges: true});
        }
    }

    selectStage(s: Stage): void {
        this.selectedStage = cloneDeep(s);
    }

    selectJob(j: Job, s: Stage): void {
        this.selectStage(s);
        this.selectedJob = cloneDeep(j);
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
                this._pipelineStore.updateJob(this.project.key, this.pipeline.name, this.selectedStage.id, job).subscribe(() => {
                    this._toast.success('', this._translate.instant('stage_job_updated'));
                    job.action.loading = false;
                    job.action.hasChanged = false;

                }, () => {
                    job.action.loading = false;
                });
                break;
            case 'delete':
                this._pipelineStore.removeJob(this.project.key, this.pipeline.name, this.selectedStage.id, job).subscribe(() => {
                    this._toast.success('', this._translate.instant('stage_job_deleted'));
                    this.selectedJob = undefined;
                }, () => {
                    job.action.loading = false;
                });
                break;
        }
    }
}
