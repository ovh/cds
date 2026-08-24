import { formatDuration } from './timeline.model';

/** A stretch of time something occupies. Zero-length is allowed, for a point in time. */
export interface TimelineBusy {
    start: number;
    end: number;
}

export interface TimelineScaleOptions {
    /**
     * Idle stretches longer than this are folded: they keep a fixed, small share of the axis whatever
     * their real length. Left at 0, time is drawn strictly proportional.
     */
    foldGapsLongerThanMs: number;
    /** Share of the axis given to a folded stretch. */
    foldedGapShare: number;
}

export const DEFAULT_SCALE_OPTIONS: TimelineScaleOptions = {
    // Short waits are worth seeing to scale; past a few minutes, a wait only needs to be readable.
    foldGapsLongerThanMs: 300000,
    foldedGapShare: 0.03
};

/** A run of time drawn at a single rate. Folded pieces are drawn at a much smaller one. */
export interface TimelineScalePiece {
    start: number;
    end: number;
    folded: boolean;
    /** Where the piece starts on the axis, in [0, 1]. */
    from: number;
    /** Where the piece ends on the axis, in [0, 1]. */
    to: number;
}

export interface TimelineTick {
    at: number;
    /** Position on the axis, in [0, 1]. */
    ratio: number;
    label: string;
}

/** The steps a time axis is allowed to be graduated with, from a second to a week. */
const TICK_STEPS = [
    1000, 2000, 5000, 10000, 15000, 30000,
    60000, 120000, 300000, 600000, 900000, 1800000,
    3600000, 7200000, 10800000, 21600000, 43200000,
    86400000, 172800000, 604800000
];

/**
 * Maps time to a position on the axis.
 *
 * What is being drawn may sit idle for hours or days between two things worth seeing, which to scale
 * would squash everything that did happen into a few pixels. Stretches where nothing happens are
 * therefore folded: they are still drawn, so that the wait is visible and stays in order, but they no
 * longer cost the axis anything near their real length. Time remains strictly proportional everywhere
 * else, so comparing two segments on either side of a fold is still comparing like with like as long as
 * no fold sits between them — which is what the fold marks are for.
 */
export class TimelineScale {
    readonly start: number;
    readonly end: number;
    readonly pieces: TimelineScalePiece[];

    private constructor(start: number, end: number, pieces: TimelineScalePiece[]) {
        this.start = start;
        this.end = end;
        this.pieces = pieces;
    }

    static build(start: number, end: number, busy: TimelineBusy[], options: TimelineScaleOptions = DEFAULT_SCALE_OPTIONS): TimelineScale {
        // A timeline of no length still has to place its segments somewhere: one millisecond wide is
        // enough for everything to land at the same spot instead of dividing by zero.
        if (!(end > start)) {
            end = start + 1;
        }

        const merged = TimelineScale.merge(busy, start, end);
        const threshold = options.foldGapsLongerThanMs > 0 ? options.foldGapsLongerThanMs : Infinity;

        // Walk the axis once, emitting what is busy and, between two busy stretches, the gap that
        // separates them. Adjacent unfolded pieces are joined so that graduations run through them.
        const pieces: Array<{ start: number, end: number, folded: boolean }> = [];
        const push = (from: number, to: number, folded: boolean) => {
            if (to < from) {
                return;
            }
            const last = pieces[pieces.length - 1];
            if (last && !last.folded && !folded) {
                last.end = to;
                return;
            }
            pieces.push({ start: from, end: to, folded });
        };

        let cursor = start;
        merged.forEach(interval => {
            if (interval.start > cursor) {
                push(cursor, interval.start, interval.start - cursor > threshold);
            }
            push(Math.max(cursor, interval.start), interval.end, false);
            cursor = Math.max(cursor, interval.end);
        });
        if (cursor < end) {
            push(cursor, end, end - cursor > threshold);
        }
        if (pieces.length === 0) {
            pieces.push({ start, end, folded: false });
        }

        // Folded stretches are sized against what is left drawn to scale, so that they stay a small
        // part of the axis however long the wait was.
        const unfolded = pieces.filter(p => !p.folded).reduce((total, p) => total + (p.end - p.start), 0);
        const foldedCount = pieces.filter(p => p.folded).length;
        const share = Math.max(options.foldedGapShare, 0);
        // Everything folded, or nothing but points: fall back to equal shares rather than a zero-length axis.
        const foldedWidth = unfolded > 0 ? unfolded * share : (foldedCount > 0 ? 1 / foldedCount : 1);
        const widths = pieces.map(p => p.folded ? foldedWidth : Math.max(p.end - p.start, 0));
        const total = widths.reduce((a, b) => a + b, 0) || 1;

        let offset = 0;
        const scaled: TimelineScalePiece[] = pieces.map((p, i) => {
            const from = offset / total;
            offset += widths[i];
            return { start: p.start, end: p.end, folded: p.folded, from, to: offset / total };
        });

        return new TimelineScale(start, end, scaled);
    }

    /** Where a date sits on the axis, in [0, 1]. */
    ratio(time: number): number {
        if (time <= this.start) {
            return 0;
        }
        if (time >= this.end) {
            return 1;
        }
        for (const piece of this.pieces) {
            if (time > piece.end) {
                continue;
            }
            const length = piece.end - piece.start;
            if (length <= 0) {
                return piece.from;
            }
            return piece.from + ((time - piece.start) / length) * (piece.to - piece.from);
        }
        return 1;
    }

    /** The date at a position of the axis. Inverse of ratio(). */
    time(ratio: number): number {
        if (ratio <= 0) {
            return this.start;
        }
        if (ratio >= 1) {
            return this.end;
        }
        for (const piece of this.pieces) {
            if (ratio > piece.to) {
                continue;
            }
            const width = piece.to - piece.from;
            if (width <= 0) {
                return piece.start;
            }
            return piece.start + ((ratio - piece.from) / width) * (piece.end - piece.start);
        }
        return this.end;
    }

    get folds(): TimelineScalePiece[] {
        return this.pieces.filter(p => p.folded);
    }

    /** How long the fold at this piece stands for, ready to be shown. */
    static foldLabel(piece: TimelineScalePiece): string {
        return formatDuration(piece.end - piece.start);
    }

    /**
     * Graduations for the visible part of the axis. Only unfolded stretches are graduated: a label
     * inside a fold would claim a precision the fold does not have.
     */
    ticks(from: number, to: number, maxCount: number): TimelineTick[] {
        if (maxCount <= 0 || !(to > from)) {
            return [];
        }

        // The step is chosen from the time actually drawn to scale in the window, so that graduations
        // keep an even spacing on screen whatever is folded away.
        let drawn = 0;
        this.pieces.filter(p => !p.folded).forEach(piece => {
            const overlap = Math.min(piece.to, to) - Math.max(piece.from, from);
            if (overlap > 0) {
                drawn += (overlap / (piece.to - piece.from)) * (piece.end - piece.start);
            }
        });
        if (drawn <= 0) {
            return [];
        }

        const wanted = drawn / maxCount;
        const step = TICK_STEPS.find(s => s >= wanted) ?? TICK_STEPS[TICK_STEPS.length - 1];
        const format = TimelineScale.labelFormat(step);

        const ticks: TimelineTick[] = [];
        this.pieces.filter(p => !p.folded).forEach(piece => {
            for (let at = Math.ceil(piece.start / step) * step; at <= piece.end; at += step) {
                const ratio = this.ratio(at);
                if (ratio >= from && ratio <= to) {
                    ticks.push({ at, ratio, label: format(new Date(at)) });
                }
            }
        });
        return ticks;
    }

    private static labelFormat(step: number): (date: Date) => string {
        const pad = (value: number) => value < 10 ? `0${value}` : `${value}`;
        if (step < 60000) {
            return d => `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
        }
        if (step < 86400000) {
            return d => `${pad(d.getHours())}:${pad(d.getMinutes())}`;
        }
        return d => `${pad(d.getDate())}/${pad(d.getMonth() + 1)} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
    }

    /** Overlapping and touching stretches joined into one, clamped to the bounds of the axis. */
    private static merge(busy: TimelineBusy[], start: number, end: number): TimelineBusy[] {
        const clamped = (busy ?? [])
            .map(b => ({ start: Math.max(b.start, start), end: Math.min(Math.max(b.end, b.start), end) }))
            .filter(b => b.start <= b.end && b.start <= end && b.end >= start)
            .sort((a, b) => a.start - b.start);

        const merged: TimelineBusy[] = [];
        clamped.forEach(b => {
            const last = merged[merged.length - 1];
            if (last && b.start <= last.end) {
                last.end = Math.max(last.end, b.end);
                return;
            }
            merged.push({ ...b });
        });
        return merged;
    }
}
