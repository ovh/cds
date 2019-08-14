import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    EventEmitter,
    Input,
    Output,
    ViewChild
} from '@angular/core';
import { ModalTemplate, SuiActiveModal, SuiModalService, TemplateModalConfig } from '@richardlt/ng2-semantic-ui';
import { AuthConsumer } from 'app/model/authentication.model';
import { Group } from 'app/model/group.model';
import { AuthentifiedUser } from 'app/model/user.model';
import { UserService } from 'app/service/user/user.service';
import { Column, Select } from 'app/shared/table/data-table.component';
import { finalize } from 'rxjs/operators/finalize';

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
    @Output() close = new EventEmitter();

    newConsumer: AuthConsumer = new AuthConsumer();
    loadingGroups: boolean;
    groups: Array<Group>;
    columnsGroups: Array<Column<Group>>;
    selectedGroupKeys: Array<string>;

    constructor(
        private _modalService: SuiModalService,
        private _userService: UserService,
        private _cd: ChangeDetectorRef,
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
        this.modal.onApprove(() => {
            this.open = false;
            this.close.emit();
        });
        this.modal.onDeny(() => {
            this.open = false;
            this.close.emit();
        });

        this.init();
    }

    clickClose() {
        this.modal.approve(true);
    }

    init(): void {
        this.newConsumer = new AuthConsumer();

        this.loadingGroups = true;
        this._cd.markForCheck();
        this._userService.getGroups(this.user.username)
            .pipe(finalize(() => {
                this.loadingGroups = false;
                this._cd.markForCheck();
            }))
            .subscribe((gs) => {
                this.groups = gs;
            });
    }

    selectGroupFunc: Select<Group> = (g: Group): boolean => {
        if (!this.selectedGroupKeys || this.selectedGroupKeys.length === 0) {
            return false;
        }
        return !!this.selectedGroupKeys.find(id => id === g.key());
    }

    selectGroupChange(e: Array<string>) {
        this.selectedGroupKeys = e;
    }
}
