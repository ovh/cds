import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnInit } from "@angular/core";
import { FormBuilder, FormControl, FormGroup, Validators } from "@angular/forms";
import { Store } from "@ngxs/store";
import { EntityType } from "app/model/entity.model";
import { HookEventWorkflowStatus, Project, ProjectRepository, RepositoryHookEvent } from "app/model/project.model";
import { VCSProject } from "app/model/vcs.model";
import { ProjectService } from "app/service/project/project.service";
import { V2WorkflowRunService } from "app/service/services.module";
import { lastValueFrom } from "rxjs";
import { V2Workflow, V2WorkflowRunManualRequest } from "../../../../../libs/workflow-graph/src/lib/v2.workflow.run.model";
import { NzMessageService } from "ng-zorro-antd/message";
import { NzDrawerRef } from "ng-zorro-antd/drawer";
import { LoadOptions, load } from "js-yaml";
import { Branch, Tag } from "app/model/repositories.model";
import { ErrorUtils } from "app/shared/error.utils";
import { ProjectV2State } from "app/store/project-v2.state";

export class ProjectV2RunStartComponentParams {
  workflow_repository: string;
  repository: string;
  workflow_ref: string;
  ref: string;
  workflow: string;
}

@Component({
  selector: 'app-projectv2-run-start',
  templateUrl: './run-start.html',
  styleUrls: ['./run-start.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class ProjectV2RunStartComponent implements OnInit {
  @Input() params: ProjectV2RunStartComponentParams;

  project: Project;
  vcss: Array<VCSProject> = [];
  repositories: { [vcs: string]: Array<ProjectRepository> } = {};
  branches: Array<Branch> = [];
  tags: Array<Tag> = [];
  sourceBranches: Array<Branch> = [];
  sourceTags: Array<Tag> = [];
  workflows: Array<string> = [];
  validateForm: FormGroup<{
    repository: FormControl<string | null>;
    ref: FormControl<string | null>;
    workflow: FormControl<string | null>;
    sourceRepository: FormControl<string | null>;
    sourceRef: FormControl<string | null>;
  }>;
  event: RepositoryHookEvent;
  loaders: {
    global: boolean,
    repository: boolean,
    ref: boolean,
    workflow: boolean
  } = {
    global: false,
    repository: false,
    ref: false,
    workflow: false
  };

  constructor(
    private _drawerRef: NzDrawerRef<string>,
    private _messageService: NzMessageService,
    private _store: Store,
    private _projectService: ProjectService,
    private _fb: FormBuilder,
    private _cd: ChangeDetectorRef,
    private _workflowRunService: V2WorkflowRunService,
  ) {
    this.project = this._store.selectSnapshot(ProjectV2State.current);
    this.validateForm = this._fb.group({
      repository: this._fb.control<string | null>(null, Validators.required),
      ref: this._fb.control<string | null>(null, Validators.required),
      workflow: this._fb.control<string | null>(null, Validators.required),
      sourceRepository: this._fb.control<string | null>({ disabled: true, value: '' }),
      sourceRef: this._fb.control<string | null>(null),
    });
  }

  ngOnInit(): void {
    this.load();
  }

  async load() {
    this.loaders.global = true;
    this._cd.markForCheck();
    try {
      this.vcss = await lastValueFrom(this._projectService.listVCSProject(this.project.key));
      const resp = await Promise.all(this.vcss.map(vcs => lastValueFrom(this._projectService.getVCSRepositories(this.project.key, vcs.name))));
      this.repositories = {};
      this.vcss.forEach((vcs, i) => {
        this.repositories[vcs.name] = resp[i];
      });
    } catch (e) {
      this._messageService.error(`Unable to list repositories: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
      this.loaders.global = false;
      this._cd.markForCheck();
      return
    }
    let selectedRepository = this.params.workflow_repository ?? (this.params.repository ?? null);
    if (!selectedRepository && this.params.workflow) {
      const splitted = this.splitWorkflow(this.params.workflow);
      selectedRepository = `${splitted.vcs}/${splitted.repo}`;
    }
    if (selectedRepository) {
      const splitted = this.splitRepository(selectedRepository);
      if (this.repositories[splitted.vcs] && this.repositories[splitted.vcs].findIndex(r => r.name === splitted.repo) !== -1) {
        this.validateForm.controls.repository.setValue(selectedRepository);
      }
    }
    this.loaders.global = false;
    this._cd.markForCheck();
  }

  async repositoryChange(value: string) {
    this.loaders.repository = true;
    this._cd.markForCheck();
    const splitted = this.splitRepository(value);
    try {
      this.branches = await lastValueFrom(this._projectService.getVCSRepositoryBranches(this.project.key, splitted.vcs, splitted.repo, 50));
      this.tags = await lastValueFrom(this._projectService.getVCSRepositoryTags(this.project.key, splitted.vcs, splitted.repo));
    } catch (e) {
      this._messageService.error(`Unable to get repository refs: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
      this.loaders.repository = false;
      this._cd.markForCheck();
      return
    }
    const selectedRef = this.params.workflow_ref ?? (this.params.ref ?? null);
    if (selectedRef && (this.branches.findIndex(b => `refs/heads/${b.display_id}` === selectedRef) !== -1 || this.tags.findIndex(t => `refs/tags/${t.tag}` === selectedRef) !== -1)) {
      this.validateForm.controls.ref.setValue(selectedRef);
    } else {
      this.validateForm.controls.ref.setValue('refs/heads/' + this.branches.find(b => b.default).display_id);
    }
    this.loaders.repository = false;
    this._cd.markForCheck();
  }

  async refChange(branch: string) {
    this.loaders.ref = true;
    this._cd.markForCheck();
    const splitted = this.splitRepository(this.validateForm.controls.repository.value);
    try {
      const resp = await lastValueFrom(this._projectService.getRepoEntities(this.project.key, splitted.vcs, splitted.repo, branch));
      this.workflows = resp.filter(e => e.type === EntityType.Workflow).map(e => e.name);
    } catch (e) {
      this._messageService.error(`Unable to get repo entities: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
      this.loaders.ref = false;
      this._cd.markForCheck();
      return
    }
    if (this.params.workflow && this.workflows.findIndex(w => `${splitted.vcs}/${splitted.repo}/${w}` === this.params.workflow) !== -1) {
      this.validateForm.controls.workflow.setValue(this.params.workflow.replace(`${splitted.vcs}/${splitted.repo}/`, ''));
    } else {
      this.validateForm.controls.workflow.reset();
    }
    this.loaders.ref = false;
    this._cd.markForCheck();
  }

  async workflowChange(workflow: string) {
    if (!workflow) {
      return;
    }
    this.loaders.workflow = true;
    this._cd.markForCheck();
    const form = this.validateForm.controls;
    const splitted = this.splitRepository(form.repository.value);
    let wkf: V2Workflow;
    try {
      const entity = await lastValueFrom(this._projectService.getRepoEntity(this.project.key, splitted.vcs, splitted.repo, EntityType.Workflow, form.workflow.value, form.ref.value));
      wkf = load(entity.data && entity.data !== '' ? entity.data : '{}', <LoadOptions>{
        onWarning: (e) => { }
      });
    } catch (e) {
      this._messageService.error(`Unable to get workflow entity from repo: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
      this.loaders.workflow = false;
      this._cd.markForCheck();
      return
    }
    if (wkf.repository) {
      this.validateForm.controls.sourceRepository.setValidators([Validators.required]);
      this.validateForm.controls.sourceRef.setValidators([Validators.required]);
      this.validateForm.controls.sourceRepository.setValue(wkf.repository.vcs + '/' + wkf.repository.name);
      this.sourceBranches = await lastValueFrom(this._projectService.getVCSRepositoryBranches(this.project.key, wkf.repository.vcs, wkf.repository.name, 50));
      this.sourceTags = await lastValueFrom(this._projectService.getVCSRepositoryTags(this.project.key, wkf.repository.vcs, wkf.repository.name));
      const selectedSourceRef = this.params.ref ?? null;
      if (selectedSourceRef && (this.sourceBranches.findIndex(b => `refs/heads/${b.display_id}` === selectedSourceRef) !== -1 || this.sourceTags.findIndex(t => `refs/tags/${t.tag}` === selectedSourceRef) !== -1)) {
        this.validateForm.controls.sourceRef.setValue(selectedSourceRef);
      } else {
        this.validateForm.controls.sourceRef.setValue('refs/heads/' + this.sourceBranches.find(b => b.default).display_id);
      }
    } else {
      this.validateForm.controls.sourceRepository.reset();
      this.validateForm.controls.sourceRef.reset();
      this.validateForm.controls.sourceRepository.clearValidators();
      this.validateForm.controls.sourceRef.clearValidators();
    }
    this.loaders.workflow = false;
    this._cd.markForCheck();
  }

  close(): void {
    this._drawerRef.close();
  }

  splitRepository(repo: string): { vcs: string, repo: string } {
    const splitted = repo.split('/');
    return {
      vcs: splitted.splice(0, 1)[0],
      repo: splitted.join('/')
    };
  }

  splitWorkflow(repo: string): { vcs: string, repo: string, workflow: string } {
    const splitted = repo.split('/');
    const workflow = splitted.splice(-1, 1)[0];
    return {
      vcs: splitted.splice(0, 1)[0],
      repo: splitted.join('/'),
      workflow
    };
  }

  async submitForm() {
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
    this._cd.markForCheck();

    const splitted = this.splitRepository(this.validateForm.controls.repository.value);
    const ref = this.validateForm.value.sourceRef ?? this.validateForm.value.ref;
    let req = <V2WorkflowRunManualRequest>{};
    if (ref.startsWith('refs/tags/')) {
      req.tag = ref.replace('refs/tags/', '');
    } else {
      req.branch = ref.replace('refs/heads/', '');
    }
    if (this.validateForm.value.sourceRef) {
      const ref = this.validateForm.value.ref;
      if (ref.startsWith('refs/tags/')) {
        req.workflow_tag = ref.replace('refs/tags/', '');
      } else {
        req.workflow_branch = ref.replace('refs/heads/', '');
      }
    }

    let hookEventUUID: string;
    try {
      const resp = await lastValueFrom(this._workflowRunService.start(this.project.key, splitted.vcs, splitted.repo, this.validateForm.value.workflow, req));
      this._messageService.success('Workflow run started', { nzDuration: 2000 });
      hookEventUUID = resp.hook_event_uuid;
    } catch (e) {
      this._messageService.error(`Unable to start the workflow: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
      this.clearForm();
      return
    }

    // Wait for workflow run to start
    let retry = 0
    while (retry < 90) {
      try {
        this.event = await lastValueFrom(this._projectService.getRepositoryEvent(this.project.key, splitted.vcs, splitted.repo, hookEventUUID));
        this._cd.markForCheck();
        if (this.event.status === HookEventWorkflowStatus.Done || this.event.status === HookEventWorkflowStatus.Error || this.event.status === HookEventWorkflowStatus.Skipped) {
          break;
        }
      } catch (e) {
        this._messageService.error(`Unable to get repository event: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
      }
      await (new Promise(resolve => setTimeout(resolve, 1000)));
      retry++;
    }
  }

  clearForm(): void {
    this.event = null;
    this.validateForm.enable();
    this._cd.markForCheck();
  }

  isLoading(): boolean {
    return Object.keys(this.loaders).map(k => this.loaders[k]).reduce((p, c) => { return p || c });
  }
}