import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnInit } from '@angular/core';
import { Key } from 'app/model/keys.model';
import { Project } from 'app/model/project.model';
import { V2ProjectService } from 'app/service/projectv2/project.service';
import { ErrorUtils } from 'app/shared/error.utils';
import { KeyEvent } from 'app/shared/keys/key.event';
import { NzMessageService } from 'ng-zorro-antd/message';
import { lastValueFrom } from 'rxjs';

@Component({
    standalone: false,
    selector: 'app-project-keys',
    templateUrl: './project.keys.html',
    styleUrls: ['./project.keys.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ProjectKeysComponent implements OnInit {
    @Input() project: Project;

    keys: Array<Key> = [];
    filteredKeys: Array<Key> = [];
    filter: string = '';
    createModalVisible: boolean = false;
    loading = { list: false, action: false };

    constructor(
        private _messageService: NzMessageService,
        private _cd: ChangeDetectorRef,
        private _v2ProjectService: V2ProjectService
    ) { }

    ngOnInit(): void {
        this.load();
    }

    async load() {
        this.loading.list = true;
        this._cd.markForCheck();
        try {
            this.keys = await lastValueFrom(this._v2ProjectService.getKeys(this.project.key));
            this.applyFilter();
        } catch (e) {
            this._messageService.error(`Unable to load keys: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
        }
        this.loading.list = false;
        this._cd.markForCheck();
    }

    async addKey(event: KeyEvent) {
        this.loading.action = true;
        this._cd.markForCheck();
        try {
            const key = await lastValueFrom(this._v2ProjectService.postKey(this.project.key, event.key));
            this.keys = this.keys.concat(key);
            this.keys.sort((a, b) => a.name < b.name ? -1 : 1);
            this.applyFilter();
            this.closeCreateModal();
        } catch (e) {
            this._messageService.error(`Unable to add key: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
        }
        this.loading.action = false;
        this._cd.markForCheck();
    }

    async deleteKey(key: Key) {
        this.loading.action = true;
        this._cd.markForCheck();
        try {
            await lastValueFrom(this._v2ProjectService.deleteKey(this.project.key, key.name));
            this.keys = this.keys.filter(k => k.name !== key.name);
            this.applyFilter();
        } catch (e) {
            this._messageService.error(`Unable to delete key: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
        }
        this.loading.action = false;
        this._cd.markForCheck();
    }

    updateFilter(value: string): void {
        this.filter = value;
        this.applyFilter();
        this._cd.markForCheck();
    }

    applyFilter(): void {
        const search = (this.filter ?? '').trim().toLowerCase();
        this.filteredKeys = search === '' ? this.keys :
            this.keys.filter(k => k.name.toLowerCase().indexOf(search) !== -1
                || (k.type ?? '').toLowerCase().indexOf(search) !== -1);
    }

    openCreateModal(): void {
        this.createModalVisible = true;
        this._cd.markForCheck();
    }

    closeCreateModal(): void {
        this.createModalVisible = false;
        this._cd.markForCheck();
    }
}
