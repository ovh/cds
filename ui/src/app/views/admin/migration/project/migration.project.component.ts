import {Component, OnInit} from '@angular/core';
import {ProjectStore} from '../../../../service/project/project.store';
import {ActivatedRoute} from '@angular/router';
import {Project} from '../../../../model/project.model';
import {first} from 'rxjs/operators';

@Component({
    selector: 'app-migration-project',
    templateUrl: './migration.project.html',
    styleUrls: ['./migration.project.scss']
})
export class MigrationProjectComponent implements OnInit {

    project: Project;

    constructor(private _projectStore: ProjectStore, private _activatedRoute: ActivatedRoute) { }

    ngOnInit(): void {
        this._activatedRoute.params.subscribe(d => {
            if (d['key']) {
                let key = d['key'];
                this._projectStore.resync(key, []).pipe(first()).subscribe(p => {
                    this.project = p;
                });
            }

        });
    }
}
