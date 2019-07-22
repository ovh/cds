import { ChangeDetectionStrategy, Component, Input } from '@angular/core';
import { environment } from 'app/../environments/environment';
import { WorkflowNodeRunArtifact, WorkflowNodeRunStaticFiles } from 'app/model/workflow.run.model';
import { Column, ColumnType, Filter } from 'app/shared/table/data-table.component';

@Component({
    selector: 'app-workflow-artifact-list',
    templateUrl: './artifact.list.html',
    styleUrls: ['./artifact.list.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class WorkflowRunArtifactListComponent {
    @Input() artifacts: Array<WorkflowNodeRunArtifact>;
    @Input() staticFiles: Array<WorkflowNodeRunStaticFiles>;

    filter: Filter<WorkflowNodeRunArtifact>;
    columns: Array<Column<WorkflowNodeRunArtifact>>;

    constructor() {
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
                        link: `${environment.apiURL}/workflow/artifact/${a.download_hash}`,
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

    getHumainFileSize(size: number): string {
        let i = Math.floor(Math.log(size) / Math.log(1024));
        let hSize = (size / Math.pow(1024, i)).toFixed(2);
        return hSize + ' ' + ['B', 'kB', 'MB', 'GB', 'TB'][i];
    }
}
