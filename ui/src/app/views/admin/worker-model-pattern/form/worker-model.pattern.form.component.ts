import { ChangeDetectionStrategy, ChangeDetectorRef, Component, EventEmitter, Input, Output } from '@angular/core';
import { Store } from '@ngxs/store';
import { ModelPattern } from 'app/model/worker-model.model';
import { WorkerModelService } from 'app/service/worker-model/worker-model.service';
import omit from 'lodash-es/omit';
import { finalize } from 'rxjs/operators';

@Component({
    selector: 'app-worker-model-pattern-form',
    templateUrl: './worker-model-pattern.form.html',
    styleUrls: ['./worker-model-pattern.form.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class WorkerModelPatternFormComponent {
    @Input() loading: boolean;

    @Input() set pattern(p: ModelPattern) {
        if (!p) {
            return;
        }

        this._pattern = p;

        this.envNames = [];
        if (this._pattern.model.envs) {
            this.envNames = Object.keys(this._pattern.model.envs);
        }

        this._cd.markForCheck();
    }
    get pattern(): ModelPattern {
        return this._pattern;
    }
    _pattern: ModelPattern;

    @Output() save = new EventEmitter<ModelPattern>();
    @Output() delete = new EventEmitter();

    loadingPatterns: boolean;
    workerModelTypes: Array<string>;
    newEnvName: string;
    newEnvValue: string;
    envNames: Array<string>;

    constructor(
        private _workerModelService: WorkerModelService,
        private _store: Store,
        private _cd: ChangeDetectorRef
    ) {

        this.loadingPatterns = true;
        this._cd.markForCheck();
        this._workerModelService.getTypes()
            .pipe(finalize(() => {
                this.loadingPatterns = false;
                this._cd.markForCheck();
            }))
            .subscribe(wmt => this.workerModelTypes = wmt);
    }

    clickSaveButton(): void {
        if (this.loading || !this._pattern || !this._pattern.name) {
            return;
        }

        this.save.emit(this._pattern);
    }

    clickDeleteButton(): void {
        if (this.loading) {
            return;
        }

        this.delete.emit();
    }

    clickAddEnv() {
        if (!this.newEnvName || !this.newEnvValue) {
            return;
        }
        if (!this._pattern.model.envs) {
            this._pattern.model.envs = {};
        }
        this._pattern.model.envs[this.newEnvName] = this.newEnvValue;
        this.envNames = Object.keys(this._pattern.model.envs);
        this.newEnvName = '';
        this.newEnvValue = '';
        this._cd.markForCheck();
    }

    clickDeleteEnv(envName: string) {
        this._pattern.model.envs = omit(this.pattern.model.envs, envName);
        this.envNames = Object.keys(this._pattern.model.envs);
        this._cd.markForCheck();
    }
}
