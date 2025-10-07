package api

import (
	"context"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/sdk"
)

func (api *API) initMCP() {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "CDS MCP server",
		Version: sdk.VERSION,
	}, nil)

	projectList := &mcp.Tool{
		Name:        "CDSGetProjects",
		Description: "List CDS projects. Can be filtered by projectKey",
		Title:       "Get CDS projects list",
	}

	mcp.AddTool(server, projectList, api.projectsList)

	api.mcpServer = server
	api.mcpHandler = mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return api.mcpServer
	}, nil)
}

type MCPProject struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type MCPProjectInput struct {
	ProjectKey string `json:"projectKey,omitempty"`
}

type MCPProjects struct {
	Projects []MCPProject `json:"projects"`
}

func (api *API) projectsList(ctx context.Context, req *mcp.CallToolRequest, input MCPProjectInput) (
	*mcp.CallToolResult,
	MCPProjects,
	error,
) {
	if input.ProjectKey != "" {
		proj, err := project.Load(ctx, api.mustDBWithCtx(ctx), input.ProjectKey)
		if err != nil {
			return nil, MCPProjects{}, err
		}
		output := MCPProjects{Projects: []MCPProject{{
			Key:         proj.Key,
			Name:        proj.Name,
			Description: proj.Description,
		}}}
		return nil, output, nil
	}
	projs, err := project.LoadAll(ctx, api.mustDBWithCtx(ctx), api.Cache)
	if err != nil {
		return nil, MCPProjects{}, err
	}
	mcpProjs := make([]MCPProject, 0, len(projs))
	for _, p := range projs {
		mcpProjs = append(mcpProjs, MCPProject{
			Key:         p.Key,
			Name:        p.Name,
			Description: p.Description,
		})
	}
	output := MCPProjects{Projects: mcpProjs}
	return nil, output, nil

}
