import {ChangeDetectionStrategy, ChangeDetectorRef, Component, inject, Input, OnInit} from '@angular/core';
import { Store } from '@ngxs/store';
import { Label, Project } from 'app/model/project.model';
import { SaveLabelsInProject } from 'app/store/project.action';
import cloneDeep from 'lodash-es/cloneDeep';
import { finalize } from 'rxjs/operators';
import {NZ_MODAL_DATA, NzModalRef} from 'ng-zorro-antd/modal';

interface IModalData {
    project: Project;
}

@Component({
    selector: 'app-labels-edit',
    templateUrl: './labels.edit.component.html',
    styleUrls: ['./labels.edit.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class LabelsEditComponent implements OnInit {
    project: Project;

    labels: Label[];
    newLabel: Label = new Label();
    loading = false;

    readonly nzModalData: IModalData = inject(NZ_MODAL_DATA);

    constructor(
        public _modal: NzModalRef,
        private store: Store,
        private _cd: ChangeDetectorRef

    ) { }

    ngOnInit() {
        this.project = this.nzModalData.project;
        if (this.project) {
            this.labels = cloneDeep(this.project.labels);
        }
    }

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
