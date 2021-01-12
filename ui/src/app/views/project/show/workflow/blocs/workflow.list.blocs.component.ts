import { ChangeDetectionStrategy, ChangeDetectorRef, Component, EventEmitter, Input, Output } from '@angular/core';
import { Store } from '@ngxs/store';
import { IdName, Label, Project } from 'app/model/project.model';
import { HelpersService } from 'app/service/helpers/helpers.service';
import { AddLabelWorkflowInProject, DeleteLabelWorkflowInProject } from 'app/store/project.action';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-project-workflows-blocs',
    templateUrl: './workflow.list.blocs.html',
    styleUrls: ['./workflow.list.blocs.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ProjectWorkflowListBlocsComponent {

  _project: Project;
  @Input() set project(data: Project) {
      this._project = data;
      if (data && data.labels) {
          let labelFilter = this.labelFilter.toLowerCase();
          this.filteredLabels = data.labels.filter((lbl) => lbl.name.toLowerCase().indexOf(labelFilter) !== -1);
      }
  }
  get project() {
      return this._project;
  }

  @Input()
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

  @Output() edit = new EventEmitter<null>();

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

  _workflows: IdName[];
  _filterLabel = '';

  workflowLabelsMap: {} = {};
  filteredLabels: Array<Label> = [];
  loadingLabel = false;

  constructor(
    private store: Store, private _cd: ChangeDetectorRef,
    private _helpersService: HelpersService
  ) { }

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
      .subscribe();
  }

  editLabels() {
    this.edit.emit(null);
  }
}
