import {
    afterNextRender,
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    ElementRef,
    EventEmitter,
    inject,
    Injector,
    Input,
    NgZone,
    OnChanges,
    OnDestroy,
    Output,
    SimpleChanges,
    TemplateRef,
    ViewChild,
    ViewContainerRef
} from '@angular/core';
import { FlexibleConnectedPositionStrategy, Overlay, OverlayRef } from '@angular/cdk/overlay';
import { TemplatePortal } from '@angular/cdk/portal';
import {
    axisEnds,
    AXIS_BAND_PX,
    flattenLanes,
    formatDuration,
    revealBy,
    TimelineActivation,
    TimelineData,
    TimelineDetail,
    shouldFollow,
    TimelineLane,
    TimelineMarker,
    TimelineSegment
} from './timeline.model';
import { DEFAULT_SCALE_OPTIONS, TimelineBusy, TimelineScale, TimelineScalePiece } from './timeline.scale';
import { timelineIcon } from './timeline.icons';

/** Everything below is what the template binds to: plain values, computed once per layout. */

interface RenderedSegment {
    id: string;
    kind: string;
    label: string;
    /** How long the segment lasted, shown next to the label when the segment can hold both. */
    duration: string;
    left: string;
    width: string;
    /** The label only fits on a segment wide enough to hold it. */
    showLabel: boolean;
    showDuration: boolean;
    /** Still going: drawn open-ended. */
    open: boolean;
}

interface RenderedMarker {
    id: string;
    kind: string;
    icon: string;
    left: string;
    /** What it says when hovered: a heading and a few facts, like a lane but about one instant. */
    label: string;
    details: Array<TimelineDetail>;
    /** How many markers this one stands for. More than one means they were too close to tell apart. */
    count: number;
    /** The span the markers behind it cover, which is what zooming into it has to show. */
    from: number;
    to: number;
}

/**
 * A run of segments drawn as one element. It is what the pointer talks to, so that moving from one of
 * its segments to the next never leaves it. Its segments and the markers that fall inside it are
 * placed relative to it, not to the track.
 */
interface RenderedGroup {
    id: string;
    left: string;
    width: string;
    segments: RenderedSegment[];
    markers: RenderedMarker[];
}

/** One line of the breakdown shown when a lane is hovered. */
interface RenderedShare {
    label: string;
    duration: string;
    /** Width of the proportion bar, as a percentage. */
    width: string;
    percent: string;
}

interface RenderedLane {
    id: string;
    label: string;
    sublabel: string;
    status: string;
    depth: number;
    expandable: boolean;
    expanded: boolean;
    activatable: boolean;
    highlighted: boolean;
    /** What the lane holds, read out for whoever cannot see the bars. */
    description: string;
    /** How long the lane runs from end to end. */
    total: string;
    /** Where that total went, segment by segment. */
    shares: RenderedShare[];
    details: Array<TimelineDetail>;
    groups: RenderedGroup[];
    /** Markers falling outside every group, placed against the track. */
    markers: RenderedMarker[];
    /** Whether anything is pinned on this lane, which is drawn above its bars and needs room for it. */
    hasMarkers: boolean;
}

interface RenderedSection {
    id: string;
    label: string;
    lanes: RenderedLane[];
}

interface RenderedFold {
    id: string;
    left: string;
    width: string;
    label: string;
    title: string;
}

interface RenderedTick {
    id: number;
    left: string;
    label: string;
}

/** The first and the last instant of the axis, marked so that no one takes them for a cut edge. */
interface RenderedCap {
    /** Position on the axis, kept so that graduations can be told to keep their distance. */
    ratio: number;
    left: string;
    label: string;
    /** The end of an axis whose last segment is still open is the present, and says so. */
    live: boolean;
    title: string;
}

/** Below this, zooming stops: the window would hold less than a pixel of the axis. */
const MIN_SPAN = 0.0002;
const ZOOM_FACTOR = 1.3;
/** Long enough that crossing the timeline on the way somewhere else opens nothing. */
const CARD_DELAY = 450;
/** How much room the labels at either end of the axis are given before a graduation may be drawn. */
const CAP_MARGIN_PX = 46;
/**
 * How far apart two markers have to be to be drawn separately. Closer than this they overlap, so they are
 * drawn as one carrying a count — see `cluster()`. A mark is `--timeline-mark-size` wide, and this is that
 * plus enough for the two to be seen as two.
 */
const MARKER_GAP_PX = 25;
/** How many of them a card lists before it stops and says how many are left. */
const CLUSTER_LISTED = 8;

/** What the hover card is open on: a group of bars, or one marker. */
interface CardSubject {
    label: string;
    /** Only a group of bars has a span to break down. */
    total?: string;
    shares?: RenderedShare[];
    details: Array<TimelineDetail>;
}

/**
 * Draws lanes of segments and markers against a time axis.
 *
 * The component knows nothing of what it draws: it is given sections, lanes, segments and markers,
 * paints them from their `kind` and tells back which one was activated. Everything it owns is about
 * showing time — the axis, folding idle stretches, zooming, panning, expanding a lane.
 */
@Component({
    standalone: false,
    selector: 'app-timeline',
    templateUrl: './timeline.html',
    styleUrls: ['./timeline.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class TimelineComponent implements OnChanges, OnDestroy {
    @Input() data: TimelineData;
    /**
     * Fold the stretches of time where no segment runs, so that a long wait does not squash the rest.
     * On by default, and there is no control for it: a timeline that cannot fold is unreadable as soon
     * as anything waits, so this is a property of the host's data rather than a user preference.
     */
    @Input() foldIdleGaps: boolean = true;
    @Input() foldGapsLongerThanMs: number = DEFAULT_SCALE_OPTIONS.foldGapsLongerThanMs;
    /** Dim every lane that is not highlighted, to bring out the ones that are. */
    @Input() highlightEnabled: boolean = false;
    /** Width of the column holding the lane labels. */
    @Input() labelWidth: string = '260px';
    @Input() label: string = 'Timeline';
    @Input() emptyLabel: string = 'Nothing to show yet';
    /** What several markers drawn as one are called: `12 files`, `12 uploads`, `12 releases`. */
    @Input() markersLabel: string = 'markers';
    /** Drawn as hovered, so that pointing at something elsewhere can point at it here. */
    @Input() hoveredLaneID: string;
    /** Drawn as the one currently open, so the view says what is being looked at. */
    @Input() selectedLaneID: string;
    /**
     * The present, for segments that have not ended. Left out, the component keeps its own clock so
     * that an open segment grows on screen.
     */
    @Input() now: number;

    @Output() activate = new EventEmitter<TimelineActivation>();

    @ViewChild('cardTemplate') cardTemplate: TemplateRef<any>;
    /** Moved by hand as the pointer goes, so nothing here is a binding. */
    @ViewChild('cursorLine') cursorLine: ElementRef<HTMLElement>;
    @ViewChild('cursorLabel') cursorLabel: ElementRef<HTMLElement>;

    /**
     * Setters rather than plain queries: the axis and the lanes only exist once there is something to
     * draw, so whatever watches them has to be attached when they appear, not when the view is built.
     */
    @ViewChild('axisTrack') set axisTrackRef(ref: ElementRef<HTMLElement>) {
        this.axisTrack = ref;
        this.observeAxis();
    }
    private axisTrack: ElementRef<HTMLElement>;

    @ViewChild('viewport') set viewportRef(ref: ElementRef<HTMLElement>) {
        this.viewport = ref;
        this.watchPointer();
    }
    viewport: ElementRef<HTMLElement>;

    sections: RenderedSection[] = [];
    ticks: RenderedTick[] = [];
    folds: RenderedFold[] = [];
    startCap: RenderedCap;
    endCap: RenderedCap;
    empty: boolean = true;
    /** Total span of the axis, shown next to the controls. */
    totalLabel: string;

    expandedAll: boolean = false;

    private scale: TimelineScale;
    private viewFrom: number = 0;
    private viewTo: number = 1;
    private expanded: { [laneID: string]: boolean } = {};
    private trackWidth: number = 0;
    /** What each end of the axis keeps clear of the track, in pixels. Decided on every layout. */
    private ends: { start: number, end: number } = { start: AXIS_BAND_PX, end: AXIS_BAND_PX };
    private clock: any;
    private tick: number = Date.now();
    private resizeObserver: ResizeObserver;
    private destroyed: boolean = false;
    private panPointer: number = null;
    private panFrom: { x: number, viewFrom: number, viewTo: number };
    /** A drag that panned must not be taken for a click on whatever it started over. */
    private panned: boolean = false;

    // The hover card.
    private cardTimer: any;
    private cardOverlay: OverlayRef;
    private cardPosition: FlexibleConnectedPositionStrategy;
    private cardContext: { $implicit: CardSubject };
    /** What the card is open on — `lane:<id>` or `marker:<id>` — so a refresh can keep it up to date. */
    private cardKey: string;
    /** Whether the pointer is in the lanes, which holds the clock so nothing slides out from under it. */
    private pointerInside: boolean = false;
    /** One scroll to the foot of the lanes pending, so a burst of updates does not queue a dozen. */
    private followQueued: boolean = false;
    /** The lane to bring into view once the render lands, and whether one scroll is already pending. */
    private revealID: string = null;
    private revealQueued: boolean = false;
    /** Where the pointer is, in client coordinates: what the guide and the card are placed against. */
    private pointer = { x: 0, y: 0 };
    private unwatchPointer: () => void;

    private _cd = inject(ChangeDetectorRef);
    private _zone = inject(NgZone);
    private _overlay = inject(Overlay);
    private _viewContainer = inject(ViewContainerRef);
    private _injector = inject(Injector);

    ngOnChanges(changes: SimpleChanges): void {
        this.layout();
        // Which lane the two views agree is being looked at outranks which one is merely being pointed at.
        const asked = changes['selectedLaneID']?.currentValue ?? changes['hoveredLaneID']?.currentValue;
        if (asked) {
            this.revealLane(asked);
        }
    }

    ngOnDestroy(): void {
        this.destroyed = true;
        this.stopClock();
        this.clearCardTimer();
        this.hideCard();
        this.unwatchPointer?.();
        this.cardOverlay?.dispose();
        this.resizeObserver?.disconnect();
    }

    /** The width of the axis decides how many graduations fit and which segments can hold a label. */
    private observeAxis(): void {
        this.resizeObserver?.disconnect();
        if (!this.axisTrack || this.destroyed) {
            return;
        }
        this.resizeObserver = new ResizeObserver(entries => {
            const width = entries[0]?.contentRect?.width ?? 0;
            if (this.destroyed || Math.abs(width - this.trackWidth) < 1) {
                return;
            }
            // Whether the callback of a ResizeObserver runs inside the zone is not something to rely
            // on: going through it makes sure the new width is drawn.
            this._zone.run(() => {
                this.trackWidth = width;
                this.layout();
                this._cd.markForCheck();
            });
        });
        this.resizeObserver.observe(this.axisTrack.nativeElement);
    }

    get zoomed(): boolean {
        return this.viewFrom > 0 || this.viewTo < 1;
    }

    get panning(): boolean {
        return this.panned;
    }

    // -- Display controls ----------------------------------------------------------------------

    clickToggleAll(): void {
        this.expandedAll = !this.expandedAll;
        this.expanded = {};
        if (this.expandedAll) {
            flattenLanes(this.data).filter(l => l.lanes?.length > 0).forEach(l => this.expanded[l.id] = true);
        }
        this.layout();
        this._cd.markForCheck();
    }

    clickToggleLane(laneID: string, event?: Event): void {
        event?.stopPropagation();
        this.expanded[laneID] = !this.expanded[laneID];
        this.layout();
        this._cd.markForCheck();
    }

    /** Zooming is on the wheel and the keys; the only button for it is the one that undoes it. */
    zoomIn(): void {
        this.zoomAt(0.5, 1 / ZOOM_FACTOR);
    }

    zoomOut(): void {
        this.zoomAt(0.5, ZOOM_FACTOR);
    }

    resetZoom(): void {
        this.viewFrom = 0;
        this.viewTo = 1;
        this.layout();
        this._cd.markForCheck();
    }

    // -- Interactions --------------------------------------------------------------------------

    /**
     * The row and the bars are asked different questions, and each answers its own.
     *
     * The row — the name, and the empty track beside the bars — is *about* the lane: clicking it opens
     * the lane up and shows what is inside. The bars *are* the lane: clicking them opens the thing
     * itself. A lane with nothing inside it has only the second answer to give.
     */
    clickRow(lane: RenderedLane, event?: Event): void {
        if (this.panned) {
            return;
        }
        // The click lands on the column around the name rather than on the name itself, so nothing would
        // be focused and every key that acts on the focused lane would do nothing until something was
        // tabbed to. Whatever was clicked, the lane it belongs to takes the focus.
        this.focusLane(event);
        if (lane.expandable) {
            this.clickToggleLane(lane.id);
            return;
        }
        this.clickLane(lane);
    }

    clickGroup(lane: RenderedLane, event: Event): void {
        event.stopPropagation();
        this.focusLane(event);
        this.clickLane(lane);
    }

    /** Puts the focus on the lane an event happened in, so the keys have something to act on. */
    private focusLane(event?: Event): void {
        const row = (event?.currentTarget as HTMLElement)?.closest('.lane');
        row?.querySelector<HTMLElement>('.lane-label')?.focus({ preventScroll: true });
    }

    /**
     * Zooming from the keyboard, wherever the focus is inside the view. A lane deals with these first and
     * stops them, so this only sees the ones that reached no lane — the focus sitting on one of the
     * controls, say, which is where it lands after using one.
     */
    onTimelineKeyDown(event: KeyboardEvent): void {
        switch (event.key) {
            case '+':
            case '=':
                event.preventDefault();
                event.stopPropagation();
                this.zoomIn();
                return;
            case '-':
                event.preventDefault();
                event.stopPropagation();
                this.zoomOut();
                return;
            case '0':
                event.preventDefault();
                event.stopPropagation();
                this.resetZoom();
                return;
        }
    }

    clickLane(lane: RenderedLane): void {
        if (this.panned || !lane.activatable) {
            return;
        }
        this.activate.emit({ laneID: lane.id });
    }

    clickMarker(lane: RenderedLane, marker: RenderedMarker, event: Event): void {
        event.stopPropagation();
        if (this.panned) {
            return;
        }
        // Several drawn as one stand for no single thing to open. Zooming into what they cover pulls them
        // apart, and each is then its own mark again — which is also why the view, and not the host, is
        // what decides they were too close in the first place.
        if (marker.count > 1) {
            this.zoomToRange(marker.from, marker.to);
            return;
        }
        this.activate.emit({ laneID: lane.id, markerID: marker.id });
    }

    /**
     * Bring a stretch of the axis into view, wide enough that what is in it has room to separate. A
     * stretch of no length cannot be pulled apart by any amount of zoom — those markers really are
     * simultaneous — so the window is left alone and the card is what tells them apart.
     */
    private zoomToRange(from: number, to: number): void {
        const a = this.scale.ratio(from);
        const b = this.scale.ratio(to);
        if (!(b > a)) {
            return;
        }
        const span = Math.min(Math.max((b - a) * 3, MIN_SPAN), 1);
        this.setWindow((a + b) / 2 - span / 2, span);
    }

    /**
     * Keys handled here are stopped from bubbling. A host page is free to listen for arrows and Enter on
     * the window to drive something of its own, and it must not act on a key that was meant for a lane.
     */
    onKeyDown(event: KeyboardEvent, lane: RenderedLane): void {
        switch (event.key) {
            case 'ArrowDown':
            case 'ArrowUp':
                event.preventDefault();
                event.stopPropagation();
                this.moveFocus(event.key === 'ArrowDown' ? 1 : -1);
                return;
            case 'ArrowRight':
                if (lane.expandable && !lane.expanded) {
                    event.preventDefault();
                    event.stopPropagation();
                    this.clickToggleLane(lane.id);
                }
                return;
            case 'ArrowLeft':
                if (lane.expandable && lane.expanded) {
                    event.preventDefault();
                    event.stopPropagation();
                    this.clickToggleLane(lane.id);
                }
                return;
            case '+':
            case '=':
                event.preventDefault();
                event.stopPropagation();
                this.zoomIn();
                return;
            case '-':
                event.preventDefault();
                event.stopPropagation();
                this.zoomOut();
                return;
            case '0':
                event.preventDefault();
                event.stopPropagation();
                this.resetZoom();
                return;
            case 'Enter':
            case ' ':
                // Opens the lane, where a click on the row unfolds it. Stopped from turning into a click,
                // which would reach the row and unfold instead; and from reaching the window, which would
                // hand it to the host page.
                event.preventDefault();
                event.stopPropagation();
                this.clickLane(lane);
                return;
        }
    }

    /**
     * Nothing here uses Shift as a modifier, on purpose. A host page may well treat a Shift keydown
     * anywhere as a shortcut of its own, and the wheel needs no focus, so a Shift wheel over the lanes
     * would reach it without the pointer ever having been near what it drives.
     */
    onWheel(event: WheelEvent): void {
        if (event.ctrlKey || event.metaKey) {
            event.preventDefault();
            const step = event.deltaY !== 0 ? event.deltaY : event.deltaX;
            const rect = this.axisTrack?.nativeElement?.getBoundingClientRect();
            const at = rect && rect.width > 0 ? (event.clientX - rect.left) / rect.width : 0.5;
            this.zoomAt(Math.min(Math.max(at, 0), 1), step > 0 ? ZOOM_FACTOR : 1 / ZOOM_FACTOR);
            return;
        }
        // Sideways on a trackpad pans; straight up and down is left to the panel, which scrolls the lanes.
        if (this.zoomed && Math.abs(event.deltaX) > Math.abs(event.deltaY)) {
            event.preventDefault();
            this.panBy(event.deltaX / 800);
        }
    }

    onPointerDown(event: PointerEvent): void {
        this.panned = false;
        if (!this.zoomed || event.button !== 0) {
            return;
        }
        if ((event.target as HTMLElement)?.closest('button')) {
            return;
        }
        this.panPointer = event.pointerId;
        this.panFrom = { x: event.clientX, viewFrom: this.viewFrom, viewTo: this.viewTo };
    }

    onPointerMove(event: PointerEvent): void {
        if (this.panPointer !== event.pointerId || !this.panFrom) {
            return;
        }
        const delta = event.clientX - this.panFrom.x;
        if (!this.panned && Math.abs(delta) < 3) {
            return;
        }
        // The pointer is only captured once the press turns out to be a drag. Capturing it on the press
        // would retarget the click that ends it to the viewport, and clicking a lane while zoomed in
        // would do nothing at all.
        if (!this.panned) {
            this.panned = true;
            this.hideCard();
            this.viewport?.nativeElement?.setPointerCapture(event.pointerId);
        }
        const span = this.panFrom.viewTo - this.panFrom.viewFrom;
        const moved = this.trackWidth > 0 ? -(delta / this.trackWidth) * span : 0;
        this.setWindow(this.panFrom.viewFrom + moved, span);
    }

    onPointerUp(event: PointerEvent): void {
        if (this.panPointer !== event.pointerId) {
            return;
        }
        if (this.panned) {
            this.viewport?.nativeElement?.releasePointerCapture(event.pointerId);
        }
        this.panPointer = null;
        this.panFrom = null;
        // Released before the click event: the click that ends a pan is dropped, the next one is not.
        setTimeout(() => this.panned = false);
    }

    // -- What a lane says when hovered ---------------------------------------------------------

    /**
     * The card is asked for by the group of bars, not by the row: pointing at the name of a lane, or at
     * the empty track next to its bars, is not asking how its time was spent.
     *
     * It goes in an overlay of its own, anchored above what it is about. Drawn inside the timeline it was
     * cut off by the panel, which is a few hundred pixels tall and clips what overflows it. It does not
     * follow the pointer: a card that moves is a card that has to be read while moving, and where it
     * will appear is worth being able to predict.
     */
    onGroupEnter(lane: RenderedLane, event: MouseEvent): void {
        this.showLaneCard(lane, event.currentTarget as HTMLElement);
    }

    private showLaneCard(lane: RenderedLane, group: HTMLElement): void {
        this.askForCard({
            label: lane.label,
            total: lane.total,
            shares: lane.shares,
            details: lane.details
        }, group, `lane:${lane.id}`);
    }

    /**
     * A marker has something of its own to say — what it is, and when — so it says it instead of letting
     * the group speak for it. It sits inside the group, so without this, pointing at one would leave the
     * group's card up over something it is not about.
     */
    onMarkerEnter(marker: RenderedMarker, event: MouseEvent): void {
        event.stopPropagation();
        this.askForCard({
            label: marker.label,
            details: marker.details
        }, event.currentTarget as HTMLElement, `marker:${marker.id}`);
    }

    /**
     * A marker sits inside its group, so leaving it does not leave the group — nothing would fire to put
     * the group's card back, and it would stay gone until the group was left and entered again. Handing
     * it back here is what keeps moving from a marker to the bar under it working.
     */
    onMarkerLeave(lane: RenderedLane, event: MouseEvent): void {
        const group = (event.currentTarget as HTMLElement)?.closest('.segment-group') as HTMLElement;
        if (!group) {
            this.onCardLeave();
            return;
        }
        this.showLaneCard(lane, group);
    }

    /**
     * Going from one segment of a group to the next never gets here: they are one group, and the group is
     * one element. This is for leaving a group or a marker, and the short wait is only so that going from
     * one to the next carries the card across instead of starting its delay again.
     */
    onCardLeave(): void {
        this.clearCardTimer();
        this.cardTimer = setTimeout(() => {
            this.cardTimer = null;
            this.hideCard();
            this._cd.markForCheck();
        }, 100);
    }

    /** Drop the card at once, for what has moved the bars out from under it. */
    dropCard(): void {
        this.clearCardTimer();
        this.hideCard();
    }

    private askForCard(subject: CardSubject, anchor: HTMLElement, key: string): void {
        this.clearCardTimer();
        if (this.panPointer !== null || this.destroyed || !anchor) {
            return;
        }
        // Already up on something else: swap it over at once, having already waited once.
        if (this.cardOverlay?.hasAttached()) {
            this.showCard(subject, anchor, key);
            return;
        }
        this.cardTimer = setTimeout(() => {
            this.cardTimer = null;
            this.showCard(subject, anchor, key);
        }, CARD_DELAY);
    }

    private showCard(subject: CardSubject, anchor: HTMLElement, key: string): void {
        if (this.destroyed || !this.cardTemplate || !anchor.isConnected) {
            return;
        }

        // Above what it is about, and lined up with the pointer where it was when the card was asked
        // for — close to what is being read, and then still. A card placed dead centre of a bar that
        // spans the view is nowhere near the pointer; one that follows it has to be read while moving.
        const rect = anchor.getBoundingClientRect();
        const origin = { x: this.pointer.x, y: rect.top, width: 0, height: rect.height };

        if (!this.cardOverlay) {
            this.cardPosition = this._overlay.position()
                .flexibleConnectedTo(origin)
                .withPositions([
                    { originX: 'center', originY: 'top', overlayX: 'center', overlayY: 'bottom', offsetY: -10 },
                    { originX: 'center', originY: 'bottom', overlayX: 'center', overlayY: 'top', offsetY: 10 }
                ])
                // Pushed back inside the window rather than flipped sideways, so it never jumps corners.
                .withPush(true)
                .withFlexibleDimensions(false)
                .withViewportMargin(8);
            this.cardOverlay = this._overlay.create({
                positionStrategy: this.cardPosition,
                scrollStrategy: this._overlay.scrollStrategies.noop(),
                hasBackdrop: false,
                disposeOnNavigation: true
            });
            // Nothing on the card is meant to be aimed at.
            this.cardOverlay.overlayElement.style.pointerEvents = 'none';
        }

        this.cardKey = key;
        this.cardPosition.setOrigin(origin);

        // The card is an embedded view of this component's own container, so it is checked when this
        // component is: marking is enough, and calling detection by hand from inside a pass would be
        // checking a view that is already being checked.
        if (this.cardOverlay.hasAttached()) {
            this.cardContext.$implicit = subject;
            this.cardOverlay.updatePosition();
            this._cd.markForCheck();
            return;
        }
        this.cardContext = { $implicit: subject };
        this.cardOverlay.attach(new TemplatePortal(this.cardTemplate, this._viewContainer, this.cardContext));
        this._cd.markForCheck();
    }

    private hideCard(): void {
        this.cardKey = null;
        if (this.cardOverlay?.hasAttached()) {
            this.cardOverlay.detach();
        }
    }

    /** Keeps what the card says in step with the data while it stays open on something still going. */
    private refreshCard(): void {
        if (!this.cardKey?.startsWith('lane:') || !this.cardContext) {
            return;
        }
        const id = this.cardKey.substring('lane:'.length);
        const lane = this.sections
            .reduce((all, section) => all.concat(section.lanes), <RenderedLane[]>[])
            .find(l => l.id === id);
        if (!lane) {
            this.hideCard();
            return;
        }
        this.cardContext.$implicit = {
            label: lane.label,
            total: lane.total,
            shares: lane.shares,
            details: lane.details
        };
    }

    /**
     * Follows the pointer across the lanes, for two things at once.
     *
     * A **guide** is drawn down the lanes where the pointer is, with the time it stands for shown up on
     * the axis: on a folded axis, working out what a position means is otherwise guesswork.
     *
     * And the **clock is held** while the pointer is in there. An axis with an open segment on it redraws
     * itself every second, which moves every bar a little to the left as it grows, and aiming at something
     * while it slides away is not something anyone should have to do.
     *
     * All of it runs outside the zone, writing the two elements it moves by hand: a mouse move changes
     * nothing else that has to be rendered, and running change detection on every one of them would cost
     * the whole page.
     */
    private watchPointer(): void {
        this.unwatchPointer?.();
        const element = this.viewport?.nativeElement;
        if (!element) {
            return;
        }

        const onMove = (event: MouseEvent) => {
            this.pointer = { x: event.clientX, y: event.clientY };
            this.drawGuide();
        };
        const onEnter = () => {
            this.pointerInside = true;
        };
        const onLeave = () => {
            this.pointerInside = false;
            this.hideGuide();
            // Catch up on everything the clock did not draw while it was held.
            this._zone.run(() => {
                this.layout();
                this._cd.markForCheck();
            });
        };

        this._zone.runOutsideAngular(() => {
            element.addEventListener('mousemove', onMove, { passive: true });
            element.addEventListener('mouseenter', onEnter, { passive: true });
            element.addEventListener('mouseleave', onLeave, { passive: true });
        });
        this.unwatchPointer = () => {
            element.removeEventListener('mousemove', onMove);
            element.removeEventListener('mouseenter', onEnter);
            element.removeEventListener('mouseleave', onLeave);
            this.unwatchPointer = null;
        };
    }

    /** Where the pointer is, and what time that is. Written straight to the DOM, outside the zone. */
    private drawGuide(): void {
        const line = this.cursorLine?.nativeElement;
        const label = this.cursorLabel?.nativeElement;
        const track = this.axisTrack?.nativeElement;
        if (!line || !label || !track || !this.scale) {
            return;
        }

        const rect = track.getBoundingClientRect();
        const x = this.pointer.x - rect.left;
        // Only over the part of the track things are drawn in: what the ends keep clear stands for no one
        // instant, so there is no time to read off it.
        if (x < this.ends.start || x > rect.width - this.ends.end) {
            this.hideGuide();
            return;
        }

        const ratio = (x - this.ends.start) / Math.max(rect.width - this.ends.start - this.ends.end, 1);
        const at = this.scale.time(this.viewFrom + ratio * (this.viewTo - this.viewFrom));

        line.style.left = `${x}px`;
        line.style.display = 'block';
        label.style.left = `${x}px`;
        label.style.display = 'block';
        label.textContent = TimelineComponent.clock(at);
    }

    private hideGuide(): void {
        const line = this.cursorLine?.nativeElement;
        const label = this.cursorLabel?.nativeElement;
        if (line) {
            line.style.display = 'none';
        }
        if (label) {
            label.style.display = 'none';
        }
    }

    private static clock(at: number): string {
        const pad = (value: number) => value < 10 ? `0${value}` : `${value}`;
        const date = new Date(at);
        return `${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`;
    }

    private clearCardTimer(): void {
        if (this.cardTimer) {
            clearTimeout(this.cardTimer);
            this.cardTimer = null;
        }
    }

    // -- Layout --------------------------------------------------------------------------------

    private zoomAt(anchor: number, factor: number): void {
        const span = this.viewTo - this.viewFrom;
        const next = Math.min(Math.max(span * factor, MIN_SPAN), 1);
        const focus = this.viewFrom + anchor * span;
        this.setWindow(focus - (focus - this.viewFrom) * (next / span), next);
    }

    private panBy(ratio: number): void {
        const span = this.viewTo - this.viewFrom;
        this.setWindow(this.viewFrom + ratio * span, span);
    }

    private setWindow(from: number, span: number): void {
        this.viewFrom = Math.min(Math.max(from, 0), 1 - span);
        this.viewTo = this.viewFrom + span;
        this.layout();
        this._cd.markForCheck();
    }

    /** Where a date lands in the window, in [0, 1] of the visible width. */
    private position(time: number): number {
        const span = this.viewTo - this.viewFrom || 1;
        return (this.scale.ratio(time) - this.viewFrom) / span;
    }

    private static pct(value: number): string {
        return `${(value * 100).toFixed(4)}%`;
    }

    /** What each end of the axis keeps clear, as a fraction of the width of the track. */
    private get endFractions(): { start: number, end: number } {
        const track = this.trackWidth || 800;
        return { start: this.ends.start / track, end: this.ends.end / track };
    }

    /**
     * Where a position of the axis is drawn across the track. Everything is squeezed into the space the
     * two ends leave, so nothing is ever drawn into what they keep clear.
     */
    private trackPct(ratio: number): string {
        const ends = this.endFractions;
        return TimelineComponent.pct(ends.start + ratio * (1 - ends.start - ends.end));
    }

    /** The same squeeze, applied to a width rather than to a position. */
    private trackSpan(width: number): string {
        const ends = this.endFractions;
        return TimelineComponent.pct(width * (1 - ends.start - ends.end));
    }

    /** Width of the part of the track things are drawn in, in pixels. */
    private get plotWidth(): number {
        return Math.max((this.trackWidth || 800) - this.ends.start - this.ends.end, 1);
    }

    private layout(): void {
        const lanes = flattenLanes(this.data);
        this.empty = lanes.length === 0;
        if (this.empty) {
            this.sections = [];
            this.ticks = [];
            this.folds = [];
            this.totalLabel = null;
            this.stopClock();
            return;
        }

        const open = lanes.some(lane => (lane.segments ?? []).some(s => s.end === null || s.end === undefined));
        if (open && this.now === undefined) {
            this.startClock();
        } else {
            this.stopClock();
        }
        const present = this.now !== undefined ? this.now : this.tick;

        const busy: TimelineBusy[] = [];
        let start = Infinity;
        let end = -Infinity;
        // What the ends of the axis have to make room for, on top of the bands themselves: whether
        // anything on the axis lasted at all, and where the outermost instants are. See `axisEnds`.
        let lasting = false;
        let firstMark = Infinity;
        let lastMark = -Infinity;
        lanes.forEach(lane => {
            (lane.segments ?? []).forEach(segment => {
                const to = segment.end ?? present;
                // An idle segment still sets the bounds of the axis, but it does not keep its stretch
                // of time from being folded: waiting is not something happening.
                if (!segment.idle) {
                    busy.push({ start: segment.start, end: Math.max(to, segment.start) });
                }
                lasting = lasting || to > segment.start;
                start = Math.min(start, segment.start);
                end = Math.max(end, to, segment.start);
            });
            // A group is indivisible: whatever sits between two of its segments holds the axis open, so
            // no fold can open inside it and its pieces keep their proportions to one another. This says
            // nothing about the inside of a segment: a long wait drawn as one idle segment is exactly what
            // folding is for, and folding straight through it has to stay possible.
            busy.push(...TimelineComponent.gapsWithinGroups(lane, present));
            // A marker holds a stretch of no length: it keeps its instant out of a fold, so that what
            // happened during a long wait stays visible instead of being folded away with it.
            (lane.markers ?? []).forEach(marker => {
                busy.push({ start: marker.at, end: marker.at });
                firstMark = Math.min(firstMark, marker.at);
                lastMark = Math.max(lastMark, marker.at);
                start = Math.min(start, marker.at);
                end = Math.max(end, marker.at);
            });
        });

        start = this.data.start ?? (isFinite(start) ? start : present);
        end = Math.max(this.data.end ?? (isFinite(end) ? end : present), start);

        // How much of the track the ends of the axis need. Decided before anything is placed, since
        // everything is placed between the two.
        this.ends = axisEnds({
            trackWidth: this.trackWidth || 800,
            lasting,
            markAtStart: firstMark <= start,
            markAtEnd: lastMark >= end
        });

        this.scale = TimelineScale.build(start, end, busy, {
            foldGapsLongerThanMs: this.foldIdleGaps ? this.foldGapsLongerThanMs : 0,
            foldedGapShare: DEFAULT_SCALE_OPTIONS.foldedGapShare
        });
        this.totalLabel = formatDuration(end - start);

        this.sections = (this.data.sections ?? []).map(section => ({
            id: section.id,
            label: section.label,
            lanes: this.renderLanes(section.lanes, 0, present, false)
        })).filter(section => section.lanes.length > 0 || !!section.label);

        this.folds = this.scale.folds
            .map((fold, i) => this.renderFold(fold, i))
            .filter(fold => !!fold);

        // The ends of the axis are marked rather than left to be guessed: a bar reaching the edge of
        // the view says nothing about whether there is more of it past that edge. When zoomed in, the marks
        // fall outside the window and are not drawn, which is itself the answer.
        this.startCap = this.renderCap(start, 'Start', false);
        this.endCap = this.renderCap(end, open ? 'Now' : 'End', open);

        const maxTicks = Math.max(Math.floor(this.plotWidth / 90), 2);
        // The labels at either end have their own room: a graduation crowding them would leave two
        // times printed on top of one another.
        const margin = CAP_MARGIN_PX / this.plotWidth;
        const capsAt = [this.startCap, this.endCap].filter(cap => !!cap).map(cap => cap.ratio);
        this.ticks = this.scale.ticks(this.viewFrom, this.viewTo, maxTicks)
            .map(t => ({ id: t.at, ratio: this.position(t.at), label: t.label }))
            .filter(t => capsAt.every(cap => Math.abs(t.ratio - cap) > margin))
            .map(t => ({ id: t.id, left: this.trackPct(t.ratio), label: t.label }));

        // A card left open on something still going would otherwise freeze on the numbers it opened with.
        this.refreshCard();

        // Decided here rather than after drawing: once the new lanes are in, the foot of the list has
        // moved and there is no telling whether it was being watched.
        this.keepUpWithTheData(open);
    }

    /**
     * Lanes are added at the foot of the list as they begin, so a view that stays where it was shows less
     * of the whole the longer it goes on. So the lanes are followed down, the way a log tail is.
     *
     * Only while nothing has been asked for, though. Zooming in, or scrolling away from the foot of the
     * list, is someone saying they are reading something in particular — and what they are reading must
     * not be dragged out from under them. Scrolling back to the foot picks the following up again, which
     * is why this asks where the list is rather than remembering that it was once left.
     */
    private keepUpWithTheData(live: boolean): void {
        const element = this.viewport?.nativeElement;
        if (!element?.offsetParent || this.followQueued) {
            return;
        }
        const room = element.scrollHeight - element.scrollTop - element.clientHeight;
        if (!shouldFollow(live, this.zoomed, this.pointerInside, room)) {
            return;
        }
        this.followQueued = true;
        afterNextRender(() => {
            this.followQueued = false;
            const view = this.viewport?.nativeElement;
            if (view && !this.pointerInside) {
                view.scrollTop = view.scrollHeight;
            }
        }, { injector: this._injector });
    }

    /**
     * Bring a lane into view. Pointing at a job in the graph, or opening one from anywhere, brings out its
     * lane here — which says nothing at all if that lane is below the fold.
     *
     * Vertically only, and by the least that does it: the axis is where someone put it and moving it
     * sideways would be answering a question nobody asked, and a lane already in view is not a reason to
     * move anything. A lane that has gone with the data it stood for is simply not found, and nothing moves.
     *
     * This never fights the lanes being followed down (§8.1), and does not have to be made to: following
     * only happens while the view is sitting at the foot of the list, which a view that has just been
     * scrolled up to a lane is not.
     */
    private revealLane(id: string): void {
        this.revealID = id;
        if (this.revealQueued) {
            return;
        }
        this.revealQueued = true;
        // After the render, because the lane asked for may be one that is only arriving with this change.
        afterNextRender(() => {
            this.revealQueued = false;
            const wanted = this.revealID;
            this.revealID = null;
            const view = this.viewport?.nativeElement;
            if (!view?.offsetParent || !wanted) {
                return;
            }
            const lane = Array.from(view.querySelectorAll<HTMLElement>('.lane'))
                .find(element => element.dataset.laneId === wanted);
            if (!lane) {
                return;
            }
            const by = revealBy(view.getBoundingClientRect(), lane.getBoundingClientRect());
            if (by === 0) {
                return;
            }
            // Somebody who has asked for less motion is not asking for the view to slide there.
            const still = window.matchMedia?.('(prefers-reduced-motion: reduce)')?.matches;
            view.scrollBy({ top: by, behavior: still ? 'auto' : 'smooth' });
        }, { injector: this._injector });
    }

    private renderCap(at: number, name: string, live: boolean): RenderedCap {
        const ratio = this.position(at);
        if (ratio < 0 || ratio > 1) {
            return null;
        }
        return {
            ratio,
            left: this.trackPct(ratio),
            label: name,
            live,
            title: `${name} of the timeline — ${new Date(at).toLocaleString()}`
        };
    }

    private renderFold(fold: TimelineScalePiece, index: number): RenderedFold {
        const from = (fold.from - this.viewFrom) / (this.viewTo - this.viewFrom || 1);
        const to = (fold.to - this.viewFrom) / (this.viewTo - this.viewFrom || 1);
        if (to < 0 || from > 1) {
            return null;
        }
        const label = TimelineScale.foldLabel(fold);
        return {
            id: `fold-${index}`,
            left: this.trackPct(Math.max(from, -0.05)),
            width: this.trackSpan(Math.min(to, 1.05) - Math.max(from, -0.05)),
            label,
            title: `Nothing ran for ${label}, folded to keep the rest readable`
        };
    }

    private renderLanes(lanes: TimelineLane[], depth: number, present: number, parentHighlighted: boolean): RenderedLane[] {
        const rendered: RenderedLane[] = [];
        (lanes ?? []).forEach(lane => {
            const expandable = (lane.lanes ?? []).length > 0;
            const expanded = expandable && !!this.expanded[lane.id];
            const span = TimelineComponent.spanOf(lane, present);
            // What is inside a highlighted lane belongs to it: dimming the lanes nested under a
            // highlighted one would hide the very thing the highlight was turned on to look at.
            const highlighted = !!lane.highlighted || parentHighlighted;
            const track = this.renderTrack(lane, present);
            rendered.push({
                id: lane.id,
                label: lane.label,
                sublabel: lane.sublabel,
                status: lane.status ?? '',
                depth,
                expandable,
                expanded,
                activatable: !!lane.activatable,
                highlighted,
                description: TimelineComponent.describe(lane, present),
                total: span > 0 ? formatDuration(span) : null,
                shares: TimelineComponent.shares(lane, present, span),
                details: lane.details ?? [],
                ...track,
                hasMarkers: track.markers.length > 0 || track.groups.some(g => g.markers.length > 0)
            });
            if (expanded) {
                rendered.push(...this.renderLanes(lane.lanes, depth + 1, present, highlighted));
            }
        });
        return rendered;
    }

    /**
     * What goes in the track of a lane: its groups, each holding the segments it is made of and the
     * markers that fall inside it, and whatever markers fall outside all of them.
     *
     * A group is placed against the track, and everything inside it against the group. Cutting the group
     * down to the part of it that can be seen keeps its width bounded however far the axis is zoomed in,
     * and its children move with it, so what is inside stays where it belongs.
     */
    private renderTrack(lane: TimelineLane, present: number): { groups: RenderedGroup[], markers: RenderedMarker[] } {
        const plotPixels = this.plotWidth;
        const groups: RenderedGroup[] = [];
        const taken = new Set<string>();

        TimelineComponent.groupsOf(lane).forEach((segments, index) => {
            const endOf = (segment: TimelineSegment) => Math.max(segment.end ?? present, segment.start);
            const opensAt = Math.min(...segments.map(s => s.start));
            const closesAt = Math.max(...segments.map(endOf));
            const from = this.position(opensAt);
            const to = this.position(closesAt);
            if (to < -0.5 || from > 1.5) {
                return;
            }

            const left = Math.max(from, -0.5);
            const right = Math.min(to, 1.5);
            const width = right - left;
            /** Where a position of the track falls inside the visible part of the group, in [0, 1]. */
            const inside = (at: number) => width > 0 ? Math.min(Math.max((at - left) / width, 0), 1) : 0;

            const rendered: RenderedSegment[] = segments.map(segment => {
                const open = segment.end === null || segment.end === undefined;
                const ends = endOf(segment);
                const starts = this.position(segment.start);
                const stops = this.position(ends);
                const pixels = (stops - starts) * plotPixels;
                const at = inside(starts);
                return <RenderedSegment>{
                    id: segment.id,
                    kind: segment.kind,
                    label: segment.label,
                    duration: formatDuration(ends - segment.start),
                    left: TimelineComponent.pct(at),
                    width: TimelineComponent.pct(Math.max(inside(stops) - at, 0)),
                    showLabel: !!segment.label && pixels > 44,
                    // Read straight off the bar when it is wide enough, rather than only on hover.
                    showDuration: pixels > 96,
                    open
                };
            });

            // A marker sitting in the span of a group is drawn inside it, so that pointing at it is still
            // pointing at the group rather than dropping out of it.
            const mine = (lane.markers ?? []).filter(marker => marker.at >= opensAt && marker.at <= closesAt);
            mine.forEach(marker => taken.add(marker.id));
            // Inside a group, so placed against the group and already squeezed with it. Its own width in
            // pixels is what says whether two of them would overlap.
            const markers = this.cluster(mine, Math.max(width, 0) * plotPixels,
                marker => inside(this.position(marker.at)), TimelineComponent.pct);

            groups.push({
                id: `${lane.id}-group-${index}`,
                left: this.trackPct(left),
                width: this.trackSpan(Math.max(width, 0)),
                segments: rendered,
                markers
            });
        });

        // A loose marker is placed against the track, so it takes the squeeze the bands impose.
        const loose = this.cluster(
            (lane.markers ?? []).filter(marker => !taken.has(marker.id) && this.position(marker.at) >= -0.02 && this.position(marker.at) <= 1.02),
            plotPixels,
            marker => this.position(marker.at),
            ratio => this.trackPct(ratio));

        return { groups, markers: loose };
    }

    /**
     * Markers too close together to be told apart are drawn as one, carrying how many it stands for.
     *
     * This belongs to the view and nowhere else: whether two of them collide is a fact about pixels — it
     * depends on the zoom, on how wide the track is and on how big a marker is drawn — none of which the
     * side providing the data knows. Forty markers a second apart overlap on an axis spanning an hour and
     * do not once it is zoomed in, so the same data has to cluster differently from one moment to the next.
     */
    private cluster(
        markers: TimelineMarker[],
        widthPixels: number,
        ratioOf: (marker: TimelineMarker) => number,
        format: (ratio: number) => string
    ): RenderedMarker[] {
        if (markers.length === 0) {
            return [];
        }
        const ordered = markers.slice().sort((a, b) => a.at - b.at);
        const clusters: TimelineMarker[][] = [];
        let last = -Infinity;
        ordered.forEach(marker => {
            const pixels = ratioOf(marker) * widthPixels;
            if (clusters.length > 0 && pixels - last < MARKER_GAP_PX) {
                clusters[clusters.length - 1].push(marker);
            } else {
                clusters.push([marker]);
            }
            last = pixels;
        });

        return clusters.map(members => {
            const first = members[0];
            const left = format(ratioOf(first));
            if (members.length === 1) {
                return <RenderedMarker>{
                    id: first.id,
                    kind: first.kind ?? 'default',
                    icon: timelineIcon(first.icon),
                    left,
                    label: first.label ?? first.kind ?? '',
                    details: first.details ?? [],
                    count: 1,
                    from: first.at,
                    to: first.at
                };
            }

            // One kind throughout keeps its own icon and its own word; a mixed handful gets a mark saying
            // only "several", and the general word, since no one kind speaks for the rest.
            const kinds = new Set(members.map(m => m.kind));
            const icons = new Set(members.map(m => m.icon));
            const plurals = new Set(members.map(m => m.plural).filter(word => !!word));
            const details: Array<TimelineDetail> = members.slice(0, CLUSTER_LISTED)
                .map(m => ({ label: m.label ?? m.kind ?? '', value: TimelineComponent.clock(m.at) }));
            if (members.length > CLUSTER_LISTED) {
                details.push({ label: `and ${members.length - CLUSTER_LISTED} more`, value: '' });
            }
            return <RenderedMarker>{
                id: `cluster-${first.id}`,
                kind: kinds.size === 1 ? first.kind ?? 'default' : 'cluster',
                icon: icons.size === 1 ? timelineIcon(first.icon) : 'appstore',
                left,
                label: `${members.length} ${plurals.size === 1 ? [...plurals][0] : this.markersLabel}`,
                details,
                count: members.length,
                from: first.at,
                to: members[members.length - 1].at
            };
        });
    }

    /**
     * The stretches that separate the segments of one group, which is what makes a group indivisible on
     * the axis. Where a group's segments run end to end — one phase handing straight over to the next —
     * there are none, and this earns its place as a guarantee rather than as a fix.
     */
    private static gapsWithinGroups(lane: TimelineLane, present: number): TimelineBusy[] {
        const gaps: TimelineBusy[] = [];
        TimelineComponent.groupsOf(lane).forEach(segments => {
            if (segments.length < 2) {
                return;
            }
            const ordered = segments.slice().sort((a, b) => a.start - b.start);
            for (let i = 1; i < ordered.length; i++) {
                const previous = Math.max(ordered[i - 1].end ?? present, ordered[i - 1].start);
                if (ordered[i].start > previous) {
                    gaps.push({ start: previous, end: ordered[i].start });
                }
            }
        });
        return gaps;
    }

    /** The segments of a lane, gathered by group. A segment with no group is a group of its own. */
    private static groupsOf(lane: TimelineLane): TimelineSegment[][] {
        const groups = new Map<string, TimelineSegment[]>();
        (lane.segments ?? []).forEach((segment, index) => {
            const key = segment.group ?? `${segment.id}#${index}`;
            if (!groups.has(key)) {
                groups.set(key, []);
            }
            groups.get(key).push(segment);
        });
        return Array.from(groups.values());
    }

    /** From the start of the first segment to the end of the last: the wall clock of the lane. */
    private static spanOf(lane: TimelineLane, present: number): number {
        const segments = lane.segments ?? [];
        if (segments.length === 0) {
            return 0;
        }
        const from = Math.min(...segments.map(s => s.start));
        const to = Math.max(...segments.map(s => Math.max(s.end ?? present, s.start)));
        return Math.max(to - from, 0);
    }

    /**
     * Where the span of a lane went. This is the point of the whole view: something that took four minutes
     * of which three were spent waiting is a different matter from one that spent them working.
     */
    private static shares(lane: TimelineLane, present: number, span: number): RenderedShare[] {
        if (span <= 0) {
            return [];
        }
        return (lane.segments ?? []).map(segment => {
            const length = Math.max((segment.end ?? present) - segment.start, 0);
            const share = length / span;
            return <RenderedShare>{
                label: segment.label ?? segment.kind,
                duration: formatDuration(length),
                width: TimelineComponent.pct(share),
                percent: `${Math.round(share * 100)}%`
            };
        });
    }

    /** What the lane holds, in words: the bars carry it for the eye, this carries it for the rest. */
    private static describe(lane: TimelineLane, present: number): string {
        const parts = (lane.segments ?? []).map(segment => {
            const to = segment.end ?? present;
            return `${segment.label ?? segment.kind} ${formatDuration(Math.max(to - segment.start, 0))}`;
        });
        const markers = (lane.markers ?? []).length;
        if (markers > 0) {
            parts.push(`${markers} marker${markers > 1 ? 's' : ''}`);
        }
        return parts.join(', ');
    }

    // -- Clock ---------------------------------------------------------------------------------

    private startClock(): void {
        if (this.clock) {
            return;
        }
        this.clock = setInterval(() => {
            // Held while the pointer is in the lanes: growing the axis moves every bar, and nothing
            // should slide away from under a pointer that is trying to reach it. Held too while the view
            // is not on screen — a tab kept alive behind another one has nothing to redraw.
            if (this.pointerInside || !this.viewport?.nativeElement?.offsetParent) {
                return;
            }
            this.tick = Date.now();
            this.layout();
            this._cd.markForCheck();
        }, 1000);
    }

    private stopClock(): void {
        if (!this.clock) {
            return;
        }
        clearInterval(this.clock);
        this.clock = null;
    }

    // -- Focus ---------------------------------------------------------------------------------

    /** Moves focus to the lane above or below, so that arrows walk the lanes without leaving them. */
    private moveFocus(direction: number): void {
        const host = this.viewport?.nativeElement;
        if (!host) {
            return;
        }
        const items = Array.from(host.querySelectorAll<HTMLElement>('.lane-label'));
        const current = items.findIndex(item => item === document.activeElement);
        const next = items[Math.min(Math.max((current === -1 ? 0 : current) + direction, 0), items.length - 1)];
        next?.focus();
    }
}
