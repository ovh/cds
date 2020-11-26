package doc

import (
	"strings"

	"github.com/pkg/errors"

	"github.com/ovh/cds/sdk/slug"
)

// CleanAndCheckURL returns cleaned URL but also an error if given url contains variables that can't be cleaned.
// Ex: /myresource/{permMyresourceID} should be transformed to /myresource/<myresource-id>, an error will be returned if permMyResourceID not converted.
func CleanAndCheckURL(url string) (string, error) {
	url = CleanURL(url)
	urlSplitted := strings.Split(url, "/")
	for i := range urlSplitted {
		u := strings.TrimSuffix(strings.TrimPrefix(urlSplitted[i], "<"), ">")
		if u == urlSplitted[i] {
			continue
		}
		if !slug.Valid(u) {
			return url, errors.Errorf("given url %s contains variable %s that was not cleaned", url, u)
		}
	}
	return url, nil
}

// CleanURL given a URL with declared variable inside, returns the same URL with harmonized variable names.
// Ex: permProjectKey -> projectKey
func CleanURL(url string) string {
	url = strings.Replace(url, "\"", "", -1)
	urlSplitted := strings.Split(url, "/")
	for i := range urlSplitted {
		u := strings.TrimSuffix(strings.TrimPrefix(urlSplitted[i], "{"), "}")
		if u == urlSplitted[i] {
			continue
		}

		switch u {
		case "consumerType":
			u = "consumer-type"
		case "key", "permProjectKey":
			u = "project-key"
		case "permWorkflowName", "workflowName":
			u = "workflow-name"
		case "workflowID":
			u = "workflow-id"
		case "applicationName":
			u = "application-name"
		case "permGroupName", "groupName":
			u = "group-name"
		case "permUsernamePublic", "permUsername":
			u = "username"
		case "permActionName", "permActionBuiltinName":
			u = "action-name"
		case "permJobID", "jobID":
			u = "job-id"
		case "pipelineKey":
			u = "pipeline-key"
		case "environmentName":
			u = "environment-name"
		case "nodeRunID":
			u = "node-run-id"
		case "runJobID":
			u = "run-job-id"
		case "stepOrder":
			u = "step-order"
		case "nodeID":
			u = "node-id"
		case "permModelName":
			u = "model-name"
		case "permTemplateSlug", "templateSlug":
			u = "template-slug"
		case "instanceID":
			u = "instance-id"
		case "bulkID":
			u = "bulk-id"
		case "permConsumerID":
			u = "consumer-id"
		case "permSessionID":
			u = "session-id"
		case "integrationID":
			u = "integration-id"
		case "integrationName":
			u = "integration-name"
		case "auditID":
			u = "audit-id"
		case "stageID":
			u = "stage-id"
		case "labelID":
			u = "label-id"
		case "nodeName":
			u = "node-name"
		case "artifactId":
			u = "artifact-id"
		case "hookRunID":
			u = "hook-run-id"
		case "vcsServer":
			u = "vcs-server"
		case "metricName":
			u = "metric-name"
		case "cloneName":
			u = "clone-name"
		case "serviceName":
			u = "service-name"
		case "sessionID":
			u = "session-id"
		}

		urlSplitted[i] = "<" + u + ">"
	}
	return strings.Join(urlSplitted, "/")
}
