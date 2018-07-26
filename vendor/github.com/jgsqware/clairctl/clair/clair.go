package clair

import (
	"strconv"
	"strings"

	"github.com/coreos/clair/api/v1"
	"github.com/coreos/pkg/capnslog"
	"github.com/jgsqware/clairctl/xstrings"
	"github.com/spf13/viper"
)

var log = capnslog.NewPackageLogger("github.com/jgsqware/clairctl", "clair")

var uri string
var healthURI string

//ImageAnalysis Full image analysis
type ImageAnalysis struct {
	Registry, ImageName, Tag string
	Layers                   []v1.LayerEnvelope
}

func (imageAnalysis ImageAnalysis) String() string {
	return imageAnalysis.Registry + "/" + imageAnalysis.ImageName + ":" + imageAnalysis.Tag
}

//MostRecentLayer returns the most recent layer of an ImageAnalysis object
func (imageAnalysis ImageAnalysis) MostRecentLayer() v1.LayerEnvelope {
	return imageAnalysis.Layers[0]
}

func fmtURI(u string, port int) string {

	if port != 0 {
		u += ":" + strconv.Itoa(port)
	}
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		u = "http://" + u
	}

	return u
}

func (imageAnalysis ImageAnalysis) ShortName(l v1.Layer) string {
	return xstrings.Substr(l.Name, 0, 12)
}

//Config configure Clair from configFile
func Config() {
	uri = fmtURI(viper.GetString("clair.uri"), viper.GetInt("clair.port")) + "/v1"
	healthURI = fmtURI(viper.GetString("clair.uri"), viper.GetInt("clair.healthPort")) + "/health"
	Report.Path = viper.GetString("clair.report.path")
	Report.Format = viper.GetString("clair.report.format")
}
