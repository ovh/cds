import { AfterViewInit, ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, ViewChild } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Select, Store } from '@ngxs/store';
import {
    ModalSize,
    ModalTemplate,
    SuiActiveModal,
    SuiModalService,
    TemplateModalConfig
} from '@richardlt/ng2-semantic-ui';
import { GroupPermission } from 'app/model/group.model';
import { Project } from 'app/model/project.model';
import { WNode, Workflow } from 'app/model/workflow.model';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { PermissionEvent } from 'app/shared/permission/permission.event.model';
import { ToastService } from 'app/shared/toast/ToastService';
import { ProjectState, ProjectStateModel } from 'app/store/project.state';
import { CloseEditModal, UpdateWorkflow } from 'app/store/workflow.action';
import { WorkflowState, WorkflowStateModel } from 'app/store/workflow.state';
import cloneDeep from 'lodash-es/cloneDeep';
import { Observable, Subscription } from 'rxjs';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-node-edit-modal',
    templateUrl: './node.edit.modal.html',
    styleUrls: ['./node.edit.modal.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class WorkflowNodeEditModalComponent implements AfterViewInit, OnDestroy {

    @Select(WorkflowState.getEditModal()) editModal$: Observable<boolean>;
    editModalSub: Subscription;

    @Select(WorkflowState.getSelectedNode()) node$: Observable<WNode>;
    node: WNode;
    nodeSub: Subscription;

    project: Project;
    workflow: Workflow;
    groups: Array<GroupPermission>;

    currentNodeName = '';
    currentNodeType: string;
    hookSelected: boolean;




    @ViewChild('nodeEditModal')
    public nodeEditModal: ModalTemplate<boolean, boolean, void>;
    modal: SuiActiveModal<boolean, boolean, void>;
    modalConfig: TemplateModalConfig<boolean, boolean, void>;

    selected: string;
    hasModification = false;

    loading = false;

    projectSubscriber: Subscription;
    storeSub: Subscription;
    readonly = true;

    constructor(private _modalService: SuiModalService, private _store: Store, private _cd: ChangeDetectorRef,
        private _translate: TranslateService, private _toast: ToastService) {
        this.projectSubscriber = this._store.select(ProjectState)
            .subscribe((projState: ProjectStateModel) => {
                this.project = projState.project;
                this._cd.markForCheck();
            });
    }

    ngOnDestroy(): void {} // Should be set to use @AutoUnsubscribe with AOT

    ngAfterViewInit(): void {
        this.nodeSub = this.node$.subscribe(n => {
            if (!n) {
                return
            }
            let stateSnap: WorkflowStateModel = this._store.selectSnapshot(WorkflowState);
            if (stateSnap.editMode) {
                this.workflow = stateSnap.editWorkflow;
            } else {
                this.workflow = stateSnap.workflow;
            }
            this.groups = cloneDeep(stateSnap.node.groups);
            this._cd.markForCheck();
        });
        this.editModalSub = this.editModal$.subscribe(b => {
            let stateSnap: WorkflowStateModel = this._store.selectSnapshot(WorkflowState);
            if (!b) {
                this.currentNodeName = '';
                delete this.currentNodeType;
                delete this.hookSelected;
                this.readonly = true;
                delete this.selected;
                if (this.modal) {
                    this.modal.approve(true);
                }
                this._cd.markForCheck();
                return;
            }
            if (stateSnap.node) {
                if (stateSnap.editMode) {
                    this.workflow = stateSnap.editWorkflow;
                } else {
                    this.workflow = stateSnap.workflow;
                }
                let open = this.currentNodeName !== '';

                this.currentNodeName = stateSnap.node.name;
                this.currentNodeType = stateSnap.node.type;
                this.groups = cloneDeep(stateSnap.node.groups);
                this.readonly = !stateSnap.canEdit || (!!this.workflow.from_template && !!this.workflow.from_repository);
                if (stateSnap.hook) {
                    this.hookSelected = true;
                }
                if (!this.selected) {
                    if (this.hookSelected) {
                        this.selected = 'hook';
                    } else {
                        switch (this.currentNodeType) {
                            case 'outgoinghook':
                                this.selected = 'outgoinghook';
                                break;
                            case 'join':
                            case 'fork':
                                this.selected = 'conditions';
                                break;
                            default:
                                this.selected = 'context';
                        }
                    }
                }
                if (!open) {
                    this.show();
                }
            }
            this._cd.markForCheck();
        });
    }

    show(): void {
        if (this.nodeEditModal) {
            this.modalConfig = new TemplateModalConfig<boolean, boolean, void>(this.nodeEditModal);
            this.modalConfig.mustScroll = true;
            this.modalConfig.size = ModalSize.Large;
            this.modalConfig.isClosable = true;
            this.modal = this._modalService.open(this.modalConfig);
            this.modal.onApprove(() => {
                this._store.dispatch(new CloseEditModal({}));
            });
            this.modal.onDeny(() => {
                this._store.dispatch(new CloseEditModal({}));
            });
        } else {
            this.modal = this._modalService.open(this.modalConfig);
            this.modal.onApprove(() => {
                this._store.dispatch(new CloseEditModal({}));
            });
            this.modal.onDeny(() => {
                this._store.dispatch(new CloseEditModal({}));
            });
        }
    }

    groupManagement(event: PermissionEvent, skip?: boolean): void {
        let snapNode = cloneDeep(this._store.selectSnapshot(WorkflowState).node);
        this.loading = true;
        switch (event.type) {
            case 'add':
                if (!snapNode.groups) {
                    snapNode.groups = [];
                }
                snapNode.groups.push(event.gp);
                break;
            case 'update':
                snapNode.groups = snapNode.groups.map((group) => {
                    if (group.group.name === event.gp.group.name) {
                        group = event.gp;
                    }
                    return group;
                });
                break;
            case 'delete':
                snapNode.groups = snapNode.groups.filter((group) => group.group.name !== event.gp.group.name);
                break;
        }
        let workflow = cloneDeep(this.workflow);
        let node = Workflow.getNodeByID(snapNode.id, workflow);
        node.groups = snapNode.groups;
        this._store.dispatch(new UpdateWorkflow({
            projectKey: this.workflow.project_key,
            workflowName: this.workflow.name,
            changes: workflow
        })).pipe(finalize(() => {
            this.loading = false;
            event.gp.updating = false;
            this._cd.markForCheck();
        })).subscribe(() => {
            this.hasModification = false;
            this._toast.success('', this._translate.instant('permission_updated'));
            this._cd.markForCheck();
        });
    }

    pushChange(b: boolean): void {
        this.hasModification = b;
    }

    changeView(newView: string): void {
        if (this.selected === newView) {
            return;
        }
        if (this.hasModification) {
            if (confirm(this._translate.instant('workflow_modal_change_view_confirm'))) {
                this.hasModification = false;
                this.selected = newView;
            }
            return;
        }
        if (newView === 'permissions') {
            let snapNode = cloneDeep(this._store.selectSnapshot(WorkflowState).node);
            this.groups = cloneDeep(snapNode.groups);
        }
        this.hasModification = false;
        this.selected = newView;
    }
}
