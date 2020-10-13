package workflow

import (
	"context"
	"database/sql"

	"github.com/go-gorp/gorp"
	"github.com/sguiheux/go-coverage"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/metrics"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func loadPreviousCoverageReport(db gorp.SqlExecutor, workflowID int64, runNumber int64, repository string, branch string, appID int64) (sdk.WorkflowNodeRunCoverage, error) {
	query := `
      SELECT * from workflow_node_run_coverage
      WHERE run_number < $1 AND repository = $2 AND branch = $3 AND workflow_id = $4 AND application_id = $5
      ORDER BY run_number DESC
      LIMIT 1
  `
	var cov Coverage
	if err := db.SelectOne(&cov, query, runNumber, repository, branch, workflowID, appID); err != nil {
		if err == sql.ErrNoRows {
			return sdk.WorkflowNodeRunCoverage{}, sdk.ErrNotFound
		}
		return sdk.WorkflowNodeRunCoverage{}, sdk.WrapError(err, "Unable to load previous coverage")
	}

	return sdk.WorkflowNodeRunCoverage(cov), nil
}

func loadLatestCoverageReport(db gorp.SqlExecutor, workflowID int64, repository string, branch string, appID int64) (sdk.WorkflowNodeRunCoverage, error) {
	query := `
      SELECT * from workflow_node_run_coverage
      WHERE workflow_id = $1 AND repository = $2 AND branch = $3 AND application_id = $4
      ORDER BY run_number DESC
      LIMIT 1
  `
	var cov Coverage
	if err := db.SelectOne(&cov, query, workflowID, repository, branch, appID); err != nil {
		if err == sql.ErrNoRows {
			return sdk.WorkflowNodeRunCoverage{}, sdk.ErrNotFound
		}
		return sdk.WorkflowNodeRunCoverage{}, sdk.WrapError(err, "Unable to load latest coverage")
	}

	return sdk.WorkflowNodeRunCoverage(cov), nil
}

// LoadCoverageReport loads a coverage report
func LoadCoverageReport(db gorp.SqlExecutor, workflowNodeRunID int64) (sdk.WorkflowNodeRunCoverage, error) {
	query := `
    SELECT * from workflow_node_run_coverage
    WHERE workflow_node_run_id = $1
  `
	var cov Coverage
	if err := db.SelectOne(&cov, query, workflowNodeRunID); err != nil {
		if err == sql.ErrNoRows {
			return sdk.WorkflowNodeRunCoverage{}, sdk.ErrNotFound
		}
		return sdk.WorkflowNodeRunCoverage{}, sdk.WrapError(err, "Unable to load coverage")
	}

	return sdk.WorkflowNodeRunCoverage(cov), nil
}

// InsertCoverage insert a coverage report for a workflow run
func InsertCoverage(db gorp.SqlExecutor, cov sdk.WorkflowNodeRunCoverage) error {
	c := Coverage(cov)
	if err := db.Insert(&c); err != nil {
		return sdk.WrapError(err, "Unable to insert code coverage report")
	}
	return nil
}

// UpdateCoverage update a coverage report for a workflow run
func UpdateCoverage(db gorp.SqlExecutor, cov sdk.WorkflowNodeRunCoverage) error {
	c := Coverage(cov)
	if _, err := db.Update(&c); err != nil {
		return sdk.WrapError(err, "Unable to update code coverage report")
	}
	return nil
}

// PostGet is a db hook on workflow_node_run_coverage
func (c *Coverage) PostGet(s gorp.SqlExecutor) error {
	var report, trend sql.NullString
	query := "SELECT report, trend FROM workflow_node_run_coverage WHERE workflow_node_run_id=$1"
	if err := s.QueryRow(query, c.WorkflowNodeRunID).Scan(&report, &trend); err != nil {
		return sdk.WrapError(err, "Unable to get report and trend")
	}

	if err := gorpmapping.JSONNullString(report, &c.Report); err != nil {
		return sdk.WrapError(err, "Unable to unmarshal report")
	}

	if err := gorpmapping.JSONNullString(trend, &c.Trend); err != nil {
		return sdk.WrapError(err, "Unable to unmarshal trend")
	}
	return nil
}

// PostInsert is a db hook on workflow_node_run_coverage
func (c *Coverage) PostInsert(s gorp.SqlExecutor) error {
	return c.PostUpdate(s)
}

// PostUpdate is a db hook on workflow_node_run_coverage
func (c *Coverage) PostUpdate(s gorp.SqlExecutor) error {
	reportS, errR := gorpmapping.JSONToNullString(c.Report)
	if errR != nil {
		return sdk.WrapError(errR, "workflow.coverage.postupdate> Unable to stringify report")
	}
	trendS, errT := gorpmapping.JSONToNullString(c.Trend)
	if errT != nil {
		return sdk.WrapError(errT, "workflow.coverage.postupdate> Unable to stringify trend")
	}

	query := `
    UPDATE workflow_node_run_coverage
    SET report=$1, trend=$2
    WHERE workflow_node_run_id=$3`
	if _, err := s.Exec(query, reportS, trendS, c.WorkflowNodeRunID); err != nil {
		return sdk.WrapError(err, "Unable to update report and trend")
	}

	return nil
}

// ComputeNewReport compute trends and import new coverage report
func ComputeNewReport(ctx context.Context, db gorpmapper.SqlExecutorWithTx, cache cache.Store, report coverage.Report, wnr *sdk.WorkflowNodeRun, proj sdk.Project) error {
	covReport := sdk.WorkflowNodeRunCoverage{
		WorkflowID:        wnr.WorkflowID,
		WorkflowRunID:     wnr.WorkflowRunID,
		WorkflowNodeRunID: wnr.ID,
		ApplicationID:     wnr.ApplicationID,
		Num:               wnr.Number,
		Repository:        wnr.VCSRepository,
		Branch:            wnr.VCSBranch,
		Report:            report,
		Trend:             sdk.WorkflowNodeRunCoverageTrends{},
	}

	// Get previous report
	previousReport, err := loadPreviousCoverageReport(db, wnr.WorkflowID, wnr.Number, wnr.VCSRepository, wnr.VCSBranch, covReport.ApplicationID)
	if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
		return sdk.WrapError(err, "unable to load previous report")
	}
	if !sdk.ErrorIs(err, sdk.ErrNotFound) {
		// remove data we don't need
		previousReport.Report.Files = nil
		covReport.Trend.CurrentBranch = previousReport.Report
	}

	if err := ComputeLatestDefaultBranchReport(ctx, db, cache, proj, wnr, &covReport); err != nil {
		return sdk.WrapError(err, "Unable to get default branch coverage report")
	}

	if err := InsertCoverage(db, covReport); err != nil {
		return sdk.WrapError(err, "Unable to insert coverage report")
	}

	return nil
}

// ComputeLatestDefaultBranchReport add the default branch coverage report into  the given report
func ComputeLatestDefaultBranchReport(ctx context.Context, db gorpmapper.SqlExecutorWithTx, cache cache.Store, proj sdk.Project, wnr *sdk.WorkflowNodeRun, covReport *sdk.WorkflowNodeRunCoverage) error {
	// Get report latest report on previous branch
	var defaultBranch string
	projectVCSServer, err := repositoriesmanager.LoadProjectVCSServerLinkByProjectKeyAndVCSServerName(ctx, db, proj.Key, wnr.VCSServer)
	if err != nil {
		return err
	}
	client, erra := repositoriesmanager.AuthorizedClient(ctx, db, cache, proj.Key, projectVCSServer)
	if erra != nil {
		return sdk.WrapError(sdk.ErrNoReposManagerClientAuth, "ComputeLatestDefaultBranchReport> Cannot get repo client %s : %s", wnr.VCSServer, erra)
	}

	branches, err := client.Branches(ctx, wnr.VCSRepository)
	if err != nil {
		return err
	}
	for _, b := range branches {
		if b.Default {
			defaultBranch = b.DisplayID
			break
		}
	}

	if defaultBranch != wnr.VCSBranch {
		defaultCoverage, errD := loadLatestCoverageReport(db, wnr.WorkflowID, wnr.VCSRepository, defaultBranch, covReport.ApplicationID)
		if errD != nil && !sdk.ErrorIs(errD, sdk.ErrNotFound) {
			return sdk.WrapError(errD, "ComputeLatestDefaultBranchReport> Cannot get latest report on default branch")
		}
		defaultCoverage.Report.Files = nil
		covReport.Trend.DefaultBranch = defaultCoverage.Report
	} else {
		metrics.PushCoverage(proj.Key, wnr.ApplicationID, wnr.WorkflowID, wnr.Number, covReport.Report)
	}

	return nil
}
