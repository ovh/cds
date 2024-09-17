import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnInit } from "@angular/core";
import { FormBuilder, FormControl, FormGroup, Validators } from "@angular/forms";
import { Store } from "@ngxs/store";
import { Project, ProjectRepository } from "app/model/project.model";
import { VCSProject } from "app/model/vcs.model";
import { ProjectService } from "app/service/project/project.service";
import { ProjectState } from "app/store/project.state";
import { lastValueFrom } from "rxjs";
import { NzMessageService } from "ng-zorro-antd/message";
import { NzDrawerRef } from "ng-zorro-antd/drawer";
import { Analysis, AnalysisRequest, AnalysisResponse, StatusAnalyzeError, StatusAnalyzeSkipped, StatusAnalyzeSucceed } from "app/model/analysis.model";
import { AnalysisService } from "app/service/analysis/analysis.service";

export class ProjectV2TriggerAnalysisComponentParams {
  repository: string;
}

@Component({
  selector: 'app-projectv2-trigger-analysis',
  templateUrl: './trigger-analysis.html',
  styleUrls: ['./trigger-analysis.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class ProjectV2TriggerAnalysisComponent implements OnInit {
  @Input() params: ProjectV2TriggerAnalysisComponentParams;

  project: Project;
  vcss: Array<VCSProject> = [];
  repositories: { [vcs: string]: Array<ProjectRepository> } = {};
  branches: Array<string> = [];
  sourceBranches: Array<string> = [];
  workflows: Array<string> = [];
  validateForm: FormGroup<{
    repository: FormControl<string | null>;
    branch: FormControl<string | null>;
  }>;
  response: AnalysisResponse;
  analysis: Analysis;

  constructor(
    private _drawerRef: NzDrawerRef<string>,
    private _messageService: NzMessageService,
    private _store: Store,
    private _projectService: ProjectService,
    private _fb: FormBuilder,
    private _cd: ChangeDetectorRef,
    private _analysisService: AnalysisService,
  ) {
    this.project = this._store.selectSnapshot(ProjectState.projectSnapshot);
    this.validateForm = this._fb.group({
      repository: this._fb.control<string | null>(null, Validators.required),
      branch: this._fb.control<string | null>(null, Validators.required),
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
    let selectedRepository = this.params.repository;
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
    this.validateForm.controls.branch.setValue(branches.find(b => b.default).display_id);
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
    let req = <AnalysisRequest>{
      projectKey: this.project.key,
      vcsName: splitted.vcs,
      repoName: splitted.repo,
      ref: 'refs/heads/' + this.validateForm.value.branch,
    };

    this.response = await lastValueFrom(this._analysisService.triggerAnalysis(req));
    this._cd.markForCheck();
    this._messageService.success('Analysis triggered', { nzDuration: 2000 });

    // Wait for analysis to be over
    let retry = 0
    while (retry < 90) {
      try {
        this.analysis = await lastValueFrom(this._analysisService.getAnalysis(this.project.key, splitted.vcs, splitted.repo, this.response.analysis_id));
        if (this.analysis.status === StatusAnalyzeSucceed || this.analysis.status === StatusAnalyzeError || this.analysis.status === StatusAnalyzeSkipped) {
          this.response = null;
          this._cd.markForCheck();
          break;
        }
      } catch (e) { }
      await (new Promise(resolve => setTimeout(resolve, 1000)));
      retry++;
    }

  }

  clearForm(): void {
    this.response = null;
    this.analysis = null;
    this.validateForm.enable();
    this._cd.markForCheck();
  }
}