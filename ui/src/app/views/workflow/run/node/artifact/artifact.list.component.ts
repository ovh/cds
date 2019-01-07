import { Component, Input, NgZone } from '@angular/core';
import { environment } from '../../../../../../environments/environment';
import { WorkflowNodeRunArtifact, WorkflowNodeRunStaticFiles } from '../../../../../model/workflow.run.model';
import { Table } from '../../../../../shared/table/table';

@Component({
    selector: 'app-workflow-artifact-list',
    templateUrl: './artifact.list.html',
    styleUrls: ['./artifact.list.scss']
})
export class WorkflowRunArtifactListComponent extends Table<WorkflowNodeRunArtifact> {

    @Input() artifacts: Array<WorkflowNodeRunArtifact>;
    @Input() staticFiles: Array<WorkflowNodeRunStaticFiles>;

    // Allow angular update from work started outside angular context
    zone: NgZone;
    filter: string;

    constructor() {
        super();
        this.zone = new NgZone({ enableLongStackTrace: false });
    }

    getData(): Array<WorkflowNodeRunArtifact> {
        if (!this.filter) {
            return this.artifacts;
        }
        return this.artifacts.filter(v => (v.name.indexOf(this.filter) !== -1 || v.sha512sum.indexOf(this.filter) !== -1));
    }

    getHumainFileSize(size: number): string {
        let i = Math.floor(Math.log(size) / Math.log(1024));
        let hSize = (size / Math.pow(1024, i)).toFixed(2);
        return hSize + ' ' + ['B', 'kB', 'MB', 'GB', 'TB'][i];
    }

    getUrl(a: WorkflowNodeRunArtifact): string {
        return environment.apiURL + '/workflow/artifact/' + a.download_hash;
    }
}
