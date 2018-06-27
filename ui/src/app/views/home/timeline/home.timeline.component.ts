import {Component, OnInit} from '@angular/core';
import {Subscription} from 'rxjs/Subscription';
import {Event} from '../../../model/event.model';
import {PipelineStatus} from '../../../model/pipeline.model';
import {TimelineFilter} from '../../../model/timeline.model';
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
    selectedTab = 'timeline';

    currentItem = 0;
    pipelineStatus = PipelineStatus;

    filter: TimelineFilter;

    constructor(private _timelineStore: TimelineStore) {
        this.filter = new TimelineFilter();
    }

    ngOnInit(): void {
        this.timelineSub = this._timelineStore.alltimeline(this.currentItem).subscribe(es => {
            if (!es) {
                return;
            }
            this.loading = false;
            this.events = es.toArray();
            this.currentItem = this.events.length;
        });
    }

    selectTab(t: string): void {
        this.selectedTab = t;
    }

    onScroll() {
        this._timelineStore.getMore(this.currentItem + 1);
    }

    updateFilter(): void {
        console.log('BIM');
    }
}
