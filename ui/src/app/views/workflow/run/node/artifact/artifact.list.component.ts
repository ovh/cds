import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnInit } from '@angular/core';
import { Select } from '@ngxs/store';
import { WorkflowNodeRun, WorkflowNodeRunArtifact, WorkflowNodeRunStaticFiles } from 'app/model/workflow.run.model';
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
export class WorkflowRunArtifactListComponent implements OnInit  {
    @Select(WorkflowState.getSelectedNodeRun()) nodeRun$: Observable<WorkflowNodeRun>;
    nodeRunSubs: Subscription;

    artifacts: Array<WorkflowNodeRunArtifact>;
    staticFiles: Array<WorkflowNodeRunStaticFiles>;

    filter: Filter<WorkflowNodeRunArtifact>;
    columns: Array<Column<WorkflowNodeRunArtifact>>;

    constructor(private _cd: ChangeDetectorRef) {
        this.filter = f => {
            const lowerFilter = f.toLowerCase();
            return d => {
                return d.name.toLowerCase().indexOf(lowerFilter) !== -1 ||
                    d.sha512sum.toLowerCase().indexOf(lowerFilter) !== -1;
            }
        };

        this.columns = [
            <Column<WorkflowNodeRunArtifact>>{
                type: ColumnType.LINK,
                name: 'artifact_name',
                selector: (a: WorkflowNodeRunArtifact) => {
                    let size = this.getHumainFileSize(a.size);
                    return {
                        link: `./cdsapi/workflow/artifact/${a.download_hash}`,
                        value: `${a.name} (${size})`
                    };
                }
            },
            <Column<WorkflowNodeRunArtifact>>{
                name: 'artifact_tag',
                selector: (a: WorkflowNodeRunArtifact) => a.tag
            },
            <Column<WorkflowNodeRunArtifact>>{
                type: ColumnType.TEXT_COPY,
                name: 'artifact_sha512',
                selector: (a: WorkflowNodeRunArtifact) => a.sha512sum
            }
        ];
    }
     ngOnInit(): void {
        this.nodeRunSubs = this.nodeRun$.subscribe(nr => {
            if (!nr) {
                return;
            }
            if ( (!this.artifacts && nr.artifacts) || (this.artifacts && nr.artifacts && this.artifacts.length !== nr.artifacts.length)) {
                this.artifacts = nr.artifacts;
                this._cd.markForCheck();
            }
            if ( (!this.staticFiles && nr.static_files) ||
                (this.staticFiles && nr.static_files && this.staticFiles.length !== nr.static_files.length )) {
                this.staticFiles = nr.static_files;
                this._cd.markForCheck();
            }
        });
     }

    getHumainFileSize(size: number): string {
        let i = Math.floor(Math.log(size) / Math.log(1024));
        let hSize = (size / Math.pow(1024, i)).toFixed(2);
        return hSize + ' ' + ['B', 'kB', 'MB', 'GB', 'TB'][i];
    }
}
