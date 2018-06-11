import {Component, Input, OnInit, OnDestroy, ViewChild} from '@angular/core';
import {Pipeline} from '../../../../model/pipeline.model';
import {Project} from '../../../../model/project.model';
import {PipelineStore} from '../../../../service/pipeline/pipeline.store';
import {PipelineCoreService} from '../../../../service/pipeline/pipeline.core.service';
import {Stage} from '../../../../model/stage.model';
import {ToastService} from '../../../../shared/toast/ToastService';
import {AutoUnsubscribe} from '../../../../shared/decorator/autoUnsubscribe';
import {TranslateService} from '@ngx-translate/core';
import {SemanticModalComponent} from 'ng-semantic/ng-semantic';
import {VariableService} from '../../../../service/variable/variable.service';
import {cloneDeep} from 'lodash';
import {Job} from '../../../../model/job.model';
import {ActionEvent} from '../../../../shared/action/action.event.model';
import {DragulaService} from 'ng2-dragula';
import {PermissionValue} from '../../../../model/permission.model';
import {first} from 'rxjs/operators';
import {Subscription} from 'rxjs';

@Component({
    selector: 'app-pipeline-workflow',
    templateUrl: './pipeline.workflow.html',
    styleUrls: ['./pipeline.workflow.scss']
})
@AutoUnsubscribe()
export class PipelineWorkflowComponent implements OnInit, OnDestroy {

    @Input() project: Project;
    @Input('currentPipeline')
    set currentPipeline (data: Pipeline) {
        this.pipeline = data;

        if (!this.pipeline) {
            return;
        }

        if (!this.selectedJob || this.pipeline.forceRefresh) {
            this.selectDefaultJob();
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
    originalPipeline: Pipeline;
    previewMode = false;

    loadingStage = false;

    pipelinePreviewSubscription: Subscription;
    asCodeEditorSubscription: Subscription;
    dragulaSubscription: Subscription;

    constructor(private _pipelineStore: PipelineStore, private _toast: ToastService, private _dragularService: DragulaService,
                private _translate: TranslateService, private _varService: VariableService, private _pipCoreService: PipelineCoreService) {
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
        this.dragulaSubscription = this._dragularService.drop.subscribe(v => {
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

        this.pipelinePreviewSubscription = this._pipCoreService.getPipelinePreview()
          .subscribe((pipPreview) => {
              if (pipPreview != null) {
                  this.originalPipeline = this.pipeline;
                  this.pipeline = null;
                  this.pipeline = Object.assign({}, this.originalPipeline, {
                      stages: pipPreview.stages,
                      previewMode: pipPreview.previewMode,
                      forceRefresh: pipPreview.forceRefresh
                  });
                  this.selectDefaultJob();
                  this.previewMode = true;
                  this._pipCoreService.toggleAsCodeEditor({open: false, save: false});
              } else if (this.originalPipeline != null) {
                  this.previewMode = false;
                  this.pipeline = this.originalPipeline;
                  this.selectDefaultJob();
              }
          });
    }

    selectDefaultJob() {
        if (this.pipeline.stages && this.pipeline.stages.length &&
              this.pipeline.stages[0].jobs && this.pipeline.stages[0].jobs.length) {
            this.selectJob(this.pipeline.stages[0].jobs[0], this.pipeline.stages[0]);
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
                this._pipelineStore.updateJob(this.project.key, this.pipeline.name, this.selectedStage.id, job).subscribe((pip) => {
                    this._toast.success('', this._translate.instant('stage_job_updated'));
                    job.action.loading = false;
                    job.action.hasChanged = false;
                    this.pipeline = pip;
                    this.selectedStage = this.pipeline.stages.find((s) => this.selectedStage.id === s.id) || this.selectedStage;
                    this.selectedJob = this.selectedStage.jobs.find((j) => this.selectedJob.action.id === j.action.id) || this.selectedJob;
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

    savePreview() {
        this.previewMode = false;
        this._pipCoreService.toggleAsCodeEditor({open: false, save: true});
    }

    showAsCodeEditor() {
        this._pipCoreService.toggleAsCodeEditor({open: true, save: false});
    }
}
