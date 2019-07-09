import {
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    EventEmitter,
    Input,
    OnInit,
    Output,
    ViewChild
} from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { ModalTemplate, SuiActiveModal, SuiModalService, TemplateModalConfig } from '@richardlt/ng2-semantic-ui';
import { AllKeys } from 'app/model/keys.model';
import { Project } from 'app/model/project.model';
import { VCSConnections, VCSStrategy } from 'app/model/vcs.model';
import { KeyService } from 'app/service/keys/keys.service';
import { KeyEvent } from 'app/shared/keys/key.event';
import { ToastService } from 'app/shared/toast/ToastService';
import { AddKeyInProject } from 'app/store/project.action';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-vcs-strategy',
    templateUrl: './vcs.strategy.html',
    styleUrls: ['./vcs.strategy.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class VCSStrategyComponent implements OnInit {
    @Input() project: Project;
    @Input() appName: string;
    @Input() loading: boolean;
    @Input() hideBranch = false;
    @Input() hideButton = false;
    @Input() createOnProject = false;
    @Input() sshWarning = false;
    @Input() projectKeysOnly = false;

    selectedPublicKey: string;

    _strategy: VCSStrategy;
    @Input('strategy')
    set strategy(data: VCSStrategy) {
        if (data) {
            this._strategy = data;
        }
    }

    get strategy() {
        return this._strategy;
    }

    @Output() strategyChange = new EventEmitter<VCSStrategy>();
    keys: AllKeys;
    connectionType = VCSConnections;
    displayVCSStrategy = false;
    defaultKeyType = 'ssh';

    @ViewChild('createKey', {static: false})
    sshModalTemplate: ModalTemplate<boolean, boolean, void>;
    sshModal: SuiActiveModal<boolean, boolean, void>;

    constructor(
        private store: Store,
        private _keyService: KeyService,
        private _modalService: SuiModalService,
        private _toast: ToastService,
        private _translate: TranslateService,
        private _cd: ChangeDetectorRef
    ) { }

    ngOnInit() {
        if (!this.strategy) {
            this.strategy = new VCSStrategy();
        }
        this.loadKeys();
    }

    loadKeys() {
        if (this.projectKeysOnly) {
            this._keyService.getProjectKeys(this.project.key)
                .pipe(finalize(() => this._cd.markForCheck()))
                .subscribe(k => {
                this.keys = k;
            });
        } else {
            this._keyService.getAllKeys(this.project.key, this.appName)
                .pipe(finalize(() => this._cd.markForCheck()))
                .subscribe(k => {
                this.keys = k;
            });
        }
    }

    saveStrategy() {
        this.strategyChange.emit(this.strategy);
    }

    openCreateKeyModal(k): void {
        this.defaultKeyType = k;
        if (this.sshModalTemplate) {
            const config = new TemplateModalConfig<boolean, boolean, void>(this.sshModalTemplate);
            this.sshModal = this._modalService.open(config);
        }
    }

    addKey(event: KeyEvent): void {
        this.loading = true;
        this.store.dispatch(new AddKeyInProject({
            projectKey: this.project.key,
            key: event.key
        })).pipe(finalize(() => {
            this.loading = false;
            this.sshModal.approve(true);
            this.loadKeys();
        })).subscribe(() => this._toast.success('', this._translate.instant('keys_added')));
    }

    updatePublicKey(keyName: string): void {
        if (this.keys) {
            let key = this.keys.ssh.find(k => k.name === keyName);
            if (key) {
                this.selectedPublicKey = key.public;
            }
        }
    }

    clickCopyKey() {
        this._toast.success('', this._translate.instant('key_copied'))
    }
}
