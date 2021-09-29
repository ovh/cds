package workflowv3

import "fmt"

type Keys map[string]Key

func (d Keys) ExistKey(keyName string, t KeyType) bool {
	k, ok := d[keyName]
	return ok && k.Type == t
}

type Key struct {
	Type KeyType `json:"type,omitempty" yaml:"type,omitempty"`
}

func (k Key) Validate(w Workflow) error {
	if err := k.Type.Validate(); err != nil {
		return err
	}
	return nil
}

type KeyType string

const (
	SSHKeyType KeyType = "ssh"
	PGPKeyType KeyType = "pgp"
)

func (k KeyType) Validate() error {
	switch k {
	case SSHKeyType, PGPKeyType:
		return nil
	default:
		return fmt.Errorf("invalid given key type %q", k)
	}
}
