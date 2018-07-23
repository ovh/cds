package sdk

// These are constants about hooks
const (
	WebHookModelName              = "WebHook"
	RepositoryWebHookModelName    = "RepositoryWebHook"
	SchedulerModelName            = "Scheduler"
	GitPollerModelName            = "Git Repository Poller"
	KafkaHookModelName            = "Kafka hook"
	RabbitMQHookModelName         = "RabbitMQ hook"
	HookConfigProject             = "project"
	HookConfigWorkflow            = "workflow"
	HookConfigWorkflowID          = "workflow_id"
	WebHookModelConfigMethod      = "method"
	RepositoryWebHookModelMethod  = "method"
	SchedulerModelCron            = "cron"
	SchedulerModelTimezone        = "timezone"
	SchedulerModelPayload         = "payload"
	HookModelPlatform             = "platform"
	KafkaHookModelConsumerGroup   = "consumer group"
	KafkaHookModelTopic           = "topic"
	RabbitMQHookModelQueue        = "queue"
	RabbitMQHookModelBindingKey   = "binding_key"
	RabbitMQHookModelExchangeType = "exchange_type"
	RabbitMQHookModelExchangeName = "exchange_name"
	RabbitMQHookModelConsumerTag  = "consumer_tag"
)

// KafkaHookModel is the builtin hooks
var (
	KafkaHookModel = WorkflowHookModel{
		Author:     "CDS",
		Type:       WorkflowHookModelBuiltin,
		Identifier: "github.com/ovh/cds/hook/builtin/kafka",
		Name:       KafkaHookModelName,
		Icon:       "Linkify",
		DefaultConfig: WorkflowNodeHookConfig{
			HookModelPlatform: {
				Value:        "",
				Configurable: true,
				Type:         HookConfigTypePlatform,
			},
			KafkaHookModelConsumerGroup: {
				Value:        "",
				Configurable: false,
				Type:         HookConfigTypeString,
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
			HookModelPlatform: {
				Value:        "",
				Configurable: true,
				Type:         HookConfigTypePlatform,
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
			SchedulerModelPayload: {
				Value:        "{}",
				Configurable: true,
				Type:         HookConfigTypeString,
			},
		},
	}
)

// Hook used to link a git repository to a given pipeline
type Hook struct {
	ID            int64    `json:"id"`
	UID           string   `json:"uid"`
	Pipeline      Pipeline `json:"pipeline"`
	ApplicationID int64    `json:"application_id"`
	Kind          string   `json:"kind"`
	Host          string   `json:"host"`
	Project       string   `json:"project"`
	Repository    string   `json:"repository"`
	Enabled       bool     `json:"enabled"`
	Link          string   `json:"link"`
}

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
	}

	return WebHookModel
}
