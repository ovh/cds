import { Component,  } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { User } from '../../../model/user.model';
import { AuthentificationStore } from '../../../service/auth/authentification.store';
import { ConfigService } from '../../../service/config/config.service';
import { PathItem } from '../../../shared/breadcrumb/breadcrumb.component';


@Component({
    selector: 'app-cdsctl',
    templateUrl: './cdsctl.html',
    styleUrls: ['./cdsctl.scss']
})
export class CdsctlComponent {
    currentUser: User;
    apiURL: string;
    arch: Array<string>;
    os: Array<string>;
    path: Array<PathItem>;
    codeMirrorConfig: any;
    tutorials: Array<string> = new Array();
    osChoice: string;
    archChoice: string;
    loading = true;

    constructor(
        private _authentificationStore: AuthentificationStore,
        private _configService: ConfigService,
        private _translate: TranslateService
    ) {
        this.loading = true;
        this._configService.getConfig().subscribe(r => {
            this.apiURL = r['url.api'];
            this.loading = false;
            this.codeMirrorConfig = {
                matchBrackets: true,
                autoCloseBrackets: true,
                mode: 'shell',
                lineWrapping: true,
                autoRefresh: true,
                readOnly: true,
                lineNumbers: true
            };

            this.os = new Array<string>('windows', 'linux', 'darwin', 'freebsd');
            this.arch = new Array<string>('amd64', '386');
            this.osChoice = 'linux';
            this.archChoice = 'amd64'
            this.currentUser = this._authentificationStore.getUser();
            this.buildData();
        });
    }

    buildData(): void {
        this.tutorials['part1'] = this._translate.instant('cdsctl_part_1',
            {apiURL: this.apiURL, osChoice: this.osChoice, archChoice: this.archChoice});
        this.tutorials['part2'] = this._translate.instant('cdsctl_part_2',
            {apiURL: this.apiURL, username: this.currentUser.username});
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
