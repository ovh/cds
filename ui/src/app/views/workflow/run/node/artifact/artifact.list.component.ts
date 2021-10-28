import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, OnInit } from '@angular/core';
import { Select, Store } from '@ngxs/store';
import {
    UIArtifact,
    WorkflowNodeRun,
    WorkflowNodeRunArtifact,
    WorkflowNodeRunStaticFiles, WorkflowRunResult,
    WorkflowRunResultArtifact, WorkflowRunResultArtifactManager, WorkflowRunResultStaticFile
} from 'app/model/workflow.run.model';

import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { Column, ColumnType, Filter } from 'app/shared/table/data-table.component';
import { WorkflowState } from 'app/store/workflow.state';
import { Observable, Subscription } from 'rxjs';
import { Workflow } from 'app/model/workflow.model';

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
                    if (a.size) {
                        let size = this.getHumainFileSize(a.size);
                        if (size) {
                            value += ` (${size})`;
                        }
                    }
                    return {
                        link,
                        value
                    };
                }
            },
            <Column<UIArtifact>>{
                name: 'Type of artifact',
                selector: (a: UIArtifact) => a.type
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
                    uiRunResults = this.toUIArtifact(w, nr.results);
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

    getHumainFileSize(size: number): string {
        if (size === 0) {
            return '';
        }
        let i = Math.floor(Math.log(size) / Math.log(1024));
        let hSize = (size / Math.pow(1024, i)).toFixed(2);
        return hSize + ' ' + ['B', 'kB', 'MB', 'GB', 'TB'][i];
    }

    private toUIArtifact(w: Workflow, results: Array<WorkflowRunResult>): Array<UIArtifact> {
        if (!results) {
            return [];
        }
        let integrationArtifactManagerURL = '';
        if (w?.integrations) {
            for (let i = 0; i < w.integrations.length; i++) {
               let integ = w.integrations[i];
               if (!integ.project_integration.model.artifact_manager) {
                   continue;
               }
               integrationArtifactManagerURL = integ?.project_integration?.config['url']?.value;
            }
        }

        return results.map(r => {
            switch (r.type) {
                case 'artifact':
                case 'coverage':
                    let data = <WorkflowRunResultArtifact>r.data;
                    let uiArtifact = new UIArtifact();
                    uiArtifact.link = `./cdscdn/item/run-result/${data.cdn_hash}/download`;
                    uiArtifact.md5 = data.md5;
                    uiArtifact.name = data.name;
                    uiArtifact.size = data.size;
                    uiArtifact.type = 'file';
                    return uiArtifact;
                case 'artifact-manager':
                    let dataAM = <WorkflowRunResultArtifactManager>r.data;
                    let uiArtifactAM = new UIArtifact();
                    uiArtifactAM.link = `${integrationArtifactManagerURL}${dataAM.repository_name}/${dataAM.path}`;
                    uiArtifactAM.md5 = dataAM.md5;
                    uiArtifactAM.name = dataAM.name;
                    uiArtifactAM.size = dataAM.size;
                    uiArtifactAM.type = dataAM.repository_type;
                    return uiArtifactAM;
                case 'static-file':
                    let dataSF = <WorkflowRunResultStaticFile>r.data;
                    let uiArtifactSF = new UIArtifact();
                    uiArtifactSF.link = dataSF.remote_url;
                    uiArtifactSF.name = dataSF.name;
                    uiArtifactSF.type = 'static file';
                    return uiArtifactSF;
            }
        });
    }
}
