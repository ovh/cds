import { ChangeDetectionStrategy, ChangeDetectorRef, Component, Input, OnInit } from "@angular/core";
import { Project } from "app/model/project.model";
import { PostProjectRepositoryHook, PostResponseCreateHook, ProjectWebHook } from "app/model/project.webhook.model";
import { VCSProject } from "app/model/vcs.model";
import { ProjectService } from "app/service/project/project.service";
import { V2ProjectService } from "app/service/projectv2/project.service";
import { ErrorUtils } from "app/shared/error.utils";
import { NzMessageService } from "ng-zorro-antd/message";
import { lastValueFrom } from "rxjs";

@Component({
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
    newWebhook: PostProjectRepositoryHook = new PostProjectRepositoryHook();
    createdHook: PostResponseCreateHook;
    errorRepository: boolean;

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
            this._messageService.success(`WebHook ${h.id} deleted`, { nzDuration: 2000 });
        } catch (e) {
            this._messageService.error(`Unable to delete webhook: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
        }
        this.loading.action = false;
        this._cd.markForCheck();
    }

    async createHook() {
        if (this.newWebhook.repository === '') {
            this.errorRepository = true;
            this._cd.markForCheck();
            return;
        }
        this.loading.action = true;
        this._cd.markForCheck();
        try {
            this.createdHook = await lastValueFrom(this._v2ProjectService.createWebhook(this.project.key, this.newWebhook))
            this.load();
            this.newWebhook = new PostProjectRepositoryHook();
        } catch (e) {
            this._messageService.error(`Unable to create webhook: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
        }
        this.loading.action = false;
        this._cd.markForCheck();
    }

    closeAlert() {
        delete this.createdHook;
    }
}