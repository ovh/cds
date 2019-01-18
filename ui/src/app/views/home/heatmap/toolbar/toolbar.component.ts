import { Component, OnInit } from '@angular/core';
import { Subject } from 'rxjs';
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
export class ToolbarComponent implements OnInit {

  projects: Array<Project>;
  searchCriterion: string;

  filter: HeatmapSearchCriterion;
  subject = new Subject<any>();
  selectedProjectKeys: Project[];
  loading: boolean;

  constructor(private _projectStore: ProjectStore) {
    this.loading = true;
    this._projectStore.getProjectsList().subscribe(ps => {
        if (ps) {
            this.projects = ps.toArray();
        }
        this.loading = false;
    });
  }

  ngOnInit() {
  }

  search() {
    this.subject.next(new HeatmapSearchCriterion(this.selectedProjectKeys, this.searchCriterion));
  }

  getFilter() {
    return this.subject;
  }
}
