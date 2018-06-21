import {Component, OnInit} from '@angular/core';
import {finalize, first} from 'rxjs/operators';
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


    constructor(private _timelineStore: TimelineStore) {

    }

    ngOnInit(): void {
        this._timelineStore.alltimeline().pipe(first(), finalize(() => this.loading = false )).subscribe(es => {
            this.events = es.toArray();
        });
    }
}
