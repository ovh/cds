import { Component, EventEmitter, Input, Output, ViewChild } from '@angular/core';
import { Store } from '@ngxs/store';
import { LinkLabelOnWorkflow, UnlinkLabelOnWorkflow } from 'app/store/workflow.action';
import { finalize } from 'rxjs/operators';
import { IdName, Label, Project } from '../../../../../model/project.model';
import { Warning } from '../../../../../model/warning.model';
import { HelpersService } from '../../../../../service/helpers/helpers.service';
import { LabelsEditComponent } from '../../../../../shared/labels/edit/labels.edit.component';

@Component({
  selector: 'app-project-workflows-blocs',
  templateUrl: './workflow.list.blocs.html',
  styleUrls: ['./workflow.list.blocs.scss']
})
export class ProjectWorkflowListBlocsComponent {

  @Input() project: Project;
  @Input() warnMap: Map<string, Array<Warning>>;
  @Input('workflows')
  set workflows(workflows: IdName[]) {
    this._workflows = workflows;
    if (workflows) {
      workflows.forEach((wf) => {
        this.workflowLabelsMap[wf.name] = {};
        if (wf.labels) {
          this.workflowLabelsMap[wf.name] = wf.labels.reduce((obj, lbl) => {
            lbl.font_color = this._helpersService.getBrightnessColor(lbl.color);
            obj[lbl.name] = true;
            return obj;
          }, {});
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

  _labels: Label[];
  _workflows: IdName[];
  _filterLabel = '';

  workflowLabelsMap: {} = {};
  filteredLabels: Array<Label> = [];
  loadingLabel = false;

  constructor(
    private store: Store,
    private _helpersService: HelpersService
  ) { }

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
