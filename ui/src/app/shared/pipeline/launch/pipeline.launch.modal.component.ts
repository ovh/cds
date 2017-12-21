import {Component, EventEmitter, Input, Output, ViewChild} from '@angular/core';
import {SemanticModalComponent} from 'ng-semantic/ng-semantic';
import {Parameter} from '../../../model/parameter.model';
import {Pipeline, PipelineBuild, PipelineRunRequest} from '../../../model/pipeline.model';
import {Application} from '../../../model/application.model';
import {Project} from '../../../model/project.model';
import {WorkflowItem} from '../../../model/application.workflow.model';
import {ApplicationPipelineService} from '../../../service/application/pipeline/application.pipeline.service';
import {Commit, Remote} from '../../../model/repositories.model';
import {cloneDeep} from 'lodash';

@Component({
    selector: 'app-pipeline-launch-modal',
    templateUrl: './pipeline.launch.modal.html',
    styleUrls: ['./pipeline.launch.modal.scss']
})
export class PipelineLaunchModalComponent {

    @Input() applicationFilter: any;
    @Input() application: Application;
    @Input() project: Project;
    @Input() workflowItem: WorkflowItem;
    @Input() remotes: Array<Remote>;

    @Output() pipelineRunEvent = new EventEmitter<PipelineBuild>();

    @ViewChild('launchModal')
    launchModal: SemanticModalComponent;

    // Manual Launch data
    launchGitParams: Array<Parameter>;
    launchPipelineParams: Array<Parameter>;
    launchParentBuildNumber = 0;
    launchOldBuilds: Array<PipelineBuild>;

    commits: { [key: string]: Array<Commit> } = {};
    currentHash: string;
    loadingCommits = false;

    constructor(private _appPipService: ApplicationPipelineService) {
    }

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

        if (this.applicationFilter.remote && this.applicationFilter.remote !== this.application.repository_fullname) {
              let remote = this.remotes.find((rem) => rem.name === this.applicationFilter.remote);

            if (remote) {
                let urlParam = new Parameter();
                urlParam.name = 'git.http_url';
                urlParam.type = 'string';
                urlParam.value = remote.url;
                this.launchGitParams.push(urlParam);

                urlParam = new Parameter();
                urlParam.name = 'git.url';
                urlParam.type = 'string';
                urlParam.value = remote.url;
                this.launchGitParams.push(urlParam);

                urlParam = new Parameter();
                urlParam.name = 'git.repository';
                urlParam.type = 'string';
                urlParam.value = remote.name;
                this.launchGitParams.push(urlParam);
            }
        }

        if (this.workflowItem.trigger) {
            this.launchPipelineParams = Pipeline.mergeParams(
                cloneDeep(this.workflowItem.pipeline.parameters),
                cloneDeep(this.workflowItem.trigger.parameters)
            );
        } else {
            this.launchPipelineParams = Pipeline.mergeParams(cloneDeep(this.workflowItem.pipeline.parameters), []);
        }

        // Init parent version
        if (this.workflowItem.parent && this.workflowItem.trigger.id > 0) {
            this._appPipService.buildHistory(
                this.project.key, this.workflowItem.trigger.src_application.name, this.workflowItem.trigger.src_pipeline.name,
                this.workflowItem.trigger.src_environment.name, 20, 'Success', this.applicationFilter.branch, this.applicationFilter.remote)
                .subscribe(pbs => {
                    this.launchOldBuilds = pbs;
                    if (Array.isArray(pbs) && pbs.length) {
                        this.launchParentBuildNumber = pbs[0].build_number;
                    }
                    this.loadCommits();
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
        request.parameters = new Array<Parameter>();
        if (this.launchPipelineParams) {
            request.parameters = request.parameters.concat(this.launchPipelineParams);
        }

        request.env = cloneDeep(this.workflowItem.environment);
        delete request.env.variables;
        delete request.env.groups;

        if (this.workflowItem.parent) {
            request.parent_application_id = this.workflowItem.parent.application_id;
            request.parent_pipeline_id = this.workflowItem.parent.pipeline_id;
            request.parent_environment_id = this.workflowItem.parent.environment_id;
            request.parent_build_number = Number(this.launchParentBuildNumber);
        }

        request.parameters.push(...this.launchGitParams);
        request.parameters = Parameter.formatForAPI(request.parameters);

        this.launchModal.hide();
        // Run pipeline
        this._appPipService.run(
            this.workflowItem.project.key,
            this.workflowItem.application.name,
            this.workflowItem.pipeline.name, request).subscribe(pipelineBuild => {
            this.pipelineRunEvent.emit(pipelineBuild);
        });
    }

    loadCommits() {
        let pb = this.launchOldBuilds.find(p => {
            return p.build_number === Number(this.launchParentBuildNumber);
        });
        this.currentHash = pb.trigger.vcs_hash;
        if (this.currentHash && this.currentHash !== '' && !this.commits[pb.trigger.vcs_hash]) {
            // Load commits
            this.loadingCommits = true;
            this._appPipService.getCommits(this.project.key, this.workflowItem.application.name, this.workflowItem.pipeline.name,
                this.workflowItem.environment.name, this.currentHash).subscribe(cs => {
                this.loadingCommits = false;
                this.commits[this.currentHash] = cs;
            }, () => {
                this.loadingCommits = false;
            });
        }
    }
}
