import { Component, Input } from '@angular/core';
import { Commit } from 'app/model/repositories.model';
import { Table } from 'app/shared/table/table';

@Component({
    selector: 'app-commit-list',
    templateUrl: './commit.list.html',
    styleUrls: ['./commit.list.scss']
})
export class CommitListComponent extends Table<Commit> {
    @Input() commits: Array<Commit>;

    constructor() {
        super();
    }

    getData(): Array<Commit> {
        return this.commits;
    }
}
