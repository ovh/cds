import { Component, EventEmitter, Input, Output, ViewChild } from '@angular/core';
import { Store } from '@ngxs/store';
import { LinkLabelOnWorkflow, UnlinkLabelOnWorkflow } from 'app/store/workflow.action';
import { cloneDeep } from 'lodash';
import { finalize } from 'rxjs/operators';
import { IdName, Label, Project } from '../../../../../model/project.model';
import { Warning } from '../../../../../model/warning.model';
import { HelpersService } from '../../../../../service/helpers/helpers.service';
import { LabelsEditComponent } from '../../../../../shared/labels/edit/labels.edit.component';

@Component({
  selector: 'app-project-workflows-labels',
  templateUrl: './workflow.list.labels.html',
  styleUrls: ['./workflow.list.labels.scss']
})
export class ProjectWorkflowListLabelsComponent {

  @Input('project')
  set project(project: Project) {
    this._project = cloneDeep(project);
    if (project) {
      let fakeLabel = new Label();
      fakeLabel.name = '...';
      if (Array.isArray(project.labels)) {
        this._project.labels.push(fakeLabel);
      } else {
        this._project.labels = [fakeLabel];
      }
    }
  }
  get project(): Project {
    return this._project;
  }
  @Input() warnMap: Map<string, Array<Warning>>;
  @Input('workflows')
  set workflows(workflows: IdName[]) {
    this._workflows = workflows;
    this.workflowLabelsMap = {};
    this.workflowLabelsMapByLabels = {};
    if (workflows) {
      workflows.forEach((wf) => {
        this.workflowLabelsMap[wf.name] = {};
        if (wf.labels && wf.labels.length > 0) {
          wf.labels.forEach((lbl) => {
            if (!lbl.font_color) {
              lbl.font_color = this._helpersService.getBrightnessColor(lbl.color);
            }
            this.workflowLabelsMap[wf.name][lbl.name] = true;
            if (!this.workflowLabelsMapByLabels[lbl.name]) {
              this.workflowLabelsMapByLabels[lbl.name] = [];
            }
            this.workflowLabelsMapByLabels[lbl.name].push(wf);
          });
        } else {
          if (!this.workflowLabelsMapByLabels['...']) {
            this.workflowLabelsMapByLabels['...'] = [];
          }
          this.workflowLabelsMapByLabels['...'].push(wf);
        }
      });
    }
  }
  get workflows(): IdName[] {
    return this._workflows;
  }
  @Input('labels')
  set labels(labels: Label[]) {
    this._labels = labels;
    if (labels) {
      let labelFilter = this.labelFilter.toLowerCase();
      this.filteredLabels = labels.filter((lbl) => lbl.name.toLowerCase().indexOf(labelFilter) !== -1);
    }
  }
  get labels(): Label[] {
    return this._labels;
  }
  @Output() edit = new EventEmitter<null>();

  set labelFilter(filter: string) {
    this._filterLabel = filter;
    if (this.labels) {
      let filterLower = filter.toLowerCase();
      this.filteredLabels = this.labels.filter((lbl) => lbl.name.toLowerCase().indexOf(filterLower) !== -1);
    }
  }
  get labelFilter(): string {
    return this._filterLabel;
  }

  // Modal
  @ViewChild('projectLabels')
  projectLabels: LabelsEditComponent;

  _project: Project;
  _labels: Label[];
  _workflows: IdName[];
  _filterLabel = '';

  workflowLabelsMap: {} = {};
  workflowLabelsMapByLabels: {} = {};
  filteredLabels: Array<Label> = [];
  loadingLabel = false;

  constructor(private store: Store, private _helpersService: HelpersService) { }

  linkLabelToWorkflow(wfName: string, label: Label) {
    this.loadingLabel = true;
    this.store.dispatch(new LinkLabelOnWorkflow({
      projectKey: this.project.key,
      workflowName: wfName,
      label
    })).pipe(finalize(() => this.loadingLabel = false))
      .subscribe();
  }

  unlinkLabelToWorkflow(wfName: string, label: Label) {
    this.loadingLabel = true;
    this.store.dispatch(new UnlinkLabelOnWorkflow({
      projectKey: this.project.key,
      workflowName: wfName,
      label
    })).pipe(finalize(() => this.loadingLabel = false))
      .subscribe();
  }

  confirmLabel(wfName: string, labelName: string) {
    let label = new Label();
    label.name = labelName;

    this.loadingLabel = true;
    this.store.dispatch(new LinkLabelOnWorkflow({
      projectKey: this.project.key,
      workflowName: wfName,
      label
    })).pipe(finalize(() => this.loadingLabel = false))
      .subscribe();
  }

  editLabels() {
    this.edit.emit(null);
  }
}
