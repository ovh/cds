import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    EventEmitter,
    Input,
    Output,
    ViewChild
} from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { ModalTemplate, SuiActiveModal, SuiModalService, TemplateModalConfig } from '@richardlt/ng2-semantic-ui';
import { AuthConsumer, AuthScope } from 'app/model/authentication.model';
import { Group } from 'app/model/group.model';
import { AuthentifiedUser } from 'app/model/user.model';
import { AuthenticationService } from 'app/service/authentication/authentication.service';
import { GroupService } from 'app/service/group/group.service';
import { UserService } from 'app/service/user/user.service';
import { Column, Select } from 'app/shared/table/data-table.component';
import { ToastService } from 'app/shared/toast/ToastService';
import { finalize } from 'rxjs/operators/finalize';

export enum CloseEventType {
    CREATED = 'CREATED',
    CLOSED = 'CLOSED'
}

@Component({
    selector: 'app-consumer-create-modal',
    templateUrl: './consumer-create-modal.html',
    styleUrls: ['./consumer-create-modal.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ConsumerCreateModalComponent {
    @ViewChild('consumerCreateModal', { static: false }) consumerDetailsModal: ModalTemplate<boolean, boolean, void>;
    modal: SuiActiveModal<boolean, boolean, void>;
    open: boolean;

    @Input() user: AuthentifiedUser;
    @Output() close = new EventEmitter<CloseEventType>();

    newConsumer: AuthConsumer = new AuthConsumer();
    signinToken: string;
    loading: boolean;
    loadingGroups: boolean;
    groups: Array<Group>;
    columnsGroups: Array<Column<Group>>;
    selectedGroupKeys: Array<string>;
    loadingScopes: boolean;
    scopes: Array<AuthScope>;
    columnsScopes: Array<Column<AuthScope>>;
    selectedScopeKeys: Array<string>;

    constructor(
        private _modalService: SuiModalService,
        private _userService: UserService,
        private _groupService: GroupService,
        private _authenticationService: AuthenticationService,
        private _cd: ChangeDetectorRef,
        private _toast: ToastService,
        private _translate: TranslateService
    ) {
        this.columnsGroups = [
            <Column<Group>>{
                name: 'common_name',
                class: 'fourteen',
                selector: (g: Group) => g.name
            }
        ];

        this.columnsScopes = [
            <Column<AuthScope>>{
                name: 'common_name',
                class: 'fourteen',
                selector: (s: AuthScope) => s.value
            }
        ];
    }

    show() {
        if (this.open) {
            return;
        }

        this.open = true;

        const config = new TemplateModalConfig<boolean, boolean, void>(this.consumerDetailsModal);
        config.mustScroll = true;
        this.modal = this._modalService.open(config);
        this.modal.onApprove(_ => { this.closeCallback() });
        this.modal.onDeny(_ => { this.closeCallback() });

        this.init();
    }

    closeCallback(): void {
        this.open = false;
        if (this.newConsumer.id) {
            this.close.emit(CloseEventType.CREATED);
        } else {
            this.close.emit(CloseEventType.CLOSED);
        }
    }

    init(): void {
        this.newConsumer = new AuthConsumer();
        this.signinToken = null;
        this.selectedGroupKeys = null;
        this.selectedScopeKeys = null;

        this.loadingGroups = true;
        this.loadingScopes = true;
        this._cd.markForCheck();

        this._groupService.getAll()
            .pipe(finalize(() => {
                this.loadingGroups = false;
                this._cd.markForCheck();
            }))
            .subscribe((gs) => {
                this.groups = gs.sort((a, b) => a.name < b.name ? -1 : 1);
            });

        this._authenticationService.getScopes()
            .pipe(finalize(() => {
                this.loadingScopes = false;
                this._cd.markForCheck();
            }))
            .subscribe((ss) => {
                this.scopes = ss.sort((a, b) => a.value < b.value ? -1 : 1);
            });
    }

    selectGroupFunc: Select<Group> = (g: Group): boolean => {
        if (!this.selectedGroupKeys || this.selectedGroupKeys.length === 0) {
            return false;
        }
        return !!this.selectedGroupKeys.find(k => k === g.key());
    }

    selectGroupChange(e: Array<string>) {
        this.selectedGroupKeys = e;
    }

    selectScopeFunc: Select<AuthScope> = (s: AuthScope): boolean => {
        if (!this.selectedScopeKeys || this.selectedScopeKeys.length === 0) {
            return false;
        }
        return !!this.selectedScopeKeys.find(k => k === s.key());
    }

    selectScopeChange(e: Array<string>) {
        this.selectedScopeKeys = e;
    }

    clickSave(): void {
        this.newConsumer.group_ids = this.groups.filter(g => this.selectedGroupKeys.find(k => k === g.key())).map(g => g.id);
        this.newConsumer.scopes = this.selectedScopeKeys;

        this.loading = true;
        this._cd.markForCheck();
        this._userService.createConsumer(this.user.username, this.newConsumer)
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(res => {
                this.newConsumer = res.consumer;
                this.signinToken = res.token;
            });
    }

    confirmCopy() {
        this._toast.success('', this._translate.instant('auth_value_copied'));
    }
}
