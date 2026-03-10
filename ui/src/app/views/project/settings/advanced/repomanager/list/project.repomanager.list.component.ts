import { ChangeDetectionStrategy, ChangeDetectorRef, Component, inject, Input, OnInit } from '@angular/core';
import { Project } from 'app/model/project.model';
import { VCSProject, VCSProjectAuth, VCSProjectOptions } from 'app/model/vcs.model';
import { ProjectService } from 'app/service/project/project.service';
import { ErrorUtils } from 'app/shared/error.utils';
import { NzMessageService } from 'ng-zorro-antd/message';
import { lastValueFrom } from 'rxjs';

@Component({
    standalone: false,
    selector: 'app-project-repomanager-list',
    templateUrl: './project.repomanager.list.html',
    styleUrls: ['./project.repomanager.list.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ProjectRepoManagerListComponent implements OnInit {

    @Input() project: Project;
    @Input() canAdd: boolean = true;

    loading = false;
    vcsProjects: VCSProject[] = [];
    selectedVCSProject: VCSProject | null = null;
    modalVisible = false;

    private _projectService = inject(ProjectService);
    private _messageService = inject(NzMessageService);
    private _cd = inject(ChangeDetectorRef);

    ngOnInit(): void {
        this.load();
    }

    async load() {
        this.loading = true;
        this._cd.markForCheck();
        try {
            this.vcsProjects = await lastValueFrom(this._projectService.listVCSProject(this.project.key));
        } catch (e) {
            this._messageService.error(`Unable to load repository managers: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
        }
        this.loading = false;
        this._cd.markForCheck();
    }

    openCreate(): void {
        const vcs = new VCSProject();
        vcs.options = new VCSProjectOptions();
        vcs.auth = new VCSProjectAuth();
        this.selectedVCSProject = vcs;
        this.modalVisible = true;
        this._cd.markForCheck();
    }

    openEdit(r: VCSProject): void {
        this._projectService.getVCSProject(this.project.key, r.name).subscribe({
            next: vcsProject => {
                this.selectedVCSProject = vcsProject;
                this.modalVisible = true;
                this._cd.markForCheck();
            },
            error: e => {
                this._messageService.error(`Unable to load repository manager: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
            }
        });
    }

    onModalClose(refresh: boolean): void {
        this.modalVisible = false;
        this.selectedVCSProject = null;
        if (refresh) {
            this.load();
        }
        this._cd.markForCheck();
    }
}
