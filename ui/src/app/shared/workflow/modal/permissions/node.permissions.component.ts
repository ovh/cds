import { Component, Input, ViewChild } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { PermissionValue } from 'app/model/permission.model';
import { ToastService } from 'app/shared/toast/ToastService';
import { ModalTemplate, SuiModalService, TemplateModalConfig } from 'ng2-semantic-ui';
import { ActiveModal } from 'ng2-semantic-ui/dist';
import { finalize } from 'rxjs/operators';
import { WNode, Workflow } from '../../../../model/workflow.model';
import { WorkflowStore } from '../../../../service/workflow/workflow.store';
import { PermissionEvent } from '../../../permission/permission.event.model';

@Component({
    selector: 'app-workflow-node-permissions',
    templateUrl: './node.permissions.html',
    styleUrls: ['./node.permissions.scss']
})
export class WorkflowNodePermissionsComponent {

    @ViewChild('permissionsModal')
    permissionsModalTemplate: ModalTemplate<boolean, boolean, void>;
    modal: ActiveModal<boolean, boolean, void>;

    @Input() node: WNode;
    @Input() workflow: Workflow;

    loading = false;
    permissionEnum = PermissionValue;

    constructor(
        private _modalService: SuiModalService,
        private _workflowStore: WorkflowStore,
        private _translate: TranslateService, private _toast: ToastService
    ) { }

    show(): void {
        const config = new TemplateModalConfig<boolean, boolean, void>(this.permissionsModalTemplate);
        this.modal = this._modalService.open(config);
    }

    groupManagement(event: PermissionEvent, skip?: boolean): void {
        this.loading = true;
        switch (event.type) {
            case 'add':
                this.node.groups.push(event.gp);
                break;
            case 'update':
                this.node.groups = this.node.groups.map((group) => {
                    if (group.group.name === event.gp.group.name) {
                        group = event.gp;
                    }
                    return group;
                });
                break;
            case 'delete':
                this.node.groups = this.node.groups.filter((group) => group.group.name !== event.gp.group.name);
                break;
        }
        this._workflowStore.updateWorkflow(this.workflow.project_key, this.workflow)
            .pipe(finalize(() => this.loading = false))
            .subscribe(() => {
                event.gp.updating = false;
                this._toast.success('', this._translate.instant('permission_updated'));
            });
    }

}
