import {Component, OnInit} from '@angular/core';
import {ProjectStore} from '../../../../service/project/project.store';
import {ActivatedRoute} from '@angular/router';
import {Project} from '../../../../model/project.model';

@Component({
    selector: 'app-migration-project',
    templateUrl: './migration.project.html',
    styleUrls: ['./migration.project.scss']
})
export class MigrationProjectComponent implements OnInit {

    project: Project;

    constructor(private _projectStore: ProjectStore, private _activatedRoute: ActivatedRoute) { }

    ngOnInit(): void {
        console.log(this._activatedRoute);
        this._activatedRoute.params.subscribe(d => {
            if (d['key']) {
                let key = d['key'];
                this._projectStore.resync(key).first().subscribe(p => {
                    this.project = p;
                });
            }

        });
    }
}
