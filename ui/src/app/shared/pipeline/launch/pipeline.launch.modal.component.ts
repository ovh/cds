import {Component, EventEmitter, Input, Output, ViewChild} from '@angular/core';
import {SemanticModalComponent} from 'ng-semantic/ng-semantic';
import {Parameter} from '../../../model/parameter.model';
import {Pipeline, PipelineBuild, PipelineRunRequest} from '../../../model/pipeline.model';
import {PipelineStore} from '../../../service/pipeline/pipeline.store';
import {Project} from '../../../model/project.model';
import {WorkflowItem} from '../../../model/application.workflow.model';
import {ApplicationPipelineService} from '../../../service/application/pipeline/application.pipeline.service';
import {Environment} from '../../../model/environment.model';

@Component({
    selector: 'app-pipeline-launch-modal',
    templateUrl: './pipeline.launch.modal.html',
    styleUrls: ['./pipeline.launch.modal.scss']
})
export class PipelineLaunchModalComponent {

    @Input() applicationFilter: any;
    @Input() project: Project;
    @Input() workflowItem: WorkflowItem;

    @Output() pipelineRunEvent = new EventEmitter<PipelineBuild>();

    @ViewChild('launchModal')
    launchModal: SemanticModalComponent;

    // Manual Launch data
    launchGitParams: Array<Parameter>;
    launchPipelineParams: Array<Parameter>;
    launchParentBuildNumber = 0;
    launchOldBuilds: Array<PipelineBuild>;

    constructor(private _pipStore: PipelineStore, private _appPipService: ApplicationPipelineService) { }

    show(data?: {}): void {
        this.launchPipelineParams = new Array<Parameter>();
        this.launchParentBuildNumber = undefined;
        this.launchOldBuilds = new Array<PipelineBuild>();
        this.launchGitParams = new Array<Parameter>();

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
            this.launchModal.show(data);
        }, 100);
    }

    hide(): void {
        if (this.launchModal) {
            this.launchModal.hide();
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
                this.pipelineRunEvent.emit(pipelineBuild);
        });
    }
}
