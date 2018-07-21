import {Component, OnInit} from '@angular/core';
import {ActivatedRoute, Router} from '@angular/router';
import {TranslateService} from '@ngx-translate/core';
import {omit} from 'lodash';
import {finalize} from 'rxjs/operators';
import {User} from '../../../../model/user.model';
import {ModelPattern} from '../../../../model/worker-model.model';
import {AuthentificationStore} from '../../../../service/auth/authentification.store';
import {WorkerModelService} from '../../../../service/worker-model/worker-model.service';
import {ToastService} from '../../../../shared/toast/ToastService';

@Component({
    selector: 'app-worker-model-pattern-edit',
    templateUrl: './worker-model-pattern.edit.html',
    styleUrls: ['./worker-model-pattern.edit.scss']
})
export class WorkerModelPatternEditComponent implements OnInit {
    loading = false;
    editLoading = false;
    pattern: ModelPattern;
    workerModelTypes: Array<string>;
    currentUser: User;
    envNames: Array<string> = [];
    newEnvName: string;
    newEnvValue: string;

    private workerModelPatternNamePattern: RegExp = new RegExp('^[a-zA-Z0-9._-]{1,}$');
    workerModelPatternError = false;

    constructor(private _workerModelService: WorkerModelService,
                private _toast: ToastService, private _translate: TranslateService,
                private _route: ActivatedRoute, private _router: Router,
                private _authentificationStore: AuthentificationStore) {
        this.currentUser = this._authentificationStore.getUser();
        this.loading = true;
        this._workerModelService.getWorkerModelTypes()
            .pipe(finalize(() => this.loading = false))
            .subscribe( wmt => this.workerModelTypes = wmt);
        this.pattern = new ModelPattern();
    }

    ngOnInit() {
        this.loading = true;
        this._workerModelService.getWorkerModelPattern(this._route.snapshot.params['type'], this._route.snapshot.params['name'])
            .pipe(finalize(() => this.loading = false))
            .subscribe(
                (pattern) => {
                    if (pattern.model.envs) {
                        this.envNames = Object.keys(pattern.model.envs);
                    }
                    this.pattern = pattern;
                },
                () => this._router.navigate(['admin', 'worker-model-pattern'])
            );
    }

    clickSaveButton(): void {
      if (!this.pattern || !this.pattern.name) {
          return;
      }

      if (!this.workerModelPatternNamePattern.test(this.pattern.name)) {
          this.workerModelPatternError = true;
          return;
      }

      this.editLoading = true;
      this._workerModelService
          .updateWorkerModelPattern(this._route.snapshot.params['type'], this._route.snapshot.params['name'], this.pattern)
          .pipe(finalize(() => this.editLoading = false))
          .subscribe((pattern) => {
              this._toast.success('', this._translate.instant('worker_model_pattern_saved'));
              this._router.navigate(['admin', 'worker-model-pattern', pattern.type, pattern.name]);
          });
    }

    delete() {
        if (this.editLoading) {
            return;
        }

        this.editLoading = true;
        this._workerModelService
            .deleteWorkerModelPattern(this._route.snapshot.params['type'], this._route.snapshot.params['name'])
            .pipe(finalize(() => this.editLoading = false))
            .subscribe(() => {
                this._toast.success('', this._translate.instant('worker_model_pattern_deleted'));
                this._router.navigate(['admin', 'worker-model-pattern']);
            });
    }

    addEnv(newEnvName: string, newEnvValue: string) {
          if (!newEnvName) {
              return;
          }
          if (!this.pattern.model.envs) {
              this.pattern.model.envs = {};
          }
          this.pattern.model.envs[newEnvName] = newEnvValue;
          this.envNames.push(newEnvName);
          this.newEnvName = '';
          this.newEnvValue = '';
      }

      deleteEnv(envName: string, index: number) {
          this.envNames.splice(index, 1);
          this.pattern.model.envs = omit(this.pattern.model.envs, envName);
      }
}
