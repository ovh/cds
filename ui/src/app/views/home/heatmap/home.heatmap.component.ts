import { Component, OnInit } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { AppService } from 'app/app.service';
import { AuthentificationStore } from 'app/service/services.module';
import { finalize } from 'rxjs/operators';
import { Subscription } from 'rxjs/Subscription';
import { Event } from '../../../model/event.model';
import { PipelineStatus } from '../../../model/pipeline.model';
import { ProjectFilter, TimelineFilter } from '../../../model/timeline.model';
import { TimelineStore } from '../../../service/timeline/timeline.store';
import { AutoUnsubscribe } from '../../../shared/decorator/autoUnsubscribe';
import { ToastService } from '../../../shared/toast/ToastService';


@Component({
    selector: 'app-home-heatmap',
    templateUrl: './home.heatmap.html',
    styleUrls: ['./home.heatmap.scss']
})
@AutoUnsubscribe()
export class HomeHeatmapComponent implements OnInit {

    loading = true;
    events: Array<Event>;
    projects: Array<string>;
    workflows = new Object();
    properties = new Array<string>();

    eventsIds = new Array();
    groupedEvents = new Object();

    timelineSub: Subscription;
    selectedTab = 'heatmap';

    currentItem = 0;
    pipelineStatus = PipelineStatus;

    filter: TimelineFilter;
    filterSub: Subscription;

    constructor(
        private _timelineStore: TimelineStore,
        private _toast: ToastService,
        public _translate: TranslateService,
        private _appService: AppService,
        private _authStore: AuthentificationStore
    ) {
        this.filter = new TimelineFilter();
    }

    ngOnInit(): void {
        if (!this._authStore.isConnected) {
            return;
        }
        this.filterSub = this._timelineStore.getFilter().subscribe(f => {
            this.filter = f;
            this._appService.initFilter(this.filter);

            if (this.timelineSub) {
                this.timelineSub.unsubscribe();
            }
            if (f) {

                this.timelineSub = this._timelineStore.alltimeline()
                    .pipe(finalize(() => this.loading = false))
                    .subscribe(es => {
                        if (!es) {
                            return;
                        }
                        this.loading = false;
                        this.events = es.toArray().filter((el, i, a) => i === a.indexOf(el));

                        this.events.forEach((event) => {
                            const allowed = ['workflow_name', 'status'];
                            this.properties = Object.keys(event).filter(p => allowed.indexOf(p) !== -1);

                            if (!this.groupedEvents[event.project_key]) {
                                this.groupedEvents[event.project_key] = new Object();
                            }
                            if (!this.groupedEvents[event.project_key][event.workflow_name]) {
                                this.groupedEvents[event.project_key][event.workflow_name] = new Array();
                            }

                            let eventsWorkflow = this.groupedEvents[event.project_key][event.workflow_name];
                            eventsWorkflow = eventsWorkflow.filter((pendingEvent: Event) => {
                                if (event.workflow_name === pendingEvent.workflow_name
                                    && event.workflow_run_num === pendingEvent.workflow_run_num) {
                                    return event.timestamp < pendingEvent.timestamp;
                                }
                                return true;
                            });
                            if (eventsWorkflow.length > 0 && event.timestamp > eventsWorkflow[0].timestamp) {
                                eventsWorkflow = eventsWorkflow.filter((pendingEvent: Event) => {
                                    if (event.workflow_name === pendingEvent.workflow_name
                                        && event.workflow_run_num === pendingEvent.workflow_run_num) {
                                        return false;
                                    }
                                    return true;
                                });
                                eventsWorkflow.push(event);
                            } else if (eventsWorkflow.length === 0) {
                                eventsWorkflow.push(event);
                            }
                            eventsWorkflow = eventsWorkflow.filter((el, i, a) => i === a.indexOf(el));
                            this.groupedEvents[event.project_key][event.workflow_name] = eventsWorkflow;
                            if (!this.workflows[event.project_key]) {
                                this.workflows[event.project_key] = new Array<string>();
                            }
                            if (this.workflows[event.project_key].lastIndexOf(event.workflow_name) === -1) {
                                this.workflows[event.project_key].push(event.workflow_name);
                            }
                        });

                        this.currentItem = this.events.length;

                        this.projects = Object.keys(this.groupedEvents).sort();
                    });
            }
        });
    }

    selectTab(t: string): void {
        this.selectedTab = t;
    }

    addFilter(project_key: string): void {
        if (!this.filter.projects) {
            this.filter.projects = new Array<ProjectFilter>();
        }

        let pFilter = this.filter.projects.find(p => p.key === project_key);
        if (!pFilter) {
            pFilter = new ProjectFilter();
            pFilter.key = project_key;
            this.filter.projects.push(pFilter);
        }
        this._timelineStore.saveFilter(this.filter).subscribe(() => {
            this._toast.success('', this._translate.instant('timeline_filter_updated'));
        });
        delete this.groupedEvents[project_key];
    }
}
