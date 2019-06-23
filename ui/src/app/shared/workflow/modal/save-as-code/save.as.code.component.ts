import { Component, Input, NgZone, ViewChild } from '@angular/core';
import { Store } from '@ngxs/store';
import { AuthenticationState } from 'app/store/authentication.state';
import { ModalTemplate, SuiModalService, TemplateModalConfig } from 'ng2-semantic-ui';
import { ActiveModal } from 'ng2-semantic-ui/dist';
import { Subscription } from 'rxjs';
import { Operation } from '../../../../model/operation.model';
import { Project } from '../../../../model/project.model';
import { Workflow } from '../../../../model/workflow.model';
import { AutoUnsubscribe } from '../../../decorator/autoUnsubscribe';
import { CDSWebWorker } from '../../../worker/web.worker';

@Component({
    selector: 'app-workflow-save-as-code-modal',
    templateUrl: './save.as.code.html',
    styleUrls: ['./save.as.code.scss']
})
@AutoUnsubscribe()
export class WorkflowSaveAsCodeComponent {

    @Input() project: Project;
    @Input() workflow: Workflow;
    ope: Operation;
    webworkerSub: Subscription;

    @ViewChild('saveAsCodeModal', { static: false })
    public saveAsCodeModal: ModalTemplate<boolean, boolean, void>;
    modalConfig: TemplateModalConfig<boolean, boolean, void>;
    modal: ActiveModal<boolean, boolean, void>;

    constructor(
        private _modalService: SuiModalService,
        private _store: Store
    ) { }

    show(ope: Operation): void {
        if (this.saveAsCodeModal) {
            this.ope = ope;
            this.modalConfig = new TemplateModalConfig<boolean, boolean, void>(this.saveAsCodeModal);
            this.modalConfig.mustScroll = true;
            this.modal = this._modalService.open(this.modalConfig);
            this.startOperationPull();
        }
    }

    startOperationPull() {
        // poll operation
        let zone = new NgZone({ enableLongStackTrace: false });
        let webworker = new CDSWebWorker('./assets/worker/web/operation.js');
        webworker.start({
            'user': this._store.selectSnapshot(AuthenticationState.user),
            // 'session': this._authStore.getSessionToken(),
            'api': '/cdsapi',
            'path': '/project/' + this.project.key + '/workflows/' + this.workflow.name + '/ascode/' + this.ope.uuid
        });
        this.webworkerSub = webworker.response().subscribe(operation => {
            if (operation) {
                zone.run(() => {
                    this.ope = JSON.parse(operation);
                    if (this.ope.status > 1) {
                        webworker.stop();
                        this.webworkerSub.unsubscribe();
                    }
                });
            }
        });
    }

}
