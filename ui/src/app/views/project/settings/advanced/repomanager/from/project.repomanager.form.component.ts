import { ChangeDetectionStrategy, ChangeDetectorRef, Component, EventEmitter, inject, Input, OnChanges, Output, SimpleChanges } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { AllKeys } from 'app/model/keys.model';
import { Project } from 'app/model/project.model';
import { VCSProject, VCSProjectAuth, VCSProjectOptions } from 'app/model/vcs.model';
import { ProjectService } from 'app/service/project/project.service';
import { V2ProjectService } from 'app/service/projectv2/project.service';
import { ErrorUtils } from 'app/shared/error.utils';
import { NzMessageService } from 'ng-zorro-antd/message';
import { lastValueFrom } from 'rxjs';

@Component({
    standalone: false,
    selector: 'app-project-repomanager-form',
    templateUrl: './project.repomanager.form.html',
    styleUrls: ['./project.repomanager.form.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ProjectRepoManagerFormComponent implements OnChanges {
    @Input() vcsProject: VCSProject;
    @Input() project: Project;
    @Input() visible: boolean;
    @Output() closed = new EventEmitter<boolean>();

    loading: boolean;
    selectedPublicKey: string;
    keys: AllKeys;

    reposManagerList: string[] = ['bitbucketcloud', 'bitbucketserver', 'github', 'gitlab', 'gitea', 'gerrit'];

    get isEditing(): boolean {
        return !!(this.vcsProject && this.vcsProject.id);
    }

    public _translate = inject(TranslateService);
    private _v2ProjectService = inject(V2ProjectService);
    private _cd = inject(ChangeDetectorRef);
    private _projectService = inject(ProjectService);
    private _messageService = inject(NzMessageService);

    ngOnChanges(changes: SimpleChanges): void {
        if (changes['vcsProject'] && this.vcsProject) {
            if (!this.vcsProject.auth) {
                this.vcsProject.auth = new VCSProjectAuth();
            }
            if (!this.vcsProject.options) {
                this.vcsProject.options = new VCSProjectOptions();
            }
        }
        if (changes['visible'] && this.visible) {
            this.selectedPublicKey = null;
            this.loadKeys();
        }
    }

    async loadKeys() {
        this.loading = true;
        this._cd.markForCheck();
        try {
            const keys = await lastValueFrom(this._v2ProjectService.getKeys(this.project.key));
            this.keys = new AllKeys();
            this.keys.ssh = keys.filter(k => k.type === 'ssh');
        } catch (e) {
            this._messageService.error(`Unable to load keys: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
        }
        this.loading = false;
        this._cd.markForCheck();
    }

    async save() {
        this.loading = true;
        this._cd.markForCheck();
        try {
            if (this.isEditing) {
                await lastValueFrom(this._projectService.saveVCSProject(this.project.key, this.vcsProject));
                this._messageService.success('Repository Manager updated');
            } else {
                await lastValueFrom(this._projectService.addVCSProject(this.project.key, this.vcsProject));
                this._messageService.success('Repository Manager added');
            }
            this.closed.emit(true);
        } catch (e) {
            this._messageService.error(`Unable to save repository manager: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
        } finally {
            this.loading = false;
            this._cd.markForCheck();
        }
    }

    async delete() {
        this.loading = true;
        this._cd.markForCheck();
        try {
            await lastValueFrom(this._projectService.deleteVCSProject(this.project.key, this.vcsProject.name));
            this._messageService.success('Repository Manager deleted');
            this.closed.emit(true);
        } catch (e) {
            this._messageService.error(`Unable to delete repository manager: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
        } finally {
            this.loading = false;
            this._cd.markForCheck();
        }
    }

    cancel(): void {
        this.closed.emit(false);
    }

    updatePublicKey(keyName: any): void {
        if (!this.keys) {
            return;
        }
        const key = this.keys.ssh.find(k => k.name === keyName);
        if (key) {
            this.selectedPublicKey = key.public;
            this.vcsProject.auth.sshKeyName = key.name;
        }
    }

    clickCopyKey() {
        this._messageService.success(this._translate.instant('key_copied'));
    }
}

