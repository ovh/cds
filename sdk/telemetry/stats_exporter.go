package telemetry

import (
	"reflect"
	"time"

	"go.opencensus.io/stats/view"
)

func (e *HTTPExporter) GetView(name string, tags map[string]string) *HTTPExporterView {
	for i := range e.Views {
		if e.Views[i].Name == name && reflect.DeepEqual(e.Views[i].Tags, tags) {
			return &e.Views[i]
		}
	}
	return nil
}

func (e *HTTPExporter) NewView(name string, tags map[string]string) *HTTPExporterView {
	v := HTTPExporterView{
		Name: name,
		Tags: tags,
	}
	e.Views = append(e.Views, v)
	return &v
}

func (e *HTTPExporter) ExportView(vd *view.Data) {
	for _, row := range vd.Rows {
		tags := make(map[string]string)
		for _, t := range row.Tags {
			tags[t.Key.Name()] = t.Value
		}
		view := e.GetView(vd.View.Name, tags)
		if view == nil {
			view = e.NewView(vd.View.Name, tags)
		}
		view.Record(row.Data)
	}
}

func (v *HTTPExporterView) Record(data view.AggregationData) {
	v.Date = time.Now()
	switch x := data.(type) {
	case *view.DistributionData:
		v.Value = x.Mean
	case *view.CountData:
		v.Value = float64(x.Value)
	case *view.SumData:
		v.Value = x.Value
	case *view.LastValueData:
		v.Value = x.Value
	}
}
