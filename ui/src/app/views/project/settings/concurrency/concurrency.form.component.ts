
import { ChangeDetectionStrategy, ChangeDetectorRef, Component, EventEmitter, Input, OnChanges, OnInit, Output, SimpleChanges } from "@angular/core";
import { Concurrency, ConcurrencyOrder } from "app/model/project.concurrency.model";
import { Project } from "app/model/project.model";
import { VariableSet, VariableSetItem } from "app/model/variablesets.model";
import { V2ProjectService } from "app/service/projectv2/project.service";
import { ErrorUtils } from "app/shared/error.utils";
import { ToastService } from "app/shared/toast/ToastService";
import { NzMessageService } from "ng-zorro-antd/message";
import { finalize, lastValueFrom } from "rxjs";

@Component({
    selector: 'app-project-concurrency-form',
    templateUrl: './concurrency.form.component.html',
    styleUrls: ['./concurrency.form.component.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ProjectConcurrencyFormComponent implements OnInit {
    @Input() project: Project;
    @Input() concurrency: Concurrency
    @Input() verticalOrientation: boolean;

    @Output() refresh: EventEmitter<boolean> = new EventEmitter();
    
    loading: boolean;
    errorName: boolean;
    errorDescription: boolean;
    errorPool: boolean;
    errorOrder: boolean;
    orders = ConcurrencyOrder.array();
    
    constructor(
        private _cd: ChangeDetectorRef,
        private _messageService: NzMessageService,
        private _v2ProjectService: V2ProjectService
    ) {}

    ngOnInit(): void {
        if (!this.concurrency) {
            this.concurrency = <Concurrency>{pool: 1, order: ConcurrencyOrder.OLDEST_FIRST}
        }
    }

    async createConcurrency() {
        this.errorName = false;
        this.errorDescription = false;
        this.errorPool = false;
        this.errorOrder = false;

        if (!this.concurrency.name || this.concurrency.name === '') {
            this.errorName = true;
        } else {
            this.concurrency.name = this.concurrency.name.trim();
        }
        if (!this.concurrency.description || this.concurrency.description === '') {
            this.errorDescription = true;
        }
        if (!this.concurrency.pool || this.concurrency.pool <= 0) {
            this.errorPool = true
        }
        if (!this.concurrency.order || this.concurrency.order === '' || (this.concurrency.order !== ConcurrencyOrder.OLDEST_FIRST && this.concurrency.order !== ConcurrencyOrder.NEWEST_FIRST)) {
            this.errorOrder = true
        }
        if (this.errorOrder || this.errorPool || this.errorDescription || this.errorName) {
            this._cd.markForCheck();
            return;
        }

        this.loading = true;
        try {
            if (this.concurrency.id && this.concurrency.id !== '') {
                await lastValueFrom(this._v2ProjectService.updateConcurrency(this.project.key, this.concurrency));
                this._messageService.success(`Concurrency ${this.concurrency.name} updated`, {nzDuration: 2000});
            } else {
                await lastValueFrom(this._v2ProjectService.createConcurrency(this.project.key, this.concurrency));
                this._messageService.success(`Concurrency ${this.concurrency.name} created`, {nzDuration: 2000});
                this.concurrency = new Concurrency();
            }
            this.refresh.emit(true);
        } catch (e) {
            this._messageService.error(`Unable to create concurrency: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
        }
        this.loading = false;
        this._cd.markForCheck();
    }
}