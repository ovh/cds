import {
	ChangeDetectionStrategy,
	ChangeDetectorRef,
	Component,
	Input,
	OnDestroy,
	OnInit,
} from '@angular/core';
import { Store } from '@ngxs/store';
import { Project } from 'app/model/project.model';
import { AuthSummary } from 'app/model/user.model';
import { AutoUnsubscribe } from 'app/shared/decorator/autoUnsubscribe';
import { AuthenticationState } from 'app/store/authentication.state';
import { PreferencesState } from 'app/store/preferences.state';
import { Subscription, forkJoin } from 'rxjs';
import * as actionPreferences from 'app/store/preferences.action';
import { ProjectRunFilter } from 'app/model/project-run-filter.model';
import { ProjectRunFilterService } from 'app/service/project/project-run-filter.service';
import { CdkDragDrop, moveItemInArray } from '@angular/cdk/drag-drop';

@Component({
    standalone: false,
	selector: 'app-projectv2-run-list-sidebar',
	templateUrl: './run-list-sidebar.html',
	styleUrls: ['./run-list-sidebar.scss'],
	changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectV2RunListSidebarComponent implements OnInit, OnDestroy {
	@Input() project: Project;

	authSummary: AuthSummary;
	searchesSubscription: Subscription;
	searches: Array<{
		name: string
		value: string
		sort: string
		order: number
		params: { [key: string]: any }
	}> = [];
	sharedFilters: Array<{
		name: string
		params: { [key: string]: any }
		order: number
	}> = [];
	canManage: boolean = false;

	constructor(
		private _cd: ChangeDetectorRef,
		private _store: Store,
		private _projectRunFilterService: ProjectRunFilterService
	) { }

	ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

	ngOnInit(): void {
		this.authSummary = this._store.selectSnapshot(AuthenticationState.summary);

		// Check permissions
		this.canManage = this.hasProjectManagePermission();

		// Load shared filters
		this._projectRunFilterService.list(this.project.key).subscribe(filters => {
			this.sharedFilters = filters.map(f => ({
				name: f.name,
				params: this.parseFilterToParams(f),
				order: f.order
			})).sort((a, b) => a.order - b.order); // Sort by order
			this._cd.markForCheck();
		}, err => {
			console.error('Error loading shared filters:', err);
		});

		// Load personal filters
		this.searchesSubscription = this._store.select(PreferencesState.selectProjectRunFilters(this.project.key)).subscribe(searches => {
			this.searches = (searches ?? []).map(s => {
				let params = {};
				s.value.split(' ').forEach(f => {
					const parts = f.split(':');
					if (parts.length === 2 && parts[1] !== '') {
						if (Array.isArray(params[parts[0]])) {
							params[parts[0]].push(parts[1]);
						} else if (params[parts[0]]) {
							params[parts[0]] = [params[parts[0]], parts[1]];
						} else {
							params[parts[0]] = parts[1];
						}
					}
				});
				if (s.sort) { params['sort'] = s.sort; }
				return {
					name: s.name,
					value: s.value,
					sort: s.sort,
					order: s.order || 0,
					params
				};
			});
			// Trier par order
			this.searches.sort((a, b) => a.order - b.order);
			this._cd.markForCheck();
		});

		this._cd.markForCheck();
	}

	parseFilterToParams(filter: ProjectRunFilter): any {
		let params = {};
		filter.value.split(' ').forEach(f => {
			const parts = f.split(':');
			if (parts.length === 2 && parts[1] !== '') {
				if (Array.isArray(params[parts[0]])) {
					params[parts[0]].push(parts[1]);
				} else if (params[parts[0]]) {
					params[parts[0]] = [params[parts[0]], parts[1]];
				} else {
					params[parts[0]] = parts[1];
				}
			}
		});
		if (filter.sort) {
			params['sort'] = filter.sort;
		}
		return params;
	}

	hasProjectManagePermission(): boolean {
		return this.project?.permissions?.writable ?? false;
	}

	// Share a personal filter (promote it to shared filter)
	sharePersonalFilter(personalFilter: any): void {
		const newFilter: any = {
			name: personalFilter.name,
			value: personalFilter.value,
			sort: personalFilter.sort || ''
		};

		this._projectRunFilterService.create(this.project.key, newFilter).subscribe({
			next: (created) => {
				// Success: delete local filter
				this._store.dispatch(new actionPreferences.DeleteProjectWorkflowRunFilter({
					projectKey: this.project.key,
					name: personalFilter.name
				}));
				// Reload shared filters
				this.reloadSharedFilters();
			},
			error: (err) => {
				console.error('Error sharing filter:', err);
			}
		});
	}

	// Delete a shared filter
	deleteSharedFilter(filterName: string): void {
		this._projectRunFilterService.delete(this.project.key, filterName).subscribe({
			next: () => {
				this.reloadSharedFilters();
			},
			error: (err) => {
				console.error('Error deleting shared filter:', err);
			}
		});
	}

	// Existing unchanged
	deleteSearch(name: string): void {
		this._store.dispatch(new actionPreferences.DeleteProjectWorkflowRunFilter({ projectKey: this.project.key, name }));
	}

	// Reorder shared filters
	onDropShared(event: CdkDragDrop<any>): void {
		if (event.previousIndex === event.currentIndex) {
			return; // no change
		}

		// Reorganize locally
		moveItemInArray(this.sharedFilters, event.previousIndex, event.currentIndex);

		// Update order and send PUT requests
		const requests = this.sharedFilters.map((f, idx) =>
			this._projectRunFilterService.update(this.project.key, f.name, { order: idx })
		);

		forkJoin(requests).subscribe({
			next: () => {
				// Update orders locally
				this.sharedFilters.forEach((f, idx) => f.order = idx);
				this._cd.markForCheck();
			},
			error: (err) => {
				console.error('Error reordering shared filters:', err);
				// In case of error, reload from API
				this.reloadSharedFilters();
			}
		});
	}

	// Reorder personal filters
	onDropPersonal(event: CdkDragDrop<any>): void {
		if (event.previousIndex === event.currentIndex) {
			return; // no change
		}

		// Reorganize locally
		moveItemInArray(this.searches, event.previousIndex, event.currentIndex);

		// Update order and save in NgXS
		const updatedSearches = this.searches.map((s, idx) => ({
			name: s.name,
			value: s.value,
			sort: s.sort || '',
			order: idx
		}));

		// Dispatch action to update all filters with their new order
		this._store.dispatch(new actionPreferences.ReorderProjectWorkflowRunFilters({
			projectKey: this.project.key,
			filters: updatedSearches
		}));

		this._cd.markForCheck();
	}

	private reloadSharedFilters(): void {
		this._projectRunFilterService.list(this.project.key).subscribe(filters => {
			this.sharedFilters = filters.map(f => ({
				name: f.name,
				params: this.parseFilterToParams(f),
				order: f.order
			})).sort((a, b) => a.order - b.order); // Trier par order
			this._cd.markForCheck();
		});
	}
}
