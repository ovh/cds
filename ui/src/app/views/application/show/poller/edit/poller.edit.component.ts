import {Component, Input, OnInit, OnDestroy} from '@angular/core';
import {Project} from '../../../../../model/project.model';
import {Application} from '../../../../../model/application.model';
import {RepositoryPoller} from '../../../../../model/polling.model';
import {ApplicationStore} from '../../../../../service/application/application.store';
import {Subscription} from 'rxjs/Subscription';

@Component({
    selector: 'app-application-poller-form',
    templateUrl: './poller.form.html',
    styleUrls: ['./poller.form.scss']
})
export class ApplicationPollerFormComponent implements OnInit {

    @Input() project: Project;
    @Input() application: Application;
    @Input() poller: RepositoryPoller;

    constructor(private _appStore: ApplicationStore) {}


    ngOnInit() {
        if (!this.application.vcs_server) {
            this._appStore.getApplications(this.project.key, this.application.name).first().subscribe( apps => {
                let appKey = this.project.key + '-' + this.application.name;
                if (apps.get(appKey)) {
                    this.application = apps.get(appKey);
                }
            });
        }
    }

    togglePoller(current: boolean) {
        this.poller.hasChanged = true;
        this.poller.enabled = !current;
    }
}
