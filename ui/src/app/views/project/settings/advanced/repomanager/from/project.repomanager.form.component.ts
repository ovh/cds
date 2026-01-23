import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnInit } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { AllKeys, Key } from 'app/model/keys.model';
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
export class ProjectRepoManagerFormComponent implements OnInit {
    @Input() vcsProjectName: string;
    @Input() project: Project;

    loading: boolean;
    public ready = false;
    public connectLoading = false;
    public verificationLoading = false;

    // Repo manager form data
    reposManagerList: string[];
    selectedRepoId: number;
    selectedRepoType: string;
    selectedPublicKey: string;

    repoModalVisible: boolean;
    addingVCSProject: boolean;
    askDeleting: boolean;
    keys: AllKeys;

    vcsProject: VCSProject;

    constructor(
        public _translate: TranslateService,
        private _v2ProjectService: V2ProjectService,
        private _cd: ChangeDetectorRef,
        private _projectService: ProjectService,
        private _messageService: NzMessageService
    ) {
        this.askDeleting = false;
        this.reposManagerList = ["bitbucketcloud", "bitbucketserver", "github", "gitlab", "gitea"];
        if (!this.vcsProjectName) {
            this.vcsProject = new VCSProject();
            this.vcsProject.options = new VCSProjectOptions();
            this.vcsProject.auth = new VCSProjectAuth();
        }
    }

    ngOnInit(): void {
        this.load();
    }

    async load() {
        this.loading = true;
        this._cd.markForCheck();
        try {
            const keys = await lastValueFrom(this._v2ProjectService.getKeys(this.project.key));
            this.keys = new AllKeys();
            this.keys.ssh = keys.filter(k => k.type === 'ssh');
        } catch (e) {
            this._messageService.error(`Unable to load integrations: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
        }
        this.loading = false;
        this._cd.markForCheck();
    }

    create(): void {
        if (this.reposManagerList[this.selectedRepoId]) {
            this.repoModalVisible = true;
            this.selectedRepoType = this.reposManagerList[this.selectedRepoId];
            this.vcsProject.type = this.reposManagerList[this.selectedRepoId];
        }
    }

    view(): void {
        if (this.vcsProjectName) {
            this._projectService.getVCSProject(this.project.key, this.vcsProjectName).subscribe(vcsProject => {
                this.vcsProject = vcsProject;
                this.repoModalVisible = true;
                this._cd.markForCheck();
            });
        }
    }

    async addVCSProject() {
        if (!this.reposManagerList[this.selectedRepoId]) {
            return;
        }

        this.loading = true;
        this._cd.markForCheck();

        try {
            await lastValueFrom(this._projectService.addVCSProject(this.project.key, this.vcsProject));
            this._messageService.success('Repository Manager added');
        } catch (e) {
            this._messageService.error(`Unable to add repository manager: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
            return;
        } finally {
            this.loading = false;
            this._cd.markForCheck();
        }

        this.loadVCSProject();
    }

    async loadVCSProject() {
        try {
            this.project.vcs_servers = await lastValueFrom(this._projectService.listVCSProject(this.project.key));
        } catch (e) {
            this._messageService.error(`Unable to load repository managers: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
            return;
        } finally {
            this.repoModalVisible = false;
            this._cd.markForCheck();
        }
    }

    async saveVCSProject() {
        this.loading = true;
        this._cd.markForCheck();

        try {
            await lastValueFrom(this._projectService.saveVCSProject(this.project.key, this.vcsProject));
            this._messageService.success('Repository Manager updated');
        } catch (e) {
            this._messageService.error(`Unable to save repository manager: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
            return;
        } finally {
            this.loading = false;
            this._cd.markForCheck();
        }

        this.loadVCSProject();
    }

    async deleteVCSProject() {
        this.loading = true;
        this._cd.markForCheck();

        try {
            await lastValueFrom(this._projectService.deleteVCSProject(this.project.key, this.vcsProject.name));
            this._messageService.success('Repository Manager deleted');
        } catch (e) {
            this._messageService.error(`Unable to delete repository manager: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
            return;
        } finally {
            this.loading = false;
            this._cd.markForCheck();
        }

        this.loadVCSProject();
    }

    updatePublicKey(keyName: string): void {
        if (!this.keys) {
            return;
        }
        let key = this.keys.ssh.find(k => k.name === keyName);
        if (key) {
            this.selectedPublicKey = key.public;
            this.vcsProject.auth.sshKeyName = key.name;
        }
    }

    clickCopyKey() {
        this._messageService.success(this._translate.instant('key_copied'));
    }
}
