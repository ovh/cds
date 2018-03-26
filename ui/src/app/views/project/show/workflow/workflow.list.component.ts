import {Component, Input} from '@angular/core';
import {Project} from '../../../../model/project.model';

@Component({
    selector: 'app-project-workflows',
    templateUrl: './workflow.list.html',
    styleUrls: ['./workflow.list.scss']
})
export class ProjectWorkflowListComponent {

  @Input()
  set project(project: Project) {
    this._project = project;
    if (project.workflow_names) {
      let filter = this.filter.toLowerCase();
      this.filteredWorkflows = project.workflow_names.filter((wf) => wf.toLowerCase().indexOf(filter) !== -1);
    }
  }
  get project(): Project {
    return this._project;
  }

  set filter(filter: string) {
    this._filter = filter;
    if (this.project.workflow_names) {
      let filterLower = filter.toLowerCase();
      this.filteredWorkflows = this.project.workflow_names.filter((wf) => wf.toLowerCase().indexOf(filterLower) !== -1);
    }
  }
  get filter(): string {
    return this._filter;
  }

  _project: Project;
  _filter = '';

  filteredWorkflows: Array<string> = [];

    constructor() { }
}
