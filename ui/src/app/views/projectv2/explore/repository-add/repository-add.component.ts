import { ChangeDetectionStrategy, ChangeDetectorRef, Component, inject, Input, OnDestroy, OnInit } from '@angular/core';
import { Store } from '@ngxs/store';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { FormBuilder, FormControl, FormGroup, Validators } from '@angular/forms';
import { lastValueFrom } from 'rxjs';
import { Project, ProjectRepository } from 'app/model/project.model';
import { ProjectService } from 'app/service/project/project.service';
import { RepoManagerService } from 'app/service/repomanager/project.repomanager.service';
import { Repository } from 'app/model/repositories.model';
import { VCSProject } from 'app/model/vcs.model';
import { ProjectV2State } from 'app/store/project-v2.state';
import { ErrorUtils } from 'app/shared/error.utils';
import { NzMessageService } from 'ng-zorro-antd/message';
import { NzDrawerRef } from 'ng-zorro-antd/drawer';

export class ProjectV2RepositoryAddComponentParams {
    vcs: string;
}

@Component({
    standalone: false,
    selector: 'app-projectv2-repository-add',
    templateUrl: './repository-add.html',
    styleUrls: ['./repository-add.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectV2RepositoryAddComponent implements OnDestroy, OnInit {
    @Input() params: ProjectV2RepositoryAddComponentParams;

    loaders: {
        global: boolean,
        vcs: boolean,
        repository: boolean
    } = {
            global: false,
            vcs: false,
            repository: false
        };

    project: Project;
    vcsProject: VCSProject;
    vcss: VCSProject[] = [];
    repositories: Repository[];
    validateForm: FormGroup<{
        vcs: FormControl<string | null>;
        repository: FormControl<string | null>;
    }>;
    result: ProjectRepository;
    error: string;

    private _drawerRef = inject(NzDrawerRef<string>);
    private _store = inject(Store);
    private _cd = inject(ChangeDetectorRef);
    private _projectService = inject(ProjectService);
    private _messageService = inject(NzMessageService);
    private _repoManagerService = inject(RepoManagerService);
    private _fb = inject(FormBuilder);

    constructor() {
        this.project = this._store.selectSnapshot(ProjectV2State.current);
        this.validateForm = this._fb.group({
            vcs: this._fb.control<string | null>(null, Validators.required),
            repository: this._fb.control<string | null>(null, Validators.required),
        });
    }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit(): void {
        this.load();
    }

    async load() {
        this.loaders.global = true;
        this._cd.markForCheck();
        try {
            this.vcss = await lastValueFrom(this._projectService.listVCSProject(this.project.key));
        } catch (e) {
            this._messageService.error(`Unable to list VCS: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
            this.loaders.global = false;
            this._cd.markForCheck();
        }
        let selectedVCS = this.params.vcs ?? null;
        if (selectedVCS && this.vcss.findIndex(v => v.name === selectedVCS) !== -1) {
            this.validateForm.controls.vcs.setValue(selectedVCS);
        }
        this.loaders.global = false;
        this._cd.markForCheck();
    }

    async loadRepositories(vcs: string, resync: boolean) {
        this.loaders.repository = true;
        this._cd.markForCheck();
        try {
            const existingProjectRepositories = await lastValueFrom(this._projectService.getVCSRepositories(this.project.key, vcs)); this.repositories = await lastValueFrom(this._repoManagerService.getV2Repositories(this.project.key, vcs, resync));
            const repositories = await lastValueFrom(this._repoManagerService.getV2Repositories(this.project.key, vcs, resync));
            this.repositories = repositories.filter(r => existingProjectRepositories.findIndex((pr) => pr.name === r.fullname) === -1);
            if (this.validateForm.value.repository && this.repositories.findIndex(r => r.fullname === this.validateForm.value.repository) === -1) {
                this.validateForm.controls.repository.reset();
            }
        } catch (e) {
            this._messageService.error(`Unable to list Repositories: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
        }
        this.loaders.repository = false;
        this._cd.markForCheck();
    }

    async vcsChange(value: string) {
        this.loaders.vcs = true;
        this._cd.markForCheck();
        this.vcsProject = this.project.vcs_servers.find(v => v.name === value) ?? null;
        this.loadRepositories(this.vcsProject.name, false);
        this.loaders.vcs = false;
        this._cd.markForCheck();
    }

    async resyncRepositories(e: Event) {
        e.preventDefault();
        e.stopPropagation();
        await this.loadRepositories(this.vcsProject.name, true);
    }

    async submitForm() {
        this.result = null;
        this.error = null;

        if (!this.validateForm.valid) {
            Object.values(this.validateForm.controls).forEach(control => {
                if (control.invalid) {
                    control.markAsDirty();
                    control.updateValueAndValidity({ onlySelf: true });
                }
            });
            return;
        }
        this.validateForm.disable();

        try {
            this.result = await lastValueFrom(this._projectService.addVCSRepository(this.project.key, this.validateForm.value.vcs, <ProjectRepository>{ name: this.validateForm.value.repository }));
        } catch (e) {
            this.error = ErrorUtils.print(e);
        }

        this._cd.markForCheck();
    }

    close(): void {
        this._drawerRef.close();
    }

    clearForm(): void {
        this.result = null;
        this.error = null;
        this.validateForm.enable();
        this._cd.markForCheck();
    }

    isLoading(): boolean {
        return Object.keys(this.loaders).map(k => this.loaders[k]).reduce((p, c) => { return p || c });
    }
}
