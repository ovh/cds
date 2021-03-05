import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, OnInit } from '@angular/core';
import { Select, Store } from '@ngxs/store';
import {
    WorkflowNodeRun,
    WorkflowNodeRunArtifact,
    WorkflowNodeRunStaticFiles,
    WorkflowRunResultArtifact
} from 'app/model/workflow.run.model';

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

    artifacts: Array<WorkflowNodeRunArtifact>;
    staticFiles: Array<WorkflowNodeRunStaticFiles>;

    filter: Filter<WorkflowNodeRunArtifact>;
    columns: Array<Column<WorkflowNodeRunArtifact>>;

    constructor(private _cd: ChangeDetectorRef, private _store: Store) {
        this.filter = f => {
            const lowerFilter = f.toLowerCase();
            return d => d.name.toLowerCase().indexOf(lowerFilter) !== -1 ||
                d.sha512sum.toLowerCase().indexOf(lowerFilter) !== -1
        };
        this.columns = [
            <Column<WorkflowNodeRunArtifact>>{
                type: ColumnType.LINK,
                name: 'artifact_name',
                selector: (a: WorkflowNodeRunArtifact) => {
                    let size = this.getHumainFileSize(a.size);
                    let link = `./cdsapi/workflow/artifact/${a.download_hash}`
                    if (!a.id) {
                        link = `./cdscdn/item/run-result/${a.download_hash}/download`
                    }
                    return {
                        link,
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
                name: 'MD5 Sum',
                selector: (a: WorkflowNodeRunArtifact) => a.md5sum
            }
        ];
    }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit(): void {
        this.nodeRunSubs = this.nodeRun$.subscribe(nr => {
            if (!nr) {
                return;
            }
            let resultArtifacts = nr?.results.filter(r => r.type === 'artifact').map(r => <WorkflowRunResultArtifact>r.data);
            if (!resultArtifacts) {
                resultArtifacts = new Array<WorkflowRunResultArtifact>();
            }
            if ( (!this.artifacts && (nr.artifacts || resultArtifacts.length > 0)) || (this.artifacts && nr.artifacts && this.artifacts.length !== (nr.artifacts.length + resultArtifacts.length))) {
                this.artifacts = new Array<WorkflowNodeRunArtifact>();
                if (nr.artifacts) {
                    this.artifacts.push(...nr.artifacts);
                }
                this.artifacts.push(...this.toWorkflowNodeRunArtifacts(resultArtifacts));
                this._cd.markForCheck();
            }
            if ((!this.staticFiles && nr.static_files) ||
                (this.staticFiles && nr.static_files && this.staticFiles.length !== nr.static_files.length)) {
                this.staticFiles = nr.static_files;
                this._cd.markForCheck();
            }
        });
    }

    toWorkflowNodeRunArtifacts(results: Array<WorkflowRunResultArtifact>): Array<WorkflowNodeRunArtifact> {
        let arts = new Array<WorkflowNodeRunArtifact>();
        results.forEach(r => {
            let a = new WorkflowNodeRunArtifact();
            a.download_hash = r.cdn_hash;
            a.md5sum = r.md5;
            a.size =  r.size;
            a.name = r.name;
            arts.push(a);
        })
        return arts;
    }

    getHumainFileSize(size: number): string {
        let i = Math.floor(Math.log(size) / Math.log(1024));
        let hSize = (size / Math.pow(1024, i)).toFixed(2);
        return hSize + ' ' + ['B', 'kB', 'MB', 'GB', 'TB'][i];
    }
}
