import { ChangeDetectionStrategy, Component, Input } from "@angular/core";
import { Project } from "app/model/project.model";

@Component({
	selector: 'app-projectv2-left-menu',
	templateUrl: './left-menu.html',
	styleUrls: ['./left-menu.scss'],
	changeDetection: ChangeDetectionStrategy.OnPush
})
export class ProjectV2LeftMenuComponent {
	@Input() project: Project;
}