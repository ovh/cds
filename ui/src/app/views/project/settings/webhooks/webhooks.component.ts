import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnInit } from "@angular/core";
import { Project } from "app/model/project.model";
import { HookType, PostProjectWebHook, PostResponseCreateHook, ProjectWebHook } from "app/model/project.webhook.model";
import { VCSProject } from "app/model/vcs.model";
import { ProjectService } from "app/service/project/project.service";
import { V2ProjectService } from "app/service/projectv2/project.service";
import { ErrorUtils } from "app/shared/error.utils";
import { NzMessageService } from "ng-zorro-antd/message"; 
import { lastValueFrom } from "rxjs";

@Component({
    standalone: false,
    selector: 'app-project-webhooks',
    templateUrl: './webhooks.html',
    styleUrls: ['./webhooks.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class ProjectWebhooksComponent implements OnInit {
    @Input() project: Project;

    loading = { list: false, action: false };
    vcss: Array<VCSProject>;
    webhooks: Array<ProjectWebHook> = [];
    filteredWebhooks: Array<ProjectWebHook> = [];
    filter: string = '';
    createModalVisible: boolean = false;
    newWebhook: PostProjectWebHook = new PostProjectWebHook();
    createdHook: PostResponseCreateHook;
    errorRepository: boolean;
    errorWorkflow: boolean;
    hookTypes: Array<string> = [HookType.Repository, HookType.Workflow];

    constructor(
        private _cd: ChangeDetectorRef,
        private _messageService: NzMessageService,
        private _v2ProjectService: V2ProjectService,
        private _projectService: ProjectService
    ) { }

    ngOnInit(): void {
        this.load();
    }

    async load() {
        this.loading.list = true;
        this._cd.markForCheck();
        try {
            this.webhooks = await lastValueFrom(this._v2ProjectService.getWebhooks(this.project.key));
            this.applyFilter();
            this.vcss = await lastValueFrom(this._projectService.listVCSProject(this.project.key));
            if (this.vcss.length > 0) {
                this.newWebhook.vcs_server = this.vcss[0].name;
            }
        } catch (e) {
            this._messageService.error(`Unable to load webhooks: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
        }
        this.loading.list = false;
        this._cd.markForCheck();
    }

    async deleteWebhook(h: ProjectWebHook) {
        this.loading.action = true;
        this._cd.markForCheck();
        try {
            await lastValueFrom(this._v2ProjectService.deleteWebhook(this.project.key, h.id))
            this.webhooks = this.webhooks.filter(s => s.id !== h.id);
            this.applyFilter();
            this._messageService.success(`WebHook ${h.id} deleted`, { nzDuration: 2000 });
        } catch (e) {
            this._messageService.error(`Unable to delete webhook: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
        }
        this.loading.action = false;
        this._cd.markForCheck();
    }

    async createHook() {
        this.errorRepository = false;
        this.errorWorkflow = false;
        if (this.newWebhook.repository === '' || !this.newWebhook.repository) {
            this.errorRepository = true;
            this._cd.markForCheck();
            return;
        }
        if (this.newWebhook.type === HookType.Workflow && (this.newWebhook.workflow === '' || !this.newWebhook.workflow)) {
            this.errorWorkflow = true;
            this._cd.markForCheck();
            return;
        }
        this.loading.action = true;
        this._cd.markForCheck();
        try {
            this.createdHook = await lastValueFrom(this._v2ProjectService.createWebhook(this.project.key, this.newWebhook))
            this.closeCreateModal();
            this.load();
            this.newWebhook = new PostProjectWebHook();
        } catch (e) {
            this._messageService.error(`Unable to create webhook: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
        }
        this.loading.action = false;
        this._cd.markForCheck();
    }

    closeAlert() {
        delete this.createdHook;
        this._cd.markForCheck();
    }

    updateFilter(value: string): void {
        this.filter = value;
        this.applyFilter();
        this._cd.markForCheck();
    }

    applyFilter(): void {
        const search = (this.filter ?? '').trim().toLowerCase();
        this.filteredWebhooks = search === '' ? this.webhooks : this.webhooks.filter(h =>
            [h.id, h.type, h.vcs_server, h.repository, h.workflow, h.username]
                .some(field => (field ?? '').toLowerCase().indexOf(search) !== -1));
    }

    openCreateModal(): void {
        this.errorRepository = false;
        this.errorWorkflow = false;
        this.newWebhook = new PostProjectWebHook();
        if (this.vcss?.length > 0) {
            this.newWebhook.vcs_server = this.vcss[0].name;
        }
        this.createModalVisible = true;
        this._cd.markForCheck();
    }

    closeCreateModal(): void {
        this.createModalVisible = false;
        this._cd.markForCheck();
    }
}
