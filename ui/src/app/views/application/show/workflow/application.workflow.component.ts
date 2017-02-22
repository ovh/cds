import {Component, OnInit, Input, Output, EventEmitter, NgZone, ViewChild} from '@angular/core';
import {ApplicationWorkflowService} from '../../../../service/application/application.workflow.service';
import {Application} from '../../../../model/application.model';
import {Project} from '../../../../model/project.model';
import {WorkflowItem} from '../../../../model/application.workflow.model';
import {PipelineType, PipelineBuild} from '../../../../model/pipeline.model';
import {ApplicationPipelineLinkComponent} from './pipeline/pipeline.link.component';
import {Branch} from '../../../../model/repositories.model';
import {Router} from '@angular/router';

declare var _: any;
declare var jQuery: any;

@Component({
    selector: 'app-application-workflow',
    templateUrl: './application.workflow.html',
    styleUrls: ['./application.workflow.scss']
})
export class ApplicationWorkflowComponent implements OnInit {

    @Input() project: Project;
    @Input() application: Application;
    @Input() applicationFilter: any;
    @Output() changeWorkerEvent = new EventEmitter<boolean>();

    // Allow angular update from work started outside angular context
    zone: NgZone;

    // Worflow to display
    workflowOrientation = 'horizontal';

    // Filter values
    branches: Array<Branch>;
    versions: Array<string|number>;

    // Modal Component to link pipeline
    @ViewChild('linkPipelineComponent')
    linkPipelineComponent: ApplicationPipelineLinkComponent;

    constructor(private _appWorkflow: ApplicationWorkflowService, private _router: Router) {
        this.zone = new NgZone({enableLongStackTrace: false});
    }

    ngOnInit(): void {
        this.generateParentInformation();
        // Load branches
        this._appWorkflow.getBranches(this.project.key, this.application.name).subscribe(branches => {
            branches.unshift(new Branch());
            this.branches = branches;

            this.branches.forEach(b => {
                if (b.default || (this.applicationFilter.branch === '' && b.display_id === 'master')) {
                    this.applicationFilter.branch = b.display_id;
                }
            });
            this.loadVersions(this.project.key, this.application.name);
        });
    }

    generateParentInformation() {
        if (this.application.workflows && this.application.workflows.length > 0) {
            this.application.workflows.forEach((item) => {
                this.generateItemContent(item);
            });
        }
    }

    /**
     * @param item Current item
     * @param parent Parent datas
     */
    generateItemContent(item: WorkflowItem, parent?: WorkflowItem): void {
        if (parent) {
            item.parent = {
                application_id: parent.application.id,
                pipeline_id: parent.pipeline.id,
                environment_id: parent.environment.id,
                buildNumber: 0,
                version: 0,
                branch: 'master'
            };
        }

        if (item.subPipelines) {
            item.subPipelines.forEach((subItem) => {
                this.generateItemContent(subItem, item);
            });
        }
    }

    switchApplication(): void {
        this.generateParentInformation();
    }

    /**
     * Refresh workflow trees.
     * @param app Application data updated
     */
    refreshWorkflow(app: Application): void {
        if (this.application.workflows) {
            this.zone.run(() => {
                this.application.workflows.forEach((w) => {
                    this.updateTree(w, app);
                });
            });
        }
    }

    /**
     * Update workflow Item
     * @param w Workflow Item to update
     * @param app Application data updated
     */
    updateTree(w: WorkflowItem, app: Application): void {
        // If same app, try to find pipeline
        if (w.application.id === app.id && app.pipelines_build) {
            let pipelineBuildUpdated = app.pipelines_build.filter(
                pb => pb.pipeline.id === w.pipeline.id && pb.environment.id === w.environment.id
            );
            // If pipeline found : update it
            if (pipelineBuildUpdated && pipelineBuildUpdated.length === 1) {
                w.pipeline.last_pipeline_build = pipelineBuildUpdated[0];
            } else if (w.environment.name === 'NoEnv' && Number(PipelineType[w.pipeline.type]) > 0) {
                // If current item is a deploy or testing pipeline without environment
                // Then add new item on workflow
                this.project.environments.forEach((env, index) => {
                    let pipelineBuild = app.pipelines_build.filter(pb => pb.pipeline.id === w.pipeline.id && pb.environment.id === env.id);
                    let pbToAssign: PipelineBuild = undefined;
                    if (pipelineBuild && pipelineBuild.length === 1) {
                        pbToAssign = pipelineBuild[0];
                    }

                    if (index === 0) {
                        w.environment = env;
                        w.pipeline.last_pipeline_build = pbToAssign;
                    } else {
                        let newItem = _.cloneDeep(w);
                        newItem.environment = env;
                        newItem.pipeline.last_pipeline_build = pbToAssign;
                        this.application.workflows.push(newItem);
                    }

                });
            }
        }
        // Update parent info
        if (w.parent && w.parent.application_id === app.id && app.pipelines_build) {
            let parentUpdated = app.pipelines_build.filter(
                pb => pb.pipeline.id === w.parent.pipeline_id && pb.environment.id === w.parent.environment_id
            );
            if (parentUpdated && parentUpdated.length === 1) {
                w.parent.buildNumber = parentUpdated[0].build_number;
                w.parent.version = parentUpdated[0].version;
                if (parentUpdated[0].trigger) {
                    w.parent.branch = parentUpdated[0].trigger.vcs_branch;
                }
            }
        }

        // Check subpipeline
        if (w.subPipelines) {
            w.subPipelines.forEach((sub) => {
                this.updateTree(sub, app);
            });
        }
    };

    /**
     * Action when changing branch
     */
    changeBranch(): void {
        // reinit verison filter
        this.applicationFilter.version = '';
        jQuery('.cdsVersion div.text')[0].textContent = '';
        this.changeVersion();

        // Load the versions of the new branch
        this.loadVersions(this.project.key, this.application.name);
    };

    /**
     * Action when changing version
     */
    changeVersion(): void {
        this.applicationFilter.branch = this.applicationFilter.branch.trim();
        if (this.applicationFilter.version.trim() === '') {
            this.applicationFilter.version = 0;
        }
        this._router.navigate(['/project/', this.project.key, 'application', this.application.name],
            {queryParams: { tab: 'workflow', branch: this.applicationFilter.branch, version: this.applicationFilter.version}});
        this.changeWorkerEvent.emit(true);
        this.clearTree(this.application.workflows);
    }

    clearTree(items: Array<WorkflowItem>): void {
        items.forEach(w => {
            delete w.pipeline.last_pipeline_build;
            if (w.subPipelines) {
                this.clearTree(w.subPipelines);
            }
        });
    }

    /**
     * Load the list of version for the current application on the selected branch
     */
    loadVersions(key: string, appName: string): void {
        this._appWorkflow.getVersions(key, appName, this.applicationFilter.branch).subscribe(versions => {
            this.versions = versions;
            this.versions.unshift(' ');
        });
    };

    openLinkPipelineModal(): void {
        if (this.linkPipelineComponent) {
            this.linkPipelineComponent.show({autofocus: false, closable: false, observeChanges: true});
        }
    }
}

