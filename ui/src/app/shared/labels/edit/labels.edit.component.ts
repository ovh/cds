import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input } from '@angular/core';
import { Store } from '@ngxs/store';
import { Label, Project } from 'app/model/project.model';
import { SaveLabelsInProject } from 'app/store/project.action';
import cloneDeep from 'lodash-es/cloneDeep';
import { finalize } from 'rxjs/operators';
import { NzModalRef } from 'ng-zorro-antd/modal';

@Component({
    selector: 'app-labels-edit',
    templateUrl: './labels.edit.component.html',
    styleUrls: ['./labels.edit.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class LabelsEditComponent {
    _project: Project;
    @Input() set project(data: Project) {
        this._project = data;
        if (this._project) {
            this.labels = cloneDeep(this.project.labels);
        }
    }
    get project() {
        return this._project;
    }

    labels: Label[];
    newLabel: Label = new Label();
    loading = false;

    constructor(
        public _modal: NzModalRef,
        private store: Store,
        private _cd: ChangeDetectorRef

    ) { }

    deleteLabel(label: Label) {
        this.labels = this.labels.filter((lbl) => lbl.name !== label.name);
    }

    createLabel() {
        if (!this.labels) {
            this.labels = [];
        }
        this.labels.push(this.newLabel);
        this.saveLabels();
    }

    saveLabels(close?: boolean) {
        this.loading = true;
        this.store.dispatch(new SaveLabelsInProject({
            projectKey: this.project.key,
            labels: this.labels
        })).pipe(finalize(() => {
            this.loading = false;
            this.newLabel = new Label();
            this._cd.markForCheck();
        })).subscribe(() => {
            if (close) {
                this._modal.destroy();
            }
        });
    }
}
