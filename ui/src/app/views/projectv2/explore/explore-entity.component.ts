import { ChangeDetectionStrategy, ChangeDetectorRef, Component, OnDestroy, OnInit } from "@angular/core";
import { Entity, EntityType, EntityTypeUtil } from "app/model/entity.model";
import { AutoUnsubscribe } from "app/shared/decorator/autoUnsubscribe";
import { Schema } from 'app/model/json-schema.model';
import { ActivatedRoute, Router } from "@angular/router";
import { VCSProject } from "app/model/vcs.model";
import { ProjectService } from "app/service/project/project.service";
import { lastValueFrom } from "rxjs";
import { RouterService } from "app/service/services.module";
import { NzMessageService } from "ng-zorro-antd/message";
import { ProjectRepository } from "app/model/project.model";
import { load } from "js-yaml";
import { ErrorUtils } from "app/shared/error.utils";

@Component({
	selector: 'app-projectv2-explore-entity',
	templateUrl: './explore-entity.html',
	styleUrls: ['./explore-entity.scss'],
	changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectV2ExploreEntityComponent implements OnInit, OnDestroy {
	loading: boolean;
	error: string;
	currentRef: string;
	projectKey: string;
	vcs: VCSProject;
	repository: ProjectRepository;
	entity: Entity;
	jsonSchema: Schema;
	isWorkflowFromTemplate: boolean;

	constructor(
		private _cd: ChangeDetectorRef,
		private _activatedRoute: ActivatedRoute,
		private _projectService: ProjectService,
		private _routerService: RouterService,
		private _router: Router,
		private _messageService: NzMessageService
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
			this.isWorkflowFromTemplate = false;
			if (this.entity.type === EntityType.Workflow) {
				const wkf = load(this.entity.data);
				if (wkf['from']) {
					this.isWorkflowFromTemplate = true;
				}
			}
		} catch (e: any) {
			this._messageService.error(`Unable to entity: ${ErrorUtils.print(e)}`, { nzDuration: 2000 });
			this._router.navigate(['/project', this.projectKey, 'explore', 'vcs', vcsName, 'repository', repoName]);
		}

		this.loading = false;
		this._cd.markForCheck();
	}

}	