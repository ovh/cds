import { IconDefinition } from '@ant-design/icons-angular';
import {
    AimOutline,
    AppstoreOutline,
    ArrowsAltOutline,
    BranchesOutline,
    BuildOutline,
    ClockCircleOutline,
    CloseCircleOutline,
    CodeOutline,
    ContainerOutline,
    DownOutline,
    ExperimentOutline,
    FieldTimeOutline,
    FileOutline,
    GlobalOutline,
    InboxOutline,
    PieChartOutline,
    QuestionCircleOutline,
    RedoOutline,
    RightOutline,
    RocketOutline,
    ShrinkOutline,
    SyncOutline,
    TagOutline,
    ThunderboltOutline,
    UnlockOutline,
    UserOutline
} from '@ant-design/icons-angular/icons';

/**
 * The controls of the view, plus the icons a marker may be drawn with. Which of the latter a marker gets
 * is the host's decision — the library only makes sure they are all registered.
 */
export const TIMELINE_ICONS: IconDefinition[] = [
    AimOutline,
    AppstoreOutline,
    ArrowsAltOutline,
    BranchesOutline,
    BuildOutline,
    ClockCircleOutline,
    CloseCircleOutline,
    CodeOutline,
    ContainerOutline,
    DownOutline,
    ExperimentOutline,
    FieldTimeOutline,
    FileOutline,
    GlobalOutline,
    InboxOutline,
    PieChartOutline,
    QuestionCircleOutline,
    RedoOutline,
    RightOutline,
    RocketOutline,
    ShrinkOutline,
    SyncOutline,
    TagOutline,
    ThunderboltOutline,
    UnlockOutline,
    UserOutline
];

/** The names those are drawn by, which is what a marker asks for. */
const NAMES: ReadonlySet<string> = new Set(TIMELINE_ICONS.map(icon => icon.name));

/**
 * The icon a marker is drawn with, or nothing when it asks for one this view does not carry — in which
 * case it is drawn as a plain mark instead.
 *
 * Asking is not enough on purpose. An unregistered name is not a missing icon: the icon component goes and
 * fetches `assets/<theme>/<name>.svg` and puts what comes back on the page, so a name that reached here
 * from data would be a way of choosing a URL to load and render. Nothing outside the list above is ever
 * passed on, whoever asked for it and however it got here.
 */
export function timelineIcon(name: string): string {
    return name && NAMES.has(name) ? name : null;
}
