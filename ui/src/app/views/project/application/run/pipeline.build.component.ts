import {Component, OnDestroy, NgZone} from '@angular/core';
import {ActivatedRoute, Router} from '@angular/router';
import {Application} from '../../../../model/application.model';
import {Project} from '../../../../model/project.model';
import {Pipeline, PipelineBuild} from '../../../../model/pipeline.model';
import {Subscription} from 'rxjs/Rx';
import {environment} from '../../../../../environments/environment';
import {AuthentificationStore} from '../../../../service/auth/authentification.store';
import {CDSWorker} from '../../../../shared/worker/worker';
import {ApplicationPipelineService} from '../../../../service/application/pipeline/application.pipeline.service';

declare var Duration: any;

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
    duration: string;

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
                private _router: Router, private _appPipService: ApplicationPipelineService) {
        this.zone = new NgZone({enableLongStackTrace: false});

        // Get from pipeline resolver
        this._activatedRoute.data.subscribe(data => {
            this.pipeline = data['pipeline'];
            this.application = data['application'];
            this.project = data['project'];
        });

        this.envName = this._activatedRoute.snapshot.queryParams['envName'];
        if (this._activatedRoute.snapshot.queryParams['tab']) {
            this.selectedTab = this._activatedRoute.snapshot.queryParams['tab'];
        } else {
            this.selectedTab = 'pipeline';
        }

        this._activatedRoute.queryParams.subscribe( q => {
            this.envName = q['envName'];
            if (q['tab']) {
                this.selectedTab = q['tab'];
            } else {
                this.selectedTab = 'pipeline';
            }
        });
        this._activatedRoute.params.subscribe(params => {
            let buildNumber = params['buildNumber'];

            if (buildNumber && this.envName) {
                this.currentBuildNumber = Number(buildNumber);

                if (this.workerSubscription) {
                    this.workerSubscription.unsubscribe();
                }
                this.worker = new CDSWorker('./assets/worker/shared/runpipeline.js', './assets/worker/web/runpipeline.js');
                this.worker.start({
                    user: this._authStore.getUser(),
                    api: environment.apiURL,
                    key: this.project.key,
                    appName: this.application.name,
                    pipName: this.pipeline.name,
                    envName: this.envName,
                    buildNumber: buildNumber
                });

                this.worker.response().subscribe( msg => {
                    if (msg.data) {
                        let build: PipelineBuild = JSON.parse(msg.data);
                        this.zone.run(() => {
                            this.currentBuild = build;

                            if (this.currentBuild.status !== 'Building') {
                                this.duration = (new Duration((new Date(this.currentBuild.done)).getTime() - new Date(this.currentBuild.start).getTime())).toString();
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
        });
    }

    ngOnDestroy(): void {
        if (this.worker) {
            this.worker.updateWorker('unsubscribe', {});
        }

        if (this.workerSubscription) {
            this.workerSubscription.unsubscribe();
        }
    }

    showTab(tab: string): void {
        this._router.navigateByUrl( '/project/' + this.project.key +
            '/application/' + this.application.name +
            '/pipeline/' + this.pipeline.name +
            '/build/' + this.currentBuildNumber +
            '?envName=' + this.envName + '&tab=' + tab);
    }

    loadHistory(pb: PipelineBuild): void {
        let env = '';
        if (pb.environment && pb.environment.name) {
            env = pb.environment.name;
        }
        this._appPipService.buildHistory(this.project.key, pb.application.name, pb.pipeline.name,
            env, 50, '', pb.trigger.vcs_branch, false).subscribe( pbs => {
            this.histories = pbs;
            this.nbHistory = this.histories.length;
        });
    }
}
