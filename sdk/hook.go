package sdk

import (
	"crypto/rand"
	"encoding/base64"
)

// These are constants about hooks
const (
	WebHookModelName              = "WebHook"
	RepositoryWebHookModelName    = "RepositoryWebHook"
	GerritHookModelName           = "GerritHook"
	SchedulerModelName            = "Scheduler"
	GitPollerModelName            = "Git Repository Poller"
	KafkaHookModelName            = "Kafka hook"
	RabbitMQHookModelName         = "RabbitMQ hook"
	WorkflowModelName             = "Workflow"
	HookConfigProject             = "project"
	HookConfigWorkflow            = "workflow"
	HookConfigTargetProject       = "target_project"
	HookConfigTargetWorkflow      = "target_workflow"
	HookConfigTargetHook          = "target_hook"
	HookConfigWorkflowID          = "workflow_id"
	HookConfigWebHookID           = "webHookID"
	HookConfigVCSType             = "vcsType"
	HookConfigVCSServer           = "vcsServer"
	HookConfigEventFilter         = "eventFilter"
	HookConfigRepoFullName        = "repoFullName"
	HookConfigModelType           = "model_type"
	HookConfigModelName           = "model_name"
	HookConfigIcon                = "hookIcon"
	WebHookModelConfigMethod      = "method"
	RepositoryWebHookModelMethod  = "method"
	SchedulerModelCron            = "cron"
	SchedulerModelTimezone        = "timezone"
	Payload                       = "payload"
	HookModelIntegration          = "integration"
	KafkaHookModelConsumerGroup   = "consumer group"
	KafkaHookModelTopic           = "topic"
	RabbitMQHookModelQueue        = "queue"
	RabbitMQHookModelBindingKey   = "binding_key"
	RabbitMQHookModelExchangeType = "exchange_type"
	RabbitMQHookModelExchangeName = "exchange_name"
	RabbitMQHookModelConsumerTag  = "consumer_tag"
	SchedulerUsername             = "cds.scheduler"
	SchedulerFullname             = "CDS Scheduler"
)

// Here are the default hooks
var (
	BuiltinHookModels = []*WorkflowHookModel{
		&WebHookModel,
		&RepositoryWebHookModel,
		&GitPollerModel,
		&SchedulerModel,
		&KafkaHookModel,
		&RabbitMQHookModel,
		&WorkflowModel,
		&GerritHookModel,
	}

	BuiltinOutgoingHookModels = []*WorkflowHookModel{
		&OutgoingWebHookModel,
		&OutgoingWorkflowModel,
	}

	KafkaHookModel = WorkflowHookModel{
		Author:     "CDS",
		Type:       WorkflowHookModelBuiltin,
		Identifier: "github.com/ovh/cds/hook/builtin/kafka",
		Name:       KafkaHookModelName,
		Icon:       "Linkify",
		DefaultConfig: WorkflowNodeHookConfig{
			HookModelIntegration: {
				Value:        "",
				Configurable: true,
				Type:         HookConfigTypeIntegration,
			},
			KafkaHookModelTopic: {
				Value:        "",
				Configurable: true,
				Type:         HookConfigTypeString,
			},
		},
	}

	RabbitMQHookModel = WorkflowHookModel{
		Author:     "CDS",
		Type:       WorkflowHookModelBuiltin,
		Identifier: "github.com/ovh/cds/hook/builtin/rabbitmq",
		Name:       RabbitMQHookModelName,
		Icon:       "Linkify",
		DefaultConfig: WorkflowNodeHookConfig{
			HookModelIntegration: {
				Value:        "",
				Configurable: true,
				Type:         HookConfigTypeIntegration,
			},
			RabbitMQHookModelQueue: {
				Value:        "",
				Configurable: true,
				Type:         HookConfigTypeString,
			},
			RabbitMQHookModelExchangeType: {
				Value:        "",
				Configurable: true,
				Type:         HookConfigTypeString,
			},
			RabbitMQHookModelExchangeName: {
				Value:        "",
				Configurable: true,
				Type:         HookConfigTypeString,
			},
			RabbitMQHookModelBindingKey: {
				Value:        "",
				Configurable: true,
				Type:         HookConfigTypeString,
			},
			RabbitMQHookModelConsumerTag: {
				Value:        "",
				Configurable: true,
				Type:         HookConfigTypeString,
			},
		},
	}

	WebHookModel = WorkflowHookModel{
		Author:     "CDS",
		Type:       WorkflowHookModelBuiltin,
		Identifier: "github.com/ovh/cds/hook/builtin/webhook",
		Name:       WebHookModelName,
		Icon:       "Linkify",
		DefaultConfig: WorkflowNodeHookConfig{
			WebHookModelConfigMethod: {
				Value:        "POST",
				Configurable: true,
				Type:         HookConfigTypeString,
			},
		},
	}

	RepositoryWebHookModel = WorkflowHookModel{
		Author:     "CDS",
		Type:       WorkflowHookModelBuiltin,
		Identifier: "github.com/ovh/cds/hook/builtin/repositorywebhook",
		Name:       RepositoryWebHookModelName,
		Icon:       "Linkify",
		DefaultConfig: WorkflowNodeHookConfig{
			RepositoryWebHookModelMethod: {
				Value:        "POST",
				Configurable: false,
				Type:         HookConfigTypeString,
			},
			HookConfigEventFilter: {
				Value:        "",
				Configurable: true,
				Type:         HookConfigTypeMultiChoice,
			},
		},
	}

	GitPollerModel = WorkflowHookModel{
		Author:     "CDS",
		Type:       WorkflowHookModelBuiltin,
		Identifier: "github.com/ovh/cds/hook/builtin/poller",
		Name:       GitPollerModelName,
		Icon:       "git square",
		DefaultConfig: WorkflowNodeHookConfig{
			"payload": {
				Value:        "{}",
				Configurable: true,
				Type:         HookConfigTypeString,
			},
		},
	}

	SchedulerModel = WorkflowHookModel{
		Author:     "CDS",
		Type:       WorkflowHookModelBuiltin,
		Identifier: "github.com/ovh/cds/hook/builtin/scheduler",
		Name:       SchedulerModelName,
		Icon:       "fa-clock-o",
		DefaultConfig: WorkflowNodeHookConfig{
			SchedulerModelCron: {
				Value:        "0 * * * *",
				Configurable: true,
				Type:         HookConfigTypeString,
			},
			SchedulerModelTimezone: {
				Value:        "UTC",
				Configurable: true,
				Type:         HookConfigTypeString,
			},
			Payload: {
				Value:        "{}",
				Configurable: true,
				Type:         HookConfigTypeString,
			},
		},
	}

	GerritHookModel = WorkflowHookModel{
		Author:     "CDS",
		Type:       WorkflowHookModelBuiltin,
		Identifier: "github.com/ovh/cds/hook/builtin/gerrit",
		Name:       GerritHookModelName,
		Icon:       "git",
		DefaultConfig: WorkflowNodeHookConfig{
			HookConfigEventFilter: {
				Value:        "",
				Configurable: true,
				Type:         HookConfigTypeMultiChoice,
			},
		},
	}

	WorkflowModel = WorkflowHookModel{
		Author:     "CDS",
		Type:       WorkflowHookModelBuiltin,
		Identifier: "github.com/ovh/cds/hook/builtin/workflowhook",
		Name:       WorkflowModelName,
		Icon:       "sitemap",
	}

	OutgoingWebHookModel = WorkflowHookModel{
		Author:     "CDS",
		Type:       WorkflowHookModelBuiltin,
		Identifier: "github.com/ovh/cds/hook/builtin/webhook",
		Name:       WebHookModelName,
		Icon:       "Linkify",
		DefaultConfig: WorkflowNodeHookConfig{
			WebHookModelConfigMethod: {
				Value:        "POST",
				Configurable: true,
				Type:         HookConfigTypeString,
			},
			"URL": {
				Configurable: true,
				Type:         HookConfigTypeString,
			},
			Payload: {
				Value:        "{}",
				Configurable: true,
				Type:         HookConfigTypeString,
			},
		},
	}

	OutgoingWorkflowModel = WorkflowHookModel{
		Author:     "CDS",
		Type:       WorkflowHookModelBuiltin,
		Identifier: "github.com/ovh/cds/hook/builtin/workflowhook",
		Name:       WorkflowModelName,
		Icon:       "sitemap",
		DefaultConfig: WorkflowNodeHookConfig{
			HookConfigTargetProject: {
				Configurable: true,
				Type:         HookConfigTypeProject,
			},
			HookConfigTargetWorkflow: {
				Configurable: true,
				Type:         HookConfigTypeWorkflow,
			},
			HookConfigTargetHook: {
				Configurable: true,
				Type:         HookConfigTypeHook,
			},
			Payload: {
				Value:        "{}",
				Configurable: true,
				Type:         HookConfigTypeString,
			},
		},
	}
)

// GetDefaultHookModel return the workflow hook model by its name
func GetDefaultHookModel(modelName string) WorkflowHookModel {
	switch modelName {
	case SchedulerModelName:
		return SchedulerModel
	case RepositoryWebHookModelName:
		return RepositoryWebHookModel
	case WebHookModelName:
		return WebHookModel
	case GitPollerModelName:
		return GitPollerModel
	case WorkflowModelName:
		return WorkflowModel
	}

	return WebHookModel
}

func GenerateHookSecret() (string, error) {
	b := make([]byte, 128)
	if _, err := rand.Read(b); err != nil {
		return "", WrapError(err, "unable to generate hook secret")
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
