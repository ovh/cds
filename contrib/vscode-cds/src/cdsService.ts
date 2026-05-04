import * as cp from "child_process";

export interface CdsRepository {
  id: string;
  repoName: string;
  vcsName: string;
  projectKey: string;
}

export interface CdsWorkflowRun {
  id: string;
  runNumber: number;
  status: string;
  started: string;
  workflowName: string;
  projectKey: string;
  username?: string;
}

const EXEC_TIMEOUT = 30_000;

function run(cmd: string, cwd?: string): Promise<string> {
  return new Promise((resolve, reject) => {
    cp.exec(
      cmd,
      { cwd, timeout: EXEC_TIMEOUT, env: { ...process.env } },
      (err, stdout, stderr) => {
        if (err) {
          reject(new Error((stderr || err.message).trim()));
        } else {
          resolve(stdout.trim());
        }
      },
    );
  });
}

/** Parse ASCII table rows from cdsctl (only data lines starting with |). */
export function parseTable(output: string): Record<string, string>[] {
  const lines = output.split("\n");
  // Find header line (first | line)
  const dataLines = lines.filter((l) => /^\|/.test(l));
  if (dataLines.length < 2) {
    return [];
  }
  const headers = dataLines[0]
    .split("|")
    .map((h) =>
      h
        .trim()
        .toLowerCase()
        .replace(/[\s_-]/g, ""),
    )
    .filter(Boolean);
  const rows: Record<string, string>[] = [];
  for (let i = 1; i < dataLines.length; i++) {
    const cols = dataLines[i]
      .split("|")
      .map((c) => c.trim())
      .filter(Boolean);
    // Skip continuation lines (only one non-empty cell)
    if (cols.length < 2) {
      continue;
    }
    const row: Record<string, string> = {};
    headers.forEach((h, idx) => {
      row[h] = cols[idx] ?? "";
    });
    rows.push(row);
  }
  return rows;
}

export function parseJson<T>(output: string): T[] {
  try {
    const parsed = JSON.parse(output);
    return Array.isArray(parsed) ? parsed : parsed ? [parsed] : [];
  } catch {
    return [];
  }
}

/** Parse a flat run record as returned by cdsctl --format json */
function flatRunToWorkflowRun(
  r: Record<string, string>,
  defaults: { projKey: string; workflowName?: string },
): CdsWorkflowRun {
  return {
    id: r["id"] ?? "",
    runNumber: parseInt(r["run_number"] ?? r["runnumber"] ?? "0", 10),
    status: r["status"] ?? "",
    started: r["started"] ?? r["start"] ?? r["last_modified"] ?? "",
    workflowName:
      r["workflow_name"] ?? r["workflowname"] ?? defaults.workflowName ?? "",
    projectKey: r["project_key"] ?? r["projectkey"] ?? defaults.projKey,
    username: r["username"] ?? "",
  };
}

export class CdsService {
  /** Returns all project keys available in the current cdsctl context. */
  async getProjectKeys(cwd?: string): Promise<string[]> {
    try {
      const out = await run("cdsctl project list --quiet", cwd);
      return out
        .split("\n")
        .map((l) => l.trim())
        .filter(Boolean);
    } catch {
      return [];
    }
  }

  /**
   * List all CDS v2 repositories across all project keys.
   */
  async listRepositories(cwd?: string): Promise<CdsRepository[]> {
    const projectKeys = await this.getProjectKeys(cwd);
    const results: CdsRepository[] = [];
    for (const projKey of projectKeys) {
      try {
        // Try JSON first, fall back to table
        let repos: Array<{ id: string; vcsName: string; repoName: string }> =
          [];
        try {
          const out = await run(
            `cdsctl experimental project repository list ${projKey} --format json`,
            cwd,
          );
          const parsed = parseJson<Record<string, string>>(out);
          repos = parsed.map((r) => ({
            id: r["id"] ?? "",
            vcsName: r["vcsName"] ?? r["vcsname"] ?? r["vcs_name"] ?? "",
            repoName: r["repoName"] ?? r["reponame"] ?? r["repo_name"] ?? "",
          }));
        } catch {
          const out = await run(
            `cdsctl experimental project repository list ${projKey}`,
            cwd,
          );
          repos = parseTable(out).map((r) => ({
            id: r["id"] ?? "",
            vcsName: r["vcsname"] ?? "",
            repoName: r["reponame"] ?? "",
          }));
        }
        for (const r of repos) {
          if (r.repoName) {
            results.push({
              id: r.id,
              repoName: r.repoName,
              vcsName: r.vcsName,
              projectKey: projKey,
            });
          }
        }
      } catch {
        // continue to next project key
      }
    }
    return results;
  }

  /**
   * Get run history for a specific v2 workflow.
   * Uses UUID for repoId to avoid shell quoting issues with slash-names.
   */
  async getWorkflowHistory(
    projKey: string,
    vcsName: string,
    repoId: string,
    workflowName: string,
    limit = 15,
    cwd?: string,
  ): Promise<CdsWorkflowRun[]> {
    const base = `cdsctl experimental workflow history ${projKey} ${vcsName} ${repoId} ${workflowName}`;
    try {
      const out = await run(`${base} --format json`, cwd);
      const items = parseJson<Record<string, string>>(out);
      return items
        .slice(0, limit)
        .map((r) => flatRunToWorkflowRun(r, { projKey, workflowName }));
    } catch {
      try {
        const out = await run(base, cwd);
        // table fallback
        return parseTable(out)
          .slice(0, limit)
          .map((r) => ({
            id: r["id"] ?? "",
            runNumber: parseInt(r["runnumber"] ?? r["run number"] ?? "0", 10),
            status: r["status"] ?? "",
            started: r["started"] ?? "",
            workflowName,
            projectKey: projKey,
            username: r["username"] ?? "",
          }));
      } catch {
        return [];
      }
    }
  }

  /**
   * List workflow names that have runs in a given repo by calling history for
   * every workflow discovered from a lightweight workflow search.
   * Returns unique workflow names scoped to a specific repository.
   * Step 1: collect all candidate names project-wide.
   * Step 2: verify each candidate belongs to this repo via --workflow vcs/repo/name filter.
   */
  async discoverRepoWorkflowNames(
    projKey: string,
    vcsName: string,
    repoName: string,
    cwd?: string,
  ): Promise<string[]> {
    // Step 1: collect candidate names across the whole project
    let candidates: string[] = [];
    try {
      const out = await run(
        `cdsctl experimental workflow search --project ${projKey} --limit 200 --format json`,
        cwd,
      );
      const items = parseJson<Record<string, string>>(out);
      const names = new Set<string>();
      for (const r of items) {
        const n = r["workflow_name"] ?? r["workflowname"] ?? "";
        if (n) {
          names.add(n);
        }
      }
      candidates = [...names];
    } catch {
      return [];
    }

    // Step 2: verify each candidate belongs to this specific repo using the
    // --workflow <vcs>/<repo>/<name> scoped filter (returns empty if not in repo)
    const verified: string[] = [];
    await Promise.all(
      candidates.map(async (name) => {
        try {
          const out = await run(
            `cdsctl experimental workflow search --project ${projKey} --workflow "${vcsName}/${repoName}/${name}" --limit 1 --format json`,
            cwd,
          );
          const items = parseJson<Record<string, string>>(out);
          if (items.length > 0) {
            verified.push(name);
          }
        } catch {
          // workflow not in this repo — skip
        }
      }),
    );
    return verified;
  }

  /** Build the cdsctl command to trigger a v2 workflow run (run in a terminal). */
  buildTriggerV2Command(
    projKey: string,
    vcsName: string,
    repoId: string,
    workflowName: string,
    branch?: string,
    tag?: string,
  ): string {
    let cmd = `cdsctl experimental workflow run ${projKey} ${vcsName} ${repoId} ${workflowName}`;
    if (branch) {
      cmd += ` --branch ${branch}`;
    }
    if (tag) {
      cmd += ` --tag ${tag}`;
    }
    return cmd;
  }

  /** Stop a running v2 workflow run. */
  async stopRun(projKey: string, runId: string, cwd?: string): Promise<void> {
    await run(`cdsctl experimental workflow stop ${projKey} ${runId}`, cwd);
  }

  /** Restart failed/stopped jobs in a v2 workflow run. */
  async restartRun(
    projKey: string,
    runId: string,
    cwd?: string,
  ): Promise<void> {
    await run(`cdsctl experimental workflow restart ${projKey} ${runId}`, cwd);
  }

  /** Build the command to download run logs (run in a terminal). */
  buildLogsCommand(projKey: string, runId: string): string {
    return `cdsctl experimental workflow logs download ${projKey} ${runId}`;
  }
}
