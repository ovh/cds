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
    selector: 'app-home-heatmap',
    templateUrl: './home.heatmap.html',
    styleUrls: ['./home.heatmap.scss']
})
@AutoUnsubscribe()
export class HomeHeatmapComponent implements OnInit {

    loading = true;
    events: Array<Event>;

    eventsIds = new Array();
    groupedEvents = new Object();

    timelineSub: Subscription;
    selectedTab = 'heatmap';

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
            //this.filter = f;

            if (this.timelineSub) {
                this.timelineSub.unsubscribe();
            }
            if (f) {

                this.timelineSub = this._timelineStore.alltimeline().subscribe(es => {
                    if (!es) {
                        return;
                    }
                    this.loading = false;
                    this.events = es.toArray().filter((el, i, a) => i === a.indexOf(el));

                    this.events.forEach((event) => {
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
                    });

                    this.currentItem = this.events.length;
                });
            }
        });
    }

    selectTab(t: string): void {
        this.selectedTab = t;
    }

    getProjects() {
        return Object.keys(this.groupedEvents);
    }

    getRuns(workflows: Array<any>) {
        const runs = new Array<Event>();
        Object.keys(workflows).forEach((workflow_name) => {
            let workflow = workflows[workflow_name];
            workflow.forEach((run) => {
                runs.push(run);
            });
        });
        return runs;
    }

    getProperties(e: Event) {
        const allowed = ['workflow_name', 'status'];
        return Object.keys(e).filter(p => allowed.indexOf(p) !== -1);
    }

    addFilter(project_key: string): void {
        if (!this.filter.projects) {
            this.filter.projects = new Array<ProjectFilter>();
        }
        console.log(this.filter);

        let pFilter = this.filter.projects.find(p => p.key === project_key);
        if (!pFilter) {
            pFilter = new ProjectFilter();
            pFilter.key = project_key;
            this.filter.projects.push(pFilter);
        }
        console.log(this.filter);
        /*this._timelineStore.saveFilter(this.filter).subscribe(() => {
            this._toast.success('', this._translate.instant('timeline_filter_updated'));
        });*/
        delete this.groupedEvents[project_key];
    }
}
