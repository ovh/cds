import {
    ChangeDetectionStrategy,
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
import { ApplicationsState } from 'app/store/applications.state';
import { AddPipeline } from 'app/store/pipelines.action';
import { PipelinesState } from 'app/store/pipelines.state';
import { AddEnvironmentInProject } from 'app/store/project.action';
import { ProjectState, ProjectStateModel } from 'app/store/project.state';
import cloneDeep from 'lodash-es/cloneDeep';
import { Observable, of as observableOf } from 'rxjs';
import { finalize, first, flatMap, map } from 'rxjs/operators';

@Component({
    selector: 'app-workflow-node-add-wizard',
    templateUrl: './node.wizard.html',
    styleUrls: ['./node.wizard.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class WorkflowNodeAddWizardComponent implements OnInit {

  @Input('project')
  set project(data: Project) {
    this._project = cloneDeep(data);
  }
  get project(): Project {
    return this._project;
  }
  @Input() hideCancel: boolean;
  @Input() hideNext: boolean;
  @Input() loading: boolean;
  @Output() nodeCreated: EventEmitter<WNode> = new EventEmitter<WNode>();
  @Output() pipelineSectionChanged: EventEmitter<string> = new EventEmitter<string>();

  _project: Project;
  node: WNode = new WNode();

  applicationsName: IdName[] = [];
  environmentsName: IdName[] = [];

  // Pipeline section
  set createNewPipeline(data: boolean) {
    this._createNewPipeline = data;
    if (data) {
      this.newPipeline = new Pipeline();
      this.node.context.pipeline_id = null;
    }
  }
  get createNewPipeline() {
    return this._createNewPipeline;
  }
  errorPipelineNamePattern = false;
  loadingCreatePipeline = false;
  newPipeline: Pipeline = new Pipeline();
  set pipelineSection(data: 'pipeline' | 'application' | 'environment' | 'integration') {
    this._pipelineSection = data;
    this.pipelineSectionChanged.emit(data);
  }
  get pipelineSection() {
    return this._pipelineSection;
  }
  _pipelineSection: 'pipeline' | 'application' | 'environment' | 'integration' = 'pipeline';
  _createNewPipeline = false;

  // Application details
  set createNewApplication(data: boolean) {
    this._createNewApplication = data;
    if (data) {
      this.newApplication = new Application();
      this.node.context.application_id = null;
    }
  }
  get createNewApplication() {
    return this._createNewApplication;
  }
  errorApplicationNamePattern = false;
  loadingCreateApplication = false;
  newApplication: Application = new Application();
  _createNewApplication = false;

  // Environment details
  set createNewEnvironment(data: boolean) {
    this._createNewEnvironment = data;
    if (data) {
      this.newEnvironment = new Environment();
      this.node.context.environment_id = null;
    }
  }
  get createNewEnvironment() {
    return this._createNewEnvironment;
  }
  loadingCreateEnvironment = false;
  newEnvironment: Environment = new Environment();
  _createNewEnvironment = false;
  integrations: IdName[] = [];
  loadingIntegrations = false;

  constructor(
    private _router: Router,
    private _translate: TranslateService,
    private _toast: ToastService,
    private _appService: ApplicationService,
    private store: Store
  ) {

  }

  ngOnInit() {
    if (!this.project.pipeline_names || !this.project.pipeline_names.length) {
      this.createNewPipeline = true;
    }
    if (!this.project.application_names || !this.project.application_names.length) {
      this.createNewApplication = true;
    }
    if (!this.project.environments || !this.project.environments.length) {
      this.createNewEnvironment = true;
    }

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
      }),
      flatMap(() => this.store.selectOnce(PipelinesState.selectPipeline(this.project.key, this.newPipeline.name))),
      map((pip) => {
        this._toast.success('', this._translate.instant('pipeline_added'));
        this.node.context.pipeline_id = pip.id;
        this.pipelineSection = 'application';
        return pip
      }));
  }

  selectOrCreatePipeline(): Observable<string> {
    if (this.createNewPipeline) {
      return this.createPipeline().pipe(
        map(() => 'application'));
    }
    this.pipelineSection = 'application';
    return observableOf('application');
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
      finalize(() => this.loadingCreateApplication = false),
      flatMap(() => this.store.selectOnce(ApplicationsState.selectApplication(this.project.key, this.newApplication.name))),
      map((app) => {
        this._toast.success('', this._translate.instant('application_created'));
        this.node.context.application_id = app.id;
        this.pipelineSection = 'environment';
        return app;
      })
    );
  }

  selectOrCreateApplication(): Observable<string> {
    if (this.createNewApplication && this.newApplication.name) {
      return this.createApplication().pipe(
        map(() => 'environment'));
    }
    this.getIntegrations();
    this.pipelineSection = 'environment';
    return observableOf('environment');
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
      finalize(() => this.loadingIntegrations = false)
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
        })
        if (this.integrations.length) {
          this.integrations.unshift(new IdName());
        }
      }
    );
  }

  createEnvironment(): Observable<Project> {
    this.loadingCreateEnvironment = true;
    return this.store.dispatch(new AddEnvironmentInProject({
      projectKey: this.project.key,
      environment: this.newEnvironment
    })).pipe(
      finalize(() => this.loadingCreateEnvironment = false),
      flatMap(() => this.store.selectOnce(ProjectState)),
      map((projState: ProjectStateModel) => {
        let proj = projState.project;
        this._toast.success('', this._translate.instant('environment_created'));
        this.node.context.environment_id = proj.environments.find((env) => env.name === this.newEnvironment.name).id;
        if (!this.node.context.application_id) {
          this.createNode();
        } else {
          this.pipelineSection = 'integration';
        }
        return proj;
      })
    );
  }

  selectOrCreateEnvironment() {
    if (this.createNewEnvironment && this.newEnvironment.name) {
      return this.createEnvironment().pipe(
        map(() => 'integration'));
    }
    let noIntegrationsAvailable = !this.loadingIntegrations && (!this.integrations || !this.integrations.length);
    if (!this.node.context.application_id || noIntegrationsAvailable) {
      this.createNode();
      return observableOf('done');
    }
    this.pipelineSection = 'integration';
    return observableOf('integration');
  }

  selectOrCreateIntegration() {
    this.createNode();
    return observableOf('done');
  }

  public goToNextSection(): Observable<string> {
    switch (this.pipelineSection) {
      case 'pipeline':
        return this.selectOrCreatePipeline();
      case 'application':
        return this.selectOrCreateApplication();
      case 'environment':
        return this.selectOrCreateEnvironment();
      case 'integration':
        return this.selectOrCreateIntegration();
    }
  }
}
