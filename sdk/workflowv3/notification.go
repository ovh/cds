package workflowv3

import "fmt"

type NotificationType string

const (
	NotificationTypeEmails NotificationType = "emails"
)

func (n NotificationType) Validate() error {
	switch n {
	case NotificationTypeEmails:
		return nil
	default:
		return fmt.Errorf("invalid given notification type %q", n)
	}
}

type Notification struct {
	Type NotificationType `json:"type,omitempty" yaml:"type,omitempty"`
	Jobs []string         `json:"jobs,omitempty" yaml:"jobs,omitempty"`
}

func (n Notification) Validate(w Workflow) error {
	if err := n.Type.Validate(); err != nil {
		return err
	}

	for _, jName := range n.Jobs {
		if !w.Jobs.ExistJob(jName) {
			return fmt.Errorf("unknown job %q", jName)
		}
	}

	return nil
}
