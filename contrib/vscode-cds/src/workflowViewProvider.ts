import * as vscode from "vscode";
import * as fsp from "fs/promises";
import * as path from "path";
import { CDS } from "./cds";
import { CdsRepository, CdsWorkflowRun } from "./cds/models";

// ─── Tree item classes ───────────────────────────────────────────────────────

export class RepoItem extends vscode.TreeItem {
  readonly kind = "repo" as const;
  constructor(
    public readonly displayName: string,
    public readonly cdsDir: string | undefined,
    public readonly repoRoot: string | undefined,
    public readonly cdsRepo: CdsRepository | undefined,
  ) {
    super(displayName, vscode.TreeItemCollapsibleState.Collapsed);
    this.id = `repo:${cdsRepo?.id ?? repoRoot ?? displayName}:${repoRoot ?? "remote"}`;
    this.description = cdsRepo
      ? `${cdsRepo.repoName}  •  ${cdsRepo.vcsName}`
      : cdsDir
        ? "local only"
        : undefined;
    this.tooltip = cdsRepo
      ? `${cdsRepo.repoName} (${cdsRepo.vcsName})\nProject: ${cdsRepo.projectKey}`
      : (repoRoot ?? displayName);
    this.iconPath = new vscode.ThemeIcon(
      cdsRepo && cdsDir ? "repo-forked" : cdsRepo ? "cloud" : "folder-opened",
    );
    this.contextValue =
      cdsRepo && cdsDir
        ? "cdsRepoWorkspace"
        : cdsRepo
          ? "cdsRepoCds"
          : "cdsRepoLocal";
  }
}

export class WorkflowItem extends vscode.TreeItem {
  readonly kind = "workflow" as const;
  constructor(
    public readonly displayLabel: string,
    public readonly cdsWorkflowName: string,
    public readonly filePath: string | undefined,
    public readonly repo: RepoItem,
  ) {
    super(displayLabel, vscode.TreeItemCollapsibleState.Collapsed);
    this.id = `wf:${repo.id}:${cdsWorkflowName}`;
    this.description = filePath
      ? path.relative(repo.repoRoot ?? "", filePath)
      : undefined;
    this.tooltip = filePath ?? cdsWorkflowName;
    this.iconPath = new vscode.ThemeIcon(
      filePath ? "file-code" : "symbol-event",
    );
    this.contextValue = repo.cdsRepo ? "cdsWorkflow" : "cdsWorkflowLocal";
    if (filePath) {
      this.command = {
        command: "vscode.open",
        title: "Open Workflow File",
        arguments: [vscode.Uri.file(filePath)],
      };
    }
  }
}

const STATUS_ICONS: Record<string, { icon: string; color: string }> = {
  success: { icon: "pass-filled", color: "charts.green" },
  fail: { icon: "error", color: "charts.red" },
  failed: { icon: "error", color: "charts.red" },
  building: { icon: "sync~spin", color: "charts.blue" },
  crafting: { icon: "sync~spin", color: "charts.blue" },
  pending: { icon: "clock", color: "charts.yellow" },
  waiting: { icon: "clock", color: "charts.yellow" },
  blocked: { icon: "lock", color: "charts.yellow" },
  skipped: { icon: "circle-slash", color: "disabledForeground" },
  stopped: { icon: "circle-slash", color: "charts.orange" },
  cancelled: { icon: "circle-slash", color: "disabledForeground" },
};

function statusIcon(status: string): vscode.ThemeIcon {
  const lc = status.toLowerCase();
  const entry = STATUS_ICONS[lc];
  if (entry) {
    return new vscode.ThemeIcon(entry.icon, new vscode.ThemeColor(entry.color));
  }
  return new vscode.ThemeIcon("circle-outline");
}

function runContextValue(status: string): string {
  const lc = status.toLowerCase();
  if (lc === "building" || lc === "pending" || lc === "waiting") {
    return "cdsRunBuilding";
  }
  if (lc === "fail" || lc === "failed") {
    return "cdsRunFailed";
  }
  if (lc === "success") {
    return "cdsRunSuccess";
  }
  return "cdsRunDone";
}

function relativeTime(iso: string): string {
  if (!iso) {
    return "";
  }
  const diff = Date.now() - new Date(iso).getTime();
  if (isNaN(diff) || diff < 0) {
    return "just now";
  }
  const s = Math.floor(diff / 1000);
  if (s < 60) {
    return `${s}s ago`;
  }
  const m = Math.floor(s / 60);
  if (m < 60) {
    return `${m}m ago`;
  }
  const h = Math.floor(m / 60);
  if (h < 24) {
    return `${h}h ago`;
  }
  return `${Math.floor(h / 24)}d ago`;
}

export class RunItem extends vscode.TreeItem {
  readonly kind = "run" as const;
  constructor(
    public readonly run: CdsWorkflowRun,
    public readonly workflow: WorkflowItem,
  ) {
    const refName = run.ref || "";
    const shortCommit = run.commit ? run.commit.substring(0, 7) : "";
    const ago = run.started ? relativeTime(run.started) : "";

    // Label: #number  ago
    super(`#${run.runNumber}  ${ago}`, vscode.TreeItemCollapsibleState.None);
    this.id = `run:${run.id}`;

    // Description: branch:ref  commit:sha
    const descParts: string[] = [];
    if (refName) { descParts.push(`branch:${refName}`); }
    if (shortCommit) { descParts.push(`commit:${shortCommit}`); }
    this.description = descParts.join("  ");

    this.tooltip = [
      `Run #${run.runNumber}`,
      `Status: ${run.status}`,
      refName ? `Ref: ${refName}` : "",
      run.commit ? `Commit: ${run.commit}` : "",
      run.username ? `By: ${run.username}` : "",
      run.started ? `Started: ${run.started}` : "",
    ]
      .filter(Boolean)
      .join("\n");
    this.iconPath = statusIcon(run.status);
    this.contextValue = runContextValue(run.status);
  }
}

// ─── Provider ────────────────────────────────────────────────────────────────

type TreeNode = RepoItem | WorkflowItem | RunItem | SeparatorItem;

/** Visual separator in the tree view. */
class SeparatorItem extends vscode.TreeItem {
  constructor(label = "────────────────") {
    super(label, vscode.TreeItemCollapsibleState.None);
    this.id = "separator:local-repos";
    this.description = "other workspace repos";
    this.iconPath = new vscode.ThemeIcon("dash");
  }
}

/** Placeholder item shown when no CDS project is selected. */
class NoProjectItem extends vscode.TreeItem {
  constructor() {
    super("Select a CDS project to browse workflows", vscode.TreeItemCollapsibleState.None);
    this.iconPath = new vscode.ThemeIcon("info");
    this.command = {
      command: "vscode-cds.setCurrentProject",
      title: "Select CDS project",
    };
  }
}

export class WorkflowViewProvider implements vscode.TreeDataProvider<TreeNode>, vscode.Disposable {
  private readonly _onDidChangeTreeData = new vscode.EventEmitter<
    TreeNode | undefined | null | void
  >();
  readonly onDidChangeTreeData = this._onDidChangeTreeData.event;

  // Current project key (set externally via setProjectKey)
  private projectKey: string | undefined;

  // Promise-based caches — undefined = not started yet
  private repoPromise: Promise<TreeNode[]> | undefined;
  private workflowPromises = new Map<string, Promise<WorkflowItem[]>>();
  private runPromises = new Map<string, Promise<RunItem[]>>();

  // Workflow names discovered per CDS repo (projectKey:repoId -> Set<workflowName>)
  private discoveredWorkflows = new Map<string, Promise<string[]>>();

  constructor() {
    /* initial load on first getChildren */
  }

  dispose(): void {
    this._onDidChangeTreeData.dispose();
  }

  /** Update the active project and refresh the tree. */
  setProjectKey(projectKey: string | undefined): void {
    if (this.projectKey !== projectKey) {
      this.projectKey = projectKey;
      this.refresh();
    }
  }

  getTreeItem(element: TreeNode): vscode.TreeItem {
    return element;
  }

  getChildren(element?: TreeNode): vscode.ProviderResult<TreeNode[]> {
    if (!element) {
      if (!this.repoPromise) {
        this.repoPromise = this.buildRepoList().catch((err) => {
          this.repoPromise = undefined;
          throw err;
        });
      }
      return this.repoPromise.then((items) =>
        items.length > 0 ? items : [new NoProjectItem()],
      );
    }
    if (element instanceof RepoItem) {
      if (!this.workflowPromises.has(element.id!)) {
        const p = this.buildWorkflowList(element).catch((err) => {
          this.workflowPromises.delete(element.id!);
          throw err;
        });
        this.workflowPromises.set(element.id!, p);
      }
      return this.workflowPromises.get(element.id!)!;
    }
    if (element instanceof WorkflowItem) {
      if (!this.runPromises.has(element.id!)) {
        this.runPromises.set(element.id!, this.buildRunList(element));
      }
      return this.runPromises.get(element.id!)!;
    }
    return [];
  }

  refresh(): void {
    this.repoPromise = undefined;
    this.workflowPromises.clear();
    this.runPromises.clear();
    this.discoveredWorkflows.clear();
    this._onDidChangeTreeData.fire();
  }

  /** Refresh only a single workflow's run list (after stop / restart). */
  refreshRuns(wf: WorkflowItem): void {
    this.runPromises.delete(wf.id!);
    this._onDidChangeTreeData.fire(wf);
  }

  // ─── Private helpers ──────────────────────────────────────────────────────

  private async buildRepoList(): Promise<TreeNode[]> {
    // Use the cached project key (set by onProjectChanged event)
    const projectKey = this.projectKey;
    if (!projectKey) {
      return [];
    }

    // 1. Scan workspace folders for .cds directories
    const cdsDirs: Array<{ cdsDir: string; repoRoot: string }> = [];
    for (const folder of vscode.workspace.workspaceFolders ?? []) {
      for (const d of await this.findCdsDirs(folder.uri.fsPath)) {
        cdsDirs.push({ cdsDir: d, repoRoot: path.dirname(d) });
      }
    }

    // 2. Get CDS repos for the selected project
    let cdsRepos: CdsRepository[] = [];
    try {
      cdsRepos = await CDS.listRepositories(projectKey);
    } catch {
      // cdsctl may not be available — continue with local-only items
    }

    // 3. Match workspace dirs to CDS repos by git remote URL
    // A single CDS repo can only be matched once to avoid duplicate tree IDs.
    const matchedIds = new Set<string>();
    const inProjectWorkspace: RepoItem[] = [];  // repos in workspace AND in project
    const localOnly: RepoItem[] = [];           // repos in workspace but NOT in project
    for (const { cdsDir, repoRoot } of cdsDirs) {
      const dirName = path.basename(repoRoot);
      const remoteSlug = await this.getGitRemoteSlug(repoRoot);
      const cdsRepo = cdsRepos.find(
        (r) =>
          !matchedIds.has(r.id) &&
          this.matchesRepo(r, dirName, remoteSlug),
      );
      if (cdsRepo) {
        matchedIds.add(cdsRepo.id);
        inProjectWorkspace.push(new RepoItem(dirName, cdsDir, repoRoot, cdsRepo));
      } else {
        localOnly.push(new RepoItem(dirName, cdsDir, repoRoot, undefined));
      }
    }

    // 4. CDS repos not found in workspace (remote only)
    const cdsOnlyItems = cdsRepos
      .filter((r) => !matchedIds.has(r.id))
      .map((r) => new RepoItem(r.repoName, undefined, undefined, r));

    // Order: 1) project repos in workspace, 2) project repos remote, 3) separator + local-only
    const result: TreeNode[] = [...inProjectWorkspace, ...cdsOnlyItems];
    if (localOnly.length > 0) {
      result.push(new SeparatorItem());
      result.push(...localOnly);
    }
    return result;
  }

  private async buildWorkflowList(repo: RepoItem): Promise<WorkflowItem[]> {
    const workflows: WorkflowItem[] = [];
    const seen = new Set<string>();

    // Local .cds/workflows/ YAML files take priority
    if (repo.cdsDir) {
      const workflowsDir = path.join(repo.cdsDir, "workflows");
      for (const filePath of await this.scanYamlFiles(workflowsDir)) {
        const wfName = path.basename(filePath).replace(/\.(yaml|yml)$/, "");
        if (!seen.has(wfName)) {
          seen.add(wfName);
          workflows.push(
            new WorkflowItem(path.basename(filePath), wfName, filePath, repo),
          );
        }
      }
    }

    // For CDS-only repos (not in workspace): discover workflows scoped to this
    // specific repo. Workspace repos use local YAML files as the source of truth.
    if (repo.cdsRepo && !repo.cdsDir) {
      const { projectKey, vcsName, repoName, id: repoId } = repo.cdsRepo;
      const cacheKey = `${projectKey}:${repoId}`;
      if (!this.discoveredWorkflows.has(cacheKey)) {
        this.discoveredWorkflows.set(
          cacheKey,
          CDS.discoverRepoWorkflowNames(projectKey, vcsName, repoName),
        );
      }
      const names = await this.discoveredWorkflows.get(cacheKey)!;
      for (const name of names) {
        if (!seen.has(name)) {
          seen.add(name);
          workflows.push(new WorkflowItem(name, name, undefined, repo));
        }
      }
    }

    return workflows;
  }

  private async buildRunList(wf: WorkflowItem): Promise<RunItem[]> {
    if (!wf.repo.cdsRepo) {
      return [];
    }
    const { projectKey, vcsName, id: repoId } = wf.repo.cdsRepo;

    // Use the repo UUID to avoid shell-quoting issues with slash-names
    const runs = await CDS.getWorkflowHistory(
      projectKey,
      vcsName,
      repoId,
      wf.cdsWorkflowName,
      15,
    );

    return runs.map((r) => new RunItem(r, wf));
  }

  private async findCdsDirs(dir: string, depth = 0): Promise<string[]> {
    if (depth > 3) {
      return [];
    }
    const found: string[] = [];
    try {
      const entries = await fsp.readdir(dir, { withFileTypes: true });
      for (const entry of entries) {
        if (!entry.isDirectory()) {
          continue;
        }
        const full = path.join(dir, entry.name);
        if (entry.name === ".cds") {
          found.push(full);
        } else if (
          !entry.name.startsWith(".") &&
          entry.name !== "node_modules"
        ) {
          found.push(...await this.findCdsDirs(full, depth + 1));
        }
      }
    } catch {
      /* skip unreadable dirs */
    }
    return found;
  }

  private async scanYamlFiles(dir: string): Promise<string[]> {
    const results: string[] = [];
    try {
      const entries = await fsp.readdir(dir, { withFileTypes: true });
      for (const entry of entries) {
        const full = path.join(dir, entry.name);
        if (entry.isDirectory()) {
          results.push(...await this.scanYamlFiles(full));
        } else if (
          entry.isFile() &&
          !entry.name.startsWith(".") &&
          (entry.name.endsWith(".yaml") || entry.name.endsWith(".yml"))
        ) {
          results.push(full);
        }
      }
    } catch {
      /* skip */
    }
    return results;
  }

  /**
   * Extract the "org/repo" slug from the git remote origin URL.
   * Handles SSH (git@host:org/repo.git) and HTTPS (https://host/org/repo.git).
   * Returns undefined if it cannot be determined.
   */
  private async getGitRemoteSlug(repoRoot: string): Promise<string | undefined> {
    try {
      const gitConfigPath = path.join(repoRoot, ".git", "config");
      const content = await fsp.readFile(gitConfigPath, "utf-8");
      // Find [remote "origin"] section and extract url
      const originMatch = content.match(
        /\[remote\s+"origin"\][^\[]*url\s*=\s*(.+)/m,
      );
      if (!originMatch) { return undefined; }
      const url = originMatch[1].trim();
      // SSH: git@host:org/repo.git or ssh://git@host/org/repo.git
      const sshMatch = url.match(/[:\/]([^/:]+\/[^/]+?)(?:\.git)?\s*$/);
      if (sshMatch) { return sshMatch[1]; }
      // HTTPS: https://host/org/repo.git
      const httpsMatch = url.match(/\/([^/]+\/[^/]+?)(?:\.git)?\s*$/);
      if (httpsMatch) { return httpsMatch[1]; }
    } catch {
      /* .git/config not readable */
    }
    return undefined;
  }

  /**
   * Match a CDS repository against a local directory.
   * Priority: git remote slug (exact match) > basename fallback.
   */
  private matchesRepo(
    cdsRepo: CdsRepository,
    dirName: string,
    remoteSlug: string | undefined,
  ): boolean {
    // If we have a git remote slug, use it for precise matching
    if (remoteSlug) {
      // cdsRepo.repoName is typically "org/repo" — compare with slug
      if (cdsRepo.repoName === remoteSlug) {
        return true;
      }
      // Some VCS store with a different prefix, try suffix match
      if (cdsRepo.repoName.endsWith("/" + remoteSlug.split("/").pop())) {
        // Only match if the org part also aligns
        return cdsRepo.repoName === remoteSlug;
      }
    }
    // Fallback to basename matching (only if no remote slug available)
    if (!remoteSlug) {
      return cdsRepo.repoName === dirName || cdsRepo.repoName.endsWith("/" + dirName);
    }
    return false;
  }
}
