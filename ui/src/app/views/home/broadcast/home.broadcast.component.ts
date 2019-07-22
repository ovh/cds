import { ChangeDetectionStrategy, Component, Input } from '@angular/core';
import { Broadcast } from 'app/model/broadcast.model';

@Component({
    selector: 'app-home-broadcast',
    templateUrl: './home.broadcast.html',
    styleUrls: ['./home.broadcast.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class HomeBroadcastComponent {

    @Input() loading: boolean;
    @Input() broadcasts:  Array<Broadcast>;
}
