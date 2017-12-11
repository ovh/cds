import {Component, Input, OnInit} from '@angular/core';
import {PipelineBuild, PipelineRunRequest, PipelineStatus} from '../../../model/pipeline.model';
import {ApplicationPipelineService} from '../../../service/application/pipeline/application.pipeline.service';
import {Router} from '@angular/router';
import {Application} from '../../../model/application.model';
import {WorkflowItem} from '../../../model/application.workflow.model';
import {Parameter} from '../../../model/parameter.model';
import {PipelineStore} from '../../../service/pipeline/pipeline.store';
import {Project} from '../../../model/project.model';
import {ToastService} from '../../../shared/toast/ToastService';
import {TranslateService} from '@ngx-translate/core';

@Component({
    selector: 'app-run-summary',
    templateUrl: './run.summary.html',
    styleUrls: ['./run.summary.scss']
})
export class RunSummaryComponent implements OnInit {

    @Input() currentBuild: PipelineBuild;
    @Input() duration: string;
    @Input() application: Application;
    @Input() project: Project;

    parent: WorkflowItem;
    currentWI: WorkflowItem;
    pipelineStatusEnum = PipelineStatus;

    loading = false;

    constructor(private _appPipService: ApplicationPipelineService, private _router: Router, private _pipStore: PipelineStore,
                private _toastSerivce: ToastService, private _translate: TranslateService) {
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
        this.loading = true;
        this._appPipService.runAgain(
            this.currentBuild.pipeline.projectKey,
            this.currentBuild.application.name,
            this.currentBuild.pipeline.name,
            this.currentBuild.build_number,
            this.currentBuild.environment.name
        ).subscribe(pb => {
            this.loading = false;
            this.navigateToBuild(pb);
        }, () => {
            this.loading = false;
        });
    }

    navigateToBuild(pb: PipelineBuild): void {
        let queryParams = {queryParams: {envName: pb.environment.name}};
        queryParams.queryParams['branch'] = this.currentBuild.trigger.vcs_branch;
        queryParams.queryParams['remote'] = this.currentBuild.trigger.vcs_remote;
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

    stop(): void {
        this.loading = true;
        this._appPipService.stop(this.project.key, this.currentBuild.application.name,
            this.currentBuild.pipeline.name, this.currentBuild.build_number, this.currentBuild.environment.name).subscribe(() => {
            this.loading = false;
            this._toastSerivce.success('', this._translate.instant('pipeline_stop'));
        }, () => {
            this.loading = false;
        });
    }

    runNew(): void {
        let request: PipelineRunRequest = new PipelineRunRequest();
        request.parameters = this.getCurrentPipelineParameters();
        request.env = this.currentWI.environment;

        if (this.parent) {
            request.parent_application_id = this.parent.application.id;
            request.parent_pipeline_id = this.parent.pipeline.id;
            request.parent_environment_id = this.parent.environment.id;
            request.parent_version = this.currentBuild.version;
        }

        // Run pipeline
        this._appPipService.run(
            this.currentBuild.pipeline.projectKey,
            this.currentBuild.application.name,
            this.currentBuild.pipeline.name,
            request
        ).subscribe(pipelineBuild => {
            this.navigateToBuild(pipelineBuild);
        });
    }

    getCurrentPipelineParameters(): Array<Parameter> {
        return this.currentBuild.parameters.filter(p => {
            return p.name.indexOf('cds.pip.') === 0 || p.name.indexOf('git.') === 0;
        });
    }
}
