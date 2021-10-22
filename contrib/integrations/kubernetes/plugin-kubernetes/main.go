package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/golang/protobuf/ptypes/empty"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/integrationplugin"
)

/*
This plugin have to be used as a deployment integration plugin

Kubernetes deployment plugin must configured as following:
	name: plugin-kubernetes-deployment
	type: integration
	author: "Benjamin Coenen"
	description: "Kubernetes Deployment Plugin"

$ cdsctl admin plugins import plugin-kubernetes-deployment.yml

Build the present binaries and import in CDS:
	os: linux
	arch: amd64
	cmd: <path-to-binary-file>

$ cdsctl admin plugins binary-add plugin-kubernetes-deployment plugin-kubernetes-deployment-bin.yml <path-to-binary-file>
*/
const (
	kubectlLink = "https://storage.googleapis.com/kubernetes-release/release/v1.13.0/bin/"
)

type kubernetesDeploymentPlugin struct {
	integrationplugin.Common
}

func (k8sPlugin *kubernetesDeploymentPlugin) Manifest(ctx context.Context, _ *empty.Empty) (*integrationplugin.IntegrationPluginManifest, error) {
	return &integrationplugin.IntegrationPluginManifest{
		Name:        "Kubernetes Deployment Plugin",
		Author:      "Benjamin Coenen",
		Description: "Kubernetes Deployment Plugin",
		Version:     sdk.VERSION,
	}, nil
}

// getStartingConfig implements ConfigAccess
func getStartingConfig(token, timeout string) clientcmd.KubeconfigGetter {
	return func() (*clientcmdapi.Config, error) {
		defaultClientConfigRules := clientcmd.NewDefaultClientConfigLoadingRules()
		overrideCfg := clientcmd.ConfigOverrides{
			AuthInfo: clientcmdapi.AuthInfo{
				Token: token,
			},
			Timeout: timeout + "s",
		}

		clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(defaultClientConfigRules, &overrideCfg)
		rawConfig, err := clientConfig.RawConfig()
		if os.IsNotExist(err) {
			return clientcmdapi.NewConfig(), nil
		}
		if err != nil {
			return nil, err
		}

		return &rawConfig, nil
	}
}

func (k8sPlugin *kubernetesDeploymentPlugin) Run(ctx context.Context, q *integrationplugin.RunQuery) (*integrationplugin.RunResult, error) {
	k8sAPIURL := q.GetOptions()["cds.integration.deployment.api_url"]
	k8sToken := q.GetOptions()["cds.integration.deployment.token"]
	if k8sToken == "" {
		k8sToken = q.GetOptions()["cds.integration.token"]
	}
	k8sCaCertificate := q.GetOptions()["cds.integration.deployment.ca_certificate"]
	deploymentFilepath := q.GetOptions()["cds.integration.deployment.deployment_files"]
	helmChart := q.GetOptions()["cds.integration.deployment.helm_chart"]

	if k8sToken == "" {
		return fail("Kubernetes token should not be empty")
	}

	certb64 := base64.StdEncoding.EncodeToString([]byte(k8sCaCertificate))
	kubecfg := fmt.Sprintf(`apiVersion: v1
kind: Config
users:
- name: cds
  user:
    token: %s
clusters:
- cluster:
    certificate-authority-data: %s
    server: %s
  name: self-hosted-cluster
contexts:
- context:
    cluster: self-hosted-cluster
    user: cds
  name: default-context
current-context: default-context`, k8sToken, certb64, k8sAPIURL)

	if _, err := os.Stat(".kube"); !os.IsNotExist(err) {
		return fail(".kube directory already exists, you have to delete it before using this action")
	}

	if err := os.Mkdir(".kube", 0755); err != nil {
		return fail("Cannot create directory .kube : %v", err)
	}
	defer func() {
		if err := os.RemoveAll(".kube"); err != nil {
			fmt.Printf("Cannot delete .kube directory : %v\n", err)
		}
	}()

	if err := ioutil.WriteFile(".kube/config", []byte(kubecfg), 0755); err != nil {
		return fail("Cannot write kubeconfig : %v", err)
	}

	switch {
	case helmChart != "":
		if err := executeHelm(q); err != nil {
			return fail(err.Error())
		}
	case deploymentFilepath != "":
		if err := executeK8s(q); err != nil {
			return fail(err.Error())
		}
	default:
		return fail("Must have deployment_files or helm_chart not empty")
	}

	return &integrationplugin.RunResult{
		Status: sdk.StatusSuccess,
	}, nil
}

func main() {
	e := kubernetesDeploymentPlugin{}
	if err := integrationplugin.Start(context.Background(), &e); err != nil {
		panic(err)
	}
}

func fail(format string, args ...interface{}) (*integrationplugin.RunResult, error) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(msg)
	return &integrationplugin.RunResult{
		Details: msg,
		Status:  sdk.StatusFail,
	}, nil
}

func executeK8s(q *integrationplugin.RunQuery) error {
	k8sAPIURL := q.GetOptions()["cds.integration.deployment.api_url"]
	k8sToken := q.GetOptions()["cds.integration.deployment.token"]
	if k8sToken == "" {
		k8sToken = q.GetOptions()["cds.integration.token"]
	}
	k8sCaCertificate := q.GetOptions()["cds.integration.deployment.ca_certificate"]
	namespace := q.GetOptions()["cds.integration.deployment.namespace"]
	deploymentFilepath := q.GetOptions()["cds.integration.deployment.deployment_files"]
	timeoutStr := q.GetOptions()["cds.integration.deployment.timeout"]
	project := q.GetOptions()["cds.project"]
	workflow := q.GetOptions()["cds.workflow"]
	if namespace == "" {
		namespace = "default"
	}

	kubectlFound := false
	if _, err := exec.LookPath("kubectl"); err == nil {
		kubectlFound = true
	}

	binaryName := "kubectl"
	if !kubectlFound {
		fmt.Println("Download kubectl in progress...")
		netClient := &http.Client{
			Timeout: time.Second * 600,
		}
		response, err := netClient.Get(kubectlLink + sdk.GOOS + "/" + sdk.GOARCH + "/kubectl")
		if err != nil {
			return fmt.Errorf("Cannot download kubectl : %v", err)
		}

		if response.StatusCode > 400 {
			return fmt.Errorf("Cannot download kubectl binary (status code %d)", response.StatusCode)
		}
		defer response.Body.Close()

		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return fmt.Errorf("Cannot read body http response: %v", err)
		}
		fmt.Println("Download kubectl done...")

		binaryName = project + "-" + workflow + "-kubectl"
		if err := ioutil.WriteFile(binaryName, body, 0755); err != nil {
			return fmt.Errorf("Cannot write file %s for kubectl : %v", binaryName, err)
		}
		defer func(binName string) {
			if err := os.Remove(binName); err != nil {
				fmt.Printf("Cannot delete binary file : %v\n", err)
			}
		}(binaryName)
		binaryName = "./" + binaryName
	}

	configK8s, err := clientcmd.BuildConfigFromKubeconfigGetter(k8sAPIURL, getStartingConfig(k8sToken, timeoutStr))
	if err != nil {
		return fmt.Errorf("Cannot build kubernetes config from config getter : %v", err)
	}
	configK8s.TLSClientConfig = rest.TLSClientConfig{
		CAData: []byte(k8sCaCertificate),
	}

	// creates the clientset
	clientset, errCl := kubernetes.NewForConfig(configK8s)
	if errCl != nil {
		return fmt.Errorf("Cannot create new config for kubernetes: %v", errCl)
	}

	if namespace != "" && namespace != apiv1.NamespaceDefault {
		if _, err := clientset.CoreV1().Namespaces().Get(context.Background(), namespace, metav1.GetOptions{}); err != nil {
			ns := apiv1.Namespace{}
			ns.SetName(namespace)
			if _, errC := clientset.CoreV1().Namespaces().Create(context.Background(), &ns, metav1.CreateOptions{}); errC != nil {
				return fmt.Errorf("Cannot create namespace %s in kubernetes: %v", namespace, errC)
			}
		}
	}

	// All files matching filePath
	filesPath, err := filepath.Glob(deploymentFilepath)
	if err != nil {
		return fmt.Errorf("Could not find paths : %v", err)
	}

	if len(filesPath) == 0 {
		return fmt.Errorf("Pattern '%s' matched no file", deploymentFilepath)
	}

	cmdSetContext := exec.Command(binaryName, "config", "set-context", "default-context")
	cmdSetContext.Stderr = os.Stderr
	cmdSetContext.Stdout = os.Stdout
	if err := cmdSetContext.Run(); err != nil {
		return fmt.Errorf("Cannot execute kubectl config set-context : %v", err)
	}

	args := append([]string{"apply", "--timeout=" + timeoutStr + "s", "--wait=true", "-f"}, filesPath...)
	cmd := exec.Command(binaryName, args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Cannot execute kubectl apply : %v", err)
	}

	return nil
}

func executeHelm(q *integrationplugin.RunQuery) error {
	project := q.GetOptions()["cds.project"]
	workflow := q.GetOptions()["cds.workflow"]
	helmVersion := q.GetOptions()["cds.integration.deployment.helm_version"]

	if helmVersion == "" {
		helmVersion = "2.12.2"
	}

	version, err := semver.Parse(helmVersion)
	if err != nil {
		return fmt.Errorf("Invalid Helm Version %s - err: %v", helmVersion, err)
	}
	supportedVersion := ">=2.0.0 <4.0.0"
	expectedRange, err := semver.ParseRange(supportedVersion)
	if err != nil {
		return fmt.Errorf("Fail to parse semver range : %v", err)
	}

	if !expectedRange(version) {
		return fmt.Errorf("Unsupported helm version, should be : %s", supportedVersion)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cannot get current working directory : %v", err)
	}

	binaryName := "helm"
	kubeCfg := "KUBECONFIG=" + path.Join(cwd, ".kube/config")

	helmFound := false
	if _, err := exec.LookPath("helm"); err == nil {
		helmFound = true
	}

	if helmFound {
		out, err := exec.Command(binaryName, "version", "--client", "--short").Output()
		if err != nil {
			return fmt.Errorf("Cannot check helm version : %v", err)
		}
		installedHelm := strings.TrimPrefix(string(out), "Client: ")
		installedVersion, err := semver.ParseTolerant(installedHelm)
		if err != nil {
			return fmt.Errorf("Invalid installed Helm Version %s - err: %v", installedHelm, err)
		}

		if !version.Equals(installedVersion) {
			fmt.Println("Helm in path is not at correct version, need installation")
			fmt.Printf("Path version : %s\n", installedVersion.String())
			fmt.Printf("Target version : %s\n", version.String())
			helmFound = false
		}
	}

	if !helmFound {
		fmt.Printf("Download helm %s in progress...\n", version.String())
		netClient := &http.Client{
			Timeout: time.Second * 600,
		}
		response, err := netClient.Get(fmt.Sprintf("https://get.helm.sh/helm-v%s-%s-%s.tar.gz", version.String(), sdk.GOOS, sdk.GOARCH))
		if err != nil {
			return fmt.Errorf("Cannot download helm : %v", err)
		}

		if response.StatusCode > 400 {
			return fmt.Errorf("Cannot download helm binary (status code %d)", response.StatusCode)
		}
		defer response.Body.Close()

		binaryName = project + "-" + workflow + "-helm"
		if err := os.Mkdir(binaryName, 0755); err != nil {
			return fmt.Errorf("Cannot write directory for helm : %v", err)
		}
		if err := writeHelmBinary(binaryName, response.Body); err != nil {
			return fmt.Errorf("Cannot write helm binary : %v", err)
		}
		fmt.Println("Download helm done...")
		defer func(binName string) {
			if err := os.RemoveAll(binName); err != nil {
				fmt.Printf("Cannot delete binary file : %v\n", err)
			}
		}(binaryName)
		binaryName = path.Join(".", binaryName, sdk.GOOS+"-"+sdk.GOARCH, "helm")
	}

	switch version.Major {
	case 2:
		return executeHelmV2(binaryName, kubeCfg, q)
	case 3:
		return executeHelmV3(binaryName, kubeCfg, q)
	}

	return fmt.Errorf("Unsupported helm version")
}

func executeHelmV2(binaryName, kubeCfg string, q *integrationplugin.RunQuery) error {
	releaseName := q.GetOptions()["cds.integration.deployment.release_name"]
	namespace := q.GetOptions()["cds.integration.deployment.namespace"]
	helmChart := q.GetOptions()["cds.integration.deployment.helm_chart"]
	helmValues := q.GetOptions()["cds.integration.deployment.helm_values"]
	timeoutStr := q.GetOptions()["cds.integration.deployment.timeout"]
	application := q.GetOptions()["cds.application"]

	if namespace == "" {
		namespace = "default"
	}
	if releaseName != "" {
		application = releaseName
	}

	if d, err := time.ParseDuration(timeoutStr); err == nil {
		timeoutStr = strconv.Itoa(int(d.Seconds()))
		fmt.Println("timeout is a duration, converting timeout in seconds to " + timeoutStr)
	}

	cmdInit := exec.Command(binaryName, "init", "--client-only")
	cmdInit.Env = os.Environ()
	cmdInit.Stderr = os.Stderr
	cmdInit.Stdout = os.Stdout
	if err := cmdInit.Run(); err != nil {
		return fmt.Errorf("Cannot execute helm init : %v", err)
	}

	if _, err := os.Stat(helmChart); err == nil {
		fmt.Println("Helm dependency update")
		cmdDependency := exec.Command(binaryName, "dependency", "update", helmChart)
		cmdDependency.Env = os.Environ()
		cmdDependency.Env = append(cmdDependency.Env, kubeCfg)
		cmdDependency.Stderr = os.Stderr
		cmdDependency.Stdout = os.Stdout
		if errCmd := cmdDependency.Run(); errCmd != nil {
			return fmt.Errorf("Cannot execute helm dependency update : %v", errCmd)
		}
	}

	cmdGet := exec.Command(binaryName, "get", application)
	cmdGet.Env = os.Environ()
	cmdGet.Env = append(cmdGet.Env, kubeCfg)
	errCmd := cmdGet.Run()

	var args []string
	if errCmd != nil { // Install
		fmt.Printf("Install helm release '%s' with chart '%s'...\n", application, helmChart)
		args = []string{"install", "--name=" + application, "--debug", "--timeout=" + timeoutStr, "--wait=true", "--namespace=" + namespace}
		if helmValues != "" {
			args = append(args, "-f", helmValues)
		}

		helmChartArgs := strings.Split(helmChart, " ")
		if len(helmChartArgs) > 1 {
			args = append(args, "--repo="+helmChartArgs[0], helmChartArgs[1])
		} else {
			args = append(args, helmChart)
		}
	} else {
		fmt.Printf("Update helm release '%s' with chart '%s'...\n", application, helmChart)
		args = []string{"upgrade", "--timeout=" + timeoutStr, "--wait=true", "--namespace=" + namespace}
		if helmValues != "" {
			args = append(args, "-f", helmValues)
		}

		helmChartArgs := strings.Split(helmChart, " ")
		if len(helmChartArgs) > 1 {
			args = append(args, "--repo="+helmChartArgs[0], application, helmChartArgs[1])
		} else {
			args = append(args, application, helmChart)
		}
	}

	fmt.Printf("Execute: helm %s\n", strings.Join(args, " "))
	cmd := exec.Command(binaryName, args...)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, kubeCfg)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Cannot execute helm install/update : %v", err)
	}

	return nil
}

func executeHelmV3(binaryName, kubeCfg string, q *integrationplugin.RunQuery) error {
	releaseName := q.GetOptions()["cds.integration.deployment.release_name"]
	namespace := q.GetOptions()["cds.integration.deployment.namespace"]
	helmChart := q.GetOptions()["cds.integration.deployment.helm_chart"]
	helmValues := q.GetOptions()["cds.integration.deployment.helm_values"]
	timeoutStr := q.GetOptions()["cds.integration.deployment.timeout"]
	application := q.GetOptions()["cds.application"]

	if namespace == "" {
		namespace = "default"
	}
	if releaseName != "" {
		application = releaseName
	}
	if _, err := time.ParseDuration(timeoutStr); err != nil {
		timeoutStr = timeoutStr + "s"
		fmt.Println("timeout is not a duration, setting timeout to " + timeoutStr)
	}

	cmdRepoAdd := exec.Command(binaryName, "repo", "add", "stable", "https://charts.helm.sh/stable")
	cmdRepoAdd.Env = os.Environ()
	cmdRepoAdd.Stderr = os.Stderr
	cmdRepoAdd.Stdout = os.Stdout
	if err := cmdRepoAdd.Run(); err != nil {
		return fmt.Errorf("Cannot execute helm repo add stable : %v", err)
	}

	if _, err := os.Stat(helmChart); err == nil {
		fmt.Println("Helm dependency update")
		cmdDependency := exec.Command(binaryName, "dependency", "update", helmChart)
		cmdDependency.Env = os.Environ()
		cmdDependency.Env = append(cmdDependency.Env, kubeCfg)
		cmdDependency.Stderr = os.Stderr
		cmdDependency.Stdout = os.Stdout
		if errCmd := cmdDependency.Run(); errCmd != nil {
			return fmt.Errorf("Cannot execute helm dependency update : %v", errCmd)
		}
	}

	cmdGet := exec.Command(binaryName, "get", "all", "--namespace="+namespace, application)
	cmdGet.Env = os.Environ()
	cmdGet.Env = append(cmdGet.Env, kubeCfg)
	errCmd := cmdGet.Run()

	var args []string
	if errCmd != nil { // Install
		fmt.Printf("Install helm release '%s' with chart '%s'...\n", application, helmChart)
		args = []string{"install", "--debug", "--timeout=" + timeoutStr, "--wait=true", "--namespace=" + namespace}
		if helmValues != "" {
			args = append(args, "-f", helmValues)
		}

		helmChartArgs := strings.Split(helmChart, " ")
		if len(helmChartArgs) > 1 {
			args = append(args, "--repo="+helmChartArgs[0], helmChartArgs[1])
			args = append(args, application, helmChartArgs[1])
		} else {
			args = append(args, application, helmChart)
		}
	} else {
		fmt.Printf("Update helm release '%s' with chart '%s'...\n", application, helmChart)
		args = []string{"upgrade", "--timeout=" + timeoutStr, "--wait=true", "--namespace=" + namespace}
		if helmValues != "" {
			args = append(args, "-f", helmValues)
		}

		helmChartArgs := strings.Split(helmChart, " ")
		if len(helmChartArgs) > 1 {
			args = append(args, "--repo="+helmChartArgs[0], application, helmChartArgs[1])
		} else {
			args = append(args, application, helmChart)
		}
	}

	fmt.Printf("Execute: helm %s\n", strings.Join(args, " "))
	cmd := exec.Command(binaryName, args...)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, kubeCfg)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Cannot execute helm install/update : %v", err)
	}

	return nil
}

func writeHelmBinary(pathname string, gzipStream io.Reader) error {
	uncompressedStream, err := gzip.NewReader(gzipStream)
	if err != nil {
		log.Fatal("writeHelmBinary: NewReader failed")
	}

	tarReader := tar.NewReader(uncompressedStream)

	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			return fmt.Errorf("writeHelmBinary: Next() failed: %s", err.Error())
		}

		path := filepath.Join(pathname, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.Mkdir(path, 0755); err != nil {
				return fmt.Errorf("writeHelmBinary: mkdir %s failed: %v", path, err)
			}
		case tar.TypeReg:
			outFile, err := os.Create(path)
			if err != nil {
				return fmt.Errorf("writeHelmBinary: file %s creation failed: %v", path, err)
			}
			if err := outFile.Chmod(0755); err != nil {
				return fmt.Errorf("cannot change permission of file : %v", err)
			}
			defer outFile.Close()
			if _, err := io.Copy(outFile, tarReader); err != nil {
				return fmt.Errorf("writeHelmBinary: copy() failed: %v", err)
			}
		default:
			return fmt.Errorf(
				"writeHelmBinary: uknown type: %v in %s",
				header.Typeflag,
				path)
		}
	}

	return nil
}
