import { Component, Input, OnInit, ViewChild } from '@angular/core';
import { TranslateService } from '@ngx-translate/core';
import { Store } from '@ngxs/store';
import { IntegrationModel, ProjectIntegration } from 'app/model/integration.model';
import { Project } from 'app/model/project.model';
import { IntegrationService } from 'app/service/integration/integration.service';
import { ThemeStore } from 'app/service/services.module';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { ToastService } from 'app/shared/toast/ToastService';
import { AddIntegrationInProject } from 'app/store/project.action';
import { finalize, first } from 'rxjs/operators';
import { Subscription } from 'rxjs/Subscription';

@Component({
    selector: 'app-project-integration-form',
    templateUrl: './project.integration.form.html',
    styleUrls: ['./project.integration.form.scss']
})
@AutoUnsubscribe()
export class ProjectIntegrationFormComponent implements OnInit {
    @ViewChild('codeMirror', {static: false}) codemirror: any;

    @Input() project: Project;

    models: Array<IntegrationModel>;
    newIntegration: ProjectIntegration;
    loading = false;
    codeMirrorConfig: any;
    themeSubscription: Subscription;

    constructor(
        private _integrationService: IntegrationService,
        private _toast: ToastService,
        private _translate: TranslateService,
        private store: Store,
        private _theme: ThemeStore
    ) {
        this.newIntegration = new ProjectIntegration();
        this.codeMirrorConfig = {
            mode: 'shell',
            lineWrapping: true,
            lineNumbers: true,
            autoRefresh: true
        };
    }

    ngOnInit() {
        this._integrationService.getIntegrationModels().pipe(first()).subscribe(platfs => {
            this.models = platfs.filter(pf => !pf.public);
        });

        this.themeSubscription = this._theme.get().subscribe(t => {
            this.codeMirrorConfig.theme = t === 'night' ? 'darcula' : 'default';
            if (this.codemirror && this.codemirror.instance) {
                this.codemirror.instance.setOption('theme', this.codeMirrorConfig.theme);
            }
        });
    }

    updateConfig(): void {
        ProjectIntegration.mergeConfig(this.newIntegration.model.default_config, this.newIntegration.config);
    }

    create(): void {
        this.loading = true;
        this.store.dispatch(new AddIntegrationInProject({ projectKey: this.project.key, integration: this.newIntegration }))
            .pipe(finalize(() => this.loading = false))
            .subscribe(() => {
                this.newIntegration = new ProjectIntegration();
                this._toast.success('', this._translate.instant('project_updated'));
            });
    }
}
