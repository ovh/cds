import { Component, Input } from '@angular/core';
import { User } from 'app/model/user.model';
import { WorkerModel } from 'app/model/worker-model.model';
import { WorkerModelService } from 'app/service/worker-model/worker-model.service';
import { SharedService } from 'app/shared/shared.service';
import { finalize } from 'rxjs/operators/finalize';

@Component({
    selector: 'app-worker-model-form',
    templateUrl: './worker-model.form.html',
    styleUrls: ['./worker-model.form.scss']
})
export class WorkerModelFormComponent {
    @Input() workerModel: WorkerModel;
    @Input() currentUser: User;
    @Input() loading: boolean;

    asCode = false;
    loadingAsCode = false;
    workerModelAsCode: string;

    constructor(
        private _sharedService: SharedService,
        private _workerModelService: WorkerModelService
    ) { }

    getDescriptionHeight(): number {
        return this._sharedService.getTextAreaheight(this.workerModel.description);
    }

    loadAsCode() {
        if (this.asCode) {
            return;
        }
        this.asCode = true;
        if (!this.workerModel.id) {
            return;
        }
        this.loadingAsCode = true
        this._workerModelService.exportWorkerModel(this.workerModel.id)
            .pipe(finalize(() => this.loadingAsCode = false))
            .subscribe((wmStr) => this.workerModelAsCode = wmStr);
    }
}
