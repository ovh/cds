package artifactory

import (
	"context"
	"encoding/json"
	"io"
	"sort"
	"strings"

	"github.com/pkg/errors"

	"github.com/ovh/cds/sdk"
)

// Artifactory refuses a query past 6000 characters with "AQL query is too long". The margin
// below that ceiling absorbs the paths, whose length varies a lot from one run to the next.
const aqlMaxQueryLength = 5000

// SearchItems returns, among the given locations, those carrying the given property key, or
// all of them when the key is empty. The property is only a filter. Asking for it in the
// answer makes artifactory count one database row per property in place of one per item.
// The checksums are item columns and come with each item.
//
// It does not paginate. The caller asks about a known set of items, and a trimmed answer is
// reported as an error.
func (c *Client) SearchItems(_ context.Context, locations []sdk.ArtifactLocation, propertyKey string) (sdk.ArtifactResults, error) {
	queries, err := buildItemsQueries(locations, propertyKey)
	if err != nil {
		return nil, err
	}

	var results sdk.ArtifactResults
	for _, q := range queries {
		page, err := c.runItemsQuery(q.query)
		if err != nil {
			return nil, err
		}
		// A trimmed batch reads like a set of items without the property
		if page.Range.Notification != "" {
			return nil, sdk.NewErrorFrom(sdk.ErrUnknownError, "artifactory trimmed the search: %s", page.Range.Notification)
		}
		if len(page.Results) > q.locations {
			return nil, sdk.NewErrorFrom(sdk.ErrUnknownError, "artifactory returned %d items for %d locations", len(page.Results), q.locations)
		}
		results = append(results, page.Results...)
	}
	return results, nil
}

func (c *Client) runItemsQuery(query string) (*sdk.ArtifactResultsSearchPage, error) {
	body, err := c.Asm.Aql(query)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	bts, err := io.ReadAll(body)
	_ = body.Close()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var page sdk.ArtifactResultsSearchPage
	if err := json.Unmarshal(bts, &page); err != nil {
		return nil, errors.WithStack(err)
	}
	return &page, nil
}

type itemsQuery struct {
	query     string
	locations int
}

// Locations sharing a repository and a directory go under one branch, with the directory
// written once. The artifacts of a run mostly sit in the same folder, and the directory is
// the longest part of a branch.
func buildItemsQueries(locations []sdk.ArtifactLocation, propertyKey string) ([]itemsQuery, error) {
	if len(locations) == 0 {
		return nil, nil
	}
	byRepository := make(map[string]map[string][]string)
	for _, l := range locations {
		if byRepository[l.Repository] == nil {
			byRepository[l.Repository] = make(map[string][]string)
		}
		byRepository[l.Repository][l.Path] = append(byRepository[l.Repository][l.Path], l.Name)
	}

	var queries []itemsQuery
	for _, repository := range sortedKeys(byRepository) {
		repositoryQueries, err := buildRepositoryQueries(repository, byRepository[repository], propertyKey)
		if err != nil {
			return nil, err
		}
		queries = append(queries, repositoryQueries...)
	}
	return queries, nil
}

func buildRepositoryQueries(repository string, namesByDirectory map[string][]string, propertyKey string) ([]itemsQuery, error) {
	head, tail, err := itemsQueryEnvelope(repository, propertyKey)
	if err != nil {
		return nil, err
	}

	budget := aqlMaxQueryLength - len(head) - len(tail)
	if budget <= 0 {
		return nil, sdk.NewErrorFrom(sdk.ErrInvalidData, "repository %q does not fit in an aql query", repository)
	}

	var (
		queries   []itemsQuery
		branches  []string
		size      int
		nbLocated int
	)
	flush := func() {
		if len(branches) == 0 {
			return
		}
		queries = append(queries, itemsQuery{
			query:     head + strings.Join(branches, ",") + tail,
			locations: nbLocated,
		})
		branches, size, nbLocated = nil, 0, 0
	}

	for _, directory := range sortedKeys(namesByDirectory) {
		names := append([]string{}, namesByDirectory[directory]...)
		sort.Strings(names)

		for len(names) > 0 {
			branch, used, err := directoryBranch(directory, names, budget-size)
			if err != nil {
				return nil, err
			}
			if used == 0 {
				// Nothing fits next to what is already queued. With an empty batch, a
				// single name is longer than a whole query.
				if len(branches) == 0 {
					return nil, sdk.NewErrorFrom(sdk.ErrInvalidData, "directory %q does not fit in an aql query", directory)
				}
				flush()
				continue
			}
			branches = append(branches, branch)
			size += len(branch) + 1 // the comma joining it to the previous branch
			nbLocated += used
			names = names[used:]
		}
	}
	flush()

	return queries, nil
}

func itemsQueryEnvelope(repository, propertyKey string) (string, string, error) {
	repositoryJSON, err := json.Marshal(repository)
	if err != nil {
		return "", "", errors.WithStack(err)
	}
	var propertyFilter string
	if propertyKey != "" {
		propertyJSON, err := json.Marshal("@" + propertyKey)
		if err != nil {
			return "", "", errors.WithStack(err)
		}
		propertyFilter = `{` + string(propertyJSON) + `:{"$match":"*"}},`
	}

	// The default of items.find is type file, and a run result can be a folder
	head := `items.find({"$and":[{"repo":{"$eq":` + string(repositoryJSON) +
		`}},{"type":{"$eq":"any"}},` + propertyFilter + `{"$or":[`
	tail := `]}]}).include("repo","path","name","type","actual_md5","actual_sha1","sha256")`
	return head, tail, nil
}

// A count of zero means not even the first name fits.
func directoryBranch(directory string, names []string, budget int) (string, int, error) {
	directoryJSON, err := json.Marshal(directory)
	if err != nil {
		return "", 0, errors.WithStack(err)
	}

	head := `{"$and":[{"path":{"$eq":` + string(directoryJSON) + `}},{"$or":[`
	tail := `]}]}`

	var clauses []string
	size := len(head) + len(tail)
	for _, name := range names {
		nameJSON, err := json.Marshal(name)
		if err != nil {
			return "", 0, errors.WithStack(err)
		}
		clause := `{"name":{"$eq":` + string(nameJSON) + `}}`

		next := size + len(clause)
		if len(clauses) > 0 {
			next++ // the comma joining it to the previous clause
		}
		if next > budget {
			break
		}
		clauses = append(clauses, clause)
		size = next
	}

	if len(clauses) == 0 {
		return "", 0, nil
	}
	return head + strings.Join(clauses, ",") + tail, len(clauses), nil
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
