import {Component, NgZone, OnDestroy} from '@angular/core';
import {ActivatedRoute, Router} from '@angular/router';
import {Subscription} from 'rxjs';
import {environment} from '../../../environments/environment';
import {Application} from '../../model/application.model';
import {Pipeline, PipelineBuild, PipelineStatus} from '../../model/pipeline.model';
import {Project} from '../../model/project.model';
import {ApplicationPipelineService} from '../../service/application/pipeline/application.pipeline.service';
import {AuthentificationStore} from '../../service/auth/authentification.store';
import {RouterService} from '../../service/router/router.service';
import {DurationService} from '../../shared/duration/duration.service';
import {CDSWorker} from '../../shared/worker/worker';

@Component({
    selector: 'app-pipeline-build',
    templateUrl: './pipeline.build.html',
    styleUrls: ['./pipeline.build.scss']
})
export class ApplicationPipelineBuildComponent implements OnDestroy {

    // Datas
    project: Project;
    application: Application;
    pipeline: Pipeline;
    currentBuildNumber: number;
    histories: Array<PipelineBuild>;
    envName: string;
    selectedTab: string;
    currentBuild: PipelineBuild;
    previousBuild: PipelineBuild;
    duration: string;
    branch: string;
    remote: string;
    appVersionFilter: number;

    // Allow angular update from work started outside angular context
    zone: NgZone;

    // Worker CDS that pull data
    worker: CDSWorker;

    // Worker subscription
    workerSubscription: Subscription;


    // tab datas
    nbTests = 0;
    nbArtifacts = 0;
    nbHistory = 0;


    constructor(private _activatedRoute: ActivatedRoute, private _authStore: AuthentificationStore,
                private _router: Router, private _appPipService: ApplicationPipelineService, private _durationService: DurationService,
                private _routerService: RouterService) {
        this.zone = new NgZone({enableLongStackTrace: false});

        // Get from pipeline resolver
        this._activatedRoute.data.subscribe(data => {
            this.pipeline = data['pipeline'];
            this.application = data['application'];
            this.project = data['project'];
        });

        this._activatedRoute.queryParams.subscribe(q => {
            this.envName = q['envName'];
            if (q['tab']) {
                this.selectedTab = q['tab'];
            } else {
                this.selectedTab = 'pipeline';
            }
            if (q['branch']) {
                this.branch = q['branch'];
            }
            if (q['version']) {
                this.appVersionFilter = q['version'];
            }
            if (q['ts'] && this.project && this.application && this.currentBuildNumber) {
                this.startWorker(this._activatedRoute);
                this.currentBuild = undefined;
                this.histories = undefined;
            }
            this.remote = q['remote'] || null;
        });
        // Current route param
        this._activatedRoute.params.subscribe(params => {
            let buildNumber = params['buildNumber'];
            if (buildNumber && this.envName) {
                this.currentBuildNumber = Number(buildNumber);
                this.startWorker(this._activatedRoute);
            }
        });
    }

    startWorker(_activatedRoute: ActivatedRoute): void {
        let paramSnap = this._routerService.getRouteSnapshotParams({}, _activatedRoute.snapshot);
        let querySnap = this._routerService.getRouteSnapshotQueryParams({}, _activatedRoute.snapshot);
        if (this.workerSubscription) {
            this.workerSubscription.unsubscribe();
        }
        if (this.worker) {
            this.worker.stop();
        }
        this.worker = new CDSWorker('./assets/worker/web/runpipeline.js');
        this.worker.start({
            user: this._authStore.getUser(),
            session: this._authStore.getSessionToken(),
            api: environment.apiURL,
            key: paramSnap['key'],
            appName: paramSnap['appName'],
            pipName: paramSnap['pipName'],
            envName: querySnap['envName'],
            remote: querySnap['remote'],
            buildNumber: paramSnap['buildNumber']
        });

        this.currentBuild = undefined;
        this.worker.response().subscribe(msg => {
            if (msg) {
                let build: PipelineBuild = JSON.parse(msg);
                this.zone.run(() => {
                    this.currentBuild = build;

                    if (this.currentBuild.status !== PipelineStatus.BUILDING) {
                        this.duration = this._durationService.duration(
                            new Date(this.currentBuild.start), new Date(this.currentBuild.done));
                    }

                    if (build.artifacts) {
                        if (build.artifacts.length !== this.nbArtifacts) {
                            this.nbArtifacts = build.artifacts.length;
                        }
                    }
                    if (build.tests) {
                        if (build.tests.total !== this.nbTests) {
                            this.nbTests = build.tests.total;
                        }
                    }
                    if (!this.histories) {
                        this.loadHistory(build);
                    }
                });
            }
        });
    }

    ngOnDestroy(): void {
        if (this.worker) {
            this.worker.stop();
        }

        if (this.workerSubscription) {
            this.workerSubscription.unsubscribe();
        }
    }

    showTab(tab: string): void {
        let url = '/project/' + this.project.key +
            '/application/' + this.application.name +
            '/pipeline/' + this.pipeline.name +
            '/build/' + this.currentBuildNumber +
            '?envName=' + this.envName + '&tab=' + tab;

        if (this.remote) {
            url += '&remote=' + this.remote;
        }
        if (this.branch) {
            url += '&branch=' + this.branch;
        }
        this._router.navigateByUrl(url);
    }

    loadHistory(pb: PipelineBuild): void {
        let env = '';
        if (pb.environment && pb.environment.name) {
            env = pb.environment.name;
        }
        this._appPipService.buildHistory(this.project.key, pb.application.name, pb.pipeline.name,
            env, 50, '', pb.trigger.vcs_branch, pb.trigger.vcs_remote).subscribe(pbs => {
            this.histories = pbs;
            this.nbHistory = this.histories.length;
        });

        this._appPipService.buildHistory(this.project.key, pb.application.name, pb.pipeline.name,
            env, 1, PipelineStatus.SUCCESS, pb.trigger.vcs_branch, pb.trigger.vcs_remote).subscribe(pbs => {
            if (pbs && pbs.length === 1) {
                this.previousBuild = pbs[0];
            }

        });
    }
}
