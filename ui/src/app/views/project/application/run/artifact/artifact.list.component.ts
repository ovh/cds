import {Component, Input, OnDestroy, OnInit, NgZone} from '@angular/core';
import {Table} from '../../../../../shared/table/table';
import {Artifact} from '../../../../../model/artifact.model';
import {CDSWorker} from '../../../../../shared/worker/worker';
import {Subscription} from 'rxjs/Rx';
import {environment} from '../../../../../../environments/environment.dev';

@Component({
    selector: 'app-artifact-list',
    templateUrl: './artifact.list.html',
    styleUrls: ['./artifact.list.scss']
})
export class ArtifactListComponent extends Table implements OnInit, OnDestroy {

    @Input() buildWorker: CDSWorker;
    artifacts: Array<Artifact>;

    workerSubscription: Subscription;

    // Allow angular update from work started outside angular context
    zone: NgZone;

    constructor() {
        super();
        this.zone = new NgZone({enableLongStackTrace: false});
    }

    ngOnInit(): void {
        this.workerSubscription = this.buildWorker.response().subscribe( msg => {
            if (msg.data) {
                this.zone.run(() => {
                    this.artifacts = JSON.parse(msg.data).artifacts;
                });
            }
        });
    }

    ngOnDestroy(): void {
        if (this.workerSubscription) {
            this.workerSubscription.unsubscribe();
        }
    }

    getData(): any[] {
        return this.artifacts;
    }

    getHumainFileSize(size: number): string {
        let i = Math.floor(Math.log(size) / Math.log(1024));
        let hSize = (size / Math.pow(1024, i)).toFixed(2);
        return hSize + ' ' + ['B', 'kB', 'MB', 'GB', 'TB'][i];
    }

    getUrl(a: Artifact): string {
        return environment.apiURL + '/artifact/' + a.download_hash;
    }
}
