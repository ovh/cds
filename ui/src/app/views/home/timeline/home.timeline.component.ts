import {Component, OnInit} from '@angular/core';
import {TranslateService} from '@ngx-translate/core';
import {Subscription} from 'rxjs/Subscription';
import {Event} from '../../../model/event.model';
import {PipelineStatus} from '../../../model/pipeline.model';
import {ProjectFilter, TimelineFilter} from '../../../model/timeline.model';
import {TimelineStore} from '../../../service/timeline/timeline.store';
import {AutoUnsubscribe} from '../../../shared/decorator/autoUnsubscribe';
import {ToastService} from '../../../shared/toast/ToastService';

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
    filterSub: Subscription;

    constructor(private _timelineStore: TimelineStore, private _translate: TranslateService,
                private _toast: ToastService) {
        this.filter = new TimelineFilter();
    }

    ngOnInit(): void {
        this.filterSub = this._timelineStore.getFilter().subscribe(f => {
            this.filter = f;

            if (this.timelineSub) {
                this.timelineSub.unsubscribe();
            }
            if (f) {
                this.timelineSub = this._timelineStore.alltimeline().subscribe(es => {
                    if (!es) {
                        return;
                    }
                    this.loading = false;
                    this.events = es.toArray();
                    this.currentItem = this.events.length;
                });
            }
        });
    }

    selectTab(t: string): void {
        this.selectedTab = t;
    }

    onScroll() {
        this._timelineStore.getMore(this.currentItem + 1, false);
    }

    addFilter(e: Event): void {
        if (!this.filter.projects) {
            this.filter.projects = new Array<ProjectFilter>();
        }
        let pFilter = this.filter.projects.find(p => p.key === e.project_key);
        if (!pFilter) {
            pFilter = new ProjectFilter();
            pFilter.key = e.project_key;
            this.filter.projects.push(pFilter);
        }

        if (!pFilter.workflow_names) {
            pFilter.workflow_names = new Array<string>();
        }
        let wName = pFilter.workflow_names.find(w => w === e.workflow_name);
        if (!wName) {
            pFilter.workflow_names.push(e.workflow_name);
        }
        this._timelineStore.saveFilter(this.filter).subscribe(() => {
            this._toast.success('', this._translate.instant('timeline_filter_updated'));
        });
    }
}
