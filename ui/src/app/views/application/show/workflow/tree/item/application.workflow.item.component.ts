import {Component, Input, ViewChild, DoCheck, OnDestroy} from '@angular/core';
import {Subscription} from 'rxjs/Subscription';
import {WorkflowItem} from '../../../../../../model/application.workflow.model';
import {Application} from '../../../../../../model/application.model';
import {ApplicationPipelineService} from '../../../../../../service/application/pipeline/application.pipeline.service';
import {NotificationService} from '../../../../../../service/notification/notification.service';
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
import {RepositoryPoller} from '../../../../../../model/polling.model';
import {PipelineLaunchModalComponent} from '../../../../../../shared/pipeline/launch/pipeline.launch.modal.component';

declare var _: any;

@Component({
    selector: 'app-application-workflow-item',
    templateUrl: './application.workflow.item.html',
    styleUrls: ['./application.workflow.item.scss']
})
export class ApplicationWorkflowItemComponent implements DoCheck, OnDestroy {

    @Input() project: Project;
    @Input() workflowItem: WorkflowItem;
    @Input() orientation: string;
    @Input() application: Application;
    @Input() applicationFilter: any;
    oldPipelineStatus: string;
    // Triggers modals
    @ViewChild('editTriggerModal')
    editTriggerModal: SemanticModalComponent;
    @ViewChild('createTriggerModal')
    createTriggerModal: SemanticModalComponent;
    triggerInModal: Trigger;
    triggerLoading = false;

    // Run pipeline modal
    @ViewChild('pipelineLaunchModal')
    launchPipelineModal: PipelineLaunchModalComponent;

    // scheduler
    @ViewChild('createSchedulerModal')
    createSchedulerModal: SemanticModalComponent;
    newScheduler: Scheduler;

    // Detach pipeline
    @ViewChild('detachPipelineModal')
    detachModalPipelineModal: SemanticModalComponent;

    notificationSubscription: Subscription;

    constructor(private _router: Router, private _appPipService: ApplicationPipelineService, private _pipStore: PipelineStore,
                private _appStore: ApplicationStore, private _toast: ToastService, private _translate: TranslateService,
                private _notification: NotificationService) {
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

        let branchParam = new Parameter();
        branchParam.name = 'git.branch';
        branchParam.type = 'string';
        branchParam.value = currentBranch;
        runRequest.parameters.push(branchParam);

        // Run pipeline
        this._appPipService.run(
            this.workflowItem.project.key,
            this.workflowItem.application.name,
            this.workflowItem.pipeline.name, runRequest).subscribe(pipelineBuild => {
            this.navigateToBuild(pipelineBuild);
        });

    }

    navigateToBuild(pb: PipelineBuild): void {
        if (this.launchPipelineModal) {
            this.launchPipelineModal.hide();
        }

        let queryParams = {queryParams: {envName: pb.environment.name}};
        queryParams.queryParams['branch'] = pb.trigger.vcs_branch;
        queryParams.queryParams['version'] = pb.version;

        this._router.navigate([
            '/project', this.workflowItem.project.key,
            'application', pb.application.name,
            'pipeline', pb.pipeline.name,
            'build', pb.build_number
        ], queryParams);
    }

    runWithParameters(): void {
        if (this.launchPipelineModal) {
            this.launchPipelineModal.show({autofocus: false, closable: false, observeChanges: true});
        }
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

    openDetachPipelineModal(): void {
        if (this.detachModalPipelineModal) {
            this.detachModalPipelineModal.show({autofocus: false, closable: false, observeChanges: true});
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

    createPoller(): void {
        if (!this.application.repositories_manager) {
            this._toast.error('', this._translate.instant('hook_repo_man_needed'));
            return;
        }
        let poller = new RepositoryPoller();
        poller.enabled = true;
        poller.pipeline = this.workflowItem.pipeline;
        poller.application = this.workflowItem.application;
        this._appStore.addPoller(this.project.key, this.workflowItem.application.name, this.workflowItem.pipeline.name, poller)
            .subscribe(() => {
                this._toast.success('', this._translate.instant('poller_added'));
            });
    }

    detachPipeline(p: Pipeline): void {
        this._appStore.detachPipeline(this.project.key, this.application.name, p.name).subscribe(() => {
            this._toast.success('', this._translate.instant('application_pipeline_detached'));
            if (this.detachModalPipelineModal) {
                this.detachModalPipelineModal.hide();
            }
        });
    }

    getTriggerSource(pb: PipelineBuild): string {
        return PipelineBuild.GetTriggerSource(pb);
    }

    handleNotification(pipeline: Pipeline): void {
        switch (pipeline.last_pipeline_build.status) {
        case 'Success':
            this.notificationSubscription = this._notification.create(this._translate.instant('notification_on_pipeline_success', {
                pipelineName: pipeline.name
            })).subscribe();
            break;
        case 'Failing':
            this.notificationSubscription = this._notification.create(this._translate.instant('notification_on_pipeline_failing', {
                pipelineName: pipeline.name
            })).subscribe();
            break;
        }
    }

    ngDoCheck() {
        if (this.workflowItem.pipeline && this.workflowItem.pipeline.last_pipeline_build &&
            this.workflowItem.pipeline.last_pipeline_build.status) {

            if (!this.oldPipelineStatus) {
                this.oldPipelineStatus = this.workflowItem.pipeline.last_pipeline_build.status;
            }

            if (this.oldPipelineStatus !== this.workflowItem.pipeline.last_pipeline_build.status) {
                this.handleNotification(this.workflowItem.pipeline);
            }

            this.oldPipelineStatus = this.workflowItem.pipeline.last_pipeline_build.status;
        }
    }

    ngOnDestroy() {
        this.notificationSubscription.unsubscribe();
    }
}
