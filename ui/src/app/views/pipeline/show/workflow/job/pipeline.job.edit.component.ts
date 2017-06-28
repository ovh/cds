import {Component, Input} from '@angular/core';
import {Project} from '../../../../../model/project.model';
import {Job} from '../../../../../model/job.model';
import {Pipeline} from '../../../../../model/pipeline.model';
import {cloneDeep} from 'lodash';

@Component({
    selector: 'app-pipeline-job',
    templateUrl: './pipeline.job.html',
    styleUrls: ['./pipeline.job.scss']
})
export class PipelineJobComponent {

    @Input() project: Project;
    @Input() edit = false;
    @Input() suggest: Array<string>;
    @Input() pipeline: Pipeline;

    @Input('job')
    set job(data: Job) {
        this.editableJob = cloneDeep(data);
    }

    editableJob: Job;

    constructor() {
    }
}
