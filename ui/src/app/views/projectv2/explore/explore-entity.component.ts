import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, OnInit, ViewChild } from "@angular/core";
import { Entity, EntityType, EntityTypeUtil } from "app/model/entity.model";
import { AutoUnsubscribe } from "app/shared/decorator/autoUnsubscribe";
import { Schema } from 'app/model/json-schema.model';
import { ActivatedRoute, Router } from "@angular/router";
import { VCSProject } from "app/model/vcs.model";
import { ProjectService } from "app/service/project/project.service";
import { lastValueFrom, Subscription } from "rxjs";
import { RouterService } from "app/service/services.module";
import { NzMessageService } from "ng-zorro-antd/message";
import { ProjectRepository } from "app/model/project.model";
import { ErrorUtils } from "app/shared/error.utils";
import { Store } from "@ngxs/store";
import * as actionPreferences from 'app/store/preferences.action';
import { EditorOptions, NzCodeEditorComponent } from "ng-zorro-antd/code-editor";
import { JSONSchema } from "app/model/schema.model";
import { PreferencesState } from "app/store/preferences.state";
import { load } from "js-yaml";
import { editor, } from 'monaco-editor';

declare const monaco: any;

@Component({
    standalone: false,
	selector: 'app-projectv2-explore-entity',
	templateUrl: './explore-entity.html',
	styleUrls: ['./explore-entity.scss'],
	changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectV2ExploreEntityComponent implements OnInit, OnDestroy {
	static PANEL_KEY = 'project-workflow-v2-entity';

	@ViewChild('editor') editor: NzCodeEditorComponent;

	currentRef: string;
	editorOption: EditorOptions;
	entity: Entity;
	jsonSchema: Schema;
	loading: boolean;
	panelSize: number | string;
	projectKey: string;
	repository: ProjectRepository;
	resizing: boolean;
	resizingSubscription: Subscription;
	showWorkflowPreview: boolean;
	vcs: VCSProject;

	constructor(
		private _activatedRoute: ActivatedRoute,
		private _cd: ChangeDetectorRef,
		private _messageService: NzMessageService,
		private _projectService: ProjectService,
		private _router: Router,
		private _routerService: RouterService,
		private _store: Store
	) { }

	ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

	ngOnInit(): void {
		this._activatedRoute.params.subscribe(_ => {
			const params = this._routerService.getRouteSnapshotParams({}, this._router.routerState.snapshot.root);
			const vcsName = params['vcsName'];
			const repoName = params['repoName'];
			const entityType = EntityTypeUtil.fromURLParam(params['entityType']);
			const entityName = params['entityName'];
			if (this.vcs?.name === vcsName && this.repository?.name === repoName && this.entity?.type === entityType && this.entity?.name === entityName) {
				return;
			}
			this.projectKey = params['key'];
			this.currentRef = this._activatedRoute.snapshot.queryParams['ref'] ?? null;

			this.load(vcsName, repoName, entityType, entityName);
		});

		this._activatedRoute.queryParams.subscribe(q => {
			if (this.currentRef === q['ref']) {
				return;
			}
			this.currentRef = q['ref'];

			const params = this._routerService.getRouteSnapshotParams({}, this._router.routerState.snapshot.root);
			const vcsName = params['vcsName'];
			const repoName = params['repoName'];
			const entityType = EntityTypeUtil.fromURLParam(params['entityType']);
			const entityName = params['entityName'];
			this.load(vcsName, repoName, entityType, entityName);
		});

		this.editorOption = {
			language: 'yaml',
			minimap: { enabled: false },
			readOnly: true,
			scrollBeyondLastLine: false
		};

		this.panelSize = this._store.selectSnapshot(PreferencesState.panelSize(ProjectV2ExploreEntityComponent.PANEL_KEY));

		this.resizingSubscription = this._store.select(PreferencesState.resizing).subscribe(resizing => {
			this.resizing = resizing;
			if (!resizing) {
				this.editor.layout();
			}
			this._cd.markForCheck();
		});

		this._cd.markForCheck();
	}

	async load(vcsName: string, repoName: string, entityType: EntityType, entityName: string) {
		this.loading = true;
		this._cd.markForCheck();

		try {
			const results = await Promise.all([
				lastValueFrom(this._projectService.getVCSProject(this.projectKey, vcsName)),
				lastValueFrom(this._projectService.getVCSRepository(this.projectKey, vcsName, repoName)),
				lastValueFrom(this._projectService.getJSONSchema(entityType))
			]);
			this.vcs = results[0];
			this.repository = results[1];
			this.jsonSchema = results[2];
			this.entity = await lastValueFrom(this._projectService.getRepoEntity(this.projectKey, this.vcs.name, this.repository.name, entityType, entityName, this.currentRef));
			this.showWorkflowPreview = false;
			if (this.entity.type === EntityType.Workflow) {
				const wkf = load(this.entity.data);
				this.showWorkflowPreview = !wkf['from'];
			}
		} catch (e: any) {
			this._messageService.error(`Unable to load entity: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
			this._router.navigate(['/project', this.projectKey, 'explore', 'vcs', vcsName, 'repository', repoName]);
		}

		this.loading = false;
		this._cd.markForCheck();
	}

	onEditorInit(e: editor.ICodeEditor | editor.IEditor): void {
		monaco.languages.json.jsonDefaults.setDiagnosticsOptions({
			schemas: [{
				uri: '',
				schema: JSONSchema.flat(this.jsonSchema)
			}]
		});
		this.editor.layout();
	}

	panelStartResize(): void {
		this._store.dispatch(new actionPreferences.SetPanelResize({ resizing: true }));
	}

	panelEndResize(size: string): void {
		this._store.dispatch(new actionPreferences.SavePanelSize({
			panelKey: ProjectV2ExploreEntityComponent.PANEL_KEY,
			size: size
		}));
		this._store.dispatch(new actionPreferences.SetPanelResize({ resizing: false }));
	}

}	
