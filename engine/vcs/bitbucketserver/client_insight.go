package bitbucketserver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ovh/cds/sdk"
)

func (b *bitbucketClient) CreateInsightReport(ctx context.Context, repo string, sha string, insightKey string, vcsReport sdk.VCSInsight) error {
	project, slug, err := getRepo(repo)
	if err != nil {
		return err
	}

	r := InsightReport{
		Title:    vcsReport.Title,
		Detail:   vcsReport.Detail,
		Data:     make([]InsightReportData, 0, len(vcsReport.Datas)),
		Reporter: "CDS",
	}
	for _, d := range vcsReport.Datas {
		data := InsightReportData{
			Title: d.Title,
		}
		if d.Href != "" {
			data.Type = "LINK"
			data.Value = InsightReportDataLink{
				Text: d.Text,
				Href: d.Href,
			}
		} else {
			data.Type = "TEXT"
			data.Value = d.Text
		}
		r.Data = append(r.Data, data)
	}

	values, err := json.Marshal(r)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/projects/%s/repos/%s/commits/%s/reports/%s", project, slug, sha, insightKey)
	return b.do(ctx, "PUT", "insights", path, nil, values, nil, Options{})
}
