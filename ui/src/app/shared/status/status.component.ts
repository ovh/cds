import {Component, Input} from '@angular/core';
import {PipelineStatus} from '../../model/pipeline.model';

@Component({
    selector: 'app-status-icon',
    templateUrl: './status.icon.html',
    styleUrls: ['./status.icon.scss']
})
export class StatusIconComponent {

    @Input() status: string;
    @Input() value: string;
    pipelineStatusEnum = PipelineStatus;

    constructor() { }
}
