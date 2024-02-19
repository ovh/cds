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
import { Subscription } from 'rxjs';

@Component({
	selector: 'app-projectv2-run-list-sidebar',
	templateUrl: './run-list-sidebar.html',
	styleUrls: ['./run-list-sidebar.scss'],
	changeDetection: ChangeDetectionStrategy.OnPush
})
@AutoUnsubscribe()
export class ProjectV2WorkflowRunListSidebarComponent implements OnInit, OnDestroy {
	@Input() project: Project;

	authSummary: AuthSummary;
	searchesSubscription: Subscription;
	searches: Array<{
		name: string
		params: { [key: string]: any }
	}> = [];

	constructor(
		private _cd: ChangeDetectorRef,
		private _store: Store
	) { }

	ngOnDestroy(): void { } // Should be set to use @AutoUnsubscribe with AOT

	ngOnInit(): void {
		this.authSummary = this._store.selectSnapshot(AuthenticationState.summary);

		this.searchesSubscription = this._store.select(PreferencesState.selectProjectRunFilters(this.project.key)).subscribe(searches => {
			this.searches = searches.map(s => {
				let params = {};
				s.value.split(' ').forEach(f => {
					const s = f.split(':');
					if (s.length === 2 && s[1] !== '') {
						if (Array.isArray(params[s[0]])) {
							params[s[0]].push(s[1]);
						} else if (params[s[0]]) {
							params[s[0]] = [params[s[0]], s[1]];
						} else {
							params[s[0]] = s[1];
						}
					}
				});
				return {
					name: s.name,
					params
				};
			});
			this._cd.markForCheck();
		});

		this._cd.markForCheck();
	}
}
