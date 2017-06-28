import {Component, Input} from '@angular/core';
import {Table} from '../table/table';
import {PipelineBuild} from '../../model/pipeline.model';
import {Project} from '../../model/project.model';

@Component({
    selector: 'app-history',
    templateUrl: './history.html',
    styleUrls: ['./history.scss']
})
export class HistoryComponent extends Table {

    @Input() project: Project;
    @Input() history: Array<PipelineBuild>;
    @Input() currentBuild: PipelineBuild;

    constructor() {
        super();
    }

    getData(): any[] {
        return this.history;
    }

    getTriggerSource(pb: PipelineBuild): string {
        return PipelineBuild.GetTriggerSource(pb);
    }
}

