import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, OnInit, ViewChild, } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { AuthSummary } from 'app/model/user.model';
import { ConfigService } from 'app/service/config/config.service';
import { ThemeStore } from 'app/service/theme/theme.store';
import { PathItem } from 'app/shared/breadcrumb/breadcrumb.component';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { AuthenticationState } from 'app/store/authentication.state';
import { Subscription } from 'rxjs';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-cdsctl',
    templateUrl: './cdsctl.html',
    styleUrls: ['./cdsctl.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class CdsctlComponent implements OnInit, OnDestroy {
    @ViewChild('codemirror1') codemirror1: any;
    @ViewChild('codemirror2') codemirror2: any;
    @ViewChild('codemirror3') codemirror3: any;
    @ViewChild('codemirror4') codemirror4: any;
    @ViewChild('codemirror5') codemirror5: any;
    @ViewChild('codemirror6') codemirror6: any;
    @ViewChild('codemirror7') codemirror7: any;
    @ViewChild('codemirror8') codemirror8: any;

    currentAuthSummary: AuthSummary;
    apiURL: string;
    arch: Array<string>;
    os: Array<string>;
    withKeychain: boolean;
    path: Array<PathItem>;
    codeMirrorConfig: any;
    tutorials: Array<string> = new Array();
    osChoice: string;
    archChoice: string;
    loading: boolean;
    themeSubscription: Subscription;

    constructor(
        private _store: Store,
        private _configService: ConfigService,
        private _translate: TranslateService,
        private _theme: ThemeStore,
        private _cd: ChangeDetectorRef
    ) {
        this.codeMirrorConfig = {
            matchBrackets: true,
            autoCloseBrackets: true,
            mode: 'shell',
            lineWrapping: true,
            autoRefresh: true,
            readOnly: true,
            lineNumbers: true
        };

        this.withKeychain = true;
        this.os = new Array<string>('windows', 'linux', 'darwin', 'freebsd');
        this.arch = new Array<string>('amd64', '386', 'arm', 'arm64');
        this.osChoice = 'linux';
        this.archChoice = 'amd64'
    }

    ngOnDestroy(): void {} // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit() {
        this.themeSubscription = this._theme.get().subscribe(t => {
            this.codeMirrorConfig.theme = t === 'night' ? 'darcula' : 'default';
            if (this.codemirror1 && this.codemirror1.instance) {
                this.codemirror1.instance.setOption('theme', this.codeMirrorConfig.theme);
            }
            if (this.codemirror2 && this.codemirror2.instance) {
                this.codemirror2.instance.setOption('theme', this.codeMirrorConfig.theme);
            }
            if (this.codemirror3 && this.codemirror3.instance) {
                this.codemirror3.instance.setOption('theme', this.codeMirrorConfig.theme);
            }
            if (this.codemirror4 && this.codemirror4.instance) {
                this.codemirror4.instance.setOption('theme', this.codeMirrorConfig.theme);
            }
            if (this.codemirror5 && this.codemirror5.instance) {
                this.codemirror5.instance.setOption('theme', this.codeMirrorConfig.theme);
            }
            if (this.codemirror6 && this.codemirror6.instance) {
                this.codemirror6.instance.setOption('theme', this.codeMirrorConfig.theme);
            }
            if (this.codemirror7 && this.codemirror7.instance) {
                this.codemirror7.instance.setOption('theme', this.codeMirrorConfig.theme);
            }
            if (this.codemirror8 && this.codemirror8.instance) {
                this.codemirror8.instance.setOption('theme', this.codeMirrorConfig.theme);
            }
            this._cd.markForCheck();
        });

        this.currentAuthSummary = this._store.selectSnapshot(AuthenticationState.summary);

        this.loading = true;
        this._configService.getConfig()
            .pipe(finalize(() => {
                this.loading = false;
                this._cd.markForCheck();
            }))
            .subscribe(r => {
                this.apiURL = r['url.api'];
                this.loading = false;
                this.buildData();
            });
    }

    buildData(): void {
        let variant = '';
        if (!this.withKeychain) {
            variant = '?variant=nokeychain'
        }
        this.tutorials['part1'] = this._translate.instant('cdsctl_part_1',
            { apiURL: this.apiURL, osChoice: this.osChoice, archChoice: this.archChoice, variant });
        this.tutorials['part2'] = this._translate.instant('cdsctl_part_2',
            { apiURL: this.apiURL, username: this.currentAuthSummary.user.username });
        this.tutorials['part3'] = this._translate.instant('cdsctl_part_3');
        this.tutorials['part4'] = this._translate.instant('cdsctl_part_4');
        this.tutorials['part5'] = this._translate.instant('cdsctl_part_5');
        this.tutorials['part6'] = this._translate.instant('cdsctl_part_6');
        this.tutorials['part7'] = this._translate.instant('cdsctl_part_7');
        this.tutorials['part8'] = this._translate.instant('cdsctl_part_8');

        this.path = [<PathItem>{
            translate: 'common_settings'
        }, <PathItem>{
            translate: 'navbar_cdsctl',
            routerLink: ['/', 'settings', 'cdsctl']
        }];
    }
}
