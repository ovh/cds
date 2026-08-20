/**
 * What a timeline is drawn from. Nothing here knows what any of it means: whoever provides the data
 * decides what a lane, a segment and a marker stand for, and gets told back which one was activated.
 */

/** A stretch of time on a lane. */
export interface TimelineSegment {
    id: string;
    /** Milliseconds since epoch. */
    start: number;
    /** Milliseconds since epoch, or left out while the stretch is still going. */
    end?: number;
    /**
     * Free-form. Becomes a `kind-<value>` class on the segment, which is how the host paints it:
     * the view gives it no meaning of its own.
     */
    kind: string;
    /**
     * Drawn inside the segment when it is wide enough, and read out as part of the lane. Hovering a
     * segment shows the whole breakdown of its lane rather than anything about the segment alone, so a
     * segment carries no tooltip of its own.
     */
    label?: string;
    /**
     * Waiting rather than working. The segment is drawn like any other, but it does not hold the axis
     * open: a stretch of time covered by nothing but idle segments can be folded.
     *
     * This is what lets a wait of days be drawn at all. Without it, a bar spanning that wait would
     * count as something happening, and the axis would have to give the wait its full length.
     */
    idle?: boolean;
    /**
     * Segments sharing a group are one thing that happens to be drawn in several pieces — successive
     * phases of the same activity, say. A group is:
     *
     * - **one thing to the pointer.** It is drawn as a single element holding its segments, so going
     *   from one piece of it to the next never leaves it, and whatever it shows on hover does not
     *   blink at every boundary.
     * - **indivisible on the axis.** No fold may open between two of its segments, so its pieces keep
     *   their true proportions to one another however much is folded away elsewhere.
     *
     * A segment with no group is its own group.
     */
    group?: string;
}

/** A fact the view cannot work out for itself, shown when what carries it is hovered. */
export interface TimelineDetail {
    label: string;
    value: string;
}

/** A point in time on a lane. */
export interface TimelineMarker {
    id: string;
    /** Milliseconds since epoch. */
    at: number;
    /** Free-form. Becomes a `kind-<value>` class on the marker. */
    kind?: string;
    /** What the marker stands for, the heading of what it says when hovered. */
    label?: string;
    /**
     * Name of an icon to draw instead of the default lozenge. Which one to ask for is the host's business,
     * but only the ones this view carries are drawn — see `timelineIcon`, which says why asking is not
     * enough. Anything else is drawn as a plain mark.
     */
    icon?: string;
    /** Shown under the label when the marker is hovered. */
    details?: Array<TimelineDetail>;
    /**
     * What several of this kind of marker are called, when they are too close together to draw apart and
     * are drawn as one. Left out, they take whatever the view was told markers are called in general.
     */
    plural?: string;
}

export interface TimelineLane {
    id: string;
    label: string;
    /** Second line under the label, for whatever the host wants to tell about the lane. */
    sublabel?: string;
    /**
     * Shown under the breakdown of the lane when it is hovered. The view works out how long each
     * segment lasted and what share of the lane it took; anything else worth saying about the lane is
     * only known to the host.
     */
    details?: Array<TimelineDetail>;
    /** Free-form. Becomes a `status-<value>` class on the lane. */
    status?: string;
    segments: TimelineSegment[];
    markers?: TimelineMarker[];
    /** Lanes drawn under this one when it is expanded. */
    lanes?: TimelineLane[];
    /** Kept bright while the others are dimmed, when the view is asked to highlight. */
    highlighted?: boolean;
    /** Whether activating the lane means anything to the host. A lane that is not stays inert. */
    activatable?: boolean;
}

/** A run of lanes under a common heading. A timeline without headings holds a single unlabelled section. */
export interface TimelineSection {
    id: string;
    /** No heading is drawn when left out. */
    label?: string;
    lanes: TimelineLane[];
}

export interface TimelineData {
    sections: TimelineSection[];
    /** Bounds of the axis. Taken from the segments and the markers when left out. */
    start?: number;
    end?: number;
}

/** What the user activated, either a lane or one of its markers. */
export interface TimelineActivation {
    laneID: string;
    markerID?: string;
}

/** Every lane of a timeline, sections and nesting flattened, parents before their children. */
export function flattenLanes(data: TimelineData): TimelineLane[] {
    const flat: TimelineLane[] = [];
    const walk = (lanes: TimelineLane[]) => (lanes ?? []).forEach(lane => {
        flat.push(lane);
        walk(lane.lanes);
    });
    (data?.sections ?? []).forEach(section => walk(section.lanes));
    return flat;
}

const DURATION_UNITS: Array<[number, string]> = [[86400000, 'd'], [3600000, 'h'], [60000, 'm'], [1000, 's']];

/** How close to the foot of the lanes still counts as being at it, for following them down. */
export const FOLLOW_SLACK_PX = 24;

/**
 * Whether a view should keep up with data that is still arriving, given where its lanes are sitting and
 * what has been asked of it.
 *
 * Lanes are added at the foot of the list as they begin, so a view that stays where it was shows less of
 * the whole the longer it goes on — hence following them down, the way a log tail is. Every clause below
 * is a case where moving the view would be wrong rather than helpful.
 *
 * Note what this does *not* do: remember that the view was once left alone. It asks where the lanes are
 * sitting now, so scrolling back to the foot picks the following up again by itself.
 */
export function shouldFollow(live: boolean, zoomed: boolean, hovering: boolean, roomBelowPx: number): boolean {
    // Nothing is arriving anymore, so there is nothing to keep up with.
    if (!live) {
        return false;
    }
    // Zoomed in on something, or pointing at something: both mean someone is reading.
    if (zoomed || hovering) {
        return false;
    }
    // Scrolled away from the foot of the list: they went to look at something further up, and it is not
    // to be dragged away from them.
    return roomBelowPx <= FOLLOW_SLACK_PX;
}

/** Width of the band walling off either end of the axis. */
export const AXIS_BAND_PX = 12;
/** How much more is kept clear on a side where a marker sits on the bound. */
export const MARKER_ROOM_PX = 14;

/** What decides how much room the ends of an axis need. */
export interface AxisRoom {
    /** Width of the whole track, in pixels. */
    trackWidth: number;
    /** Whether anything on the axis lasted, rather than everything on it being an instant. */
    lasting: boolean;
    /** Whether a marker sits on the first, respectively the last, instant the axis covers. */
    markAtStart: boolean;
    markAtEnd: boolean;
}

/**
 * How much of the track is kept clear at either end, in pixels. Everything else is drawn between the two.
 *
 * This is the only way an axis is given room, and deliberately so: taking it out of the *track* leaves
 * time meaning what it means. Making room by widening the stretch of time instead would invent time that
 * has to be either drawn to scale — where a long axis turns a few pixels of room into hours that then take
 * the whole view — or folded away, which is the room being taken straight back.
 *
 * Beyond the bands themselves, two things ask for room. A marker sitting on a bound is drawn half outside
 * the axis, being a glyph with a size but no length. And content that never lasted — a handful of instants
 * and not one bar, which is what a run that failed before queueing a job comes to — leaves an axis that is
 * nothing but its own two ends: its marks would sit against the two edges of an otherwise empty view, so
 * it is given the middle third and reads as being somewhere instead.
 */
export function axisEnds(axis: AxisRoom): { start: number, end: number } {
    let start = AXIS_BAND_PX + (axis.markAtStart ? MARKER_ROOM_PX : 0);
    let end = AXIS_BAND_PX + (axis.markAtEnd ? MARKER_ROOM_PX : 0);
    const track = Math.max(axis.trackWidth, 1);

    if (!axis.lasting) {
        const third = Math.max((track - start - end) / 3, 0);
        start += third;
        end += third;
    }

    // However narrow the view gets, the plot keeps a fifth of it: a view with no room left to draw in
    // answers nothing at all.
    const room = track * 0.8;
    if (start + end > room) {
        const shrink = room / (start + end);
        return { start: start * shrink, end: end * shrink };
    }
    return { start, end };
}

/**
 * Compact rendering of a duration: the largest unit that carries something, and the one below it. Two
 * days and two seconds reads as `2d`, not as `2d 2s` — at that scale the seconds say nothing.
 */
export function formatDuration(ms: number): string {
    if (!isFinite(ms) || ms < 0) {
        return '';
    }
    const rounded = Math.floor(ms / 1000) * 1000;
    const largest = DURATION_UNITS.findIndex(([size]) => rounded >= size);
    if (largest === -1) {
        return '<1s';
    }

    const [size, suffix] = DURATION_UNITS[largest];
    const value = Math.floor(rounded / size);
    const parts = [`${value}${suffix}`];

    const below = DURATION_UNITS[largest + 1];
    if (below) {
        const rest = Math.floor((rounded - value * size) / below[0]);
        if (rest > 0) {
            parts.push(`${rest}${below[1]}`);
        }
    }
    return parts.join(' ');
}
