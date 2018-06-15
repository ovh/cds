import {Component, Input} from '@angular/core';
import {TranslateService} from '@ngx-translate/core';
import {cloneDeep} from 'lodash';
import {finalize} from 'rxjs/operators';
import {Project} from '../../../../model/project.model';
import {Warning} from '../../../../model/warning.model';
import {WarningStore} from '../../../../service/warning/warning.store';
import {Table} from '../../../../shared/table/table';
import {ToastService} from '../../../../shared/toast/ToastService';

@Component({
    selector: 'app-project-warnings',
    templateUrl: './project.warnings.html',
    styleUrls: ['./project.warnings.scss']
})
export class ProjectWarningsComponent extends Table {

    @Input() project: Project;
    @Input() warnings: Array<Warning>;

    constructor(private _warningStore: WarningStore, private _translate: TranslateService, private _toastService: ToastService) {
        super();
    }

    getData(): any[] {
        if (this.warnings) {
            let warningsDisplayed = this.warnings.filter(w => !w.ignored);
            warningsDisplayed.push(...this.warnings.filter(w => w.ignored));
            return warningsDisplayed;
        }
        return null;
    }

    updateWarning(w: Warning): void {
        w.loading = true;
        let warning = cloneDeep(w);
        warning.ignored = !warning.ignored;
        this._warningStore.updateWarning(this.project.key, warning).pipe(finalize(() => w.loading = false)).subscribe(() => {
            this._toastService.success('', this._translate.instant('warning_updated'));
        });
    }
}
