import { Component, Input} from '@angular/core';
import {Project, IdName} from '../../../../model/project.model';

@Component({
    selector: 'app-project-pipelines',
    templateUrl: './pipeline.list.html',
    styleUrls: ['./pipeline.list.scss']
})
export class ProjectPipelinesComponent {

  @Input()
  set project(project: Project) {
    this._project = project;
    if (project.pipeline_names) {
      let filter = this.filter.toLowerCase();
      this.filteredPipelines = project.pipeline_names.filter((pip) => pip.name.toLowerCase().indexOf(filter) !== -1);
    }
  }
  get project(): Project {
    return this._project;
  }

  set filter(filter: string) {
    this._filter = filter;
    if (this.project.pipeline_names) {
      let filterLower = filter.toLowerCase();
      this.filteredPipelines = this.project.pipeline_names.filter((pip) => pip.name.toLowerCase().indexOf(filterLower) !== -1);
    }
  }
  get filter(): string {
    return this._filter;
  }

  _project: Project;
  _filter = '';

  filteredPipelines: Array<IdName> = [];

    constructor() {

    }
}
