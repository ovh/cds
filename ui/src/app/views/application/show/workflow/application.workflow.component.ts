import {Component, EventEmitter, Input, NgZone, OnInit, OnDestroy, Output, ViewChild} from '@angular/core';
import {ApplicationWorkflowService} from '../../../../service/application/application.workflow.service';
import {Application} from '../../../../model/application.model';
import {Project} from '../../../../model/project.model';
import {WorkflowItem, WorkflowStatusResponse} from '../../../../model/application.workflow.model';
import {PipelineBuild, PipelineType} from '../../../../model/pipeline.model';
import {ApplicationPipelineLinkComponent} from './pipeline/link/pipeline.link.component';
import {Branch} from '../../../../model/repositories.model';
import {Router} from '@angular/router';
import {cloneDeep} from 'lodash';
import {Observable} from 'rxjs/Observable';

@Component({
    selector: 'app-application-workflow',
    templateUrl: './application.workflow.html',
    styleUrls: ['./application.workflow.scss']
})
export class ApplicationWorkflowComponent implements OnInit, OnDestroy {
    readonly ORIENTATION_KEY = 'CDS-ORIENTATION';

    @Input() project: Project;
    @Input() application: Application;
    @Input() applicationFilter: any;
    @Output() changeWorkerEvent = new EventEmitter<boolean>();

    // Allow angular update from work started outside angular context
    zone: NgZone;

    // Worflow to display
    private _workflowOrientationValue = localStorage.getItem(this.ORIENTATION_KEY) || 'horizontal';
    set workflowOrientation(orientation: string) {
        this._workflowOrientationValue = orientation;
        localStorage.setItem(this.ORIENTATION_KEY, orientation);
    }
    get workflowOrientation() {
        return this._workflowOrientationValue;
    }

    // Filter values
    branches: Array<Branch>;
    versions: Array<string>;

    // Modal Component to link pipeline
    @ViewChild('linkPipelineComponent')
    linkPipelineComponent: ApplicationPipelineLinkComponent;

    constructor(private _appWorkflow: ApplicationWorkflowService, private _router: Router) {
        this.zone = new NgZone({enableLongStackTrace: false});
    }

    ngOnDestroy(): void {
        this.changeWorkerEvent.emit(true);
    }

    ngOnInit(): void {
        this.generateParentInformation();
        // Load branches
        this._appWorkflow.getBranches(this.project.key, this.application.name).subscribe(branches => {
            this.branches = branches;
            this.branches.forEach(b => {
                if (b.default && !this.applicationFilter.branch) {
                    this.applicationFilter.branch = b.display_id;
                }
            });

            this.loadVersions(this.project.key, this.application.name).subscribe();
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
    refreshWorkflow(resp: WorkflowStatusResponse): void {
        if (this.application.workflows) {
            this.zone.run(() => {
                this.application.workflows.forEach((w) => {
                    this.updateTreeStatus(w, resp);
                });
            });
        }
    }

    updateTreeStatus(w: WorkflowItem, resp: WorkflowStatusResponse): void {
        // Find pipeline build for current workflow item
        if (resp.builds) {
            let pb = resp.builds.find(p => {
                return p.application.id === w.application.id &&
                    p.pipeline.id === w.pipeline.id &&
                    p.environment.id === w.environment.id;
            });

            if (pb) {
                w.pipeline.last_pipeline_build = pb;
                if (w.schedulers && resp.schedulers && resp.schedulers.length > 0) {
                    w.schedulers.forEach(s => {
                        let sInApp = resp.schedulers.find(sc => {
                            return sc.id === s.id;
                        });
                        if (sInApp && sInApp.next_execution) {
                            s.next_execution = sInApp.next_execution;
                        }
                    });
                }
                if (w.poller && resp.pollers && resp.pollers.length > 0) {
                    let poller = resp.pollers.find(p => {
                        return p.application.id === w.poller.application.id
                            && p.pipeline.id === w.poller.pipeline.id;
                    });
                    if (poller && poller.next_execution) {
                        w.poller.next_execution = poller.next_execution;
                    }
                }
            }
        }

        if (w.environment.name === 'NoEnv' && Number(PipelineType[w.pipeline.type]) > 0) {
            // If current item is a deploy or testing pipeline without environment
            // Then add new item on workflow
            this.project.environments.forEach((env, index) => {
                let pbToAssign: PipelineBuild;
                if (resp.builds) {
                    let pipelineBuild = resp.builds.filter(p => p.application.id === w.application.id &&
                    p.pipeline.id === w.pipeline.id &&
                    p.environment.id === env.id);

                    if (pipelineBuild && pipelineBuild.length === 1) {
                        pbToAssign = pipelineBuild[0];
                    }
                }
                if (index === 0) {
                    w.environment = env;
                    w.pipeline.last_pipeline_build = pbToAssign;
                } else {
                    let newItem = cloneDeep(w);
                    newItem.environment = env;
                    newItem.pipeline.last_pipeline_build = pbToAssign;
                    this.application.workflows.push(newItem);
                }
            });
        }

        // Update parent info
        if (w.parent) {
            let parentUpdated: Array<PipelineBuild>;
            if (resp.builds) {
                parentUpdated = resp.builds.filter(
                    p => p.pipeline.id === w.parent.pipeline_id &&
                    p.environment.id === w.parent.environment_id &&
                    p.application.id === w.parent.application_id
                );
            }
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
                this.updateTreeStatus(sub, resp);
            });
        }
    }

    /**
     * Action when changing branch
     */
    changeBranch(): void {
        // Load the versions of the new branch
        this.loadVersions(this.project.key, this.application.name)
            .subscribe(() => this.changeVersion());
    }

    /**
     * Action when changing version
     */
    changeVersion(version?: string): void {
        this.applicationFilter.branch = this.applicationFilter.branch.trim();

        if (!version && Array.isArray(this.versions) && this.versions.length) {
            this.applicationFilter.version = this.versions[0];
        }

        if (version) {
            this.applicationFilter.version = version;
        }

        this._router.navigate(['/project/', this.project.key, 'application', this.application.name],
            {queryParams: {tab: 'workflow', branch: this.applicationFilter.branch, version: this.applicationFilter.version}});
        this.changeWorkerEvent.emit(false);
        this.clearTree(this.application.workflows);
    }

    /**
     * Load the list of version for the current application on the selected branch
     */
    loadVersions(key: string, appName: string): Observable<Array<string>> {
        return this._appWorkflow.getVersions(key, appName, this.applicationFilter.branch)
            .map((versions) => this.versions = [' ', ...versions.map((v) => v.toString())]);
    }

    clearTree(items: Array<WorkflowItem>): void {
        items.forEach(w => {
            delete w.pipeline.last_pipeline_build;
            if (w.subPipelines) {
                this.clearTree(w.subPipelines);
            }
        });
    }

    openLinkPipelineModal(): void {
        if (this.linkPipelineComponent) {
            this.linkPipelineComponent.show({autofocus: false, closable: false, observeChanges: true});
        }
    }
}
