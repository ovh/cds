import {Component} from '@angular/core';
import {Router} from '@angular/router';
import {AuthentificationStore} from '../../../../service/auth/authentification.store';
import {ModelPattern} from '../../../../model/worker-model.model';
import {WorkerModelService} from '../../../../service/worker-model/worker-model.service';
import {ToastService} from '../../../../shared/toast/ToastService';
import {TranslateService} from '@ngx-translate/core';
import {User} from '../../../../model/user.model';
import {finalize} from 'rxjs/operators';

@Component({
    selector: 'app-worker-model-pattern-add',
    templateUrl: './worker-model-pattern.add.html',
    styleUrls: ['./worker-model-pattern.add.scss']
})
export class WorkerModelPatternAddComponent {
    loading = false;
    addLoading = false;
    pattern: ModelPattern;
    workerModelTypes: Array<string>;
    currentUser: User;

    private workerModelPatternNamePattern: RegExp = new RegExp('^[a-zA-Z0-9._-]{1,}$');
    private workerModelPatternError = false;

    constructor(private _workerModelService: WorkerModelService,
                private _toast: ToastService,
                private _translate: TranslateService,
                private _router: Router,
                private _authentificationStore: AuthentificationStore) {
        this.currentUser = this._authentificationStore.getUser();
        this.loading = true;
        this._workerModelService.getWorkerModelTypes()
            .pipe(finalize(() => this.loading = false))
            .subscribe( wmt => this.workerModelTypes = wmt);
        this.pattern = new ModelPattern();
    }

    clickSaveButton(): void {
      if (this.addLoading || !this.pattern || !this.pattern.name) {
          return;
      }

      if (!this.workerModelPatternNamePattern.test(this.pattern.name)) {
          this.workerModelPatternError = true;
          return;
      }

      this.addLoading = true;
      this._workerModelService.createWorkerModelPattern(this.pattern)
          .pipe(finalize(() => this.addLoading = false))
          .subscribe((pattern) => {
              this._toast.success('', this._translate.instant('worker_model_pattern_saved'));
              this._router.navigate(['admin', 'worker-model-pattern', pattern.type, pattern.name]);
          });
    }
}
