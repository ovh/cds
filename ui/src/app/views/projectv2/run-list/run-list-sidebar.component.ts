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

		// Vérifier les permissions (simplifié - à adapter selon RBAC existant)
		this.canManage = this.hasProjectManagePermission();

		// Charger les filtres partagés
		this._projectRunFilterService.list(this.project.key).subscribe(filters => {
			this.sharedFilters = filters.map(f => ({
				name: f.name,
				params: this.parseFilterToParams(f),
				order: f.order
			})).sort((a, b) => a.order - b.order); // Trier par order
			this._cd.markForCheck();
		}, err => {
			console.error('Error loading shared filters:', err);
		});

		// Charger les filtres personnels
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
		// TODO: implémenter la vraie vérification RBAC
		// Pour l'instant, simplifié
		return true;
	}

	// Partager un filtre personnel (le promouvoir en filtre projet)
	sharePersonalFilter(personalFilter: any): void {
		const newFilter: any = {
			name: personalFilter.name,
			value: personalFilter.value,
			sort: personalFilter.sort || ''
		};

		this._projectRunFilterService.create(this.project.key, newFilter).subscribe({
			next: (created) => {
				// Succès : supprimer le filtre local
				this._store.dispatch(new actionPreferences.DeleteProjectWorkflowRunFilter({
					projectKey: this.project.key,
					name: personalFilter.name
				}));
				// Recharger les filtres partagés
				this.reloadSharedFilters();
			},
			error: (err) => {
				console.error('Error sharing filter:', err);
			}
		});
	}

	// Supprimer un filtre partagé
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

	// Existant inchangé
	deleteSearch(name: string): void {
		this._store.dispatch(new actionPreferences.DeleteProjectWorkflowRunFilter({ projectKey: this.project.key, name }));
	}

	// Réordonner les filtres partagés
	onDropShared(event: CdkDragDrop<any>): void {
		if (event.previousIndex === event.currentIndex) {
			return; // pas de changement
		}

		// Réorganiser localement
		moveItemInArray(this.sharedFilters, event.previousIndex, event.currentIndex);

		// Mettre à jour les order et envoyer les requêtes PUT
		const requests = this.sharedFilters.map((f, idx) =>
			this._projectRunFilterService.update(this.project.key, f.name, { order: idx })
		);

		forkJoin(requests).subscribe({
			next: () => {
				// Mettre à jour les order localement
				this.sharedFilters.forEach((f, idx) => f.order = idx);
				this._cd.markForCheck();
			},
			error: (err) => {
				console.error('Error reordering shared filters:', err);
				// En cas d'erreur, recharger depuis l'API
				this.reloadSharedFilters();
			}
		});
	}

	// Réordonner les filtres personnels
	onDropPersonal(event: CdkDragDrop<any>): void {
		if (event.previousIndex === event.currentIndex) {
			return; // pas de changement
		}

		// Réorganiser localement
		moveItemInArray(this.searches, event.previousIndex, event.currentIndex);

		// Mettre à jour les order et sauvegarder dans NgXS
		const updatedSearches = this.searches.map((s, idx) => ({
			name: s.name,
			value: s.value,
			sort: s.sort || '',
			order: idx
		}));

		// Dispatch une action pour mettre à jour tous les filtres avec leur nouvel ordre
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
