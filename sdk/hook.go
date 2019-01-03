package sdk

// These are constants about hooks
const (
	WebHookModelName              = "WebHook"
	RepositoryWebHookModelName    = "RepositoryWebHook"
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
	HookConfigModelType           = "model_type"
	HookConfigModelName           = "model_name"
	WebHookModelConfigMethod      = "method"
	RepositoryWebHookModelMethod  = "method"
	SchedulerModelCron            = "cron"
	SchedulerModelTimezone        = "timezone"
	Payload                       = "payload"
	HookModelPlatform             = "platform"
	KafkaHookModelConsumerGroup   = "consumer group"
	KafkaHookModelTopic           = "topic"
	RabbitMQHookModelQueue        = "queue"
	RabbitMQHookModelBindingKey   = "binding_key"
	RabbitMQHookModelExchangeType = "exchange_type"
	RabbitMQHookModelExchangeName = "exchange_name"
	RabbitMQHookModelConsumerTag  = "consumer_tag"
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
			Payload: {
				Value:        "{}",
				Configurable: true,
				Type:         HookConfigTypeString,
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
