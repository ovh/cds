import {Component, Input, OnInit, ViewChild} from '@angular/core';
import {IdName, Label, Project} from '../../../../model/project.model';
import {Warning} from '../../../../model/warning.model';
import {ProjectStore} from '../../../../service/project/project.store';
import {LabelsEditComponent} from '../../../../shared/labels/edit/labels.edit.component';

@Component({
    selector: 'app-project-workflows',
    templateUrl: './workflow.list.html',
    styleUrls: ['./workflow.list.scss']
})
export class ProjectWorkflowListComponent implements OnInit {

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
    if (project.labels) {
      let labelFilter = this.labelFilter.toLowerCase();
      this.filteredLabels = project.labels.filter((lbl) => lbl.name.toLowerCase().indexOf(labelFilter) !== -1);
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

  set labelFilter(filter: string) {
    this._filterLabel = filter;
    if (this.project.labels) {
      let filterLower = filter.toLowerCase();
      this.filteredLabels = this.project.labels.filter((lbl) => lbl.name.toLowerCase().indexOf(filterLower) !== -1);
    }
  }
  get labelFilter(): string {
    return this._filterLabel;
  }

  // Modal
  @ViewChild('projectLabels')
  projectLabels: LabelsEditComponent;

  _project: Project;
  _filter = '';
  _filterLabel = '';

  viewMode: 'blocs'|'labels'|'lines' = 'blocs';
  filteredWorkflows: Array<IdName> = [];
  filteredLabels: Array<Label> = [];
  loadingLabel = false;

  constructor(private _projectStore: ProjectStore) {

  }

  ngOnInit() {
    this.viewMode = this._projectStore.getWorkflowViewMode(this.project.key);
  }

  editLabels() {
    if (this.projectLabels && this.projectLabels.show) {
      this.projectLabels.show();
    }
  }

  setViewMode(mode: 'blocs'|'labels'|'lines') {
    this.viewMode = mode;
    this._projectStore.setWorkflowViewMode(this.project.key, mode);
  }
}
