import {Component, Input, OnInit} from '@angular/core';
import {finalize, first} from 'rxjs/operators';
import {LoadOpts, Project} from '../../../../model/project.model';
import {ProjectFilter, TimelineFilter} from '../../../../model/timeline.model';
import {ProjectStore} from '../../../../service/project/project.store';
import {AutoUnsubscribe} from '../../../../shared/decorator/autoUnsubscribe';

@Component({
    selector: 'app-home-timeline-filter',
    templateUrl: './home.timeline.filter.html',
    styleUrls: ['./home.timeline.filter.scss']
})
@AutoUnsubscribe()
export class HomeTimelineFilterComponent {

    @Input() filter: TimelineFilter;

    projects: Array<Project>;

    constructor(private _projectStore: ProjectStore) {
        this._projectStore.getProjectsList().subscribe(ps => {
            if (ps) {
                this.projects = ps.toArray();
                this.projects.forEach(p => {
                   let pf = new ProjectFilter();
                   pf.key = p.key;
                   this.filter.projects.set(pf.key, pf);
                });
            }
        });
    }

    loadProjectWorkflow(p: Project) {
        if (p.workflow_names) {
            return;
        }
        p.loading = true;
        let opts = new Array<LoadOpts>();
        opts.push(new LoadOpts('withWorkflowNames', 'workflow_names'));
        this._projectStore.resync(p.key, opts).pipe(first(), finalize(() => p.loading = false)).subscribe(proj => {
           p.workflow_names = proj.workflow_names;
           let pf = this.filter.projects.get(p.key);
           if (proj.workflow_names && pf.allWorkflows) {
               proj.workflow_names.forEach(w => {
                  pf.workflowName.push(w.name);
               });
           }
        });
    }

}
