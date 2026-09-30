import { ChangeDetectionStrategy, ChangeDetectorRef, Component, inject, OnDestroy } from '@angular/core';
import { Router } from '@angular/router';
import { NzMessageService } from 'ng-zorro-antd/message';
import { lastValueFrom } from 'rxjs';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { ProjectService } from 'app/service/project/project.service';
import { apiErrorMessage, RepositoryContextService } from './repository-context.service';

/** What the project knows of the repository, and the one action that changes it: removing it. */
@Component({
    standalone: false,
    selector: 'app-projectv2-repository-settings',
    templateUrl: './repository-settings.html',
    styleUrls: ['./repository-settings.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectV2RepositorySettingsComponent implements OnDestroy {
    ctx = inject(RepositoryContextService);
    removing: boolean = false;

    private _cd = inject(ChangeDetectorRef);
    private _router = inject(Router);
    private _projectService = inject(ProjectService);
    private _messageService = inject(NzMessageService);

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    get canManage(): boolean {
        return this.ctx.project?.permissions?.writable ?? false;
    }

    /** The repository leaves the project; its page has nothing left to show, the overview takes over. */
    async removeRepositoryFromProject() {
        this.removing = true;
        this._cd.markForCheck();
        try {
            await lastValueFrom(this._projectService.deleteVCSRepository(this.ctx.project.key, this.ctx.vcsName, this.ctx.repoName));
            this._messageService.success('Repository has been removed');
            this._router.navigate(['/project', this.ctx.project.key, 'explore']);
        } catch (e) {
            this._messageService.error(`Unable to remove repository: ${apiErrorMessage(e)}`, { nzDuration: 2000 });
        }
        this.removing = false;
        this._cd.markForCheck();
    }
}
