import {Component, Input, OnInit, ViewChild} from '@angular/core';
import {PipelineBuild, Pipeline, PipelineRunRequest} from '../../../model/pipeline.model';
import {ApplicationPipelineService} from '../../../service/application/pipeline/application.pipeline.service';
import {Router} from '@angular/router';
import {Application} from '../../../model/application.model';
import {WorkflowItem} from '../../../model/application.workflow.model';
import {Parameter} from '../../../model/parameter.model';
import {SemanticModalComponent} from 'ng-semantic/ng-semantic';
import {PipelineStore} from '../../../service/pipeline/pipeline.store';
import {Environment} from '../../../model/environment.model';

@Component({
    selector: 'app-run-summary',
    templateUrl: './run.summary.html',
    styleUrls: ['./run.summary.scss']
})
export class RunSummaryComponent implements OnInit {

    @Input() currentBuild: PipelineBuild;
    @Input() duration: string;
    @Input() application: Application;

    // For run NEw

    @ViewChild('launchModal')
    launchModal: SemanticModalComponent;

    launchPipelineParams = new Array<Parameter>();
    launchParentBuildNumber = undefined;
    launchOldBuilds = new Array<PipelineBuild>();
    launchGitParams = new Array<Parameter>();

    parent: WorkflowItem;
    currentWI: WorkflowItem;

    constructor(private _appPipService: ApplicationPipelineService, private _router: Router, private _pipStore: PipelineStore) {
    }

    ngOnInit(): void {

        this.application.workflows.forEach(wi => {
            if (wi.application.id === this.currentBuild.application.id &&
                wi.pipeline.id === this.currentBuild.pipeline.id &&
                wi.environment.id === this.currentBuild.environment.id) {

                this.currentWI = wi;
            }
            this.checkParent(wi);
        });

    }

    getAuthor(): string {
        if (this.currentBuild) {
            return PipelineBuild.GetTriggerSource(this.currentBuild);
        }

    }

    checkParent(wi: WorkflowItem): void {
        if (wi.subPipelines) {
            wi.subPipelines.forEach(sWi => {
                if (sWi.application.id === this.currentBuild.application.id &&
                    sWi.pipeline.id === this.currentBuild.pipeline.id &&
                    sWi.environment.id === this.currentBuild.environment.id) {
                    this.currentWI = sWi;
                    this.parent = wi;
                    return wi;
                } else {
                    this.checkParent(sWi);
                }
            });
        }
        return;
    }

    runAgain(): void {
        this._appPipService.runAgain(
            this.currentBuild.pipeline.projectKey,
            this.currentBuild.application.name,
            this.currentBuild.pipeline.name,
            this.currentBuild.build_number,
            this.currentBuild.environment.name).subscribe(pb => {
            this.navigateToBuild(pb);
        });
    }

    navigateToBuild(pb: PipelineBuild): void {
        let queryParams = {queryParams: {envName: pb.environment.name}};
        queryParams.queryParams['branch'] = this.currentBuild.trigger.vcs_branch;
        queryParams.queryParams['version'] = this.currentBuild.version;

        // Force url change
        queryParams.queryParams['ts'] = (new Date()).getTime();

        this._router.navigate([
            '/project', this.currentBuild.pipeline.projectKey,
            'application', pb.application.name,
            'pipeline', pb.pipeline.name,
            'build', pb.build_number
        ], queryParams);
    }

    runNew(): void {

        // ReInit
        this.launchPipelineParams = new Array<Parameter>();
        this.launchParentBuildNumber = undefined;
        this.launchOldBuilds = new Array<PipelineBuild>();
        this.launchGitParams = new Array<Parameter>();

        if (this.launchModal) {
            // Init Git parameters
            let gitBranchParam: Parameter = new Parameter();
            gitBranchParam.name = 'git.branch';
            gitBranchParam.value = this.currentBuild.trigger.vcs_branch;
            gitBranchParam.description = 'Git branch to use';
            gitBranchParam.type = 'string';
            this.launchGitParams.push(gitBranchParam);

            // Init pipeline parameters
            this._pipStore.getPipelines(this.currentBuild.pipeline.projectKey, this.currentBuild.pipeline.name).subscribe(pips => {
                let pipKey = this.currentBuild.pipeline.projectKey + '-' + this.currentBuild.pipeline.name;
                if (pips && pips.get(pipKey)) {
                    let pipeline = pips.get(pipKey);
                    if (this.currentWI.trigger) {
                        this.launchPipelineParams = Pipeline.mergeParams(pipeline.parameters, this.currentWI.trigger.parameters);
                    } else {
                        this.launchPipelineParams = Pipeline.mergeParams(pipeline.parameters, []);
                    }
                }
            });

            // Init parent version
            if (this.parent && this.currentWI.trigger.id > 0) {
                this._appPipService.buildHistory(
                    this.currentBuild.pipeline.projectKey, this.currentWI.trigger.src_application.name,
                    this.currentWI.trigger.src_pipeline.name,
                    this.currentWI.trigger.src_environment.name, 20, 'Success', this.currentBuild.trigger.vcs_branch)
                    .subscribe(pbs => {
                        this.launchOldBuilds = pbs;
                        this.launchParentBuildNumber = pbs[0].build_number;
                    });
            }
            setTimeout(() => {
                this.launchModal.show({autofocus: false, closable: false, observeChanges: true});
            }, 100);
        } else {
            console.log('Error loading modal');
        }
    }

    runManual(): void {
        let request: PipelineRunRequest = new PipelineRunRequest();
        request.parameters = this.launchPipelineParams;
        request.env = new Environment();
        request.env = this.currentWI.environment;

        if (this.parent) {
            request.parent_application_id = this.parent.application.id;
            request.parent_pipeline_id = this.parent.pipeline.id;
            request.parent_environment_id = this.parent.environment.id;
            request.parent_build_number = this.launchParentBuildNumber;
        } else {
            request.parameters.push(...this.launchGitParams);
        }
        this.launchModal.hide();
        // Run pipeline
        this._appPipService.run(
            this.currentBuild.pipeline.projectKey,
            this.currentBuild.application.name,
            this.currentBuild.pipeline.name, request).subscribe(pipelineBuild => {
            this.navigateToBuild(pipelineBuild);
        });
    }
}
