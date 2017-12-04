import {Component, Input, OnInit} from '@angular/core';
import {Scheduler} from '../../../../../model/scheduler.model';
import {Project} from '../../../../../model/project.model';
import {Application} from '../../../../../model/application.model';
import {Pipeline} from '../../../../../model/pipeline.model';
import {unionBy, differenceBy} from 'lodash';

@Component({
    selector: 'app-application-scheduler-form',
    templateUrl: './scheduler.form.html',
    styleUrls: ['./scheduler.form.scss']
})
export class ApplicationSchedulerFormComponent implements OnInit {

    @Input() application: Application;
    @Input() project: Project;
    @Input() edit: boolean;
    @Input() scheduler: Scheduler;
    @Input() pipeline: Pipeline;

    constructor() {

    }

    ngOnInit() {
        if (this.scheduler && this.pipeline) {
            let filteredPipelineParams = this.pipeline.parameters ? this.pipeline.parameters.filter((p) => p.type !== 'list') : [];
            this.scheduler.args = unionBy(this.scheduler.args || [], filteredPipelineParams, 'id');

            let missingParams = differenceBy(this.pipeline.parameters, this.scheduler.args, 'id');
            missingParams = missingParams.map((param) => {
                let paramsSplitted = param.value.split(';');
                if (!paramsSplitted.length) {
                    return param;
                }

                return Object.assign({}, param, {value: paramsSplitted[0]});
            });
            this.scheduler.args = this.scheduler.args.concat(missingParams);
        }
    }
}
