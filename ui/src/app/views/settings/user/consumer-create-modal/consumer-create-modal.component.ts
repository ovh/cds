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
import { AuthConsumer, AuthConsumerScopeDetail } from 'app/model/authentication.model';
import { Group } from 'app/model/group.model';
import { AuthentifiedUser } from 'app/model/user.model';
import { AuthenticationService } from 'app/service/authentication/authentication.service';
import { GroupService } from 'app/service/group/group.service';
import { UserService } from 'app/service/user/user.service';
import { Column, Select } from 'app/shared/table/data-table.component';
import { ToastService } from 'app/shared/toast/ToastService';
import { finalize } from 'rxjs/operators';

export enum CloseEventType {
    CREATED = 'CREATED',
    CLOSED = 'CLOSED'
}

export enum FormStepName {
    INFORMATIONS = 0,
    GROUPS = 1,
    SCOPES = 2,
    TOKEN = 3
}

@Component({
    selector: 'app-consumer-create-modal',
    templateUrl: './consumer-create-modal.html',
    styleUrls: ['./consumer-create-modal.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ConsumerCreateModalComponent {
    @ViewChild('consumerCreateModal') consumerDetailsModal: ModalTemplate<boolean, boolean, void>;
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
    scopes: Array<AuthConsumerScopeDetail>;
    selectedScopeDetails: Array<AuthConsumerScopeDetail>;

    formStepName = FormStepName;
    activeStep: FormStepName;
    maxActivedStep: FormStepName;

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
    }

    show() {
        if (this.open) {
            return;
        }

        this.open = true;

        const config = new TemplateModalConfig<boolean, boolean, void>(this.consumerDetailsModal);
        config.mustScroll = true;
        this.modal = this._modalService.open(config);
        this.modal.onApprove(_ => {
 this.closeCallback()
});
        this.modal.onDeny(_ => {
 this.closeCallback()
});

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
        this.selectedScopeDetails = [];

        this.activeStep = FormStepName.INFORMATIONS;
        this.maxActivedStep = this.activeStep;

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
                this.scopes = ss.sort((a, b) => a.scope < b.scope ? -1 : 1);
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

    save(): void {
        this.newConsumer.group_ids = this.groups.filter(g => this.selectedGroupKeys.find(k => k === g.key())).map(g => g.id);
        this.newConsumer.scope_details = this.selectedScopeDetails;

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

                this.activeStep = FormStepName.TOKEN;
            });
    }

    filterGroups(f: string) {
        const lowerFilter = f.toLowerCase();
        return (g: Group) => g.name.toLowerCase().indexOf(lowerFilter) !== -1
    }

    clickBack() {
        switch (this.activeStep) {
            case FormStepName.GROUPS:
                this.activeStep = FormStepName.INFORMATIONS;
                break;
            case FormStepName.SCOPES:
                this.activeStep = FormStepName.GROUPS;
                break;
            default:
                return;
        }
        this._cd.markForCheck();
    }

    clickNext() {
        if (!this.isValidStep(this.activeStep)) {
            return;
        }

        switch (this.activeStep) {
            case FormStepName.INFORMATIONS:
                this.activeStep = FormStepName.GROUPS;
                break;
            case FormStepName.GROUPS:
                this.activeStep = FormStepName.SCOPES;
                break;
            case FormStepName.SCOPES:
                this.save();
                return;
            default:
                return;
        }
        if (this.maxActivedStep < this.activeStep) {
            this.maxActivedStep = this.activeStep;
        }

        this._cd.markForCheck();
    }

    clickOpenStep(step: FormStepName) {
        if (step > this.activeStep && !this.isValidStep(this.activeStep)) {
            return;
        }
        if (step === this.activeStep) {
            return;
        }
        if (step <= this.maxActivedStep) {
            this.activeStep = step;
        }
        this._cd.markForCheck();
    }

    isValidStep(step: FormStepName): boolean {
        switch (step) {
            case FormStepName.INFORMATIONS:
                return this.newConsumer.name && this.newConsumer.name !== '';
            case FormStepName.GROUPS:
                return true;
            case FormStepName.SCOPES:
                return this.selectedScopeDetails.length > 0;
            default:
                return false;
        }
    }

    onScopeDetailChange(detail: AuthConsumerScopeDetail) {
        for (let i = 0; i < this.selectedScopeDetails.length; i++) {
            if (this.selectedScopeDetails[i].scope === detail.scope) {
                this.selectedScopeDetails[i] = detail;
                return
            }
        }
        this.selectedScopeDetails.push(detail);
    }

    clickClose() {
        this.modal.approve(true);
    }
}
