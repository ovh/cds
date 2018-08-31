import {Component, Input, ViewChild} from '@angular/core';
import {cloneDeep} from 'lodash';
import {ModalTemplate, SuiModalService, TemplateModalConfig} from 'ng2-semantic-ui';
import {ActiveModal} from 'ng2-semantic-ui/dist';
import {finalize} from 'rxjs/operators';
import {PermissionValue} from '../../../model/permission.model';
import {Label, Project} from '../../../model/project.model';
import {ProjectStore} from '../../../service/project/project.store';

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
    permission = PermissionValue;
    loading = false;

    constructor(private _suiService: SuiModalService, private _projectStore: ProjectStore) {

    }

    show() {
        if (!this.project) {
            return;
        }
        this.labels = cloneDeep(this.project.labels);
        this.modalConfig = new TemplateModalConfig<boolean, boolean, void>(this.labelsEditModal);
        this.modalConfig.mustScroll = true;
        this.modal = this._suiService.open(this.modalConfig);
    }

    deleteLabel(label: Label) {
        this.labels = this.labels.filter((lbl) => lbl.name !== label.name);
    }

    saveLabels() {
        this.loading = true;
        this._projectStore.updateLabels(this.project.key, this.labels)
            .pipe(finalize(() => this.loading = false))
            .subscribe((proj) => {
                this.project.labels = proj.labels;
                this.modal.approve(true);
            });
    }
}
