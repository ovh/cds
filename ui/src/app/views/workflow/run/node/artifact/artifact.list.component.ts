import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, OnInit } from '@angular/core';
import { Store } from '@ngxs/store';
import { ProjectIntegration } from 'app/model/integration.model';
import {
    UIArtifact,
    WorkflowRunResult
} from 'app/model/workflow.run.model';
import { WorkflowHelper } from 'app/service/workflow/workflow.helper';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { Column, ColumnType, Filter } from 'app/shared/table/data-table.component';
import { WorkflowState } from 'app/store/workflow.state';
import { Subscription } from 'rxjs';

@Component({
    selector: 'app-workflow-artifact-list',
    templateUrl: './artifact.list.html',
    styleUrls: ['./artifact.list.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowRunArtifactListComponent implements OnInit, OnDestroy {
    nodeRunSubs: Subscription;

    runResult: Array<WorkflowRunResult>
    uiArtifacts: Array<UIArtifact>;

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
        this.nodeRunSubs = this._store.select(WorkflowState.getSelectedNodeRun()).subscribe(nr => {
            if (!nr) {
                return;
            }

            let computeArtifact = false;
            if (nr.results && (!this.runResult || nr.results.length !== this.runResult.length)) {
                computeArtifact = true;
            }
            if (computeArtifact) {
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

                this._cd.markForCheck();
            }
        });
    }
}
