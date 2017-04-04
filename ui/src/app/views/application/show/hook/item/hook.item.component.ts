import {Component, Input, ViewChild, EventEmitter, Output} from '@angular/core';
import {Hook} from '../../../../../model/hook.model';
import {Project} from '../../../../../model/project.model';
import {Application} from '../../../../../model/application.model';
import {Pipeline} from '../../../../../model/pipeline.model';
import {SemanticModalComponent} from 'ng-semantic/ng-semantic';
import {HookEvent} from '../hook.event';
import {ApplicationStore} from '../../../../../service/application/application.store';
import {ToastService} from '../../../../../shared/toast/ToastService';
import {TranslateService} from 'ng2-translate';

declare var _: any;

@Component({
    selector: 'app-application-hook-item',
    templateUrl: './hook.item.html',
    styleUrls: ['./hook.item.scss']
})
export class ApplicationHookItemComponent {

    @Input() project: Project;
    @Input() application: Application;
    @Input() pipeline: Pipeline;
    @Input() hook: Hook;

    @ViewChild('editHookModal')
    editHookModal: SemanticModalComponent;

    editableHook: Hook;

    constructor(private _appStore: ApplicationStore, private _toast: ToastService, private _translate: TranslateService) { }

    editHook(): void {
        if (this.editHookModal) {
            this.editableHook = _.cloneDeep(this.hook);
            setTimeout(() => {
                this.editHookModal.show();
            }, 100);
        }
    }

    close(): void {
        if (this.editHookModal) {
            this.editHookModal.hide();
        }
    }

    updateHook(): void {
        this.editableHook.updating = true;
        this._appStore.updateHook(this.project, this.application, this.pipeline.name, this.editableHook).subscribe(() => {
            this._toast.success('', this._translate.instant('hook_updated'));
            this.close();
        }, () => {
            this.editableHook.updating = false;
        });
    }

    deleteHook(): void {
        this.hook.updating = true;
        this._appStore.removeHook(this.project, this.application, this.editableHook).subscribe(() => {
            this._toast.success('', this._translate.instant('hook_deleted'));
            this.close();
        }, () => {
            this.editableHook.updating = false;
        });
    }
}
