import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnInit } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { APIConfig } from 'app/model/config.service';
import { Project } from 'app/model/project.model';
import { RepositoriesManager } from 'app/model/repositories.model';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { ToastService } from 'app/shared/toast/ToastService';
import { ConfigState } from 'app/store/config.state';
import { Subscription } from 'rxjs';

@Component({
    selector: 'app-project-repomanager-list',
    templateUrl: './project.repomanager.list.html',
    styleUrls: ['./project.repomanager.list.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectRepoManagerComponent implements OnInit {

    @Input() project: Project;
    @Input() reposmanagers: RepositoriesManager[];

    public deleteLoading = false;
    loadingDependencies = false;
    repoNameToDelete: string;
    confirmationMessage: string;
    deleteModal: boolean;
    apiConfig: APIConfig;
    configSubscription: Subscription;

    constructor(
        private _toast: ToastService,
        public _translate: TranslateService,
        private _store: Store,
        private _cd: ChangeDetectorRef
    ) { }

    ngOnInit(): void {
        this.configSubscription = this._store.select(ConfigState.api).subscribe(c => {
            this.apiConfig = c;
            this._cd.markForCheck();
        });
    }
}
