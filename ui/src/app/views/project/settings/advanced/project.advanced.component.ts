import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnChanges, OnInit, SimpleChanges } from '@angular/core';
import { Router } from '@angular/router';
import { Store } from '@ngxs/store';
import { APIConfig } from 'app/model/config.service';
import { Project } from 'app/model/project.model';
import { V2ProjectService } from 'app/service/projectv2/project.service';
import { ConfigState } from 'app/store/config.state';
import { lastValueFrom } from 'rxjs';
import { SetCurrentProjectV2 } from 'app/store/project-v2.action';
import { NzMessageService } from 'ng-zorro-antd/message';
import { ErrorUtils } from 'app/shared/error.utils';
import { ToastService } from 'app/shared/toast/ToastService';
import { FormBuilder, FormControl, FormGroup, Validators } from '@angular/forms';

@Component({
    selector: 'app-project-advanced',
    templateUrl: './project.advanced.html',
    styleUrls: ['./project.advanced.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ProjectAdvancedComponent implements OnInit, OnChanges {
    @Input() project: Project;

    loading = false;
    fileTooLarge = false;
    apiConfig: APIConfig;
    validateForm: FormGroup<{
        name: FormControl<string | null>;
        description: FormControl<string | null>;
        icon: FormControl<string | null>;
        retention: FormControl<number | null>;
    }>;

    constructor(
        private _v2ProjectService: V2ProjectService,
        public _messageService: NzMessageService,
        private _router: Router,
        private _toast: ToastService,
        private _store: Store,
        private _cd: ChangeDetectorRef,
        private _fb: FormBuilder
    ) {
        this.validateForm = this._fb.group({
            name: this._fb.control<string | null>(null, Validators.required),
            description: this._fb.control<string | null>(null),
            icon: this._fb.control<string | null>(null),
            retention: this._fb.control<number | null>(null, Validators.required)
        });
    }

    ngOnInit(): void {
        if (!this.project.permissions.writable) {
            this._router.navigate(['/project', this.project.key]);
        }
        this.apiConfig = this._store.selectSnapshot(ConfigState.api);
    }

    ngOnChanges(changes: SimpleChanges): void {
        this.validateForm.controls.name.setValue(this.project.name);
        this.validateForm.controls.description.setValue(this.project.description);
        this.validateForm.controls.icon.setValue(this.project.icon);
        this.validateForm.controls.retention.setValue(this.project.workflow_retention);
    }

    async onSubmitProjectUpdate() {
        this.fileTooLarge = false;
        if (!this.validateForm.valid) {
            Object.values(this.validateForm.controls).forEach(control => {
                if (control.invalid) {
                    control.markAsDirty();
                    control.updateValueAndValidity({ onlySelf: true });
                }
            });
            return;
        }
        this.validateForm.disable();
        this.loading = true;
        this._cd.markForCheck();
        try {
            const p = await lastValueFrom(this._v2ProjectService.put({
                ...this.project,
                name: this.validateForm.value.name,
                description: this.validateForm.value.description,
                icon: this.validateForm.value.icon,
                workflow_retention: this.validateForm.value.retention,
            }));
            this._store.dispatch(new SetCurrentProjectV2(p));
            this._toast.success('', 'Project updated');
        } catch (e) {
            this._messageService.error(`Unable to update project: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
        }
        this.loading = false;
        this.validateForm.enable();
        this._cd.markForCheck();
    }

    async deleteProject() {
        this.loading = true;
        this._cd.markForCheck();
        try {
            await lastValueFrom(this._v2ProjectService.delete(this.project.key));
            this._toast.success('', 'Project deleted');
            this._router.navigate(['/']);
        } catch (e) {
            this._messageService.error(`Unable to delete project: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
        }
        this.loading = false;
        this._cd.markForCheck();
    }

    fileEvent(event: { content: string, file: File }) {
        this.fileTooLarge = event.file.size > 100000;
        this._cd.markForCheck();
        if (this.fileTooLarge) {
            return;
        }
        this.validateForm.controls.icon.setValue(event.content);
    }
}
