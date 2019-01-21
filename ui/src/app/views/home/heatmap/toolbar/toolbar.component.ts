import { Component } from '@angular/core';
import { Subject } from 'rxjs';
import { finalize } from 'rxjs/operators';
import { HeatmapSearchCriterion } from '../../../../model/heatmap.model';
import { Project } from '../../../../model/project.model';
import { ProjectStore } from '../../../../service/project/project.store';
import { AutoUnsubscribe } from '../../../../shared/decorator/autoUnsubscribe';

@Component({
  selector: 'app-heatmap-toolbar',
  templateUrl: './toolbar.component.html',
  styleUrls: ['./toolbar.component.scss']
})
@AutoUnsubscribe()
export class ToolbarComponent {

  projects: Array<Project>;
  searchCriterion: string;

  filter: HeatmapSearchCriterion;
  subject = new Subject<any>();
  selectedProjectKeys: Project[];
  loading: boolean;

  constructor(private _projectStore: ProjectStore) {
    this.loading = true;
    this._projectStore.getProjectsList()
      .pipe(finalize(() => this.loading = false))
      .subscribe(ps => {
        if (ps) {
            this.projects = ps.toArray();
        }
    });
  }

  search() {
    this.subject.next(new HeatmapSearchCriterion(this.selectedProjectKeys, this.searchCriterion));
  }

  getFilter() {
    return this.subject;
  }
}
