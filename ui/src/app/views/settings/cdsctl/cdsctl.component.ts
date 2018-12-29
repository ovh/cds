import { Component,  } from '@angular/core';
import { environment } from '../../../../environments/environment';
import { User } from '../../../model/user.model';
import { AuthentificationStore } from '../../../service/auth/authentification.store';
import { PathItem } from '../../../shared/breadcrumb/breadcrumb.component';


@Component({
    selector: 'app-cdsctl',
    templateUrl: './cdsctl.html',
    styleUrls: ['./cdsctl.scss']
})
export class CdsctlComponent {
    currentUser: User;
    apiURL: string;
    path: Array<PathItem>;
    codeMirrorConfig: any;
    tutorials: Array<string> = new Array();

    constructor(
        private _authentificationStore: AuthentificationStore,
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

        this.currentUser = this._authentificationStore.getUser();
        this.apiURL = environment.apiURL;

        this.tutorials['part1'] = '# replace `linux` with your OS. Available: windows, darwin, freebsd, linux\n';
        this.tutorials['part1'] += '# replace `amd64` with by `386` if you don\'t want to use a keychain\n';
        this.tutorials['part1'] += 'curl ' + this.apiURL + '/download/cdsctl/linux/amd64 -o cdsctl\n\n';
        this.tutorials['part1'] += '# add execution rights on cdsctl\n';
        this.tutorials['part1'] += 'chmod +x cdsctl\n\n';
        this.tutorials['part1'] += '# check if binary is ok, this command will display cdsctl version\n';
        this.tutorials['part1'] += './cdsctl version';

        this.tutorials['part2'] = '# The cdsctl login command will store credentials in your keychain\n';
        this.tutorials['part2'] += './cdsctl login --api-url ' + this.apiURL + ' -u ' + this.currentUser.username + '\n\n';
        this.tutorials['part2'] += '# this command will display your username, fullname, email\n';
        this.tutorials['part2'] += '# if you see them, cdsctl is well configured\n';
        this.tutorials['part2'] += './cdsctl user me';

        this.tutorials['part3'] = '# The cdsctl shell to browse your projects and workflows without the need to open a browser.\n';
        this.tutorials['part3'] += '# see https://ovh.github.io/cds/cli/cdsctl/shell/\n';
        this.tutorials['part3'] += 'cdsctl shell\n\n';
        this.tutorials['part3'] += '# Display monitoring view\n';
        this.tutorials['part3'] += 'cdsctl ui\n\n';
        this.tutorials['part3'] += '# help is available on each cdsctl command\n';
        this.tutorials['part3'] += 'cdsctl --help';

        this.tutorials['part4'] = '# Launch a workflow, consider running cdsctl ' +
            'from a directory containing a git repository known from cds\n';
        this.tutorials['part4'] += 'cdsctl workflow run\n\n';
        this.tutorials['part4'] += '# You can explicitely set a projet and workflow\n';
        this.tutorials['part4'] += 'cdsctl workflow run PRJ_KEY WORKFLOW_NAME\n\n';
        this.tutorials['part4'] += '# flags --interactive or --open-web-browser may interest you, help for all details\n';
        this.tutorials['part4'] += 'cdsctl workflow run --help';

        this.tutorials['part5'] = '# Check status of a run\n';
        this.tutorials['part5'] += 'cdsctl workflow status\n\n';
        this.tutorials['part5'] += '# --track command will wait the end of the workflow run\n';
        this.tutorials['part5'] += 'cdsctl workflow status --track';

        this.tutorials['part6'] = '# Add a git alias in your gitconfig file\n';
        this.tutorials['part6'] += '$ cat ~/.gitconfig\n';
        this.tutorials['part6'] += '[alias]\n';
        this.tutorials['part6'] += '   track = !cdsctl workflow status --track\n';

        this.path = [<PathItem>{
            translate: 'common_settings'
        }, <PathItem>{
            translate: 'navbar_cdsctl',
            routerLink: ['/', 'settings', 'cdsctl']
        }];
    }
}
