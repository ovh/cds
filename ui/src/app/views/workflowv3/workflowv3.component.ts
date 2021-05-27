import { ChangeDetectionStrategy, Component, OnDestroy } from '@angular/core';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';

@Component({
    selector: 'app-workflowv3',
    templateUrl: './workflowv3.html',
    styleUrls: ['./workflowv3.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowV3Component implements OnDestroy {
    constructor() { }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT
}
