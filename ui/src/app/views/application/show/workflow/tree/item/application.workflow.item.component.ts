import {Component, Input, ViewChild} from '@angular/core';
import {WorkflowItem} from '../../../../../../model/application.workflow.model';
import {Application} from '../../../../../../model/application.model';
import {ApplicationPipelineService} from '../../../../../../service/application/pipeline/application.pipeline.service';
import {Router} from '@angular/router';
import {PipelineRunRequest, PipelineBuild, Pipeline} from '../../../../../../model/pipeline.model';
import {SemanticModalComponent} from 'ng-semantic/ng-semantic';
import {Project} from '../../../../../../model/project.model';
import {Parameter} from '../../../../../../model/parameter.model';
import {PipelineStore} from '../../../../../../service/pipeline/pipeline.store';
import {Environment} from '../../../../../../model/environment.model';
import {Trigger} from '../../../../../../model/trigger.model';
import {ApplicationStore} from '../../../../../../service/application/application.store';
import {ToastService} from '../../../../../../shared/toast/ToastService';
import {TranslateService} from 'ng2-translate';
import {Scheduler} from '../../../../../../model/scheduler.model';
import {Hook} from '../../../../../../model/hook.model';

declare var _: any;

@Component({
    selector: 'app-application-workflow-item',
    templateUrl: './application.workflow.item.html',
    styleUrls: ['./application.workflow.item.scss']
})
export class ApplicationWorkflowItemComponent {

    @Input() project: Project;
    @Input() workflowItem: WorkflowItem;
    @Input() orientation: string;
    @Input() application: Application;
    @Input() applicationFilter: any;

    @ViewChild('launchModal')
    launchModal: SemanticModalComponent;

    // Manual Launch data
    launchGitParams: Array<Parameter>;
    launchPipelineParams: Array<Parameter>;
    launchParentBuildNumber = 0;
    launchOldBuilds: Array<PipelineBuild>;

    // Triggers modals
    @ViewChild('editTriggerModal')
    editTriggerModal: SemanticModalComponent;
    @ViewChild('createTriggerModal')
    createTriggerModal: SemanticModalComponent;
    triggerInModal: Trigger;
    triggerLoading = false;

    // scheduler
    @ViewChild('createSchedulerModal')
    createSchedulerModal: SemanticModalComponent;
    newScheduler: Scheduler;

    constructor(private _router: Router, private _appPipService: ApplicationPipelineService, private _pipStore: PipelineStore,
                private _appStore: ApplicationStore, private _toast: ToastService, private _translate: TranslateService) {
    }


    runPipeline(): void {
        // If no parents and have parameters without value, go to manual launch
        if (this.workflowItem.trigger.manual
            || (Pipeline.hasParameterWithoutValue(this.workflowItem.pipeline) && !this.workflowItem.parent)) {
            return this.runWithParameters();
        }

        let parentBranch: string;
        let currentBranch: string = this.applicationFilter.branch;

        let runRequest: PipelineRunRequest = new PipelineRunRequest();

        // Set env
        runRequest.env = this.workflowItem.environment;

        // Set parent information
        if (this.workflowItem.parent) {
            runRequest.parent_application_id = this.workflowItem.parent.application_id;
            runRequest.parent_build_number = this.workflowItem.parent.buildNumber;
            runRequest.parent_environment_id = this.workflowItem.parent.environment_id;
            runRequest.parent_pipeline_id = this.workflowItem.parent.pipeline_id;

            runRequest.parameters.push(...this.workflowItem.trigger.parameters);

            parentBranch = this.workflowItem.parent.branch;
        } else if (this.workflowItem.pipeline.parameters) {
            runRequest.parameters.push(...this.workflowItem.pipeline.parameters);
        }

        // Branch checker
        if (currentBranch === '' && this.workflowItem.pipeline.last_pipeline_build
            && this.workflowItem.pipeline.last_pipeline_build.trigger) {
            currentBranch = this.workflowItem.pipeline.last_pipeline_build.trigger.vcs_branch;
        }
        if (this.workflowItem.parent && currentBranch !== parentBranch) {
            return this.runWithParameters();
        }

        // Run pipeline
        this._appPipService.run(
            this.workflowItem.project.key,
            this.workflowItem.application.name,
            this.workflowItem.pipeline.name, runRequest).subscribe(pipelineBuild => {
            this.navigateToBuild(pipelineBuild);
        });

    }

    navigateToBuild(pb: PipelineBuild): void {
        let queryParams = {queryParams: {envName: pb.environment.name}};
        if (this.applicationFilter.branch !== '') {
            queryParams.queryParams['branch'] = this.applicationFilter.branch;
        }
        if (this.applicationFilter.version !== 0) {
            queryParams.queryParams['version'] = this.applicationFilter.version;
        }
        this._router.navigate([
            '/project', this.workflowItem.project.key,
            'application', pb.application.name,
            'pipeline', pb.pipeline.name,
            'build', pb.build_number
        ], queryParams);
    }

    runWithParameters(): void {
        // ReInit
        this.launchPipelineParams = new Array<Parameter>();
        this.launchParentBuildNumber = undefined;
        this.launchOldBuilds = new Array<PipelineBuild>();
        this.launchGitParams = new Array<Parameter>();

        if (this.launchModal) {
            // Init Git parameters
            let gitBranchParam: Parameter = new Parameter();
            gitBranchParam.name = 'git.branch';
            gitBranchParam.value = this.applicationFilter.branch;
            gitBranchParam.description = 'Git branch to use';
            gitBranchParam.type = 'string';
            this.launchGitParams.push(gitBranchParam);

            // Init pipeline parameters
            this._pipStore.getPipelines(this.project.key, this.workflowItem.pipeline.name).subscribe(pips => {
                let pipKey = this.project.key + '-' + this.workflowItem.pipeline.name;
                if (pips && pips.get(pipKey)) {
                    let pipeline = pips.get(pipKey);
                    if (this.workflowItem.trigger) {
                        this.launchPipelineParams = Pipeline.mergeParams(pipeline.parameters, this.workflowItem.trigger.parameters);
                    } else {
                        this.launchPipelineParams = Pipeline.mergeParams(pipeline.parameters, []);
                    }
                }
            });

            // Init parent version
            if (this.workflowItem.parent && this.workflowItem.trigger.id > 0) {
                this._appPipService.buildHistory(
                    this.project.key, this.workflowItem.trigger.src_application.name, this.workflowItem.trigger.src_pipeline.name,
                    this.workflowItem.trigger.src_environment.name, 20, 'Success', this.applicationFilter.branch)
                    .subscribe(pbs => {
                        this.launchOldBuilds = pbs;
                        this.launchParentBuildNumber = pbs[0].build_number;
                    });
            }
            setTimeout(() => {
                this.launchModal.show({autofocus: false, closable: false});
            }, 100);
        } else {
            console.log('Error loading modal');
        }
    }

    runManual() {
        let request: PipelineRunRequest = new PipelineRunRequest();
        request.parameters = this.launchPipelineParams;
        request.env = new Environment();
        request.env = this.workflowItem.environment;

        if (this.workflowItem.parent) {
            request.parent_application_id = this.workflowItem.parent.application_id;
            request.parent_pipeline_id = this.workflowItem.parent.pipeline_id;
            request.parent_environment_id = this.workflowItem.parent.environment_id;
            request.parent_build_number = this.launchParentBuildNumber;
        } else {
            request.parameters.push(...this.launchGitParams);
        }
        this.launchModal.hide();
        // Run pipeline
        this._appPipService.run(
            this.workflowItem.project.key,
            this.workflowItem.application.name,
            this.workflowItem.pipeline.name, request).subscribe(pipelineBuild => {
            this.navigateToBuild(pipelineBuild);
        });
    }

    rollback(): void {
        let runRequest: PipelineRunRequest = new PipelineRunRequest();
        runRequest.env = this.workflowItem.environment;
        this._appPipService.rollback(
            this.workflowItem.project.key,
            this.workflowItem.application.name,
            this.workflowItem.pipeline.name,
            runRequest
        ).subscribe(pb => {
            this.navigateToBuild(pb);
        });
    }

    editPipeline(): void {
        this._router.navigate([
            '/project', this.workflowItem.project.key,
            'pipeline', this.workflowItem.pipeline.name
        ], {queryParams: {application: this.workflowItem.application.name}});
    }

    /**
     * Init new trigger and open modal
     */
    openCreateTriggerModal(): void {
        this.triggerInModal = new Trigger();
        this.triggerInModal.src_project = this.project;
        this.triggerInModal.src_application = this.workflowItem.application;
        this.triggerInModal.src_pipeline = this.workflowItem.pipeline;
        this.triggerInModal.src_environment = new Environment();
        this.triggerInModal.src_environment.name = this.workflowItem.environment.name;
        this.triggerInModal.dest_project = this.project;
        setTimeout(() => {
            this.createTriggerModal.show({autofocus: false, closable: false, observeChanges: true});
        }, 100);
    }

    /**
     * Manage action on trigger
     * @param type Type of action
     */
    triggerEvent(type: string): void {
        switch (type) {
            case 'add':
                this.createTriggerModal.hide();
                this._appStore.addTrigger(
                    this.project.key,
                    this.workflowItem.application.name,
                    this.workflowItem.pipeline.name,
                    this.triggerInModal).subscribe(() => {
                    this._toast.success('', this._translate.instant('trigger_added'));
                });
                break;
            case 'update':
                this.editTriggerModal.hide();
                this._appStore.updateTrigger(
                    this.project.key,
                    this.workflowItem.application.name,
                    this.workflowItem.pipeline.name,
                    this.triggerInModal).subscribe(() => {
                    this._toast.success('', this._translate.instant('trigger_updated'));
                });
                break;
            case 'delete':
                this.triggerLoading = true;
                this.editTriggerModal.hide();
                this._appStore.removeTrigger(
                    this.project.key,
                    this.triggerInModal.src_application.name,
                    this.triggerInModal.src_pipeline.name,
                    this.triggerInModal).subscribe(() => {
                    this._toast.success('', this._translate.instant('trigger_deleted'));
                    this.triggerLoading = false;
                }, () => {
                    this.triggerLoading = false;
                });
                break;
        }
    }

    openEditTriggerModal(): void {
        this.triggerInModal = _.cloneDeep(this.workflowItem.trigger);
        setTimeout(() => {
            this.editTriggerModal.show({autofocus: false, closable: false, observeChanges: true});
        }, 100);
    }

    openCreateSchedulerModal(): void {
        this.newScheduler = new Scheduler();
        if (this.createSchedulerModal) {
            setTimeout(() => {
                this.createSchedulerModal.show({autofocus: false, closable: false, observeChanges: true});
            }, 100);
        }
    }

    createScheduler(scheduler: Scheduler): void {
        this._appStore.addScheduler(this.project.key, this.application.name, this.workflowItem.pipeline.name, scheduler)
            .subscribe(() => {
                this._toast.success('', this._translate.instant('scheduler_added'));
                if (this.createSchedulerModal) {
                    this.createSchedulerModal.hide();
                }
            });
    }

    createHook(): void {
        if (!this.application.repositories_manager) {
            this._toast.error('', this._translate.instant('hook_repo_man_needed'));
            return;
        }
        let hook = new Hook();
        hook.pipeline = this.workflowItem.pipeline;
        hook.enabled = true;
        this._appStore.addHook(this.project, this.application, hook)
            .subscribe(() => {
                this._toast.success('', this._translate.instant('hook_added'));
            });
    }

    detachPipeline(p: Pipeline): void {
        this._appStore.detachPipeline(this.project.key, this.application.name, p.name).subscribe(() => {
            this._toast.success('', this._translate.instant('application_pipeline_detached'));
        });
    }

    getTriggerSource(pb: PipelineBuild): string {
        return PipelineBuild.GetTriggerSource(pb);
    }
}
