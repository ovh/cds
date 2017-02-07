import {Component, Input} from '@angular/core';
import {Table} from '../table/table';
import {PipelineBuild} from '../../model/pipeline.model';

@Component({
    selector: 'app-history',
    templateUrl: './history.html',
    styleUrls: ['./history.scss']
})
export class HistoryComponent extends Table {

    @Input() history: Array<PipelineBuild>;
    @Input() currentBuild: PipelineBuild;

    constructor() {
        super();
    }

    getData(): any[] {
        return this.history;
    }

    getTriggerSource(pb: PipelineBuild): string {
        if (pb.trigger.scheduled_trigger) {
            return 'CDS scheduler';
        }
        if (pb.trigger.triggered_by && pb.trigger.triggered_by.username && pb.trigger.triggered_by.username !== '') {
            return pb.trigger.triggered_by.username;
        }
        if (pb.trigger.vcs_author) {
            return pb.trigger.vcs_author;
        }
        return '';
    }
}

