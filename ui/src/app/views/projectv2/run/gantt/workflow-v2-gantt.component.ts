import {
    AfterViewInit,
    ChangeDetectionStrategy,
    ChangeDetectorRef,
    Component,
    ElementRef,
    EventEmitter,
    Input,
    OnChanges,
    OnDestroy,
    OnInit,
    Output,
    SimpleChanges,
    ViewChild
} from '@angular/core';
import { V2WorkflowRun, V2WorkflowRunJob, V2WorkflowRunJobStatus, WorkflowRunResult } from '../../../../../../libs/workflow-graph/src/lib/v2.workflow.run.model';
import { GanttThemeService, GanttTheme } from './gantt-theme.service';

interface GanttTimeSegment {
    type: 'queued' | 'worker_init' | 'step' | 'completed';
    startTime: Date;
    endTime: Date;
    status?: V2WorkflowRunJobStatus;
    name?: string;
    color: string;
}

interface GanttJobRow {
    id: string;  // job.id for opening the panel
    jobId: string;  // job.job_id for display
    jobName: string;
    status: V2WorkflowRunJobStatus;
    segments: GanttTimeSegment[];
    needs: string[];
    matrixKey?: string;
    stage?: string;  // stage name
    y: number;
    gateInputs?: any;  // Gate inputs if present
}

interface GanttStage {
    name: string;
    y: number;
    jobs: GanttJobRow[];
}

interface GateClickArea {
    x: number;
    y: number;
    width: number;
    height: number;
    gateName: string;
    gateInputs: any;
    jobId: string;
}

@Component({
    selector: 'app-workflow-v2-gantt',
    templateUrl: './workflow-v2-gantt.component.html',
    styleUrls: ['./workflow-v2-gantt.component.scss'],
    changeDetection: ChangeDetectionStrategy.OnPush
})
export class WorkflowV2GanttComponent implements OnInit, OnChanges, AfterViewInit, OnDestroy {
    @ViewChild('canvas', { static: true }) canvasRef: ElementRef<HTMLCanvasElement>;
    @ViewChild('container', { static: true }) containerRef: ElementRef<HTMLDivElement>;

    @Input() workflowRun: V2WorkflowRun;
    @Input() jobs: V2WorkflowRunJob[];
    @Input() results: WorkflowRunResult[];
    @Input() height: number = 600;

    @Output() onJobSelect = new EventEmitter<string>();
    @Output() onHookClick = new EventEmitter<string>();
    @Output() onResultClick = new EventEmitter<WorkflowRunResult>();

    private canvas: HTMLCanvasElement;
    private ctx: CanvasRenderingContext2D;
    
    rows: GanttJobRow[] = [];
    stages: GanttStage[] = [];
    showStageHeaders: boolean = false;
    timelineStart: Date;
    timelineEnd: Date;
    
    private hookClickArea: { x: number, y: number, width: number, height: number } = null;
    private resultClickAreas: Array<{ x: number, y: number, width: number, height: number, result: WorkflowRunResult }> = [];
    private gateClickAreas: Array<GateClickArea> = [];
    
    private viewport = {
        zoom: 1,
        offsetX: 0,
        offsetY: 0,
        pixelsPerMs: 0.1
    };
    
    tooltip = {
        segment: null as GanttTimeSegment,
        jobName: '',
        x: 0,
        y: 0,
        visible: false
    };
    
    gatePopover = {
        visible: false,
        x: 0,
        y: 0,
        gateName: '',
        gateInputs: null as any,
        jobId: ''
    };
    
    private selectedJobId: string;
    private isDragging = false;
    private lastMousePos = { x: 0, y: 0 };
    private eventListenersAttached = false;
    
    // Theme properties - will be populated from the theme service
    theme: GanttTheme;
    
    constructor(
        private cd: ChangeDetectorRef,
        private themeService: GanttThemeService
    ) {
        this.theme = this.themeService.getCurrentTheme();
    }

    // Theme property accessors
    get rowHeight(): number { return this.theme.dimensions.rowHeight; }
    get rowSpacing(): number { return this.theme.dimensions.rowSpacing; }
    get stageHeaderHeight(): number { return this.theme.dimensions.stageHeaderHeight; }
    get stageSpacing(): number { return this.theme.dimensions.stageSpacing; }
    get timelineHeight(): number { return this.theme.dimensions.timelineHeight; }
    get leftMargin(): number { return this.theme.dimensions.leftMargin; }

    // Method to get status color
    getStatusColor(status: V2WorkflowRunJobStatus): string {
        return this.themeService.getStatusColor(status);
    }

    // Method to get segment color
    getSegmentColor(type: 'queued' | 'worker_init' | 'step' | 'completed'): string {
        return this.themeService.getSegmentColor(type);
    }

    // Update theme when needed
    private updateTheme(): void {
        this.theme = this.themeService.getCurrentTheme();
    }

    ngOnInit(): void {
        // Don't build data here as inputs might not be ready yet
    }

    ngOnChanges(changes: SimpleChanges): void {
        // Update theme when inputs change (in case theme switching happened)
        this.updateTheme();
        
        // Rebuild data when inputs change
        if ((changes['jobs'] || changes['workflowRun'] || changes['results']) && this.workflowRun && this.jobs && this.jobs.length > 0) {
            this.buildGanttData();
            
            // If canvas is already initialized, setup everything
            if (this.canvas && this.ctx && this.timelineStart && this.timelineEnd) {
                this.setupCanvas();
                this.attachEventListeners();
                this.render();
            }
        }
    }

    ngAfterViewInit(): void {
        this.canvas = this.canvasRef.nativeElement;
        this.ctx = this.canvas.getContext('2d');
        
        if (!this.canvas || !this.ctx) {
            console.error('Failed to initialize canvas or context');
            return;
        }

        // Reset event listeners flag when component is reinitalized
        this.eventListenersAttached = false;
        
        // Build data if we have the necessary inputs
        if (this.workflowRun && this.jobs && this.jobs.length > 0 && this.rows.length === 0) {
            this.buildGanttData();
        }
        
        // Only setup canvas if we have data
        if (this.rows.length > 0 && this.timelineStart && this.timelineEnd) {
            this.setupCanvas();
            this.attachEventListeners();
            this.render();
        }
        
        // Listen for theme changes
        this.setupThemeListener();
    }

    private themeObserver: MutationObserver;

    private setupThemeListener(): void {
        // Watch for class changes on document.body to detect theme changes
        this.themeObserver = new MutationObserver((mutations) => {
            mutations.forEach((mutation) => {
                if (mutation.type === 'attributes' && mutation.attributeName === 'class') {
                    // Theme changed, re-render canvas
                    if (this.canvas && this.ctx && this.rows.length > 0) {
                        this.render();
                    }
                }
            });
        });

        this.themeObserver.observe(document.body, {
            attributes: true,
            attributeFilter: ['class']
        });
    }

    ngOnDestroy(): void {
        this.eventListenersAttached = false;
        
        // Clean up theme observer
        if (this.themeObserver) {
            this.themeObserver.disconnect();
        }
    }

    private sortStagesTopologically(stageMap: Map<string, V2WorkflowRunJob[]>): string[] {
        const stageNames = Array.from(stageMap.keys());
        
        // If only one stage or default stage, no need to sort
        if (stageNames.length === 1 || (stageNames.length === 1 && stageNames[0] === 'default')) {
            return stageNames;
        }

        // Build a map of which jobs belong to which stage
        const jobToStage = new Map<string, string>();
        stageMap.forEach((jobs, stageName) => {
            jobs.forEach(job => jobToStage.set(job.job_id, stageName));
        });

        // Calculate dependencies between stages
        const stageDependencies = new Map<string, Set<string>>();
        stageNames.forEach(stage => stageDependencies.set(stage, new Set()));

        stageMap.forEach((jobs, stageName) => {
            jobs.forEach(job => {
                if (job.job?.needs) {
                    job.job.needs.forEach(needJobId => {
                        const dependencyStage = jobToStage.get(needJobId);
                        if (dependencyStage && dependencyStage !== stageName) {
                            // This stage depends on another stage
                            stageDependencies.get(stageName).add(dependencyStage);
                        }
                    });
                }
            });
        });

        // Calculate depth for each stage
        const depths = new Map<string, number>();
        const visited = new Set<string>();

        const calculateStageDepth = (stageName: string): number => {
            if (depths.has(stageName)) {
                return depths.get(stageName);
            }
            if (visited.has(stageName)) {
                return 0; // Circular dependency
            }

            visited.add(stageName);
            const deps = stageDependencies.get(stageName);
            
            if (!deps || deps.size === 0) {
                depths.set(stageName, 0);
                visited.delete(stageName);
                return 0;
            }

            let maxDepth = 0;
            deps.forEach(depStageName => {
                const depDepth = calculateStageDepth(depStageName);
                maxDepth = Math.max(maxDepth, depDepth + 1);
            });

            depths.set(stageName, maxDepth);
            visited.delete(stageName);
            return maxDepth;
        };

        // Calculate depth for all stages
        stageNames.forEach(stage => calculateStageDepth(stage));

        // Sort stages by depth, then by earliest job start time
        return stageNames.sort((a, b) => {
            const depthA = depths.get(a) || 0;
            const depthB = depths.get(b) || 0;

            if (depthA !== depthB) {
                return depthA - depthB;
            }

            // Same depth, sort by earliest start time in the stage
            const jobsA = stageMap.get(a);
            const jobsB = stageMap.get(b);

            const earliestTimeA = Math.min(...jobsA.map(j => 
                j.started ? new Date(j.started).getTime() : 
                j.queued ? new Date(j.queued).getTime() : Infinity
            ));
            const earliestTimeB = Math.min(...jobsB.map(j => 
                j.started ? new Date(j.started).getTime() : 
                j.queued ? new Date(j.queued).getTime() : Infinity
            ));

            return earliestTimeA - earliestTimeB;
        });
    }

    private sortJobsTopologically(jobs: V2WorkflowRunJob[]): V2WorkflowRunJob[] {
        // Create a map of job_id to job for quick lookup
        const jobMap = new Map<string, V2WorkflowRunJob>();
        jobs.forEach(job => jobMap.set(job.job_id, job));

        // Calculate depth (dependency level) for each job
        const depths = new Map<string, number>();
        const visited = new Set<string>();

        const calculateDepth = (jobId: string): number => {
            if (depths.has(jobId)) {
                return depths.get(jobId);
            }
            if (visited.has(jobId)) {
                return 0; // Circular dependency, return 0
            }

            visited.add(jobId);
            const job = jobMap.get(jobId);
            if (!job || !job.job?.needs || job.job.needs.length === 0) {
                depths.set(jobId, 0);
                visited.delete(jobId);
                return 0;
            }

            let maxDepth = 0;
            job.job.needs.forEach(needJobId => {
                const depthOfDependency = calculateDepth(needJobId);
                maxDepth = Math.max(maxDepth, depthOfDependency + 1);
            });

            depths.set(jobId, maxDepth);
            visited.delete(jobId);
            return maxDepth;
        };

        // Calculate depth for all jobs
        jobs.forEach(job => calculateDepth(job.job_id));

        // Group jobs into dependency chains
        const rootJobs = jobs.filter(job => !job.job?.needs || job.job.needs.length === 0);
        const result: V2WorkflowRunJob[] = [];
        const processed = new Set<string>();

        // Function to add job and all its descendants
        const addJobChain = (job: V2WorkflowRunJob) => {
            if (processed.has(job.job_id)) {
                return;
            }
            
            processed.add(job.job_id);
            result.push(job);
            
            // Find direct descendants (jobs that depend on this one)
            const descendants = jobs.filter(j => 
                j.job?.needs?.includes(job.job_id) && !processed.has(j.job_id)
            );
            
            // Sort descendants by start time and add them
            descendants
                .sort((a, b) => {
                    const timeA = a.started ? new Date(a.started).getTime() : 
                                 a.queued ? new Date(a.queued).getTime() : 0;
                    const timeB = b.started ? new Date(b.started).getTime() : 
                                 b.queued ? new Date(b.queued).getTime() : 0;
                    return timeA - timeB;
                })
                .forEach(descendant => addJobChain(descendant));
        };

        // Start with root jobs, sorted by start time
        rootJobs
            .sort((a, b) => {
                const timeA = a.started ? new Date(a.started).getTime() : 
                             a.queued ? new Date(a.queued).getTime() : 0;
                const timeB = b.started ? new Date(b.started).getTime() : 
                             b.queued ? new Date(b.queued).getTime() : 0;
                return timeA - timeB;
            })
            .forEach(rootJob => addJobChain(rootJob));

        return result;
    }

    private buildGanttData(): void {
        if (!this.jobs || this.jobs.length === 0) {
            return;
        }

        let minTime = new Date(this.workflowRun.started);
        let maxTime = new Date(this.workflowRun.started);

        this.jobs.forEach(job => {
            if (job.queued) {
                const queueTime = new Date(job.queued);
                if (queueTime < minTime) minTime = queueTime;
            }
            if (job.ended) {
                const endTime = new Date(job.ended);
                if (endTime > maxTime) maxTime = endTime;
            } else if (job.started) {
                const startTime = new Date(job.started);
                if (startTime > maxTime) maxTime = new Date();
            }
        });

        this.timelineStart = new Date(minTime.getTime() - 5000);
        this.timelineEnd = new Date(maxTime.getTime() + 5000);

        // Group jobs by stage
        const stageMap = new Map<string, V2WorkflowRunJob[]>();
        
        this.jobs.forEach(job => {
            const stageName = job.job?.stage || 'default';
            if (!stageMap.has(stageName)) {
                stageMap.set(stageName, []);
            }
            stageMap.get(stageName).push(job);
        });

        // Sort stages topologically if there are dependencies between stages
        const sortedStageNames = this.sortStagesTopologically(stageMap);
        
        // Determine if we should show stage headers
        this.showStageHeaders = sortedStageNames.length > 1 || 
                                (sortedStageNames.length === 1 && sortedStageNames[0] !== 'default');

        this.rows = [];
        this.stages = [];
        let yPosition = this.timelineHeight + this.stageSpacing;

        // Process each stage in topological order
        sortedStageNames.forEach(stageName => {
            const stageJobs = stageMap.get(stageName);
            const stageStartY = yPosition;
            
            // Add stage header space only if we're showing stage headers
            if (this.showStageHeaders) {
                yPosition += this.stageHeaderHeight;
            }

            const stageRows: GanttJobRow[] = [];

            // Sort jobs within stage topologically and chronologically
            const sortedStageJobs = this.sortJobsTopologically(stageJobs);

            // Group sorted jobs by matrix
            const matrixGroups = new Map<string, V2WorkflowRunJob[]>();
            const standaloneJobs: V2WorkflowRunJob[] = [];

            sortedStageJobs.forEach(job => {
                const matrixKey = this.getMatrixKey(job);
                if (matrixKey) {
                    if (!matrixGroups.has(matrixKey)) {
                        matrixGroups.set(matrixKey, []);
                    }
                    matrixGroups.get(matrixKey).push(job);
                } else {
                    standaloneJobs.push(job);
                }
            });

            // Add standalone jobs first (already sorted)
            standaloneJobs.forEach(job => {
                const row = this.createGanttRow(job, yPosition);
                row.stage = stageName;
                stageRows.push(row);
                this.rows.push(row);
                yPosition += this.rowHeight + this.rowSpacing;
            });

            // Then add matrix groups (preserve topological order within each matrix)
            matrixGroups.forEach((jobs, matrixKey) => {
                jobs.forEach(job => {
                    const row = this.createGanttRow(job, yPosition);
                    row.stage = stageName;
                    row.matrixKey = matrixKey;
                    stageRows.push(row);
                    this.rows.push(row);
                    yPosition += this.rowHeight + this.rowSpacing;
                });
            });

            // Create stage object
            this.stages.push({
                name: stageName,
                y: stageStartY,
                jobs: stageRows
            });

            // Add spacing after stage
            yPosition += this.stageSpacing;
        });

        this.cd.markForCheck();
    }

    private getMatrixKey(job: V2WorkflowRunJob): string | null {
        if (job.job_id.includes('[') && job.job_id.includes(']')) {
            return job.job_id.substring(0, job.job_id.indexOf('['));
        }
        return null;
    }

    private createGanttRow(job: V2WorkflowRunJob, y: number): GanttJobRow {
        const segments: GanttTimeSegment[] = [];
        
        const queuedTime = job.queued ? new Date(job.queued) : null;
        const startedTime = job.started ? new Date(job.started) : null;
        const endedTime = job.ended ? new Date(job.ended) : null;

        if (queuedTime && startedTime) {
            segments.push({
                type: 'queued',
                startTime: queuedTime,
                endTime: startedTime,
                color: this.getSegmentColor('queued')
            });
        }

        // Convert steps_status object to array with step IDs
        const stepsEntries = job.steps_status ? Object.entries(job.steps_status) : [];
        
        if (startedTime && stepsEntries.length > 0) {
            const firstStepStart = stepsEntries[0][1].started ? new Date(stepsEntries[0][1].started) : null;
            if (firstStepStart && firstStepStart > startedTime) {
                segments.push({
                    type: 'worker_init',
                    startTime: startedTime,
                    endTime: firstStepStart,
                    color: this.getSegmentColor('worker_init')
                });
            }
        }

        if (stepsEntries.length > 0) {
            stepsEntries.forEach(([stepId, step]) => {
                if (step.started) {
                    const stepStart = new Date(step.started);
                    const stepEnd = step.ended ? new Date(step.ended) : new Date();
                    
                    segments.push({
                        type: 'step',
                        startTime: stepStart,
                        endTime: stepEnd,
                        status: step.conclusion as V2WorkflowRunJobStatus,
                        name: stepId,
                        color: this.getStepColor(step.conclusion as V2WorkflowRunJobStatus)
                    });
                }
            });
        }

        if (segments.length === 0 && startedTime) {
            const endTime = endedTime || new Date();
            segments.push({
                type: 'completed',
                startTime: startedTime,
                endTime: endTime,
                status: job.status,
                color: this.getStatusColor(job.status)
            });
        }

        return {
            id: job.id,  // For opening the panel
            jobId: job.job_id,  // For display
            jobName: job.job_id,
            status: job.status,
            segments,
            needs: job.job?.needs || [],
            gateInputs: job.gate_inputs,  // Capture gate inputs
            y
        };
    }

    private getStepColor(status: V2WorkflowRunJobStatus): string {
        return this.getStatusColor(status);
    }

    private setupCanvas(): void {
        const container = this.containerRef.nativeElement;
        const dpr = window.devicePixelRatio || 1;
        
        this.canvas.width = container.clientWidth * dpr;
        this.canvas.height = Math.max(
            this.height,
            this.rows.length * (this.rowHeight + this.rowSpacing) + this.timelineHeight + 100
        ) * dpr;
        
        this.canvas.style.width = container.clientWidth + 'px';
        this.canvas.style.height = (this.canvas.height / dpr) + 'px';
        
        this.ctx.scale(dpr, dpr);
        
        const timeRange = this.timelineEnd.getTime() - this.timelineStart.getTime();
        const availableWidth = container.clientWidth - this.leftMargin - 40;
        this.viewport.pixelsPerMs = availableWidth / timeRange;
    }

    private render(): void {
        if (!this.ctx || !this.rows.length) {
            return;
        }

        const width = this.canvas.width / (window.devicePixelRatio || 1);
        const height = this.canvas.height / (window.devicePixelRatio || 1);

        this.ctx.clearRect(0, 0, width, height);

        this.drawTimeline();
        
        // Draw stage headers only if we have multiple stages or non-default stage
        if (this.showStageHeaders) {
            this.stages.forEach(stage => {
                this.drawStageHeader(stage);
            });
        }

        this.rows.forEach(row => {
            this.drawJobRow(row);
        });

        this.rows.forEach(row => {
            this.drawDependencies(row);
        });

        // Draw hook event line
        this.drawHookLine();
        
        // Draw gate inputs
        this.drawGates();
        
        // Draw results
        this.drawResults();
    }

    private drawTimeline(): void {
        const width = this.canvas.width / (window.devicePixelRatio || 1);
        const colors = this.themeService.getTimelineColors();
        
        this.ctx.fillStyle = colors.background;
        this.ctx.fillRect(0, 0, width, this.timelineHeight);
        
        this.ctx.strokeStyle = colors.border;
        this.ctx.lineWidth = 1;
        this.ctx.beginPath();
        this.ctx.moveTo(0, this.timelineHeight);
        this.ctx.lineTo(width, this.timelineHeight);
        this.ctx.stroke();

        const timeRange = this.timelineEnd.getTime() - this.timelineStart.getTime();
        const numMarkers = Math.min(20, Math.floor(width / 100));
        const markerInterval = timeRange / numMarkers;

        this.ctx.fillStyle = colors.text;
        this.ctx.font = this.theme.fonts.base;
        this.ctx.textAlign = 'center';

        for (let i = 0; i <= numMarkers; i++) {
            const time = new Date(this.timelineStart.getTime() + markerInterval * i);
            const x = this.timeToX(time);

            this.ctx.strokeStyle = colors.stroke;
            this.ctx.beginPath();
            this.ctx.moveTo(x, this.timelineHeight - 10);
            this.ctx.lineTo(x, this.timelineHeight);
            this.ctx.stroke();

            const timeStr = this.formatTime(time);
            this.ctx.fillText(timeStr, x, this.timelineHeight - 15);
        }
    }

    private drawStageHeader(stage: GanttStage): void {
        const width = this.canvas.width / (window.devicePixelRatio || 1);
        const colors = this.themeService.getStageHeaderColors();
        
        // Draw stage background
        this.ctx.fillStyle = colors.background;
        this.ctx.fillRect(0, stage.y, this.leftMargin, this.stageHeaderHeight);
        
        // Draw stage name
        this.ctx.fillStyle = colors.text;
        this.ctx.font = this.theme.fonts.large;
        this.ctx.textAlign = 'left';
        this.ctx.textBaseline = 'middle';
        this.ctx.fillText(`Stage: ${stage.name}`, 10, stage.y + this.stageHeaderHeight / 2);
        
        // Draw bottom border
        this.ctx.strokeStyle = colors.border;
        this.ctx.lineWidth = 1;
        this.ctx.beginPath();
        this.ctx.moveTo(0, stage.y + this.stageHeaderHeight);
        this.ctx.lineTo(width, stage.y + this.stageHeaderHeight);
        this.ctx.stroke();
    }

    private drawJobRow(row: GanttJobRow): void {
        const width = this.canvas.width / (window.devicePixelRatio || 1);
        const colors = this.themeService.getJobRowColors();
        const themeColors = this.themeService.getThemeColors();
        
        this.ctx.fillStyle = row.jobId === this.selectedJobId ? colors.selected : colors.background;
        this.ctx.fillRect(0, row.y, this.leftMargin, this.rowHeight);
        
        this.ctx.fillStyle = colors.text;
        this.ctx.font = this.theme.fonts.medium;
        this.ctx.textAlign = 'left';
        this.ctx.textBaseline = 'middle';
        
        const jobName = row.matrixKey 
            ? row.jobName.replace(row.matrixKey, '').trim() 
            : row.jobName;
        const truncatedName = this.truncateText(jobName, this.leftMargin - 20);
        this.ctx.fillText(truncatedName, 10, row.y + this.rowHeight / 2);

        row.segments.forEach(segment => {
            const startX = this.timeToX(segment.startTime);
            const endX = this.timeToX(segment.endTime);
            const segmentWidth = Math.max(2, endX - startX);

            this.ctx.fillStyle = segment.color;
            this.ctx.fillRect(
                startX,
                row.y + 5,
                segmentWidth,
                this.rowHeight - 10
            );

            if (segment.status === V2WorkflowRunJobStatus.Stopped) {
                this.ctx.save();
                this.ctx.strokeStyle = themeColors.line;
                this.ctx.lineWidth = 1;
                for (let i = 0; i < segmentWidth; i += 5) {
                    this.ctx.beginPath();
                    this.ctx.moveTo(startX + i, row.y + 5);
                    this.ctx.lineTo(startX + i + 3, row.y + this.rowHeight - 5);
                    this.ctx.stroke();
                }
                this.ctx.restore();
            }

            this.ctx.strokeStyle = colors.border;
            this.ctx.lineWidth = 1;
            this.ctx.strokeRect(startX, row.y + 5, segmentWidth, this.rowHeight - 10);

            // Draw step ID label if this is a step and there's enough space
            if (segment.name && segment.type === 'step' && segmentWidth > 40) {
                this.ctx.save();
                this.ctx.fillStyle = themeColors.background;
                this.ctx.font = this.theme.fonts.base;
                this.ctx.textAlign = 'left';
                this.ctx.textBaseline = 'middle';
                
                const text = this.truncateText(segment.name, segmentWidth - 8);
                const textWidth = this.ctx.measureText(text).width;
                const textX = startX + 4;
                const textY = row.y + this.rowHeight / 2;
                
                // Draw text with shadow for better readability
                this.ctx.shadowColor = themeColors.border;
                this.ctx.shadowBlur = 2;
                this.ctx.fillText(text, textX, textY);
                
                this.ctx.restore();
            }
        });

        this.ctx.strokeStyle = colors.border;
        this.ctx.lineWidth = 1;
        this.ctx.beginPath();
        this.ctx.moveTo(0, row.y + this.rowHeight);
        this.ctx.lineTo(width, row.y + this.rowHeight);
        this.ctx.stroke();
    }

    private drawDependencies(row: GanttJobRow): void {
        if (!row.needs || row.needs.length === 0) {
            return;
        }

        const colors = this.themeService.getThemeColors();

        row.needs.forEach(needJobId => {
            const dependencyRow = this.rows.find(r => r.jobId === needJobId);
            if (!dependencyRow) {
                return;
            }

            const depLastSegment = dependencyRow.segments[dependencyRow.segments.length - 1];
            const currentFirstSegment = row.segments[0];

            if (!depLastSegment || !currentFirstSegment) {
                return;
            }

            const x1 = this.timeToX(depLastSegment.endTime);
            const y1 = dependencyRow.y + this.rowHeight / 2;
            const x2 = this.timeToX(currentFirstSegment.startTime);
            const y2 = row.y + this.rowHeight / 2;

            this.ctx.strokeStyle = colors.line;
            this.ctx.lineWidth = 2;
            this.ctx.setLineDash([5, 5]);
            this.ctx.beginPath();
            this.ctx.moveTo(x1, y1);
            this.ctx.lineTo(x2, y2);
            this.ctx.stroke();
            this.ctx.setLineDash([]);

            const arrowSize = 14;
            const angle = Math.atan2(y2 - y1, x2 - x1);
            this.ctx.fillStyle = colors.line;
            this.ctx.beginPath();
            this.ctx.moveTo(x2, y2);
            this.ctx.lineTo(
                x2 - arrowSize * Math.cos(angle - Math.PI / 6),
                y2 - arrowSize * Math.sin(angle - Math.PI / 6)
            );
            this.ctx.lineTo(
                x2 - arrowSize * Math.cos(angle + Math.PI / 6),
                y2 - arrowSize * Math.sin(angle + Math.PI / 6)
            );
            this.ctx.closePath();
            this.ctx.fill();
        });
    }

    private drawHookLine(): void {
        if (!this.workflowRun || !this.workflowRun.started) {
            return;
        }

        const hookTime = new Date(this.workflowRun.started);
        const x = this.timeToX(hookTime);
        const height = this.canvas.height / (window.devicePixelRatio || 1);
        const colors = this.themeService.getThemeColors();
        
        // Calculate position for label: after the last job row
        const lastRowY = this.rows.length > 0 
            ? this.rows[this.rows.length - 1].y + this.rowHeight + 20
            : this.timelineHeight + 100;

        // Draw label first to know its position
        const label = (this.workflowRun.event?.hook_type || 'manual') + ' trigger';
        const labelY = lastRowY + 10;
        
        // Draw dashed vertical line from timeline to top of label area
        this.ctx.strokeStyle = colors.line;
        this.ctx.lineWidth = 2;
        this.ctx.setLineDash([8, 4]);
        this.ctx.beginPath();
        this.ctx.moveTo(x, this.timelineHeight);
        this.ctx.lineTo(x, labelY - 5); // Stop at the top of the label background
        this.ctx.stroke();
        this.ctx.setLineDash([]);

        // Draw label after the last job row
        
        // Measure text to calculate proper background size
        this.ctx.font = this.theme.fonts.bold;
        const labelWidth = Math.max(this.ctx.measureText(label).width + 20, 80);
        
        // Draw background for label
        this.ctx.fillStyle = colors.background;
        this.ctx.fillRect(x - labelWidth / 2, labelY - 5, labelWidth, 24);
        
        // Draw border around label
        this.ctx.strokeStyle = colors.border;
        this.ctx.lineWidth = 1;
        this.ctx.strokeRect(x - labelWidth / 2, labelY - 5, labelWidth, 24);
        
        // Draw label text
        this.ctx.fillStyle = colors.text;
        this.ctx.textAlign = 'center';
        this.ctx.textBaseline = 'top';
        this.ctx.fillText(label, x, labelY);
        
        // Store clickable area (with extra padding for easier clicking)
        this.hookClickArea = {
            x: x - labelWidth / 2 - 10,
            y: labelY - 15,
            width: labelWidth + 20,
            height: 44
        };
    }

    private drawGates(): void {
        // Clear previous gate click areas
        this.gateClickAreas = [];
        
        const colors = this.themeService.getThemeColors();
        
        this.rows.forEach(row => {
            if (!row.gateInputs) {
                return; // Skip jobs without gate inputs
            }
            
            // Extract gate name from gate_inputs
            let gateName = 'Gate';
            if (row.gateInputs && typeof row.gateInputs === 'object') {
                // Try to find a gate name in the inputs
                if (row.gateInputs.gate_name) {
                    gateName = row.gateInputs.gate_name;
                } else if (row.gateInputs.name) {
                    gateName = row.gateInputs.name;
                } else {
                    // Use the first key as gate name if available
                    const keys = Object.keys(row.gateInputs);
                    if (keys.length > 0) {
                        gateName = keys[0];
                    }
                }
            }
            
            // Find the first segment to position the gate at the beginning of the job
            if (row.segments.length === 0) {
                return;
            }
            
            const firstSegment = row.segments[0];
            const x = this.timeToX(firstSegment.startTime);
            const y = row.y + this.rowHeight / 2;
            
            // Draw gate diamond/indicator
            this.drawGateDiamond(x, y);
            
            // Draw gate label
            this.ctx.font = this.theme.fonts.bold;
            const labelText = `${gateName}`;
            const labelWidth = Math.max(this.ctx.measureText(labelText).width + 16, 60);
            
            // Position label slightly to the left of the gate indicator
            const labelX = x - labelWidth - 10;
            const labelY = y - 10;
            
            // Ensure label doesn't go off the left edge
            const actualLabelX = Math.max(5, labelX);
            
            // Draw background for label
            this.ctx.fillStyle = colors.background;
            this.ctx.fillRect(actualLabelX, labelY, labelWidth, 20);
            
            // Draw border around label
            this.ctx.strokeStyle = colors.border;
            this.ctx.lineWidth = 1;
            this.ctx.strokeRect(actualLabelX, labelY, labelWidth, 20);
            
            // Draw label text
            this.ctx.fillStyle = colors.text;
            this.ctx.textAlign = 'center';
            this.ctx.textBaseline = 'middle';
            this.ctx.fillText(labelText, actualLabelX + labelWidth / 2, labelY + 10);
            
            // Store clickable area
            this.gateClickAreas.push({
                x: actualLabelX - 5,
                y: labelY - 5,
                width: labelWidth + 10,
                height: 30,
                gateName: gateName,
                gateInputs: row.gateInputs,
                jobId: row.jobId
            });
        });
    }

    private drawResultDiamond(x: number, y: number): void {
        const colors = this.themeService.getThemeColors();
        this.drawDiamond(x, y, 6, colors.background, colors.border, 1);
    }

    private drawGateDiamond(x: number, y: number): void {
        const colors = this.themeService.getThemeColors();
        this.drawDiamond(x, y, 8, colors.background, colors.border, 2);
    }

    private drawDiamond(x: number, y: number, size: number, fillColor: string, borderColor: string, borderWidth: number): void {
        this.ctx.save();
        
        // Draw diamond shape
        this.ctx.beginPath();
        this.ctx.moveTo(x, y - size);      // Top point
        this.ctx.lineTo(x + size, y);      // Right point
        this.ctx.lineTo(x, y + size);      // Bottom point
        this.ctx.lineTo(x - size, y);      // Left point
        this.ctx.closePath();
        
        // Fill with specified color
        this.ctx.fillStyle = fillColor;
        this.ctx.fill();
        
        // Add border
        this.ctx.strokeStyle = borderColor;
        this.ctx.lineWidth = borderWidth;
        this.ctx.stroke();
        
        this.ctx.restore();
    }



    private drawResults(): void {
        if (!this.results || this.results.length === 0) {
            return;
        }

        // Clear previous result click areas
        this.resultClickAreas = [];
        
        
        const height = this.canvas.height / (window.devicePixelRatio || 1);
        const colors = this.themeService.getThemeColors();
        
        // Calculate starting position: after hook or after the last job row if no hook
        let baseY = this.rows.length > 0 
            ? this.rows[this.rows.length - 1].y + this.rowHeight + 20
            : this.timelineHeight + 100;
            
        // If we have a hook, start results below it
        if (this.hookClickArea) {
            baseY = this.hookClickArea.y + this.hookClickArea.height + 20;
        }

        this.results.forEach((result, index) => {
            // For results, we need to check if there's a created timestamp or similar
            // Let's assume the result has a created time - we may need to adjust this based on the actual data structure
            if (!result.detail || !result.detail.data) {
                return;
            }
            
            // Try to find a timestamp in the result data
            let resultTime: Date = null;
            
            // Try different possible timestamp fields - first check root level, then detail.data
            if ((result as any).issued_at) {
                resultTime = new Date((result as any).issued_at);
            } else if (result.detail && result.detail.data) {
                if (result.detail.data.issued_at) {
                    resultTime = new Date(result.detail.data.issued_at);
                } else if (result.detail.data.created) {
                    resultTime = new Date(result.detail.data.created);
                } else if (result.detail.data.timestamp) {
                    resultTime = new Date(result.detail.data.timestamp);
                } else if (result.detail.data.created_at) {
                    resultTime = new Date(result.detail.data.created_at);
                }
            }
            
            // Fallback to workflow run times
            if (!resultTime) {
                if (this.workflowRun.last_modified) {
                    resultTime = new Date(this.workflowRun.last_modified);
                } else if (this.workflowRun.started) {
                    // Place results slightly after the start time
                    resultTime = new Date(new Date(this.workflowRun.started).getTime() + (index + 1) * 60000); // 1 minute intervals
                } else {
                    return; // Skip if no timestamp available
                }
            }

            const x = this.timeToX(resultTime);
            const labelY = baseY + (index * 50);

            // Find the job and segment that corresponds to this result
            const jobRunId = (result as any).workflow_run_job_id;
            const jobRow = this.rows.find(row => row.id === jobRunId);
            
            let startY = this.timelineHeight; // Default to timeline if job not found
            
            if (jobRow) {
                // Find the segment that was active when the result was created
                const activeSegment = jobRow.segments.find(segment => 
                    resultTime >= segment.startTime && resultTime <= segment.endTime
                );
                
                if (activeSegment) {
                    // Start from the middle of the job segment
                    startY = jobRow.y + (this.rowHeight / 2);
                } else {
                    // If no active segment, start from the middle of the job row
                    startY = jobRow.y + (this.rowHeight / 2);
                }
            }

            // Draw small diamond indicator at the segment level
            this.drawResultDiamond(x, startY);
            
            // Calculate where the line will end
            const lineEndY = Math.min(labelY + 15, height - 30); // Leave space for label
            
            // Draw dashed vertical line from segment to label area
            this.ctx.strokeStyle = colors.line;
            this.ctx.lineWidth = 2;
            this.ctx.setLineDash([8, 4]);
            this.ctx.beginPath();
            this.ctx.moveTo(x, startY + 8); // Start slightly below the diamond
            this.ctx.lineTo(x, lineEndY);
            this.ctx.stroke();
            this.ctx.setLineDash([]);

            // Use result label or type as display text
            const label = result.label || result.type || 'Result';
            
            // Measure text to calculate proper background size
            this.ctx.font = this.theme.fonts.bold;
            const labelWidth = Math.max(this.ctx.measureText(label).width + 20, 80);
            
            // Position label at the end of the line
            const actualLabelY = lineEndY + 5;
            
            // Draw background for label
            this.ctx.fillStyle = colors.background;
            this.ctx.fillRect(x - labelWidth / 2, actualLabelY - 5, labelWidth, 24);
            
            // Draw border around label
            this.ctx.strokeStyle = colors.border;
            this.ctx.lineWidth = 1;
            this.ctx.strokeRect(x - labelWidth / 2, actualLabelY - 5, labelWidth, 24);
            
            // Draw label text
            this.ctx.fillStyle = colors.text;
            this.ctx.textAlign = 'center';
            this.ctx.textBaseline = 'top';
            this.ctx.fillText(label, x, actualLabelY);
            
            // Store clickable area (with extra padding for easier clicking)
            this.resultClickAreas.push({
                x: x - labelWidth / 2 - 10,
                y: actualLabelY - 15,
                width: labelWidth + 20,
                height: 44,
                result: result
            });
        });
    }

    private timeToX(time: Date): number {
        const relativeTime = time.getTime() - this.timelineStart.getTime();
        return this.leftMargin + (relativeTime * this.viewport.pixelsPerMs) + this.viewport.offsetX;
    }

    private formatTime(time: Date): string {
        const hours = time.getUTCHours().toString().padStart(2, '0');
        const minutes = time.getUTCMinutes().toString().padStart(2, '0');
        const seconds = time.getUTCSeconds().toString().padStart(2, '0');
        return `${hours}:${minutes}:${seconds}`;
    }

    formatDuration(ms: number): string {
        if (ms < 1000) {
            return `${ms}ms`;
        } else if (ms < 60000) {
            return `${(ms / 1000).toFixed(1)}s`;
        } else if (ms < 3600000) {
            const minutes = Math.floor(ms / 60000);
            const seconds = Math.floor((ms % 60000) / 1000);
            return `${minutes}m ${seconds}s`;
        } else {
            const hours = Math.floor(ms / 3600000);
            const minutes = Math.floor((ms % 3600000) / 60000);
            return `${hours}h ${minutes}m`;
        }
    }

    private truncateText(text: string, maxWidth: number): string {
        let width = this.ctx.measureText(text).width;
        if (width <= maxWidth) {
            return text;
        }
        
        let truncated = text;
        while (width > maxWidth - 20 && truncated.length > 0) {
            truncated = truncated.slice(0, -1);
            width = this.ctx.measureText(truncated + '...').width;
        }
        return truncated + '...';
    }

    private attachEventListeners(): void {
        if (this.eventListenersAttached) {
            return;
        }
        
        this.canvas.addEventListener('mousedown', this.onMouseDown.bind(this));
        this.canvas.addEventListener('mousemove', this.onMouseMove.bind(this));
        this.canvas.addEventListener('mouseup', this.onMouseUp.bind(this));
        this.canvas.addEventListener('mouseleave', this.onMouseLeave.bind(this));
        this.canvas.addEventListener('wheel', this.onWheel.bind(this));
        this.canvas.addEventListener('click', this.onClick.bind(this));
        
        this.eventListenersAttached = true;
    }

    private onMouseDown(event: MouseEvent): void {
        this.isDragging = true;
        this.lastMousePos = { x: event.clientX, y: event.clientY };
    }

    private onMouseMove(event: MouseEvent): void {
        const rect = this.canvas.getBoundingClientRect();
        const x = event.clientX - rect.left;
        const y = event.clientY - rect.top;

        if (this.isDragging) {
            const dx = event.clientX - this.lastMousePos.x;
            this.viewport.offsetX += dx;
            this.lastMousePos = { x: event.clientX, y: event.clientY };
            this.render();
        } else {
            let cursorSet = false;
            
            // Check if hovering over hook area
            if (this.hookClickArea) {
                const inHookArea = x >= this.hookClickArea.x && 
                    x <= this.hookClickArea.x + this.hookClickArea.width &&
                    y >= this.hookClickArea.y && 
                    y <= this.hookClickArea.y + this.hookClickArea.height;
                
                if (inHookArea) {
                    this.canvas.style.cursor = 'pointer';
                    cursorSet = true;
                }
            }
            
            // If not over hook, check if hovering over gate areas
            if (!cursorSet) {
                let gateHovered = false;
                for (const gateArea of this.gateClickAreas) {
                    if (x >= gateArea.x && 
                        x <= gateArea.x + gateArea.width &&
                        y >= gateArea.y && 
                        y <= gateArea.y + gateArea.height) {
                        this.canvas.style.cursor = 'pointer';
                        cursorSet = true;
                        gateHovered = true;
                        
                        // Show popover on hover
                        this.gatePopover = {
                            visible: true,
                            x: x,
                            y: y,
                            gateName: gateArea.gateName,
                            gateInputs: gateArea.gateInputs,
                            jobId: gateArea.jobId
                        };
                        
                        this.cd.markForCheck();
                        break;
                    }
                }
                
                // Hide popover if not hovering over any gate
                if (!gateHovered && this.gatePopover.visible) {
                    this.gatePopover.visible = false;
                    this.cd.markForCheck();
                }
            }
            
            // If not over hook or gates, check if hovering over result areas
            if (!cursorSet) {
                for (const resultArea of this.resultClickAreas) {
                    if (x >= resultArea.x && 
                        x <= resultArea.x + resultArea.width &&
                        y >= resultArea.y && 
                        y <= resultArea.y + resultArea.height) {
                        this.canvas.style.cursor = 'pointer';
                        cursorSet = true;
                        break;
                    }
                }
            }
            
            // If not over hook, gates or results, check if hovering over job segments
            if (!cursorSet) {
                for (const row of this.rows) {
                    if (y >= row.y && y <= row.y + this.rowHeight) {
                        for (const segment of row.segments) {
                            const startX = this.timeToX(segment.startTime);
                            const endX = this.timeToX(segment.endTime);
                            
                            if (x >= startX && x <= endX) {
                                this.canvas.style.cursor = 'pointer';
                                cursorSet = true;
                                break;
                            }
                        }
                        if (cursorSet) break;
                    }
                }
            }
            
            // Default cursor if not over interactive elements
            if (!cursorSet) {
                this.canvas.style.cursor = 'grab';
            }
            
            this.updateTooltip(x, y);
        }
    }

    private onMouseUp(): void {
        this.isDragging = false;
    }

    private onMouseLeave(): void {
        this.isDragging = false;
        this.tooltip.visible = false;
        this.gatePopover.visible = false;
        this.cd.markForCheck();
    }

    private onWheel(event: WheelEvent): void {
        event.preventDefault();
        
        const zoomFactor = event.deltaY > 0 ? 0.9 : 1.1;
        this.viewport.pixelsPerMs *= zoomFactor;
        
        this.viewport.pixelsPerMs = Math.max(0.001, Math.min(10, this.viewport.pixelsPerMs));
        
        this.render();
    }

    private onClick(event: MouseEvent): void {
        const rect = this.canvas.getBoundingClientRect();
        const x = event.clientX - rect.left;
        const y = event.clientY - rect.top;

        // Check if click is on hook area
        if (this.hookClickArea && 
            x >= this.hookClickArea.x && 
            x <= this.hookClickArea.x + this.hookClickArea.width &&
            y >= this.hookClickArea.y && 
            y <= this.hookClickArea.y + this.hookClickArea.height) {
            const hookType = this.workflowRun.event?.hook_type || 'manual';
            this.onHookClick.emit(hookType);
            this.cd.markForCheck();
            return;
        }

        // Check if click is on a result area
        for (const resultArea of this.resultClickAreas) {
            if (x >= resultArea.x && 
                x <= resultArea.x + resultArea.width &&
                y >= resultArea.y && 
                y <= resultArea.y + resultArea.height) {
                this.onResultClick.emit(resultArea.result);
                this.cd.markForCheck();
                return;
            }
        }

        // Check if click is on a job row segment (not just the entire row)
        const row = this.rows.find(r => y >= r.y && y <= r.y + this.rowHeight);
        if (row) {
            // Check if click is specifically on a segment
            const clickedSegment = row.segments.find(segment => {
                if (!segment.startTime || !segment.endTime) return false;
                
                const startX = this.timeToX(segment.startTime);
                const endX = this.timeToX(segment.endTime);
                const width = Math.max(1, endX - startX);
                
                return x >= startX && x <= startX + width && 
                       y >= row.y + 5 && y <= row.y + this.rowHeight - 5;
            });
            
            // Only select job if click is on a segment
            if (clickedSegment) {
                this.selectedJobId = row.jobId;
                this.onJobSelect.emit(row.id);  // Emit job.id for opening the panel
                this.render();
            }
        }
        
        // Hide gate popover on any click (since gates now only show on hover)
        if (this.gatePopover.visible) {
            this.gatePopover.visible = false;
            this.cd.markForCheck();
        }
    }

    private updateTooltip(x: number, y: number): void {
        for (const row of this.rows) {
            if (y >= row.y && y <= row.y + this.rowHeight) {
                for (const segment of row.segments) {
                    const startX = this.timeToX(segment.startTime);
                    const endX = this.timeToX(segment.endTime);
                    
                    if (x >= startX && x <= endX) {
                        this.tooltip.segment = segment;
                        this.tooltip.jobName = row.jobName;
                        this.tooltip.x = x;
                        this.tooltip.y = y;
                        this.tooltip.visible = true;
                        this.cd.markForCheck();
                        return;
                    }
                }
            }
        }

        this.tooltip.visible = false;
        this.cd.markForCheck();
    }

    zoomToFit(): void {
        const container = this.containerRef.nativeElement;
        const timeRange = this.timelineEnd.getTime() - this.timelineStart.getTime();
        const availableWidth = container.clientWidth - this.leftMargin - 40;
        this.viewport.pixelsPerMs = availableWidth / timeRange;
        this.viewport.offsetX = 0;
        this.render();
    }

    public refresh(): void {
        if (this.workflowRun && this.jobs && this.jobs.length > 0) {
            this.buildGanttData();
            if (this.canvas && this.ctx && this.timelineStart && this.timelineEnd) {
                this.setupCanvas();
                if (!this.eventListenersAttached) {
                    this.attachEventListeners();
                }
                this.render();
            }
        }
    }

    getGateInputKeys(): string[] {
        if (!this.gatePopover.gateInputs || typeof this.gatePopover.gateInputs !== 'object') {
            return [];
        }
        return Object.keys(this.gatePopover.gateInputs);
    }

    formatInputValue(value: any): string {
        if (value === null || value === undefined) {
            return 'null';
        }
        if (typeof value === 'boolean') {
            return value ? 'true' : 'false';
        }
        if (typeof value === 'string') {
            return value;
        }
        if (typeof value === 'number') {
            return value.toString();
        }
        if (typeof value === 'object') {
            return JSON.stringify(value);
        }
        return String(value);
    }

    getInputValueColor(value: any): string {
        if (value === null || value === undefined) {
            return 'default';
        }
        if (typeof value === 'boolean') {
            return value ? 'green' : 'red';
        }
        if (typeof value === 'string') {
            return 'blue';
        }
        if (typeof value === 'number') {
            return 'orange';
        }
        if (typeof value === 'object') {
            return 'purple';
        }
        return 'default';
    }

    getSegmentTypeColor(type: string): string {
        switch (type) {
            case 'queued':
                return 'orange';
            case 'worker_init':
                return 'blue';
            case 'step':
                return 'cyan';
            case 'completed':
                return 'green';
            default:
                return 'default';
        }
    }

    getStatusTagColor(status: V2WorkflowRunJobStatus): string {
        switch (status) {
            case V2WorkflowRunJobStatus.Success:
                return 'green';
            case V2WorkflowRunJobStatus.Building:
                return 'blue';
            case V2WorkflowRunJobStatus.Fail:
                return 'red';
            case V2WorkflowRunJobStatus.Waiting:
                return 'orange';
            case V2WorkflowRunJobStatus.Scheduling:
                return 'purple';
            case V2WorkflowRunJobStatus.Stopped:
                return 'default';
            default:
                return 'default';
        }
    }
}
