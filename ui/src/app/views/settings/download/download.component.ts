import {Component} from '@angular/core';
import {Download} from 'app/model/download.model';
import {environment} from '../../../../environments/environment';
import {DownloadService} from '../../../service/download/download.service';

@Component({
    selector: 'app-download',
    templateUrl: './download.html',
    styleUrls: ['./download.scss']
})
export class DownloadComponent {

    downloads: Array<Download>;
    loading = false;
    apiURL: string;

    constructor(private _downloadService: DownloadService) {
        this.loading = true;
        this._downloadService.getDownloads()
            .subscribe(r => {
                this.downloads = r;
                this.apiURL = environment.apiURL;
                this.loading = false;
            });
    }
}
