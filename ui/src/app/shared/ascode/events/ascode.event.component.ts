import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input } from '@angular/core';
import { AsCodeEvents } from 'app/model/ascode.model';
import { Project } from 'app/model/project.model';
import { Workflow } from 'app/model/workflow.model';
import { AscodeService } from 'app/service/ascode/ascode.service';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-ascode-event',
    templateUrl: './ascode.event.html',
    styleUrls: ['./ascode.event.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class AsCodeEventComponent {
    @Input() events: Array<AsCodeEvents>;
    @Input() project: Project;
    @Input() workflow: Workflow;

    loadingPopupButton = false;

    constructor(
        private _ascodeService: AscodeService,
        private _cd: ChangeDetectorRef
    ) { }

    resyncEvents(): void {
        this.loadingPopupButton = true;
        this._ascodeService.resyncPRAsCode(this.project.key, this.workflow.name)
            .pipe(finalize(() => {
                this.loadingPopupButton = false;
                this._cd.markForCheck();
            })).subscribe();
    }
}
