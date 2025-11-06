import { Injectable } from '@angular/core';
import { V2WorkflowRunJobStatus } from '../../../../../../libs/workflow-graph/src/lib/v2.workflow.run.model';

export interface GanttColors {
    // Status colors
    success: string;
    building: string;
    fail: string;
    waiting: string;
    scheduling: string;
    
    // Segment colors
    queued: string;
    workerInit: string;
    step: string;
    completed: string;
    
    // UI colors
    background: string;
    border: string;
    text: string;
    textSecondary: string;
    hover: string;
    selected: string;
    
    // Timeline colors
    timelineBackground: string;
    timelineBorder: string;
    timelineText: string;
    
    // Stage colors
    stageBackground: string;
    stageBorder: string;
    stageText: string;
    
    // Job row colors
    jobRowBackground: string;
    jobRowBackgroundEven: string;
    jobRowBorder: string;
    jobRowText: string;
    jobRowSelected: string;
    
    // Hook colors
    hookBackground: string;
    hookBorder: string;
    hookText: string;
    
    // Gate colors
    gateBackground: string;
    gateBorder: string;
    gateText: string;
}

export interface GanttDimensions {
    rowHeight: number;
    rowSpacing: number;
    stageHeaderHeight: number;
    stageSpacing: number;
    timelineHeight: number;
    leftMargin: number;
}

export interface GanttFonts {
    base: string;
    small: string;
    medium: string;
    large: string;
    bold: string;
}

export interface GanttTheme {
    colors: GanttColors;
    dimensions: GanttDimensions;
    fonts: GanttFonts;
}

@Injectable({
    providedIn: 'root'
})
export class GanttThemeService {
    
    private readonly baseFontFamily = '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif';
    private customTheme: {
        colors?: Partial<GanttColors>;
        dimensions?: Partial<GanttDimensions>;
        fonts?: Partial<GanttFonts>;
    } | null = null;
    
    private readonly lightTheme: GanttTheme = {
        colors: {
            // Status colors (using CDS colors)
            success: '#21BA45',      // $cds_color_green
            building: '#4fa3e3',     // $cds_color_teal
            fail: '#FF4F60',         // $cds_color_red
            waiting: '#FE9A76',      // $cds_color_orange
            scheduling: '#4fa3e3',   // $cds_color_teal
            
            // Segment colors
            queued: '#FE9A76',       // $cds_color_orange
            workerInit: '#4fa3e3',   // $cds_color_teal
            step: '#4fa3e3',         // $cds_color_teal
            completed: '#21BA45',    // $cds_color_green
            
            // UI colors
            background: '#ffffff',
            border: '#f0f0f0',
            text: '#000000',
            textSecondary: '#666666',
            hover: '#f5f5f5',
            selected: '#e6f7ff',
            
            // Timeline colors
            timelineBackground: '#fafafa',
            timelineBorder: '#f0f0f0',
            timelineText: '#666666',
            
            // Stage colors
            stageBackground: '#f9f9f9',
            stageBorder: '#e8e8e8',
            stageText: '#333333',
            
            // Job row colors
            jobRowBackground: '#ffffff',
            jobRowBackgroundEven: '#fafafa',
            jobRowBorder: '#f0f0f0',
            jobRowText: '#333333',
            jobRowSelected: '#e6f7ff',
            
            // Hook colors
            hookBackground: '#fff7e6',
            hookBorder: '#ffd591',
            hookText: '#d46b08',
            
            // Gate colors
            gateBackground: '#f6ffed',
            gateBorder: '#b7eb8f',
            gateText: '#52c41a'
        },
        dimensions: {
            rowHeight: 40,
            rowSpacing: 10,
            stageHeaderHeight: 30,
            stageSpacing: 15,
            timelineHeight: 60,
            leftMargin: 200
        },
        fonts: {
            base: `11px ${this.baseFontFamily}`,
            small: `10px ${this.baseFontFamily}`,
            medium: `13px ${this.baseFontFamily}`,
            large: `bold 14px ${this.baseFontFamily}`,
            bold: `bold 12px ${this.baseFontFamily}`
        }
    };
    
    private readonly darkTheme: GanttTheme = {
        colors: {
            // Status colors (same as light theme)
            success: '#21BA45',
            building: '#4fa3e3',
            fail: '#FF4F60',
            waiting: '#FE9A76',
            scheduling: '#4fa3e3',
            
            // Segment colors (same as light theme)
            queued: '#FE9A76',
            workerInit: '#4fa3e3',
            step: '#4fa3e3',
            completed: '#21BA45',
            
            // UI colors (dark theme)
            background: '#141414',
            border: '#303030',
            text: '#ffffff',
            textSecondary: '#d9d9d9',
            hover: '#262626',
            selected: '#1f1f1f',
            
            // Timeline colors
            timelineBackground: '#1f1f1f',
            timelineBorder: '#303030',
            timelineText: '#d9d9d9',
            
            // Stage colors
            stageBackground: '#1a1a1a',
            stageBorder: '#303030',
            stageText: '#d9d9d9',
            
            // Job row colors
            jobRowBackground: '#141414',
            jobRowBackgroundEven: '#1a1a1a',
            jobRowBorder: '#303030',
            jobRowText: '#d9d9d9',
            jobRowSelected: '#1f1f1f',
            
            // Hook colors
            hookBackground: '#2d1b0e',
            hookBorder: '#8b4513',
            hookText: '#ffa940',
            
            // Gate colors
            gateBackground: '#162312',
            gateBorder: '#389e0d',
            gateText: '#95de64'
        },
        dimensions: {
            rowHeight: 40,
            rowSpacing: 10,
            stageHeaderHeight: 30,
            stageSpacing: 15,
            timelineHeight: 60,
            leftMargin: 200
        },
        fonts: {
            base: `11px ${this.baseFontFamily}`,
            small: `10px ${this.baseFontFamily}`,
            medium: `13px ${this.baseFontFamily}`,
            large: `bold 14px ${this.baseFontFamily}`,
            bold: `bold 12px ${this.baseFontFamily}`
        }
    };
    
    getCurrentTheme(): GanttTheme {
        const baseTheme = this.getBaseTheme();
        
        // Merge with custom theme if provided
        if (this.customTheme) {
            return {
                colors: { ...baseTheme.colors, ...(this.customTheme.colors || {}) },
                dimensions: { ...baseTheme.dimensions, ...(this.customTheme.dimensions || {}) },
                fonts: { ...baseTheme.fonts, ...(this.customTheme.fonts || {}) }
            };
        }
        
        return baseTheme;
    }
    
    private getBaseTheme(): GanttTheme {
        // Detect if dark mode is active
        if (typeof document !== 'undefined') {
            const isDarkMode = document.body.classList.contains('night') || 
                              document.documentElement.classList.contains('night');
            return isDarkMode ? this.darkTheme : this.lightTheme;
        }
        return this.lightTheme;
    }
    
    /**
     * Apply custom theme overrides
     * @param customTheme Partial theme configuration to override defaults
     */
    setCustomTheme(customTheme: {
        colors?: Partial<GanttColors>;
        dimensions?: Partial<GanttDimensions>;
        fonts?: Partial<GanttFonts>;
    }): void {
        this.customTheme = customTheme;
    }
    
    /**
     * Reset to default theme
     */
    resetTheme(): void {
        this.customTheme = null;
    }
    
    getStatusColor(status: V2WorkflowRunJobStatus): string {
        const theme = this.getCurrentTheme();
        switch (status) {
            case V2WorkflowRunJobStatus.Success:
                return theme.colors.success;
            case V2WorkflowRunJobStatus.Building:
                return theme.colors.building;
            case V2WorkflowRunJobStatus.Fail:
                return theme.colors.fail;
            case V2WorkflowRunJobStatus.Waiting:
                return theme.colors.waiting;
            case V2WorkflowRunJobStatus.Scheduling:
                return theme.colors.scheduling;
            default:
                return theme.colors.textSecondary;
        }
    }
    
    getSegmentColor(type: 'queued' | 'worker_init' | 'step' | 'completed'): string {
        const theme = this.getCurrentTheme();
        switch (type) {
            case 'queued':
                return theme.colors.queued;
            case 'worker_init':
                return theme.colors.workerInit;
            case 'step':
                return theme.colors.step;
            case 'completed':
                return theme.colors.completed;
            default:
                return theme.colors.textSecondary;
        }
    }
    
    getTimelineColors() {
        const theme = this.getCurrentTheme();
        return {
            background: theme.colors.timelineBackground,
            border: theme.colors.timelineBorder,
            text: theme.colors.timelineText,
            stroke: theme.colors.timelineBorder  // Use border color for stroke
        };
    }
    
    getStageHeaderColors() {
        const theme = this.getCurrentTheme();
        return {
            background: theme.colors.stageBackground,
            border: theme.colors.stageBorder,
            text: theme.colors.stageText
        };
    }
    
    getJobRowColors() {
        const theme = this.getCurrentTheme();
        return {
            background: theme.colors.jobRowBackground,
            backgroundEven: theme.colors.jobRowBackgroundEven,
            border: theme.colors.jobRowBorder,
            text: theme.colors.jobRowText,
            selected: theme.colors.jobRowSelected
        };
    }
    
    getThemeColors() {
        const theme = this.getCurrentTheme();
        const isDark = document.body.classList.contains('night') || 
                      document.querySelector('.night') !== null;
        if (isDark) {
            return {
                background: '#2d2c2c',      // $darkTheme_grey_0
                border: '#434141',          // $darkTheme_grey_3
                text: '#cccccc',           // $darkTheme_grey_6
                line: '#595959'            // $darkTheme_grey_4
            };
        }
        return {
            background: '#ffffff',
            border: '#d9d9d9',
            text: '#262626',
            line: '#8c8c8c'
        };
    }
}