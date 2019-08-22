package ui

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/log"
)

// New returns a new service
func New() *Service {
	s := new(Service)
	s.Router = &api.Router{
		Mux: mux.NewRouter(),
	}
	return s
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

	s.Client = cdsclient.NewService(s.Cfg.API.HTTP.URL, 60*time.Second, s.Cfg.API.HTTP.Insecure)
	s.API = s.Cfg.API.HTTP.URL
	s.Name = s.Cfg.Name
	s.HTTPURL = s.Cfg.URL
	s.Token = s.Cfg.API.Token
	s.Type = services.TypeUI
	s.MaxHeartbeatFailures = s.Cfg.API.MaxHeartbeatFailures
	s.ServiceName = "cds-ui"

	// HTMLDir must contains the ui dist directory.
	// ui.tar.gz contains the dist directory
	s.HTMLDir = s.Cfg.Staticdir + "/dist"

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
func (s *Service) Serve(c context.Context) error {
	log.Info("ui> Starting service %s %s...", s.Cfg.Name, sdk.VERSION)
	s.StartupTime = time.Now()

	if err := s.checkStaticFiles(); err != nil {
		return err
	}

	s.indexHTMLReplaceVar()

	//Init the http server
	s.initRouter(c)
	server := &http.Server{
		Addr:           fmt.Sprintf("%s:%d", s.Cfg.HTTP.Addr, s.Cfg.HTTP.Port),
		Handler:        s.Router.Mux,
		ReadTimeout:    10 * time.Minute,
		WriteTimeout:   10 * time.Minute,
		MaxHeaderBytes: 1 << 20,
	}

	// Start the http server
	log.Info("ui> Starting HTTP Server on port %d", s.Cfg.HTTP.Port)
	if err := server.ListenAndServe(); err != nil {
		log.Error("ui> Listen and serve failed: %s", err)
	}

	// Gracefully shutdown the http server
	<-c.Done()
	log.Info("ui> Shutdown HTTP Server")
	if err := server.Shutdown(c); err != nil {
		return fmt.Errorf("unable to shutdown server: %v", err)
	}

	return c.Err()
}

func (s *Service) checkStaticFiles() error {
	fs := http.Dir(s.HTMLDir)

	if _, err := fs.Open("index.html"); os.IsNotExist(err) {
		log.Warning("ui> CDS UI static files were not found in directory %v", s.HTMLDir)

		if err := s.askForGettingStaticFiles(sdk.VERSION); err != nil {
			return err
		}

		// reset the fs, HTMLDir could be updated by user answer
		fs = http.Dir(s.HTMLDir)
		// recheck file after user answer
		if _, err := fs.Open("index.html"); os.IsNotExist(err) {
			return fmt.Errorf("CDS UI static files were not found in directory %v", s.HTMLDir)
		}
	}
	log.Info("ui> CDS UI static files were found in directory %v", s.HTMLDir)

	return nil
}

func (s *Service) indexHTMLReplaceVar() error {
	indexHTML := s.HTMLDir + "/index.html"

	read, err := ioutil.ReadFile(indexHTML)
	if err != nil {
		return err
	}

	indexContent := strings.Replace(string(read), "<base href=\"/\">", "<base href=\""+s.Cfg.BaseURL+"\">", -1)
	indexContent = strings.Replace(string(read), "window.cds_sentry_url = '';", "window.cds_sentry_url = '"+s.Cfg.SentryURL+"';", -1)
	indexContent = strings.Replace(string(read), "window.cds_version = '';", "window.cds_version='"+sdk.VERSION+"';", -1)

	return ioutil.WriteFile(indexHTML, []byte(indexContent), 0)
}

func (s *Service) askForGettingStaticFiles(version string) error {
	answerLatestRelease := fmt.Sprintf("Download files into %s from the latest GitHub CDS Release", s.Cfg.Staticdir)
	answerVersionRelease := fmt.Sprintf("Download files into %s from the GitHub CDS Release %s", s.Cfg.Staticdir, version)
	answerBuildFromSource := "Build from source ../ui with node"
	useExistingBuildFromSource := "Use existing ../ui/dist/"
	answerDoNothing := "Do nothing - exit now"
	opts := []string{}

	if strings.Contains(version, "snapshot") {
		opts = append(opts, answerBuildFromSource)
		if _, err := os.Stat("../ui/dist/index.html"); err == nil {
			opts = append(opts, useExistingBuildFromSource)
		}
		opts = append(opts, answerLatestRelease)
	} else {
		opts = append(opts, answerVersionRelease)
	}

	opts = append(opts, answerDoNothing)

	ask := fmt.Sprintf("What do you want to do?")

	selected := cli.MultiChoice(ask, opts...)

	switch opts[selected] {
	case answerDoNothing:
		return nil
	case answerLatestRelease:
		return s.downloadStaticFilesFromGitHub("latest")
	case answerVersionRelease:
		return s.downloadStaticFilesFromGitHub(version)
	case answerBuildFromSource:
		return s.buildFromSource()
	case useExistingBuildFromSource:
		s.HTMLDir = "../ui/dist"
	}
	return nil
}

func (s *Service) buildFromSource() error {
	if _, err := os.Stat("../ui"); os.IsNotExist(err) {
		return fmt.Errorf("ui> You must have the directory ../ui with the ui source code")
	}

	if err := s.execCommand("npm install --no-audit"); err != nil {
		return err
	}
	if err := s.execCommand("node --max_old_space_size=6000 node_modules/@angular/cli/bin/ng build --prod"); err != nil {
		return err
	}
	s.HTMLDir = "../ui/dist"
	return nil
}

func (s *Service) execCommand(command string) error {
	log.Info("ui> running %s...", command)
	parts := strings.Fields(command)
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Dir = "../ui"

	stderr, _ := cmd.StderrPipe()
	cmd.Start()

	scanner := bufio.NewScanner(stderr)
	scanner.Split(bufio.ScanWords)
	for scanner.Scan() {
		m := scanner.Text()
		fmt.Print(m + " ")
	}
	fmt.Println()
	cmd.Wait()
	return nil
}

func (s *Service) downloadStaticFilesFromGitHub(version string) error {
	if _, err := os.Stat(s.Cfg.Staticdir); os.IsNotExist(err) {
		log.Info("ui> creating directory %s", s.Cfg.Staticdir)
		if err := os.Mkdir(s.Cfg.Staticdir, 0740); err != nil {
			return fmt.Errorf("Error while creating directory: %v", err)
		}
	}

	urlFiles := fmt.Sprintf("https://github.com/ovh/cds/releases/download/%s/ui.tar.gz", version)
	if version == "latest" {
		var err error
		urlFiles, err = s.Client.DownloadURLFromGithub("ui.tar.gz")
		if err != nil {
			return fmt.Errorf("ui> Error while getting ui.tar.gz from Github err:%s", err)
		}
	}

	log.Info("ui> Downloading from %s...", urlFiles)

	resp, err := http.Get(urlFiles)
	if err != nil {
		return fmt.Errorf("Error while getting ui.tar.gz from GitHub: %v", err)
	}
	defer resp.Body.Close()

	if err := sdk.CheckContentTypeBinary(resp); err != nil {
		return fmt.Errorf(err.Error())
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("Error while getting ui.tar.gz from GitHub - HTTP code: %d", resp.StatusCode)
	}

	log.Info("ui> Download in success, we are decompressing the archive now...")

	return untarGZ(s.Cfg.Staticdir, resp.Body)
}

// untarGZ takes a destination path and a reader; a tar reader loops over the tarfile
// creating the file structure at 'dst' along the way, and writing any files
func untarGZ(dst string, r io.Reader) error {

	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()

		switch {

		// if no more files are found return
		case err == io.EOF:
			return nil

		// return any other error
		case err != nil:
			return err

		// if the header is nil, just skip it (not sure how this happens)
		case header == nil:
			continue
		}

		// the target location where the dir/file should be created
		target := filepath.Join(dst, header.Name)

		// the following switch could also be done using fi.Mode(), not sure if there
		// a benefit of using one vs. the other.
		// fi := header.FileInfo()

		// check the file type
		switch header.Typeflag {

		// if its a dir and it doesn't exist create it
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
			}

		// if it's a file create it
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			// copy over contents
			if _, err := io.Copy(f, tr); err != nil {
				return err
			}

			// manually close here after each file operation; defering would cause each file close
			// to wait until all operations have completed.
			f.Close()
		}
	}
}
