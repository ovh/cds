import { Component, EventEmitter, Input, OnInit, Output, ViewChild } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { ModalTemplate, SuiModalService, TemplateModalConfig } from 'ng2-semantic-ui';
import { ActiveModal } from 'ng2-semantic-ui/dist';
import { finalize, first } from 'rxjs/operators';
import { AllKeys } from '../../model/keys.model';
import { Project } from '../../model/project.model';
import { VCSConnections, VCSStrategy } from '../../model/vcs.model';
import { KeyService } from '../../service/keys/keys.service';
import { ProjectStore } from '../../service/project/project.store';
import { KeyEvent } from '../keys/key.event';
import { ToastService } from '../toast/ToastService';

@Component({
    selector: 'app-vcs-strategy',
    templateUrl: './vcs.strategy.html',
    styleUrls: ['./vcs.strategy.scss']
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

    @ViewChild('createKey')
    sshModalTemplate: ModalTemplate<boolean, boolean, void>;
    sshModal: ActiveModal<boolean, boolean, void>;

    constructor(
        private _keyService: KeyService,
        private _modalService: SuiModalService,
        private _toast: ToastService,
        private _translate: TranslateService,
        private _projectStore: ProjectStore
    ) { }

    ngOnInit() {
        if (!this.strategy) {
            this.strategy = new VCSStrategy();
        }
        this.loadKeys();
    }

    loadKeys() {
        if (this.projectKeysOnly) {
            this._keyService.getProjectKeys(this.project.key).subscribe(k => {
                this.keys = k;
            })
        } else {
            this._keyService.getAllKeys(this.project.key, this.appName).subscribe(k => {
                this.keys = k;
            })
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
        this._projectStore.addKey(this.project.key, event.key).pipe(first(), finalize(() => {
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
