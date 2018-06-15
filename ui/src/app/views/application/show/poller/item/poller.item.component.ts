import {Component, Input, ViewChild} from '@angular/core';
import {TranslateService} from '@ngx-translate/core';
import {cloneDeep} from 'lodash';
import {SemanticModalComponent} from 'ng-semantic/ng-semantic';
import {Application} from '../../../../../model/application.model';
import {Pipeline} from '../../../../../model/pipeline.model';
import {RepositoryPoller} from '../../../../../model/polling.model';
import {Project} from '../../../../../model/project.model';
import {ApplicationStore} from '../../../../../service/application/application.store';
import {ToastService} from '../../../../../shared/toast/ToastService';

@Component({
    selector: 'app-application-poller-item',
    templateUrl: './poller.item.html',
    styleUrls: ['./poller.item.scss']
})
export class ApplicationPollerItemComponent {

    @Input() project: Project;
    @Input() application: Application;
    @Input() pipeline: Pipeline;
    @Input() poller: RepositoryPoller;

    editablePoller: RepositoryPoller;

    // Schedulers modals
    @ViewChild('editPollerModal')
    editPollerModal: SemanticModalComponent;

    constructor(private _appStore: ApplicationStore, private _toast: ToastService, private _translate: TranslateService) { }

    editPoller(): void {
        if (this.editPollerModal) {
            this.editablePoller = cloneDeep(this.poller);
            setTimeout(() => {
                this.editPollerModal.show();
            }, 100);
        }
    }

    close(): void {
        if (this.editPollerModal) {
            this.editPollerModal.hide();
        }
    }

    updatePoller(): void {
        this.editablePoller.updating = true;
        this._appStore.updatePoller(this.project.key, this.application.name, this.pipeline.name, this.editablePoller).subscribe(() => {
            this._toast.success('', this._translate.instant('poller_updated'));
            this.close();
        }, () => {
            this.editablePoller.updating = false;
        });
    }

    deletePoller(): void {
        this.editablePoller.updating = true;
        this._appStore.deletePoller(this.project.key, this.application.name, this.editablePoller).subscribe(() => {
            this._toast.success('', this._translate.instant('poller_deleted'));
            this.close();
        }, () => {
            this.editablePoller.updating = false;
        });
    }
}
