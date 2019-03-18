import { Component, Input, ViewChild } from '@angular/core';
import { Store } from '@ngxs/store';
import { SaveLabelsInProject } from 'app/store/project.action';
import { cloneDeep } from 'lodash';
import { ModalTemplate, SuiModalService, TemplateModalConfig } from 'ng2-semantic-ui';
import { ActiveModal } from 'ng2-semantic-ui/dist';
import { finalize } from 'rxjs/operators';
import { PermissionValue } from '../../../model/permission.model';
import { Label, Project } from '../../../model/project.model';

@Component({
    selector: 'app-labels-edit',
    templateUrl: './labels.edit.component.html',
    styleUrls: ['./labels.edit.component.scss']
})
export class LabelsEditComponent {
    @Input() project: Project;

    @ViewChild('labelsEditModal')
    public labelsEditModal: ModalTemplate<boolean, boolean, void>;
    modal: ActiveModal<boolean, boolean, void>;
    modalConfig: TemplateModalConfig<boolean, boolean, void>;

    labels: Label[];
    newLabel: Label;
    permission = PermissionValue;
    loading = false;

    constructor(
        private store: Store,
        private _suiService: SuiModalService
    ) {

    }

    show() {
        if (!this.project) {
            return;
        }
        this.newLabel = new Label();
        this.labels = cloneDeep(this.project.labels);
        this.modalConfig = new TemplateModalConfig<boolean, boolean, void>(this.labelsEditModal);
        this.modalConfig.mustScroll = true;
        this.modal = this._suiService.open(this.modalConfig);
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
        })).subscribe(() => {
            if (close) {
                this.modal.approve(true);
            }
        });
    }
}
