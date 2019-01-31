import { Component, Input } from '@angular/core';
import { Commit } from '../../model/repositories.model';
import { Table } from '../table/table';

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
