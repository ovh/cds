package ui

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/spf13/afero"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/log"
)

// New returns a new service
func New() *Service {
	s := new(Service)
	s.GoRoutines = sdk.NewGoRoutines()
	s.Router = &api.Router{
		Mux: mux.NewRouter(),
	}
	return s
}

func (s *Service) Init(config interface{}) (cdsclient.ServiceConfig, error) {
	var cfg cdsclient.ServiceConfig
	sConfig, ok := config.(Configuration)
	if !ok {
		return cfg, sdk.WithStack(fmt.Errorf("invalid ui service configuration"))
	}

	cfg.Host = sConfig.API.HTTP.URL
	cfg.Token = sConfig.API.Token
	cfg.InsecureSkipVerifyTLS = sConfig.API.HTTP.Insecure
	cfg.RequestSecondsTimeout = sConfig.API.RequestTimeout
	return cfg, nil
}

// ApplyConfiguration apply an object of type hooks.Configuration after checking it
func (s *Service) ApplyConfiguration(config interface{}) error {
	if err := s.CheckConfiguration(config); err != nil {
		return err
	}
	var ok bool
	s.Cfg, ok = config.(Configuration)
	if !ok {
		return fmt.Errorf("Invalid configuration")
	}

	s.ServiceName = s.Cfg.Name
	s.ServiceType = sdk.TypeUI
	s.HTTPURL = s.Cfg.URL
	s.MaxHeartbeatFailures = s.Cfg.API.MaxHeartbeatFailures

	// HTMLDir must contains the ui dist directory.
	// ui.tar.gz contains the dist directory
	s.HTMLDir = filepath.Join(s.Cfg.Staticdir, "dist")
	s.DocsDir = filepath.Join(s.Cfg.Staticdir, "docs")
	s.Cfg.BaseURL = strings.TrimSpace(s.Cfg.BaseURL)
	if s.Cfg.BaseURL == "" { // s.Cfg.BaseURL could not be empty
		s.Cfg.BaseURL = "/"
	}

	return nil
}

// CheckConfiguration checks the validity of the configuration object
func (s *Service) CheckConfiguration(config interface{}) error {
	sConfig, ok := config.(Configuration)
	if !ok {
		return fmt.Errorf("Invalid configuration")
	}

	if sConfig.URL == "" {
		return fmt.Errorf("your CDS configuration seems to be empty. Please use environment variables, file or Consul to set your configuration")
	}
	if sConfig.Name == "" {
		return fmt.Errorf("please enter a name in your ui configuration")
	}

	return nil
}

// Serve will start the http ui server
func (s *Service) BeforeStart(ctx context.Context) error {
	log.Info(ctx, "ui> Starting service %s %s...", s.Cfg.Name, sdk.VERSION)
	s.StartupTime = time.Now()

	fromTmpl, err := s.prepareIndexHTML()
	if err != nil {
		return err
	}

	if err := s.checkStaticFiles(ctx); err != nil {
		return err
	}

	if fromTmpl {
		// if we have a index.tmpl, it's from a ui.tar.gz
		// we can check the checksum or files based on FILES_UI
		if err := s.checkChecksumFiles(); err != nil {
			return err
		}
	}

	if err := s.indexHTMLReplaceVar(); err != nil {
		return err
	}

	//Init the http server
	s.initRouter(ctx)
	s.Server = &http.Server{
		Addr:           fmt.Sprintf("%s:%d", s.Cfg.HTTP.Addr, s.Cfg.HTTP.Port),
		Handler:        s.Router.Mux,
		ReadTimeout:    10 * time.Minute,
		WriteTimeout:   10 * time.Minute,
		MaxHeaderBytes: 1 << 20,
	}

	// Start the http server
	s.GoRoutines.Run(ctx, "ui-http-serve", func(ctx context.Context) {
		log.Info(ctx, "ui> Starting HTTP Server on port %d", s.Cfg.HTTP.Port)
		if err := s.Server.ListenAndServe(); err != nil {
			log.Error(ctx, "ui> Listen and serve failed: %s", err)
		}
	})

	return nil
}

func (s *Service) Serve(ctx context.Context) error {
	// Gracefully shutdown the http server
	<-ctx.Done()
	log.Info(ctx, "ui> Shutdown HTTP Server")
	if err := s.Server.Shutdown(ctx); err != nil {
		return fmt.Errorf("unable to shutdown server: %v", err)
	}

	return ctx.Err()
}

// checkChecksumFiles checks the sha512 values.
// a ui.tar.gz contains a FILES_UI, with a lines as:
// filename;shar512values
// for each line, we check that the files in dist have the same sha512
func (s *Service) checkChecksumFiles() error {
	log.Debug("ui> checking checksum files...")

	filesUI := filepath.Join(s.HTMLDir, "FILES_UI")
	content, err := ioutil.ReadFile(filesUI)
	if err != nil {
		return sdk.WrapError(err, "error while reading file %s", filesUI)
	}
	lines := strings.Split(string(content), "\n")

	for _, lineValues := range lines {
		line := strings.Split(lineValues, ";")
		if len(line) != 2 {
			continue
		}
		sha512sum, err512 := sdk.FileSHA512sum(filepath.Join(s.HTMLDir, line[0]))
		if err512 != nil {
			return sdk.WrapError(err512, "error while compute sha512 on %s", line[0])
		}
		if line[1] != sha512sum {
			return fmt.Errorf("file %s sha512:%s computed:%s", line[0], line[1], sha512sum)
		}
	}
	log.Debug("ui> checking checksum files OK")
	return nil
}

func (s *Service) checkStaticFiles(ctx context.Context) error {
	fs := http.Dir(s.HTMLDir)

	if _, err := fs.Open("index.html"); os.IsNotExist(err) {
		log.Warning(ctx, "ui> CDS UI static files were not found in directory %v", s.HTMLDir)

		if err := s.askForGettingStaticFiles(ctx, sdk.VERSION); err != nil {
			return err
		}

		// reset the fs, HTMLDir could be updated by user answer
		fs = http.Dir(s.HTMLDir)
		// recheck file after user answer
		if _, err := fs.Open("index.html"); os.IsNotExist(err) {
			return fmt.Errorf("CDS UI static files were not found in directory %v", s.HTMLDir)
		}
	}
	log.Info(ctx, "ui> CDS UI static files were found in directory %v", s.HTMLDir)

	return nil
}

// prepareIndexHTML writes index.html file if index.tmpl exists
// index.tmpl is created at build time (ui.tar.gz). It's the copy of index.html file
// with the value version
// from a release: index.tmpl exists. This func copy it with the value sentryURL and baseURL rewritted.
// from source: index.tmpl does not exist. this func do nothing
func (s *Service) prepareIndexHTML() (bool, error) {
	indexTMPL := filepath.Join(s.HTMLDir, "index.tmpl")
	indexTMPLFile, err := os.Open(indexTMPL)
	if os.IsNotExist(err) {
		return false, nil
	}
	defer indexTMPLFile.Close()

	indexHTML := filepath.Join(s.HTMLDir, "index.html")
	indexHTMLFile, err := os.Create(indexHTML)
	if err != nil {
		return true, sdk.WrapError(err, "error while creating %s file", indexHTML)
	}
	defer indexHTMLFile.Close()
	_, err = io.Copy(indexHTMLFile, indexTMPLFile)
	return true, sdk.WrapError(err, "error while copy index.tmpl to index.html file")
}

func (s *Service) indexHTMLReplaceVar() error {
	indexHTML := filepath.Join(s.HTMLDir, "index.html")

	read, err := ioutil.ReadFile(indexHTML)
	if err != nil {
		return sdk.WrapError(err, "error while reading %s file", indexHTML)
	}

	regexBaseHref, err := regexp.Compile("<base href=\".*\">")
	if err != nil {
		return sdk.WrapError(err, "cannot parse base href regex")
	}
	indexContent := regexBaseHref.ReplaceAllString(string(read), "<base href=\""+s.Cfg.BaseURL+"\">")
	indexContent = strings.Replace(indexContent, "window.cds_sentry_url = '';", "window.cds_sentry_url = '"+s.Cfg.SentryURL+"';", -1)
	indexContent = strings.Replace(indexContent, "window.cds_version = '';", "window.cds_version='"+sdk.VERSION+"';", -1)
	return ioutil.WriteFile(indexHTML, []byte(indexContent), 0)
}

func (s *Service) askForGettingStaticFiles(ctx context.Context, version string) error {
	answerLatestRelease := fmt.Sprintf("Download files into %s from the latest GitHub CDS Release", s.Cfg.Staticdir)
	answerVersionRelease := fmt.Sprintf("Download files into %s from the GitHub CDS Release %s", s.Cfg.Staticdir, version)
	answerBuildFromSource := fmt.Sprintf("Build from source %s with node", filepath.Join("..", "ui"))
	useExistingBuildFromSource := fmt.Sprintf("Use existing %s", filepath.Join("..", "ui", "dist"))
	answerDoNothing := "Do nothing - exit now"
	opts := []string{}

	if strings.Contains(version, "snapshot") {
		opts = append(opts, answerBuildFromSource)
		if _, err := os.Stat(filepath.Join("..", "ui", "dist", "index.html")); err == nil {
			opts = append(opts, useExistingBuildFromSource)
		}
		opts = append(opts, answerLatestRelease)
	} else {
		opts = append(opts, answerVersionRelease)
	}

	opts = append(opts, answerDoNothing)

	ask := "What do you want to do?"

	selected := cli.AskChoice(ask, opts...)

	switch opts[selected] {
	case answerDoNothing:
		return nil
	case answerLatestRelease:
		return s.downloadStaticFilesFromGitHub(ctx, "latest")
	case answerVersionRelease:
		return s.downloadStaticFilesFromGitHub(ctx, version)
	case answerBuildFromSource:
		return s.buildFromSource(ctx)
	case useExistingBuildFromSource:
		s.HTMLDir = filepath.Join("..", "ui", "dist")
	}
	return nil
}

func (s *Service) buildFromSource(ctx context.Context) error {
	if _, err := os.Stat(filepath.Join("..", "ui")); os.IsNotExist(err) {
		return fmt.Errorf("You must have the directory ../ui with the ui source code")
	}

	if _, err := s.execCommand("npm install --no-audit"); err != nil {
		return err
	}
	if _, err := s.execCommand("node --max_old_space_size=6000 node_modules/@angular/cli/bin/ng build --prod"); err != nil {
		return err
	}
	s.HTMLDir = filepath.Join("..", "ui", "dist") // ../ui/dist
	return nil
}

func (s *Service) execCommand(command string) (string, error) {
	log.Info(context.Background(), "ui> running %s...", command)

	parts := strings.Fields(command)
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Dir = filepath.Join("..", "ui") // ../ui

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", sdk.WrapError(err, "could not get stderr")
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", sdk.WrapError(err, "could not get stdout")
	}
	var output string
	go func() {
		merged := io.MultiReader(stderr, stdout)
		scanner := bufio.NewScanner(merged)
		for scanner.Scan() {
			msg := scanner.Text()
			output = output + msg + "\n"
			fmt.Println(msg)
		}
	}()
	if err := cmd.Run(); err != nil {
		return "", sdk.WrapError(err, "could not run command")
	}
	return output, nil
}

func (s *Service) downloadStaticFilesFromGitHub(ctx context.Context, version string) error {
	if _, err := os.Stat(s.Cfg.Staticdir); os.IsNotExist(err) {
		log.Info(ctx, "ui> creating directory %s", s.Cfg.Staticdir)
		if err := os.Mkdir(s.Cfg.Staticdir, 0740); err != nil {
			return fmt.Errorf("Error while creating directory: %v", err)
		}
	}

	urlFiles := fmt.Sprintf("https://github.com/ovh/cds/releases/download/%s/ui.tar.gz", version)
	if version == "latest" {
		var err error
		urlFiles, err = s.Client.DownloadURLFromGithub("ui.tar.gz")
		if err != nil {
			return fmt.Errorf("Error while getting ui.tar.gz from Github err:%s", err)
		}
	}

	log.Info(ctx, "ui> Downloading from %s...", urlFiles)

	resp, err := http.Get(urlFiles)
	if err != nil {
		return fmt.Errorf("Error while getting ui.tar.gz from GitHub: %v", err)
	}
	defer resp.Body.Close()

	if err := sdk.CheckContentTypeBinary(resp); err != nil {
		return sdk.WrapError(err, "Error while checking Content-Type of %s", urlFiles)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("Error while getting ui.tar.gz from GitHub - HTTP code: %d", resp.StatusCode)
	}

	log.Info(ctx, "ui> Download successful, decompressing the archive file...")

	return sdk.UntarGz(afero.NewOsFs(), s.Cfg.Staticdir, resp.Body)
}
