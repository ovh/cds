import {Component, Input} from '@angular/core';
import {Project, IdName} from '../../../../model/project.model';
import {Warning} from '../../../../model/warning.model';

@Component({
    selector: 'app-project-workflows',
    templateUrl: './workflow.list.html',
    styleUrls: ['./workflow.list.scss']
})
export class ProjectWorkflowListComponent {

   warnMap: Map<string, Array<Warning>>;
  @Input('warnings')
  set warnings(data: Array<Warning>) {
      if (data) {
          this.warnMap = new Map<string, Array<Warning>>();
          data.forEach(w => {
              let arr = this.warnMap.get(w.workflow_name);
              if (!arr) {
                  arr = new Array<Warning>();
              }
              arr.push(w);
              this.warnMap.set(w.workflow_name, arr);
          });
      }
  }

  @Input()
  set project(project: Project) {
    this._project = project;
    if (project.workflow_names) {
      let filter = this.filter.toLowerCase();
      this.filteredWorkflows = project.workflow_names.filter((wf) => wf.name.toLowerCase().indexOf(filter) !== -1);
    }
  }
  get project(): Project {
    return this._project;
  }

  set filter(filter: string) {
    this._filter = filter;
    if (this.project.workflow_names) {
      let filterLower = filter.toLowerCase();
      this.filteredWorkflows = this.project.workflow_names.filter((wf) => wf.name.toLowerCase().indexOf(filterLower) !== -1);
    }
  }
  get filter(): string {
    return this._filter;
  }

  _project: Project;
  _filter = '';

  filteredWorkflows: Array<IdName> = [];

    constructor() { }
}
