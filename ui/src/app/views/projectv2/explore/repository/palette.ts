/**
 * How anything on the page went, in one word; the icon and the colour follow from it alone, so that a
 * row, a step and an analysis in the same state look the same.
 * pending: not reached yet · filtered: ruled out by the definitions themselves · none: nothing to do here
 */
export type Tone = 'success' | 'error' | 'warning' | 'processing' | 'pending' | 'filtered' | 'none';

export function toneIcon(tone: Tone): string {
    switch (tone) {
        case 'success': return 'check-circle';
        case 'error': return 'close-circle';
        case 'warning': return 'exclamation-circle';
        case 'processing': return 'sync';
        case 'pending': return 'clock-circle';
        case 'filtered': return 'stop';
        default: return 'minus-circle';
    }
}

/** The preset Ant Design tags and badges know for a tone; the grey tones share `default`. */
export function toneColor(tone: Tone): string {
    switch (tone) {
        case 'success': case 'error': case 'warning': case 'processing': return tone;
        default: return 'default';
    }
}
