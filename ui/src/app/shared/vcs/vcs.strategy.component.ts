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
import { AllKeys } from 'app/model/keys.model';
import { Project } from 'app/model/project.model';
import { VCSConnections, VCSStrategy } from 'app/model/vcs.model';
import { KeyService } from 'app/service/keys/keys.service';
import { KeyEvent } from 'app/shared/keys/key.event';
import { ToastService } from 'app/shared/toast/ToastService';
import { finalize } from 'rxjs/operators';
import { NzModalService } from 'ng-zorro-antd/modal';

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
    @Input() hideButton = false;
    @Input() sshWarning = false;
    @Input() projectKeysOnly = false;
    @Input() withoutForm = false;

    selectedPublicKey: string;

    _strategy: VCSStrategy;
    @Input()
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

    constructor(
        private store: Store,
        private _keyService: KeyService,
        private _toast: ToastService,
        private _translate: TranslateService,
        private _cd: ChangeDetectorRef,
        private _modalService: NzModalService
    ) { }

    ngOnInit() {
        if (!this.strategy) {
            this.strategy = new VCSStrategy();
        }
        this.loadKeys();
    }

    loadKeys() {
        if (this.projectKeysOnly) {
            this._keyService.getAllKeys(this.project.key)
                .pipe(finalize(() => this._cd.markForCheck()))
                .subscribe(k => {
                    this.keys = k;
                    if (this.strategy?.ssh_key) {
                        this.updatePublicKey(this.strategy.ssh_key);
                    }
                });
        } else {
            this._keyService.getAllKeys(this.project.key, this.appName)
                .pipe(finalize(() => this._cd.markForCheck()))
                .subscribe(k => {
                    this.keys = k;
                    if (this.strategy?.ssh_key) {
                        this.updatePublicKey(this.strategy.ssh_key);
                    }
                });
        }
    }

    saveStrategy() {
        this.strategyChange.emit(this.strategy);
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
        this._toast.success('', this._translate.instant('key_copied'));
    }
}
