import { ChangeDetectionStrategy, ChangeDetectorRef, Component, EventEmitter, Input, Output } from '@angular/core';
import { Store } from '@ngxs/store';
import { IdName, Label, Project } from 'app/model/project.model';
import { HelpersService } from 'app/service/helpers/helpers.service';
import { AddLabelWorkflowInProject, DeleteLabelWorkflowInProject } from 'app/store/project.action';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-project-workflows-lines',
    templateUrl: './workflow.list.lines.html',
    styleUrls: ['./workflow.list.lines.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ProjectWorkflowListLinesComponent {

  @Input() project: Project;
  @Input()
  set workflows(workflows: IdName[]) {
    this._workflows = workflows;
    if (workflows) {
      workflows.forEach((wf) => {
        this.workflowLabelsMap[wf.name] = {};
        if (wf.labels) {
          this.workflowLabelsMap[wf.name] = wf.labels.reduce((obj, lbl) => {
            if (!lbl.font_color) {
              lbl.font_color = this._helpersService.getBrightnessColor(lbl.color);
            }
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
  @Input()
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

  _labels: Label[];
  _workflows: IdName[];
  _filterLabel = '';

  workflowLabelsMap: {} = {};
  filteredLabels: Array<Label> = [];
  loadingLabel = false;

  constructor(private store: Store, private _helpersService: HelpersService, private _cd: ChangeDetectorRef) { }

  linkLabelToWorkflow(wfName: string, label: Label) {
    this.loadingLabel = true;
    this.store.dispatch(new AddLabelWorkflowInProject({
      workflowName: wfName,
      label
    })).pipe(finalize(() => {
        this.loadingLabel = false;
        this._cd.markForCheck();
    }))
      .subscribe();
  }

  unlinkLabelToWorkflow(wfName: string, label: Label) {
    this.loadingLabel = true;
    this.store.dispatch(new DeleteLabelWorkflowInProject({
      workflowName: wfName,
      labelId: label.id
    })).pipe(finalize(() => {
        this.loadingLabel = false;
        this._cd.markForCheck();
    }))
      .subscribe();
  }

  confirmLabel(wfName: string, labelName: string) {
    let label = new Label();
    label.name = labelName;

    this.loadingLabel = true;
    this.store.dispatch(new AddLabelWorkflowInProject({
      workflowName: wfName,
      label
    })).pipe(finalize(() => {
        this.loadingLabel = false;
        this._cd.markForCheck();
    }))
      .subscribe(() => this.labelFilter = '');
  }

  editLabels() {
    this.edit.emit(null);
  }
}
