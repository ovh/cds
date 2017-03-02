package web

import (
	"fmt"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/fsamin/go-dump"
	"github.com/mitchellh/mapstructure"
	"github.com/sclevine/agouti"

	"github.com/runabove/venom"
	"github.com/runabove/venom/context/webctx"
)

// Name of executor
const Name = "web"

// New returns a new Executor
func New() venom.Executor {
	return &Executor{}
}

// Executor struct
type Executor struct {
	Action     Action `json:"action,omitempty" yaml:"action"`
	Screenshot string `json:"screenshot,omitempty" yaml:"screenshot"`
}

// Result represents a step result
type Result struct {
	Executor    Executor `json:"executor,omitempty" yaml:"executor,omitempty"`
	Find        int      `json:"find,omitempty" yaml:"find,omitempty"`
	HTML        string   `json:"html,omitempty" yaml:"html,omitempty"`
	TimeSeconds float64  `json:"timeseconds,omitempty" yaml:"timeseconds,omitempty"`
	TimeHuman   string   `json:"timehuman,omitempty" yaml:"timehuman,omitempty"`
	Title       string   `json:"title,omitempty" yaml:"title,omitempty"`
	URL         string   `json:"url,omitempty" yaml:"url,omitempty"`
}

// Run execute TestStep
func (Executor) Run(testCaseContext venom.TestCaseContext, l *log.Entry, step venom.TestStep) (venom.ExecutorResult, error) {
	var ctx *webctx.WebTestCaseContext
	switch testCaseContext.(type) {
	case *webctx.WebTestCaseContext:
		ctx = testCaseContext.(*webctx.WebTestCaseContext)
	default:
		return nil, fmt.Errorf("Web executor need a Web context")
	}

	start := time.Now()

	// transform step to Executor Instance
	var t Executor
	if err := mapstructure.Decode(step, &t); err != nil {
		return nil, err
	}
	r := &Result{Executor: t}

	// Check action to realise
	if t.Action.Click != "" {
		s, err := find(ctx.Page, t.Action.Click, r)
		if err != nil {
			return nil, err
		}
		if err := s.Click(); err != nil {
			return nil, err
		}
	} else if t.Action.Fill != nil {
		for _, f := range t.Action.Fill {
			s, err := findOne(ctx.Page, f.Find, r)
			if err != nil {
				return nil, err
			}
			if err := s.Fill(f.Text); err != nil {
				return nil, err
			}
		}

	} else if t.Action.Find != "" {
		_, err := find(ctx.Page, t.Action.Find, r)
		if err != nil {
			return nil, err
		}
	} else if t.Action.Navigate != "" {
		if err := ctx.Page.Navigate(t.Action.Navigate); err != nil {
			return nil, err
		}
	}

	// take a screenshot
	if t.Screenshot != "" {
		if err := ctx.Page.Screenshot(t.Screenshot); err != nil {
			return nil, err
		}
	}

	// get page title
	title, err := ctx.Page.Title()
	if err != nil {
		return nil, err
	}
	r.Title = title

	url, errU := ctx.Page.URL()
	if errU != nil {
		return nil, fmt.Errorf("Cannot get URL: %s", errU)
	}
	r.URL = url

	elapsed := time.Since(start)
	r.TimeSeconds = elapsed.Seconds()
	r.TimeHuman = fmt.Sprintf("%s", elapsed)

	return dump.ToMap(r, dump.WithDefaultLowerCaseFormatter())
}

func find(page *agouti.Page, search string, r *Result) (*agouti.Selection, error) {
	s := page.Find(search)
	if s == nil {
		return nil, fmt.Errorf("Cannot find element %s", search)
	}
	nbElement, errC := s.Count()
	if errC != nil {
		if !strings.Contains(errC.Error(), "element not found") {
			return nil, fmt.Errorf("Cannot count element %s: %s", search, errC)
		}
		nbElement = 0
	}
	r.Find = nbElement
	return s, nil
}

func findOne(page *agouti.Page, search string, r *Result) (*agouti.Selection, error) {
	s := page.Find(search)
	if s == nil {
		return nil, fmt.Errorf("Cannot find element %s", search)
	}
	nbElement, errC := s.Count()
	if errC != nil {
		return nil, fmt.Errorf("Cannot find element %s: %s", search, errC)
	}
	if nbElement != 1 {
		return nil, fmt.Errorf("Find %s elements", nbElement)
	}
	return s, nil
}
