import { ChangeDetectionStrategy, Component, Input } from '@angular/core';
import { PipelineStatus } from 'app/model/pipeline.model';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';

@Component({
    selector: 'app-workflow-wnode-fork',
    templateUrl: './node.fork.html',
    styleUrls: ['./node.fork.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowWNodeForkComponent {
    @Input() noderunStatus: string;
    pipelineStatus = PipelineStatus;
}
