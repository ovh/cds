import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    Input, OnInit,
} from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { AuthConsumer, AuthConsumerScopeDetail, AuthConsumerValidityPeriod } from 'app/model/authentication.model';
import { Group } from 'app/model/group.model';
import { AuthentifiedUser } from 'app/model/user.model';
import { AuthenticationService } from 'app/service/authentication/authentication.service';
import { GroupService } from 'app/service/group/group.service';
import { UserService } from 'app/service/user/user.service';
import { Column, Select } from 'app/shared/table/data-table.component';
import { ToastService } from 'app/shared/toast/ToastService';
import { finalize } from 'rxjs/operators';
import { NzModalRef } from 'ng-zorro-antd/modal';

export enum CloseEventType {
    CREATED = 'CREATED',
    CLOSED = 'CLOSED'
}

export enum FormStepName {
    INFORMATIONS = 0,
    GROUPS = 1,
    SCOPES = 2,
    SERVICE = 3,
    TOKEN = 4
}

@Component({
    selector: 'app-consumer-create-modal',
    templateUrl: './consumer-create-modal.html',
    styleUrls: ['./consumer-create-modal.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ConsumerCreateModalComponent implements OnInit {

    @Input() user: AuthentifiedUser;

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

    panels = [true, false, false, false, false];

    constructor(
        private _modal: NzModalRef,
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

    closeCallback(): void {
        if (this.newConsumer.id) {
            this._modal.triggerOk();
        } else {
            this._modal.triggerCancel();
        }
    }

    ngOnInit(): void {
        this.newConsumer = new AuthConsumer();
        this.newConsumer.validity_periods = new Array<AuthConsumerValidityPeriod>();
        // Init a default validity period of 15 days
        var ttl =  new AuthConsumerValidityPeriod();
        ttl.issued_at = new Date().toISOString();
        ttl.duration = 15;
        this.newConsumer.validity_periods.push(ttl);
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
        // transform validity_period.duration from days to nanosecond
        this.newConsumer.validity_periods[0].duration = this.newConsumer.validity_periods[0].duration * 8.64e13
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
        return (g: Group) => g.name.toLowerCase().indexOf(lowerFilter) !== -1;
    }

    clickBack() {
        switch (this.activeStep) {
            case FormStepName.GROUPS:
                this.panels[this.activeStep] = false;
                this.panels[FormStepName.INFORMATIONS] = true;
                this.activeStep = FormStepName.INFORMATIONS;
                break;
            case FormStepName.SCOPES:
                this.panels[this.activeStep] = false;
                this.panels[FormStepName.GROUPS] = true;
                this.activeStep = FormStepName.GROUPS;
                break;
            case FormStepName.SERVICE:
                this.panels[this.activeStep] = false;
                this.panels[FormStepName.SCOPES] = true;
                this.activeStep = FormStepName.SCOPES;
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
                this.panels[this.activeStep] = false;
                this.panels[FormStepName.GROUPS] = true;
                this.activeStep = FormStepName.GROUPS;
                break;
            case FormStepName.GROUPS:
                this.panels[this.activeStep] = false;
                this.panels[FormStepName.SCOPES] = true;
                this.activeStep = FormStepName.SCOPES;
                break;
            case FormStepName.SCOPES:
                this.panels[this.activeStep] = false;
                this.panels[FormStepName.SERVICE] = true;
                    this.activeStep = FormStepName.SERVICE;
                    break;
            case FormStepName.SERVICE:
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
        if (step > this.activeStep) {
            if (step > 0) {
                for (let i = 0; i < step ; i++) {
                    if (!this.isValidStep(i)) {
                        this.panels[this.activeStep] = true;
                        this.panels[step] = false;
                        this._cd.markForCheck();
                        return;
                    }
                }
            }
        }
        if (step === this.activeStep) {
            return;
        }
        this.activeStep = step;
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
            case FormStepName.SERVICE:
                return true;
            default:
                return false;
        }
    }

    onScopeDetailChange(detail: AuthConsumerScopeDetail) {
        for (let i = 0; i < this.selectedScopeDetails.length; i++) {
            if (this.selectedScopeDetails[i].scope === detail.scope) {
                this.selectedScopeDetails[i] = detail;
                return;
            }
        }
        this.selectedScopeDetails.push(detail);
    }
}
