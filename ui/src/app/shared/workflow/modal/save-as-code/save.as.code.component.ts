import {Component, Input, NgZone, ViewChild} from '@angular/core';
import { Operation } from 'app/model/operation.model';
import { Project } from 'app/model/project.model';
import { Workflow } from 'app/model/workflow.model';
import { AuthentificationStore } from 'app/service/auth/authentification.store';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { CDSWebWorker } from 'app/shared/worker/web.worker';
import { environment } from 'environments/environment';
import {ModalTemplate, SuiModalService, TemplateModalConfig} from 'ng2-semantic-ui';
import {ActiveModal} from 'ng2-semantic-ui/dist';
import {Subscription} from 'rxjs';

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

    @ViewChild('saveAsCodeModal', {static: false})
    public saveAsCodeModal: ModalTemplate<boolean, boolean, void>;
    modalConfig: TemplateModalConfig<boolean, boolean, void>;
    modal: ActiveModal<boolean, boolean, void>;

    constructor(private _modalService: SuiModalService, private _authStore: AuthentificationStore) {
    }

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
            'user': this._authStore.getUser(),
            'session': this._authStore.getSessionToken(),
            'api': environment.apiURL,
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
