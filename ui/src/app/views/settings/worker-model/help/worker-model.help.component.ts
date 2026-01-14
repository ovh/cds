import { ChangeDetectionStrategy, Component } from '@angular/core';

@Component({
    standalone: false,
    selector: 'app-worker-model-help',
    templateUrl: './worker-model.help.html',
    styleUrls: ['./worker-model.help.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class WorkerModelHelpComponent { }
