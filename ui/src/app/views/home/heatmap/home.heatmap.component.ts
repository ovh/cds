import { AfterViewInit, Component, ViewChild } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { AppService } from 'app/app.service';
import { AuthentificationStore } from 'app/service/services.module';
import { finalize } from 'rxjs/operators';
import { Subscription } from 'rxjs/Subscription';
import { Event } from '../../../model/event.model';
import { HeatmapSearchCriterion } from '../../../model/heatmap.model';
import { PipelineStatus } from '../../../model/pipeline.model';
import { ProjectFilter, TimelineFilter } from '../../../model/timeline.model';
import { TimelineStore } from '../../../service/timeline/timeline.store';
import { AutoUnsubscribe } from '../../../shared/decorator/autoUnsubscribe';
import { ToastService } from '../../../shared/toast/ToastService';
import { ToolbarComponent } from './toolbar/toolbar.component';

@Component({
    selector: 'app-home-heatmap',
    templateUrl: './home.heatmap.html',
    styleUrls: ['./home.heatmap.scss']
})
@AutoUnsubscribe()
export class HomeHeatmapComponent implements AfterViewInit {

    loading = true;
    events: Array<Event>;
    projects: Array<string>;
    workflows = new Object();
    unfilteredWorkflows = new Object();
    properties = new Array<string>();

    eventsIds = new Array();
    groupedEvents = new Object();
    unfilteredGroupedEvents = new Object();

    timelineSub: Subscription;

    currentItem = 0;
    pipelineStatus = PipelineStatus;

    filter: TimelineFilter;
    filterSub: Subscription;

    heatmapSearch: HeatmapSearchCriterion;

    @ViewChild('toolbar', { read: ToolbarComponent, static: false }) toolbar: ToolbarComponent;

    static clone(objectToCopy) {
        return (JSON.parse(JSON.stringify(objectToCopy)));
    }

    constructor(
        private _timelineStore: TimelineStore,
        private _toast: ToastService,
        public _translate: TranslateService,
        private _appService: AppService,
        private _authStore: AuthentificationStore
    ) {
        this.filter = new TimelineFilter();
    }

    ngAfterViewInit() {
        if (!this._authStore.isConnected) {
            return;
        }
        this.filterSub = this._timelineStore.getFilter().subscribe(f => {
            this.filter = f;
            this._appService.initFilter(this.filter);

            this.toolbar.getFilter().subscribe((filter) => {
                this.heatmapSearch = filter;
                this.filterEvents();
            });

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
                            const allowed = ['workflow_name', 'status', 'tag'];
                            this.properties = Object.keys(event).filter(p => allowed.indexOf(p) !== -1);
                            if (event['tag']) {
                                event['tag'] = event['tag'].map((tag) => {
                                    return tag.value ? tag.value : tag;
                                });
                            }
                            if (!this.unfilteredGroupedEvents[event.project_key]) {
                                this.unfilteredGroupedEvents[event.project_key] = new Object();
                            }
                            if (!this.unfilteredGroupedEvents[event.project_key][event.workflow_name]) {
                                this.unfilteredGroupedEvents[event.project_key][event.workflow_name] = new Array();
                            }

                            let eventsWorkflow = this.unfilteredGroupedEvents[event.project_key][event.workflow_name];
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
                            this.unfilteredGroupedEvents[event.project_key][event.workflow_name] = eventsWorkflow;
                            if (!this.unfilteredWorkflows[event.project_key]) {
                                this.unfilteredWorkflows[event.project_key] = new Array<string>();
                            }
                            if (this.unfilteredWorkflows[event.project_key].lastIndexOf(event.workflow_name) === -1) {
                                this.unfilteredWorkflows[event.project_key].push(event.workflow_name);
                            }
                        });

                        this.currentItem = this.events.length;

                        this.projects = Object.keys(this.unfilteredGroupedEvents).sort();
                        this.filterEvents();
                    });
            }
        });

    }

    /**
     * Make a copy fetched datas and filter on it.
     * This keep all events to let search them on demand.
     */
    filterEvents() {
        const ONE_HOUR = 60 * 60 * 1000;
        Object.keys(this.unfilteredGroupedEvents).forEach(proj => {
          Object.keys(this.unfilteredGroupedEvents[proj]).forEach(workflow => {
            this.unfilteredGroupedEvents[proj][workflow] = this.unfilteredGroupedEvents[proj][workflow].filter(event => {
              let diffWithNow = new Date().getTime() - new Date(event.timestamp).getTime();
              if (event.pipelineStatus === 'Building') {
                return true;
              }
              return diffWithNow < ONE_HOUR;
            });
            if (this.unfilteredGroupedEvents[proj][workflow].length === 0) {
              delete this.unfilteredGroupedEvents[proj][workflow];
            }
          });
          if (Object.keys(this.unfilteredGroupedEvents[proj]).length === 0) {
            delete this.unfilteredGroupedEvents[proj];
          }
        });
        this.workflows = HomeHeatmapComponent.clone(this.unfilteredWorkflows);
        this.projects = Object.keys(this.unfilteredGroupedEvents).sort();
        this.groupedEvents = HomeHeatmapComponent.clone(this.unfilteredGroupedEvents);
        if (this.heatmapSearch) {
            // filter projects list
            if (this.heatmapSearch.projects && this.heatmapSearch.projects.length > 0) {
                this.projects = Object.keys(this.groupedEvents).filter((p) => {
                    return this.heatmapSearch.projects.indexOf(p) !== -1;
                }).sort();
                const projectsToFilter = Object.keys(this.unfilteredWorkflows).filter(proj => this.projects.indexOf(proj) === -1);
                projectsToFilter.forEach(proj => {
                    delete this.workflows[proj];
                });
            }

            if (this.heatmapSearch.searchCriteria) {
                const searchingCriteriaLowerCase = this.heatmapSearch.searchCriteria.toLowerCase();

                const projectsToFilter = new Array<string>();
                // filter events non matched
                Object.keys(this.groupedEvents).forEach((project) => {
                    let projectLength = 0;
                    if (this.groupedEvents[project]) {
                        Object.keys(this.groupedEvents[project]).forEach((workflow) => {
                            const workflowName = workflow.toLowerCase();
                            if (this.groupedEvents[project][workflow]) {
                                this.groupedEvents[project][workflow] = this.groupedEvents[project][workflow].filter(event => {
                                    const tags = JSON.stringify(event.tag).toLowerCase();
                                    return workflowName.indexOf(searchingCriteriaLowerCase) !== -1 ||
                                        tags.indexOf(searchingCriteriaLowerCase) !== -1;
                                });
                                projectLength += this.groupedEvents[project][workflow].length;
                            }
                        });
                        if (projectLength === 0 && this.groupedEvents[project]) {
                            projectsToFilter.push(project);
                        }
                    }
                });

                // delete empty projects
                projectsToFilter.forEach(proj => {
                    delete this.workflows[proj];
                });
                this.projects = Object.keys(this.groupedEvents).filter((p) => {
                    return projectsToFilter.indexOf(p) === -1;
                }).sort();
            }
        }
    };

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
