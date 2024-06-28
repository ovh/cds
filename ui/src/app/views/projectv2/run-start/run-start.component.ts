import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnInit } from "@angular/core";
import { FormBuilder, FormControl, FormGroup, Validators } from "@angular/forms";
import { Store } from "@ngxs/store";
import { EntityType } from "app/model/entity.model";
import { HookEventWorkflowStatus, Project, ProjectRepository, RepositoryHookEvent } from "app/model/project.model";
import { VCSProject } from "app/model/vcs.model";
import { ProjectService } from "app/service/project/project.service";
import { V2WorkflowRunService } from "app/service/services.module";
import { ProjectState } from "app/store/project.state";
import { lastValueFrom } from "rxjs";
import { V2Workflow, V2WorkflowRunManualRequest } from "../../../../../libs/workflow-graph/src/lib/v2.workflow.run.model";
import { NzMessageService } from "ng-zorro-antd/message";
import { NzDrawerRef } from "ng-zorro-antd/drawer";
import { LoadOptions, load } from "js-yaml";

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
  branches: Array<string> = [];
  sourceBranches: Array<string> = [];
  workflows: Array<string> = [];
  validateForm: FormGroup<{
    repository: FormControl<string | null>;
    branch: FormControl<string | null>;
    workflow: FormControl<string | null>;
    sourceRepository: FormControl<string | null>;
    sourceBranch: FormControl<string | null>;
  }>;
  event: RepositoryHookEvent;

  constructor(
    private _drawerRef: NzDrawerRef<string>,
    private _messageService: NzMessageService,
    private _store: Store,
    private _projectService: ProjectService,
    private _fb: FormBuilder,
    private _cd: ChangeDetectorRef,
    private _workflowRunService: V2WorkflowRunService,
  ) {
    this.project = this._store.selectSnapshot(ProjectState.projectSnapshot);
    this.validateForm = this._fb.group({
      repository: this._fb.control<string | null>(null, Validators.required),
      branch: this._fb.control<string | null>(null, Validators.required),
      workflow: this._fb.control<string | null>(null, Validators.required),
      sourceRepository: this._fb.control<string | null>({ disabled: true, value: '' }),
      sourceBranch: this._fb.control<string | null>(null),
    });
  }

  ngOnInit(): void {
    this.load();
  }

  async load() {
    this.vcss = await lastValueFrom(this._projectService.listVCSProject(this.project.key));
    const resp = await Promise.all(this.vcss.map(vcs => lastValueFrom(this._projectService.getVCSRepositories(this.project.key, vcs.name))));
    this.repositories = {};
    this.vcss.forEach((vcs, i) => {
      this.repositories[vcs.name] = resp[i];
    });
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
    this._cd.markForCheck();
  }

  async repositoryChange(value: string) {
    const splitted = this.splitRepository(value);
    const branches = await lastValueFrom(this._projectService.getVCSRepositoryBranches(this.project.key, splitted.vcs, splitted.repo, 50));
    this.branches = branches.map(b => b.display_id);
    const selectedBranch = this.params.workflow_ref ?? (this.params.ref ?? null);
    if (selectedBranch && branches.findIndex(b => `refs/heads/${b.display_id}` === selectedBranch) !== -1) {
      this.validateForm.controls.branch.setValue(selectedBranch.replace('refs/heads/', ''));
    } else {
      this.validateForm.controls.branch.setValue(branches.find(b => b.default).display_id);
    }
    this._cd.markForCheck();
  }

  async branchChange(branch: string) {
    const splitted = this.splitRepository(this.validateForm.controls.repository.value);
    const resp = await lastValueFrom(this._projectService.getRepoEntities(this.project.key, splitted.vcs, splitted.repo, branch));
    this.workflows = resp.filter(e => e.type === EntityType.Workflow).map(e => e.name);
    if (this.params.workflow && this.workflows.findIndex(w => `${splitted.vcs}/${splitted.repo}/${w}` === this.params.workflow) !== -1) {
      this.validateForm.controls.workflow.setValue(this.params.workflow.replace(`${splitted.vcs}/${splitted.repo}/`, ''));
    } else {
      this.validateForm.controls.workflow.reset();
    }
    this._cd.markForCheck();
  }

  async workflowChange(workflow: string) {
    if (!workflow) {
      return;
    }
    const form = this.validateForm.controls;
    const splitted = this.splitRepository(form.repository.value);
    const entity = await lastValueFrom(this._projectService.getRepoEntity(this.project.key, splitted.vcs, splitted.repo, EntityType.Workflow, form.workflow.value, form.branch.value));
    let wkf: V2Workflow;
    try {
      wkf = load(entity.data && entity.data !== '' ? entity.data : '{}', <LoadOptions>{
        onWarning: (e) => { }
      });
    } catch (e) {
      console.error("Invalid workflow:", entity.data, e)
    }
    if (wkf.repository) {
      this.validateForm.controls.sourceRepository.setValidators([Validators.required]);
      this.validateForm.controls.sourceBranch.setValidators([Validators.required]);
      this.validateForm.controls.sourceRepository.setValue(wkf.repository.vcs + '/' + wkf.repository.name);
      const branches = await lastValueFrom(this._projectService.getVCSRepositoryBranches(this.project.key, wkf.repository.vcs, wkf.repository.name, 50));
      this.sourceBranches = branches.map(b => b.display_id);
      const selectedSourceBranch = this.params.ref ?? null;
      if (selectedSourceBranch && branches.findIndex(b => `refs/heads/${b.display_id}` === selectedSourceBranch) !== -1) {
        this.validateForm.controls.sourceBranch.setValue(selectedSourceBranch.replace('refs/heads/', ''));
      } else {
        this.validateForm.controls.sourceBranch.setValue(branches.find(b => b.default).display_id);
      }
    } else {
      this.validateForm.controls.sourceRepository.reset();
      this.validateForm.controls.sourceBranch.reset();
      this.validateForm.controls.sourceRepository.clearValidators();
      this.validateForm.controls.sourceBranch.clearValidators();
    }
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
    const splitted = this.splitRepository(this.validateForm.controls.repository.value);
    let req = <V2WorkflowRunManualRequest>{
      branch: this.validateForm.value.sourceBranch ?? this.validateForm.value.branch,
    };
    if (this.validateForm.value.sourceBranch) { req.workflow_branch = this.validateForm.value.branch; }
    const resp = await lastValueFrom(this._workflowRunService.start(this.project.key, splitted.vcs, splitted.repo, this.validateForm.value.workflow, req));
    this._messageService.success('Workflow run started', { nzDuration: 2000 });

    // Wait for workflow run to start
    let retry = 0
    while (retry < 90) {
      try {
        this.event = await lastValueFrom(this._projectService.getRepositoryEvent(this.project.key, splitted.vcs, splitted.repo, resp.hook_event_uuid));
        this._cd.markForCheck();
        if (this.event.status === HookEventWorkflowStatus.Done || this.event.status === HookEventWorkflowStatus.Error || this.event.status === HookEventWorkflowStatus.Skipped) {
          break;
        }
      } catch (e) { }
      await (new Promise(resolve => setTimeout(resolve, 1000)));
      retry++;
    }
  }

  clearForm(): void {
    this.event = null;
    this._cd.markForCheck();
  }
}