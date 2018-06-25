import {Component, OnInit} from '@angular/core';
import {Subscription} from 'rxjs/Subscription';
import {Event} from '../../../model/event.model';
import {TimelineStore} from '../../../service/timeline/timeline.store';
import {AutoUnsubscribe} from '../../../shared/decorator/autoUnsubscribe';

@Component({
    selector: 'app-home-timeline',
    templateUrl: './home.timeline.html',
    styleUrls: ['./home.timeline.scss']
})
@AutoUnsubscribe()
export class HomeTimelineComponent implements OnInit {

    loading = true;
    events: Array<Event>;

    timelineSub: Subscription;


    constructor(private _timelineStore: TimelineStore) {

    }

    ngOnInit(): void {
        this.timelineSub = this._timelineStore.alltimeline().subscribe(es => {
            if (!es) {
                return;
            }
            this.loading = false;
            this.events = es.toArray();
        });
    }
}
