import {Component, Input} from '@angular/core';
import {Broadcast} from '../../../model/broadcast.model';

@Component({
    selector: 'app-home-broadcast',
    templateUrl: './home.broadcast.html',
    styleUrls: ['./home.broadcast.scss']
})
export class HomeBroadcastComponent {

    @Input() loading: boolean;
    @Input() broadcasts:  Array<Broadcast>;
}
