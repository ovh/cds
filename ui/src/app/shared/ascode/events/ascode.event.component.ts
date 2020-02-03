import { ChangeDetectionStrategy, Component, Input } from '@angular/core';
import { AsCodeEvents } from 'app/model/ascode.model';

@Component({
    selector: 'app-ascode-event',
    templateUrl: './ascode.event.html',
    styleUrls: ['./ascode.event.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class AsCodeEventComponent {

    @Input() events: Array<AsCodeEvents>;
    @Input() repo: string;


    resyncEvents(): void {

    }
}
