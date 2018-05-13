package tracing

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"
	"go.opencensus.io/exporter/jaeger"
	"go.opencensus.io/trace"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/feature"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var exporter trace.Exporter

/*
	Init the tracer
	Start jarger with:
	docker run -d -e \
		COLLECTOR_ZIPKIN_HTTP_PORT=9411 \
		-p 5775:5775/udp \
		-p 6831:6831/udp \
		-p 6832:6832/udp \
		-p 5778:5778 \
		-p 16686:16686 \
		-p 14268:14268 \
		-p 9411:9411 \
		jaegertracing/all-in-one:latest
*/
func Init() error {
	var err error
	exporter, err = jaeger.NewExporter(jaeger.Options{
		Endpoint:    "http://localhost:14268",
		ServiceName: "cds-tracing",
	})
	if err != nil {
		return err
	}
	trace.RegisterExporter(exporter)
	trace.ApplyConfig(
		trace.Config{
			DefaultSampler: trace.ProbabilitySampler(.07),
		},
	)

	return nil
}

//Options is the options struct for a new tracing span
type Options struct {
	Name     string
	Enable   bool
	User     *sdk.User
	Worker   *sdk.Worker
	Hatchery *sdk.Hatchery
}

// Start may start a tracing span
func Start(ctx context.Context, w http.ResponseWriter, req *http.Request, opt Options, db gorp.SqlExecutor, store cache.Store) (context.Context, error) {
	if !opt.Enable {
		return ctx, nil
	}

	log.Debug("tracing.Start> staring a new %s span", opt.Name)

	tags := []trace.Attribute{trace.StringAttribute("path", req.URL.Path)}
	if opt.Worker != nil {
		tags = append(tags, trace.StringAttribute("worker", opt.Worker.Name))
	}
	if opt.Hatchery != nil {
		tags = append(tags, trace.StringAttribute("hatchery", opt.Hatchery.Name))
	}
	if opt.User != nil {
		tags = append(tags, trace.StringAttribute("user", opt.User.Username))
	}

	var span *trace.Span

	vars := mux.Vars(req)
	pkey := vars["key"]
	if pkey == "" {
		pkey = vars["permProjectKey"]
	}

	if pkey == "" {
		id, _ := strconv.ParseInt(vars["id"], 10, 64)
		//The ID found may be a node run job, let's try to find the project key behing
		if id <= 0 {
			id, _ = strconv.ParseInt(vars["permID"], 10, 64)
		}
		if id != 0 {
			var err error
			cacheKey := cache.Key("api:FindProjetKeyForNodeRunJob:", fmt.Sprintf("%v", id))
			if !store.Get(cacheKey, &pkey) {
				pkey, err = findProjetKeyForNodeRunJob(db, id)
				if err != nil {
					log.Error("tracingMiddleware> %v", err)
					return ctx, nil
				}
				store.SetWithTTL(cacheKey, pkey, 60*15)
			}
		}
	}

	if pkey != "" {
		tags = append(tags, trace.StringAttribute("project_key", pkey))
	}

	if pkey == "" || !feature.IsEnabled(store, feature.FeatEnableTracing, pkey) {
		ctx, span = trace.StartSpan(ctx, opt.Name)
		span.AddAttributes(tags...)
		return ctx, nil
	}

	ctx, span = trace.StartSpan(ctx, opt.Name, trace.WithSampler(trace.AlwaysSample()))
	span.AddAttributes(tags...)
	return ctx, nil
}

// End may close a tracing span
func End(ctx context.Context, w http.ResponseWriter, req *http.Request) (context.Context, error) {
	span := trace.FromContext(ctx)
	if span == nil {
		return ctx, nil
	}
	span.End()
	return ctx, nil
}

// Current return the current span
func Current(ctx context.Context, tags ...trace.Attribute) *trace.Span {
	if ctx == nil {
		return nil
	}
	span := trace.FromContext(ctx)
	if span == nil {
		return nil
	}
	if len(tags) > 0 {
		span.AddAttributes(tags...)
	}
	return span
}

// Tag is helper function to instanciate trace.Attribute
func Tag(key string, value interface{}) trace.Attribute {
	return trace.StringAttribute(key, fmt.Sprintf("%v", value))
}

// Span start a new span from the parent context
func Span(ctx context.Context, name string, tags ...trace.Attribute) (context.Context, func()) {
	if ctx == nil {
		return nil, func() {}
	}
	var span *trace.Span
	ctx, span = trace.StartSpan(ctx, name)
	if len(tags) > 0 {
		span.AddAttributes(tags...)
	}
	return ctx, span.End
}

// findProjetKeyForNodeRunJob load the project key from a workflow_node_run_job ID
func findProjetKeyForNodeRunJob(db gorp.SqlExecutor, id int64) (string, error) {
	query := `select project.projectkey from project
	join workflow on workflow.project_id = project.id
	join workflow_run on workflow_run.workflow_id = workflow.id
	join workflow_node_run on workflow_node_run.workflow_run_id = workflow_run.id
	join workflow_node_run_job on workflow_node_run_job.workflow_node_run_id = workflow_node_run.id
	where workflow_node_run_job.id = $1`
	pkey, err := db.SelectNullStr(query, id)
	if err != nil {
		return "", sdk.WrapError(err, "FindProjetKeyForNodeRunJob")
	}
	if pkey.Valid {
		return pkey.String, nil
	}
	log.Warning("FindProjetKeyForNodeRunJob> project key not found for node run job %d", id)
	return "", nil
}
