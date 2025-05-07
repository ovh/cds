import {
  ChangeDetectionStrategy, ChangeDetectorRef,
  Component,
  EventEmitter,
  Input,
  OnInit,
  Output
} from '@angular/core';
import { Router } from '@angular/router';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { Application } from 'app/model/application.model';
import { Environment } from 'app/model/environment.model';
import { Pipeline } from 'app/model/pipeline.model';
import { IdName, Project } from 'app/model/project.model';
import { WNode, WNodeType } from 'app/model/workflow.model';
import { ApplicationService } from 'app/service/application/application.service';
import { ToastService } from 'app/shared/toast/ToastService';
import { AddApplication } from 'app/store/applications.action';
import { ApplicationsState, ApplicationStateModel } from 'app/store/applications.state';
import { AddEnvironment } from 'app/store/environment.action';
import { AddPipeline } from 'app/store/pipelines.action';
import { PipelinesState, PipelinesStateModel } from 'app/store/pipelines.state';
import { ProjectState } from 'app/store/project.state';
import cloneDeep from 'lodash-es/cloneDeep';
import { Observable, of as observableOf } from 'rxjs';
import { filter, finalize, first, map, switchMap } from 'rxjs/operators';

@Component({
  selector: 'app-workflow-node-add-wizard',
  templateUrl: './node.wizard.html',
  styleUrls: ['./node.wizard.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class WorkflowNodeAddWizardComponent implements OnInit {

  @Input()
  set project(data: Project) {
    this._project = cloneDeep(data);
  }
  get project(): Project {
    return this._project;
  }
  @Input() display: boolean;
  @Input() hideCancel: boolean;
  @Input() hideNext: boolean;
  @Input() loading: boolean;
  @Output() nodeCreated: EventEmitter<WNode> = new EventEmitter<WNode>();
  @Output() pipelineSectionChanged: EventEmitter<number> = new EventEmitter<number>();

  _project: Project;
  node: WNode = new WNode();

  applicationsName: IdName[] = [];
  environmentsName: IdName[] = [];

  pipIndexTab: number;
  appIndexTab: number;
  envIndexTab: number;

  errorPipelineNamePattern = false;
  loadingCreatePipeline = false;
  newPipeline: Pipeline = new Pipeline();
  set pipelineSection(data: number) {
    this._pipelineSection = data;
    this.pipelineSectionChanged.emit(data);
  }
  get pipelineSection() {
    return this._pipelineSection;
  }
  _pipelineSection: number = 0;


  errorApplicationNamePattern = false;
  loadingCreateApplication = false;
  newApplication: Application = new Application();

  loadingCreateEnvironment = false;
  newEnvironment: Environment = new Environment();
  integrations: IdName[] = [];
  loadingIntegrations = false;
  createFork = false;

  constructor(
    private _router: Router,
    private _translate: TranslateService,
    private _toast: ToastService,
    private _appService: ApplicationService,
    private store: Store,
    private _cd: ChangeDetectorRef
  ) { }

  ngOnInit() {
    if (Array.isArray(this.project.application_names)) {
      let voidApp = new IdName();
      voidApp.id = 0;
      voidApp.name = ' ';
      this.applicationsName = [voidApp, ...this.project.application_names];
    }
    if (Array.isArray(this.project.environments)) {
      let voidEnv = new IdName();
      voidEnv.id = 0;
      voidEnv.name = ' ';
      this.environmentsName = [voidEnv, ...this.project.environments];
    }
  }

  goToProject(): void {
    this._router.navigate(['/project', this.project.key], { queryParams: { tab: 'workflows' } });
  }

  createNode(): void {
    if (this.node.context.pipeline_id) {
      this.node.type = WNodeType.PIPELINE;
      this.node.context.pipeline_id = Number(this.node.context.pipeline_id);
    } else {
      this.node.type = WNodeType.FORK;
    }
    if (this.node.context.application_id) {
      this.node.context.application_id = Number(this.node.context.application_id);
    }
    if (this.node.context.environment_id) {
      this.node.context.environment_id = Number(this.node.context.environment_id);
    }
    if (this.node.context.project_integration_id) {
      this.node.context.project_integration_id = Number(this.node.context.project_integration_id);
    }
    if (!this.node.ref) {
      this.node.ref = new Date().getTime().toString();
    }
    this.nodeCreated.emit(this.node);
  }

  createPipeline(): Observable<Pipeline> {
    if (!Pipeline.checkName(this.newPipeline.name)) {
      this.errorPipelineNamePattern = true;
      return observableOf(null);
    }

    this.loadingCreatePipeline = true;
    return this.store.dispatch(new AddPipeline({
      projectKey: this.project.key,
      pipeline: this.newPipeline
    })).pipe(
      finalize(() => {
        this.loadingCreatePipeline = false;
        this._cd.markForCheck();
      }),
      switchMap(() => this.store.selectOnce(PipelinesState.current)),
      map((pip: PipelinesStateModel) => {
        this._toast.success('', this._translate.instant('pipeline_added'));
        this.node.context.pipeline_id = pip.pipeline.id;
        this.pipelineSection = 1;
        return pip.pipeline;
      }));
  }

  selectOrCreatePipeline(): Observable<number> {
    if (this.pipIndexTab === 1) {
      return this.createPipeline().pipe(
        map(() => 1));
    }
    this.pipelineSection = 1;
    this._cd.markForCheck();
    return observableOf(1);
  }

  createApplication(): Observable<Application> {
    if (!Application.checkName(this.newApplication.name)) {
      this.errorApplicationNamePattern = true;
      return observableOf(null);
    }

    this.loadingCreateApplication = true;

    return this.store.dispatch(new AddApplication({
      projectKey: this.project.key,
      application: this.newApplication
    })).pipe(
      finalize(() => {
        this.loadingCreateApplication = false;
        this._cd.markForCheck();
      }),
      switchMap(() => this.store.selectOnce(ApplicationsState.current)),
      filter((s: ApplicationStateModel) => s.application != null && s.application.name === this.newApplication.name),
      map((s: ApplicationStateModel) => {
        this._toast.success('', this._translate.instant('application_created'));
        this.node.context.application_id = s.application.id;
        this.pipelineSection = 2;
        return s.application;
      })
    );
  }

  selectOrCreateApplication(): Observable<number> {
    if (this.appIndexTab && this.newApplication.name) {
      return this.createApplication().pipe(
        map(() => 2));
    }
    this.getIntegrations();
    this.pipelineSection = 2;
    this._cd.markForCheck();
    return observableOf(2);
  }

  getIntegrations() {
    this.loadingIntegrations = true;
    if (!this.node.context.application_id) {
      this.loadingIntegrations = false;
      return;
    }
    let app = this.project.application_names.find((a) => a.id === Number(this.node.context.application_id));
    if (!app) {
      this.loadingIntegrations = false;
      return;
    }

    // TODO: to update with store
    this._appService.getDeploymentStrategies(this.project.key, app.name).pipe(
      first(),
      finalize(() => {
        this.loadingIntegrations = false;
        this._cd.markForCheck();
      })
    ).subscribe(
      (data) => {
        this.integrations = [];
        let pfNames = Object.keys(data);
        pfNames.forEach(s => {
          let pf = this.project.integrations.find(p => p.name === s);
          if (pf) {
            let idName = new IdName();
            idName.id = pf.id;
            idName.name = pf.name;
            this.integrations.push(idName);
          }
        });
      }
    );
  }

  createEnvironment(): Observable<Project> {
    this.loadingCreateEnvironment = true;
    return this.store.dispatch(new AddEnvironment({
      projectKey: this.project.key,
      environment: this.newEnvironment
    })).pipe(
      finalize(() => {
        this.loadingCreateEnvironment = false;
        this._cd.markForCheck();
      }),
      switchMap(() => this.store.selectOnce(ProjectState.projectSnapshot)),
      map((proj: Project) => {
        this._toast.success('', this._translate.instant('environment_created'));
        this.node.context.environment_id = proj.environments.find((env) => env.name === this.newEnvironment.name).id;
        if (!this.node.context.application_id) {
          this.createNode();
        } else {
          this.pipelineSection = 3;
        }
        return proj;
      })
    );
  }

  selectOrCreateEnvironment() {
    if (this.envIndexTab === 1 && this.newEnvironment.name) {
      return this.createEnvironment().pipe(
        map(() => 3));
    }
    let noIntegrationsAvailable = !this.loadingIntegrations && (!this.integrations || !this.integrations.length);
    if (!this.node.context.application_id || noIntegrationsAvailable) {
      this.createNode();
      return observableOf(4);
    }
    this.pipelineSection = 3;
    this._cd.markForCheck();
    return observableOf(3);
  }

  selectOrCreateIntegration() {
    this.createNode();
    return observableOf(4);
  }

  public goToPreviousSection() {
    this.pipelineSection--;
    this._cd.markForCheck();

  }

  public goToNextSection(): Observable<number> {
    switch (this.pipelineSection) {
      case 0:
        return this.selectOrCreatePipeline();
      case 1:
        return this.selectOrCreateApplication();
      case 2:
        return this.selectOrCreateEnvironment();
      case 3:
        return this.selectOrCreateIntegration();
    }
  }
}
