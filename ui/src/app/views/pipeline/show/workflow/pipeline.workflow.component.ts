import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnDestroy, OnInit, ViewChild } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { Job } from 'app/model/job.model';
import { AllKeys } from 'app/model/keys.model';
import { Pipeline } from 'app/model/pipeline.model';
import { Project } from 'app/model/project.model';
import { Stage } from 'app/model/stage.model';
import { KeyService } from 'app/service/keys/keys.service';
import { PipelineCoreService } from 'app/service/pipeline/pipeline.core.service';
import { VariableService } from 'app/service/variable/variable.service';
import { ActionEvent } from 'app/shared/action/action.event.model';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { ToastService } from 'app/shared/toast/ToastService';
import * as pipelineActions from 'app/store/pipelines.action';
import { PipelinesStateModel } from 'app/store/pipelines.state';
import cloneDeep from 'lodash-es/cloneDeep';
import { SemanticModalComponent } from 'ng-semantic/ng-semantic';
import { DragulaService } from 'ng2-dragula-sgu';
import { Subscription } from 'rxjs';
import { finalize, first } from 'rxjs/operators';

@Component({
    selector: 'app-pipeline-workflow',
    templateUrl: './pipeline.workflow.html',
    styleUrls: ['./pipeline.workflow.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class PipelineWorkflowComponent implements OnInit, OnDestroy {
    @Input() project: Project;
    @Input() editMode: boolean;
    @Input() readOnly: boolean;

    @Input()
    set currentPipeline(data: Pipeline) {
        this.pipeline = cloneDeep(data);

        this.originalPipeline = this.pipeline;
        if (!this.pipeline) {
            return;
        }

        if (!this.selectedJob || this.pipeline.forceRefresh) {
            this.selectDefaultJob();
        }

        if (!this.pipeline.stages) {
            this.pipeline.stages = new Array<Stage>();
        }

        if (this.pipeline.preview) {
            this.previewMode = true;
            this.pipeline = this.pipeline.preview;
            this.selectDefaultJob();
            this._pipCoreService.toggleAsCodeEditor({ open: false, save: false });
        } else {
            let jobFound: Job;
            let stageFound: Stage;
            if (this.editMode) {
                if (this.selectedStage) {
                    if (data && data.stages) {
                        stageFound = data.stages.find((stage) => stage.ref === this.selectedStage.ref);
                    } else {
                        delete this.selectedStage;
                    }
                    if (this.selectedJob && stageFound) {
                        jobFound = stageFound.jobs.find(j => j.ref === this.selectedJob.ref)
                    } else {
                        delete this.selectedJob;
                    }
                }
            } else {
                if (this.selectedStage) {
                    if (data && data.stages) {
                        stageFound = this.selectedStage && data.stages && data.stages.find((stage) => stage.id === this.selectedStage.id);
                        if (stageFound && this.selectedJob && stageFound.jobs) {
                            jobFound = stageFound.jobs.find((job) => job.pipeline_action_id === this.selectedJob.pipeline_action_id);
                        } else {
                            delete this.selectedJob;
                        }
                    } else {
                        delete this.selectedStage;
                    }
                }
            }
            if (jobFound && stageFound) {
                this.selectJob(jobFound, stageFound);
            } else if (stageFound) {
                this.selectStage(stageFound);
            }
            if (stageFound && !jobFound) {
                this.selectDefaultJobInStage(this.selectedStage);
            }
            if (!stageFound && !jobFound) {
                this.selectDefaultJob();
            }
            this.previewMode = false;
        }
    }
    @Input() queryParams: {};

    @ViewChild('editStageModal')
    editStageModal: SemanticModalComponent;

    pipeline: Pipeline;
    selectedStage: Stage;
    selectedJob: Job;
    suggest: Array<string>;
    originalPipeline: Pipeline;
    keys: AllKeys;
    previewMode = false;

    loadingStage = false;

    dragulaSubscription: Subscription;

    constructor(
        private store: Store,
        private _toast: ToastService,
        private _dragularService: DragulaService,
        private _translate: TranslateService,
        private _varService: VariableService,
        private _pipCoreService: PipelineCoreService,
        private _keyService: KeyService,
        private _cd: ChangeDetectorRef
    ) {
        this._dragularService.createGroup('bag-stage', {
            moves(el, source, handle) {
                return handle.classList.contains('move');
            },
            accepts(el, target, source, sibling) {
                if (sibling === null) {
                    return false;
                }
                return true;
            }
        });
        this.dragulaSubscription = this._dragularService.drop('bag-stage').subscribe(({ el, source }) => {
            setTimeout(() => {
                let stageMovedBuildOrder = Number(el.id.replace('step', ''));
                let stageMoved: Stage;
                for (let i = 0; i < this.pipeline.stages.length; i++) {
                    if (this.pipeline.stages[i].build_order === stageMovedBuildOrder) {
                        stageMoved = this.pipeline.stages[i];
                        stageMoved.build_order = i + 1;
                        break;
                    }
                }
                this.store.dispatch(new pipelineActions.MovePipelineStage({
                    projectKey: this.project.key,
                    pipeline: this.pipeline,
                    stage: stageMoved
                })).pipe(finalize(() => {
                    this._cd.markForCheck();
                })).subscribe(() => {
                    if (!this.editMode) {
                        this._toast.success('', this._translate.instant('pipeline_stage_moved'))
                    }
                });
            });
        });
    }

    selectDefaultJob() {
        if (this.pipeline.stages && this.pipeline.stages.length &&
            this.pipeline.stages[0].jobs && this.pipeline.stages[0].jobs.length) {
            this.selectJob(this.pipeline.stages[0].jobs[0], this.pipeline.stages[0]);
        }
    }

    selectDefaultJobInStage(stage: Stage) {
        if (stage.jobs && stage.jobs.length) {
            this.selectJob(stage.jobs[0], stage);
        }
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

        this._keyService.getAllKeys(this.project.key).subscribe(k => {
            this.keys = k;
        });

        this._varService.getContextVariable(this.project.key, this.pipeline.id)
            .pipe(first(), finalize(() => this._cd.markForCheck())).subscribe(s => this.suggest = s);
    }

    addStageAndJob(): void {
        let s = new Stage();
        s.enabled = true;
        if (!this.pipeline.stages) {
            this.pipeline.stages = new Array<Stage>();
        }
        s.build_order = this.pipeline.stages.length + 1;
        s.name = 'Stage ' + s.build_order;

        this.store.dispatch(new pipelineActions.AddPipelineStage({
            projectKey: this.project.key,
            pipelineName: this.pipeline.name,
            stage: s
        })).subscribe(st => {
            if (!st['pipelines']) {
                return;
            }
            let pipStateModel = <PipelinesStateModel>st['pipelines'];
            if (!pipStateModel.editMode) {
                this._toast.success('', this._translate.instant('stage_added'));
                this.selectStage(pipStateModel.pipeline.stages[pipStateModel.pipeline.stages.length - 1]);
            } else {
                this._toast.info('', this._translate.instant('pipeline_ascode_updated'));
                this.selectStage(pipStateModel.editPipeline.stages[pipStateModel.editPipeline.stages.length - 1]);
            }
            this.addJob(this.selectedStage);
            this._cd.markForCheck();
        });
    }

    addJob(s: Stage): void {
        let jobToAdd = new Job();
        jobToAdd.action.name = 'New Job';
        jobToAdd.enabled = true;
        jobToAdd.pipeline_stage_id = s.id;
        jobToAdd.action.type = 'Joined';
        this.store.dispatch(new pipelineActions.AddPipelineJob({
            projectKey: this.project.key,
            pipelineName: this.pipeline.name,
            stage: s,
            job: jobToAdd
        })).subscribe((st) => {
            if (!st['pipelines']) {
                return;
            }
            let pipStateModel = <PipelinesStateModel>st['pipelines'];
            if (!pipStateModel.editMode) {
                this._toast.success('', this._translate.instant('stage_job_added'));
                this.selectStage(pipStateModel.pipeline.stages.find((stage) => this.selectedStage.id === stage.id));
            } else {
                this._toast.info('', this._translate.instant('pipeline_ascode_updated'));
                this.selectStage(pipStateModel.editPipeline.stages.find((stage) => this.selectedStage.ref === stage.ref));
            }
            if (this.selectedStage) {
                this.selectJob(this.selectedStage.jobs[this.selectedStage.jobs.length - 1], this.selectedStage);
                this._cd.markForCheck();
            }
        });
    }

    /**
     * Event on stage
     *
     * @param type Type of event (update/delete)
     */
    stageEvent(type: string): void {
        this.loadingStage = true;
        switch (type) {
            case 'update':
                if (this.selectedStage.conditions.lua_script && this.selectedStage.conditions.lua_script !== '') {
                    this.selectedStage.conditions.plain = null;
                } else {
                    this.selectedStage.conditions.lua_script = '';
                    if (this.selectedStage.conditions.plain) {
                        this.selectedStage.conditions.plain.forEach(cc => {
                            cc.value = cc.value.toString();
                        });
                    }
                }

                this.store.dispatch(new pipelineActions.UpdatePipelineStage({
                    projectKey: this.project.key,
                    pipelineName: this.pipeline.name,
                    changes: this.selectedStage
                })).pipe(finalize(() => {
                    this.loadingStage = false;
                    this._cd.markForCheck();
                }))
                    .subscribe(() => {
                        if (!this.editMode) {
                            this._toast.success('', this._translate.instant('stage_updated'));
                        } else {
                            this._toast.info('', this._translate.instant('pipeline_ascode_updated'));
                        }
                        this.editStageModal.hide();
                    });
                break;
            case 'delete':
                this.store.dispatch(new pipelineActions.DeletePipelineStage({
                    projectKey: this.project.key,
                    pipelineName: this.pipeline.name,
                    stage: this.selectedStage
                })).pipe(finalize(() => {
                    this.loadingStage = false;
                    this._cd.markForCheck();
                }))
                    .subscribe(() => {
                        if (!this.editMode) {
                            this._toast.success('', this._translate.instant('stage_deleted'));
                        } else {
                            this._toast.info('', this._translate.instant('pipeline_ascode_updated'));
                        }
                        this.editStageModal.hide();
                        this.selectedStage = null;
                        this.selectedJob = null;
                    });
                break;
        }
    }

    openEditModal(s: Stage): void {
        this.selectedStage = cloneDeep(s);
        if (this.editStageModal) {
            this.editStageModal.show({ autofocus: false, closable: false, observeChanges: true });
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
     *
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
                this.store.dispatch(new pipelineActions.UpdatePipelineJob({
                    projectKey: this.project.key,
                    pipelineName: this.pipeline.name,
                    stage: this.selectedStage,
                    changes: job
                })).pipe(finalize(() => {
                    job.action.loading = false;
                    job.action.hasChanged = false;
                    this._cd.markForCheck();
                })).subscribe(() => {
                    if (!this.editMode) {
                        this._toast.success('', this._translate.instant('stage_job_updated'));
                        this.selectedStage = this.pipeline.stages
                            .find((s) => this.selectedStage.id === s.id) || this.selectedStage;
                        this.selectedJob = this.selectedStage.jobs
                            .find((j) => this.selectedJob.action.id === j.action.id) || this.selectedJob;
                    } else {
                        this._toast.info('', this._translate.instant('pipeline_ascode_updated'));
                        this.selectedStage = this.pipeline.stages
                            .find((s) => this.selectedStage.ref === s.ref) || this.selectedStage;
                        this.selectedJob = this.selectedStage.jobs
                            .find((j) => this.selectedJob.action.id === j.action.id) || this.selectedJob;
                    }
                });
                break;
            case 'delete':
                this.store.dispatch(new pipelineActions.DeletePipelineJob({
                    projectKey: this.project.key,
                    pipelineName: this.pipeline.name,
                    stage: this.selectedStage,
                    job: this.selectedJob
                })).pipe(finalize(() => {
                    this.selectedJob = undefined;
                    this._cd.detectChanges();
                }))
                    .subscribe(() => {
                        if (!this.editMode) {
                            this._toast.success('', this._translate.instant('stage_job_deleted'));
                        } else {
                            this._toast.info('', this._translate.instant('pipeline_ascode_updated'));
                        }
                    });
                break;
        }
    }

    savePreview() {
        this.previewMode = false;
        this._pipCoreService.toggleAsCodeEditor({ open: false, save: true });
    }

    showAsCodeEditor() {
        this._pipCoreService.toggleAsCodeEditor({ open: true, save: false });
    }
}
