import { ChangeDetectionStrategy, Component, Input } from '@angular/core';
import { PipelineStatus } from 'app/model/pipeline.model';

@Component({
    selector: 'app-status-icon',
    templateUrl: './status.icon.html',
    styleUrls: ['./status.icon.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class StatusIconComponent {
    @Input() status: string;
    @Input() value: string;
    @Input() optional: boolean;
    @Input() loader = true;
    pipelineStatusEnum = PipelineStatus;

    constructor() { }
}
