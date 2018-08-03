
import {Component, EventEmitter, Input, OnInit, Output} from '@angular/core';
import {Router} from '@angular/router';
import {TranslateService} from '@ngx-translate/core';
import {cloneDeep} from 'lodash';
import {Observable, of as observableOf} from 'rxjs';
import {finalize, first, map} from 'rxjs/operators';
import {Application} from '../../../../model/application.model';
import {Environment} from '../../../../model/environment.model';
import {Pipeline} from '../../../../model/pipeline.model';
import {IdName, Project} from '../../../../model/project.model';
import {WorkflowNode} from '../../../../model/workflow.model';
import {ApplicationStore} from '../../../../service/application/application.store';
import {PipelineStore} from '../../../../service/pipeline/pipeline.store';
import {ProjectStore} from '../../../../service/project/project.store';
import {ToastService} from '../../../../shared/toast/ToastService';

@Component({
    selector: 'app-workflow-node-add-wizard',
    templateUrl: './node.wizard.html',
    styleUrls: ['./node.wizard.scss']
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
    @Output() nodeCreated: EventEmitter<WorkflowNode> = new EventEmitter<WorkflowNode>();
    @Output() pipelineSectionChanged: EventEmitter<string> = new EventEmitter<string>();

    _project: Project;
    node: WorkflowNode = new WorkflowNode();
    loading = false;
    applicationsName: IdName[] = [];
    environmentsName: IdName[] = [];

    // Pipeline section
    set createNewPipeline(data: boolean) {
      this._createNewPipeline = data;
      if (data) {
        this.newPipeline = new Pipeline();
        this.node.pipeline_id = null;
      }
    }
    get createNewPipeline() {
      return this._createNewPipeline;
    }
    errorPipelineNamePattern = false;
    loadingCreatePipeline = false;
    newPipeline: Pipeline = new Pipeline();
    set pipelineSection(data: 'pipeline'|'application'|'environment'|'platform') {
        this._pipelineSection = data;
        this.pipelineSectionChanged.emit(data);
    }
    get pipelineSection() {
        return this._pipelineSection;
    }
    _pipelineSection: 'pipeline'|'application'|'environment'|'platform' = 'pipeline';
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
    platforms: IdName[] = [];
    loadingPlatforms = false;

    constructor(private _router: Router,
                private _translate: TranslateService,
                private _toast: ToastService,
                private _pipStore: PipelineStore,
                private _appStore: ApplicationStore,
                private _projectStore: ProjectStore) {

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
        this._router.navigate(['/project', this.project.key], {queryParams: {tab: 'workflows'}});
    }

    createNode(): void {
        this.loading = true;
        if (this.node.pipeline_id) {
            this.node.pipeline_id = Number(this.node.pipeline_id);
        }
        if (this.node.context.application_id) {
            this.node.context.application_id = Number(this.node.context.application_id);
        }
        if (this.node.context.environment_id) {
            this.node.context.environment_id = Number(this.node.context.environment_id);
        }
        if (this.node.context.project_platform_id) {
            this.node.context.project_platform_id = Number(this.node.context.project_platform_id);
        }

        this.nodeCreated.emit(this.node);
    }

    createPipeline(): Observable<Pipeline> {
        if (!Pipeline.checkName(this.newPipeline.name)) {
          this.errorPipelineNamePattern = true;
          return observableOf(null);
        }

        this.loadingCreatePipeline = true;
        this.newPipeline.type = 'deployment';
        return this._pipStore.createPipeline(this.project.key, this.newPipeline)
            .pipe(
              first(),
              finalize(() => this.loadingCreatePipeline = false)
            ).pipe(
            map((pip) => {
                this._toast.success('', this._translate.instant('pipeline_added'));
                this.node.pipeline_id = pip.id;
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
      return this._appStore.createApplication(this.project.key, this.newApplication)
          .pipe(
            first(),
            finalize(() => this.loadingCreateApplication = false)
          ).pipe(
          map((app) => {
              this._toast.success('', this._translate.instant('application_created'));
              this.node.context.application_id = app.id;
              this.pipelineSection = 'environment';
              return app;
          }));
    }

    selectOrCreateApplication(): Observable<string> {
      if (this.createNewApplication && this.newApplication.name) {
        return this.createApplication().pipe(
          map(() => 'environment'));
      }
      this.getPlatforms();
      this.pipelineSection = 'environment';
      return observableOf('environment');
    }

    getPlatforms() {
      this.loadingPlatforms = true;
      let app = this.project.application_names.find((a) => a.id === this.node.context.application_id);
      this._appStore.getDeploymentStrategies(this.project.key, app.name).pipe(
          first(),
          finalize(() => this.loadingPlatforms = false)
      ).subscribe(
          (data) => {
              this.platforms = [new IdName()];
              let pfNames = Object.keys(data);
              pfNames.forEach(s => {
                  let pf = this.project.platforms.find(p => p.name === s);
                  if (pf) {
                      let idName = new IdName();
                      idName.id = pf.id;
                      idName.name = pf.name;
                      this.platforms.push(idName);
                  }
              })
          }
      );
    }

    createEnvironment(): Observable<Project>  {
      this.loadingCreateEnvironment = true;
      return this._projectStore.addProjectEnvironment(this.project.key, this.newEnvironment)
          .pipe(
            first(),
            finalize(() => this.loadingCreateEnvironment = false)
          ).pipe(
          map((proj) => {
              this._toast.success('', this._translate.instant('environment_created'));
              this.node.context.environment_id = proj.environments.find((env) => env.name === this.newEnvironment.name).id;
              if (!this.node.context.application_id) {
                  this.createNode();
              } else {
                  this.pipelineSection = 'platform';
              }
              return proj;
          }));
    }

    selectOrCreateEnvironment() {
      if (this.createNewEnvironment && this.newEnvironment.name) {
        return this.createEnvironment().pipe(
          map(() => 'platform'));
      }
      if (!this.node.context.application_id) {
          this.createNode();
          return observableOf('done');
      }
      this.pipelineSection = 'platform';
      return observableOf('platform');
    }

    selectOrCreatePlatform() {
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
        case 'platform':
          return this.selectOrCreatePlatform();
      }
    }
}
