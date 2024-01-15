import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Project } from 'app/model/project.model';
import { VCSProject, VCSProjectAuth, VCSProjectOptions } from 'app/model/vcs.model';
import { ProjectService } from 'app/service/project/project.service';
import { ToastService } from 'app/shared/toast/ToastService';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-repomanager-form',
    templateUrl: './repomanager.form.html',
    styleUrls: ['./repomanager.form.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class RepoManagerFormComponent {
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

    repoModalVisible: boolean;
    addingVCSProject: boolean;
    askDeleting: boolean;

    vcsProject: VCSProject;

    constructor(
        public _translate: TranslateService,
        private _toastService: ToastService,
        private _cd: ChangeDetectorRef,
        private _projectService: ProjectService) {
            this.askDeleting = false;
            this.reposManagerList = ["bitbucketcloud", "bitbucketserver", "github", "gitlab", "gitea", "gerrit"];
            if (!this.vcsProjectName) {
                this.vcsProject = new VCSProject();
                this.vcsProject.options = new VCSProjectOptions();
                this.vcsProject.auth = new VCSProjectAuth();
            }
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

    addVCSProject(): void {
        if (!this.reposManagerList[this.selectedRepoId]) {
            return;
        }
        this.loading = true;
        this._cd.markForCheck();
        this._projectService.addVCSProject(this.project.key, this.vcsProject).pipe(finalize(() => {
            this.loading = false;
            this._cd.markForCheck();
        })).subscribe(r => {
            this._toastService.success('Repository Manager added', '');
            this._projectService.listVCSProject(this.project.key).subscribe(vcsProjects => {
                this.repoModalVisible = false;
                this.project.vcs_servers = vcsProjects;
                this._cd.markForCheck();
            });
        });
    }

    saveVCSProject(): void {
        this.loading = true;
        this._cd.markForCheck();
        this._projectService.addVCSProject(this.project.key, this.vcsProject).pipe(finalize(() => {
            this.loading = false;
            this._cd.markForCheck();
        })).subscribe(r => {
            this._toastService.success('Repository Manager updated', '');
            this._projectService.listVCSProject(this.project.key).subscribe(vcsProjects => {
                this.repoModalVisible = false;
                this.project.vcs_servers = vcsProjects;
                this._cd.markForCheck();
            });
        });
    }

    deleteVCSProject(): void {
        this.loading = true;
        this._cd.markForCheck();
        this._projectService.deleteVCSProject(this.project.key, this.vcsProject.name).pipe(finalize(() => {
            this.loading = false;
            this._cd.markForCheck();
        })).subscribe(r => {
            this._toastService.success('Repository Manager deleted', '');
            this._projectService.listVCSProject(this.project.key).subscribe(vcsProjects => {
                this.repoModalVisible = false;
                this.project.vcs_servers = vcsProjects;
                this._cd.markForCheck();
            });
        });
    }
}
