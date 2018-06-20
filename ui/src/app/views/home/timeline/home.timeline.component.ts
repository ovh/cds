import {Component, OnInit} from '@angular/core';
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

    events: Array<Event>;


    constructor(private _timelineStore: TimelineStore) {

    }

    ngOnInit(): void {
        this._timelineStore.alltimeline().subscribe(es => {
            this.events = es.toArray();
        });
    }
}
