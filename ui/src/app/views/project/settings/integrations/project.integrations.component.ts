import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnDestroy, OnInit, ViewChild } from '@angular/core';
import { Project } from '../../../../model/project.model';
import { Store } from '@ngxs/store';
import { IntegrationModel, ProjectIntegration } from 'app/model/integration.model';
import { V2ProjectService } from 'app/service/projectv2/project.service';
import { ErrorUtils } from 'app/shared/error.utils';
import { PreferencesState } from 'app/store/preferences.state';
import { NzMessageService } from 'ng-zorro-antd/message';
import { Subscription, lastValueFrom } from 'rxjs';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { IntegrationService } from 'app/service/services.module';
import { NsAutoHeightTableDirective } from 'app/shared/directives/ns-auto-height-table.directive';

@Component({
    standalone: false,
    selector: 'app-project-integrations',
    templateUrl: './project.integrations.html',
    styleUrls: ['./project.integrations.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectIntegrationsComponent implements OnInit, OnDestroy {
    @ViewChild('codeMirror') codemirror: any;
    @ViewChild('autoHeightDirective') autoHeightDirective: NsAutoHeightTableDirective;

    @Input() project: Project;

    loading = { list: false, action: false };
    codeMirrorConfig: any;
    codeMirrorConfigRO: any;
    themeSubscription: Subscription;
    integrations: Array<ProjectIntegration> = [];
    filteredIntegrations: Array<ProjectIntegration> = [];
    filter: string = '';
    createModalVisible: boolean = false;
    models: Array<IntegrationModel> = [];
    newIntegration: ProjectIntegration;

    constructor(
        private _v2ProjectService: V2ProjectService,
        private _messageService: NzMessageService,
        private _store: Store,
        private _cd: ChangeDetectorRef,
        private _integrationService: IntegrationService
    ) {
        this.newIntegration = new ProjectIntegration();
        this.codeMirrorConfig = {
            mode: 'shell',
            lineWrapping: true,
            lineNumbers: true,
            autoRefresh: true
        };
        this.codeMirrorConfigRO = {
            ...this.codeMirrorConfig,
            readOnly: true
        };
    }

    ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

    ngOnInit(): void {
        this.themeSubscription = this._store.select(PreferencesState.theme).subscribe(t => {
            this.codeMirrorConfig.theme = t === 'night' ? 'darcula' : 'default';
            this.codeMirrorConfigRO.theme = t === 'night' ? 'darcula' : 'default';
            if (this.codemirror && this.codemirror.instance) {
                this.codemirror.instance.setOption('theme', this.codeMirrorConfig.theme);
            }
            this._cd.markForCheck();
        });
        this.load();
    }

    async load() {
        this.loading.list = true;
        this._cd.markForCheck();
        try {
            const models = await lastValueFrom(this._integrationService.getIntegrationModels());
            this.models = models.filter(pf => !pf.public);
            this.integrations = await lastValueFrom(this._v2ProjectService.getIntegrations(this.project.key));
            this.integrations = this.integrations.map(integ => this.setDefaultConfig(integ));
            this.integrations.sort((a, b) => a.name < b.name ? -1 : 1);
            this.applyFilter();
        } catch (e) {
            this._messageService.error(`Unable to load integrations: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
        }
        this.loading.list = false;
        this._cd.markForCheck();
    }

    async deleteIntegration(p: ProjectIntegration) {
        this.loading.action = true;
        this._cd.markForCheck();
        try {
            await lastValueFrom(this._v2ProjectService.deleteIntegration(this.project.key, p.name));
            this.integrations = this.integrations.filter(i => i.name !== p.name);
            this.applyFilter();
        } catch (e) {
            this._messageService.error(`Unable to delete integration: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
        }
        this.loading.action = false;
        this._cd.markForCheck();
    }

    async updateIntegration(p: ProjectIntegration) {
        this.loading.action = true;
        this._cd.markForCheck();
        try {
            const integration = await lastValueFrom(this._v2ProjectService.putIntegration(this.project.key, p));
            this.integrations = this.integrations.map(i => i.name === integration.name ? integration : i);
            this.applyFilter();
        } catch (e) {
            this._messageService.error(`Unable to update integration: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
        }
        this.loading.action = false;
        this._cd.markForCheck();
    }

    async create() {
        this.loading.action = true;
        this._cd.markForCheck();
        try {
            const integration = await lastValueFrom(this._v2ProjectService.postIntegration(this.project.key, this.newIntegration));
            this.integrations = this.integrations.concat(this.setDefaultConfig(integration));
            this.integrations.sort((a, b) => a.name < b.name ? -1 : 1);
            this.applyFilter();
            this.closeCreateModal();
            if (this.autoHeightDirective) { this.autoHeightDirective.onResize(null); }
        } catch (e) {
            this._messageService.error(`Unable to create integration: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
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
        this.filteredIntegrations = search === '' ? this.integrations :
            this.integrations.filter(i => [i.name, i.model?.name, ...this.modelTypes(i.model)]
                .some(field => (field ?? '').toLowerCase().indexOf(search) !== -1));
    }

    modelTypes(model: IntegrationModel): Array<string> {
        if (!model) {
            return [];
        }
        return [
            model.storage ? 'storage' : null,
            model.hook ? 'hook' : null,
            model.event ? 'event' : null,
            model.deployment ? 'deployment' : null
        ].filter(t => !!t);
    }

    openCreateModal(): void {
        this.newIntegration = new ProjectIntegration();
        this.createModalVisible = true;
        this._cd.markForCheck();
    }

    closeCreateModal(): void {
        this.createModalVisible = false;
        this.newIntegration = new ProjectIntegration();
        this._cd.markForCheck();
    }

    setDefaultConfig(integ: ProjectIntegration): ProjectIntegration {
        const cp = { ...integ };
        if (!integ.model.default_config) {
            return cp;
        }
        const keys = Object.keys(cp.model.default_config);
        if (keys) {
            keys.forEach(k => {
                if (!cp.config) {
                    cp.config = {};
                }
                if (!cp.config[k]) {
                    cp.config[k] = cp.model.default_config[k];
                }
            });
        }
        return cp;
    }

    updateConfig(model: IntegrationModel) {
        this.newIntegration.model = model;
        this.newIntegration = this.setDefaultConfig(this.newIntegration);
        if (this.autoHeightDirective) { this.autoHeightDirective.onResize(null); }
        this._cd.markForCheck();
    }
}
