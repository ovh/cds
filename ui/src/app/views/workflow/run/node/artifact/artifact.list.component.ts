import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, OnInit } from '@angular/core';
import { Select, Store } from '@ngxs/store';
import { ProjectIntegration } from 'app/model/integration.model';
import {
    UIArtifact,
    WorkflowNodeRun,
    WorkflowNodeRunArtifact,
    WorkflowNodeRunStaticFiles, WorkflowRunResult
} from 'app/model/workflow.run.model';
import { WorkflowHelper } from 'app/service/workflow/workflow.helper';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { Column, ColumnType, Filter } from 'app/shared/table/data-table.component';
import { WorkflowState } from 'app/store/workflow.state';
import { Observable, Subscription } from 'rxjs';

@Component({
    selector: 'app-workflow-artifact-list',
    templateUrl: './artifact.list.html',
    styleUrls: ['./artifact.list.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowRunArtifactListComponent implements OnInit, OnDestroy {
    @Select(WorkflowState.getSelectedNodeRun()) nodeRun$: Observable<WorkflowNodeRun>;
    nodeRunSubs: Subscription;

    runResult: Array<WorkflowRunResult>
    artifacts: Array<WorkflowNodeRunArtifact>;

    uiArtifacts: Array<UIArtifact>;
    staticFiles: Array<WorkflowNodeRunStaticFiles>;

    filter: Filter<UIArtifact>;
    columns: Array<Column<UIArtifact>>;

    constructor(private _cd: ChangeDetectorRef, private _store: Store) {
        this.filter = f => {
            const lowerFilter = f.toLowerCase();
            return d => d.name.toLowerCase().indexOf(lowerFilter) !== -1 ||
                d.md5.toLowerCase().indexOf(lowerFilter) !== -1;
        };
        this.columns = [
            <Column<UIArtifact>>{
                type: ColumnType.LINK,
                name: 'artifact_name',
                selector: (a: UIArtifact) => {
                    let link = a.link;
                    let value = a.name;
                    if (a.human_size) {
                        value += ` (${a.human_size})`;
                    }
                    return {
                        link,
                        value
                    };
                }
            },
            <Column<UIArtifact>>{
                name: 'Type of artifact',
                selector: (a: UIArtifact) => a.file_type
            },
            <Column<UIArtifact>>{
                type: ColumnType.TEXT_COPY,
                name: 'MD5 Sum',
                selector: (a: UIArtifact) => a.md5
            }
        ];
    }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit(): void {
        this.nodeRunSubs = this.nodeRun$.subscribe(nr => {
            if (!nr) {
                return;
            }

            let computeArtifact = false;
            if (nr.results && (!this.runResult || nr.results.length !== this.runResult.length)) {
                computeArtifact = true;
            }
            if (nr.artifacts && (!this.artifacts || nr.artifacts.length !== this.artifacts.length)) {
                computeArtifact = true;
            }
            if (computeArtifact) {
                let uiArtifacts: Array<UIArtifact>;
                let uiRunResults: Array<UIArtifact>;
                this.uiArtifacts = new Array<UIArtifact>();
                if (nr.results) {
                    let w = this._store.selectSnapshot(WorkflowState.workflowRunSnapshot).workflow;

                    let integrationArtifactManager: ProjectIntegration;
                    if (w?.integrations) {
                        for (let i = 0; i < w.integrations.length; i++) {
                            let integ = w.integrations[i];
                            if (!integ.project_integration.model.artifact_manager) {
                                continue;
                            }
                            integrationArtifactManager = integ?.project_integration;
                        }
                    }

                    uiRunResults = WorkflowHelper.toUIArtifact(nr.results, integrationArtifactManager);
                    this.uiArtifacts.push(...uiRunResults);
                }

                if (nr.artifacts) {
                    uiArtifacts = nr.artifacts.map(a => {
                        let uiArt = new UIArtifact();
                        uiArt.name = a.name;
                        uiArt.size = a.size;
                        uiArt.md5 = a.md5sum;
                        uiArt.type = 'file';
                        uiArt.link = `./cdsapi/workflow/artifact/${a.download_hash}`;
                        uiArt.file_type = uiArt.type;
                        return uiArt;
                    });
                    this.uiArtifacts.push(...uiArtifacts);
                }
                this._cd.markForCheck();
            }
            if ((!this.staticFiles && nr.static_files) ||
                (this.staticFiles && nr.static_files && this.staticFiles.length !== nr.static_files.length)) {
                this.staticFiles = nr.static_files;
                this._cd.markForCheck();
            }
        });
    }
}
