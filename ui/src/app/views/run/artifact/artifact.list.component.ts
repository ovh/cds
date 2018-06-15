import {Component, Input, NgZone} from '@angular/core';
import {environment} from '../../../../environments/environment';
import {Artifact} from '../../../model/artifact.model';
import {Table} from '../../../shared/table/table';

@Component({
    selector: 'app-artifact-list',
    templateUrl: './artifact.list.html',
    styleUrls: ['./artifact.list.scss']
})
export class ArtifactListComponent extends Table {

    @Input() artifacts: Array<Artifact>;

    // Allow angular update from work started outside angular context
    zone: NgZone;

    constructor() {
        super();
        this.zone = new NgZone({enableLongStackTrace: false});
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
