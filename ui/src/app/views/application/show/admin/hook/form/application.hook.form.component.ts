import {Component, OnInit, ViewChild} from '@angular/core';
import {Input} from '@angular/core/src/metadata/directives';
import {Project} from '../../../../../../model/project.model';
import {Application} from '../../../../../../model/application.model';
import {ApplicationStore} from '../../../../../../service/application/application.store';
import {ToastService} from '../../../../../../shared/toast/ToastService';
import {TranslateService} from 'ng2-translate';
import {Hook} from '../../../../../../model/hook.model';
import {WarningModalComponent} from '../../../../../../shared/modal/warning/warning.component';

@Component({
    selector: 'app-application-hook-form',
    templateUrl: './application.hook.form.html',
    styleUrls: ['./application.hook.form.scss']
})
export class ApplicationHookFormComponent implements OnInit {

    @Input() project: Project;
    @Input() application: Application;
    @ViewChild('addHookWarning') warningModal: WarningModalComponent;

    public loading = false;
    newHook: Hook = new Hook();

    constructor(private _appStore: ApplicationStore, private _toast: ToastService, public _translate: TranslateService) { }

    ngOnInit() {
        this.newHook.pipeline = this.project.pipelines[0];
    }

    addHook(skip?: boolean): void {
        if (!skip && this.application.externalChange) {
            this.warningModal.show();
        } else {
            this.loading = true;
            this._appStore.addHook(this.project, this.application, this.newHook).subscribe(() => {
                this._toast.success('', this._translate.instant('application_poller_add_ok'));
                this.loading = false;
            }, () => {
                this.loading = false;
            });
        }
    }
}
