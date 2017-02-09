import {Component, OnInit, ViewChild} from '@angular/core';
import {Input} from '@angular/core/src/metadata/directives';
import {Project} from '../../../../../../model/project.model';
import {Application} from '../../../../../../model/application.model';
import {RepositoryPoller} from '../../../../../../model/polling.model';
import {ApplicationStore} from '../../../../../../service/application/application.store';
import {ToastService} from '../../../../../../shared/toast/ToastService';
import {TranslateService} from 'ng2-translate';
import {WarningModalComponent} from '../../../../../../shared/modal/warning/warning.component';

@Component({
    selector: 'app-application-poller-form',
    templateUrl: './application.poller.form.html',
    styleUrls: ['./application.poller.form.scss']
})
export class ApplicationPollerFormComponent implements OnInit {

    @Input() project: Project;
    @Input() application: Application;
    @ViewChild('formWarning') warningModal: WarningModalComponent;

    public loading = false;
    newPoller: RepositoryPoller = new RepositoryPoller();

    constructor(private _appStore: ApplicationStore, private _toast: ToastService, public _translate: TranslateService) { }

    ngOnInit() {
        this.newPoller.pipeline = this.project.pipelines[0];
        this.newPoller.name = this.application.repositories_manager.name;
    }

    addPoller(skip?: boolean): void {
        if (!skip && this.application.externalChange) {
            this.warningModal.show();
        } else {
            this.loading = true;
            this._appStore.addPoller(this.project.key, this.application.name, this.newPoller.pipeline.name, this.newPoller)
                .subscribe(() => {
                    this._toast.success('', this._translate.instant('application_poller_add_ok'));
                    this.loading = false;
                }, () => {
                this.loading = false;
            });
        }
    }
}
