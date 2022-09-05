package sdk

import "fmt"

type WorkerModelTemplate struct {
	Name   string                     `json:"name"`
	Docker *WorkerModelTemplateDocker `json:"docker,omitempty"`
	VM     *WorkerModelTemplateVM     `json:"vm,omitempty"`
}

type WorkerModelTemplateDocker struct {
	Cmd   string            `json:"cmd"`
	Shell string            `json:"shell"`
	Envs  map[string]string `json:"envs"`
}

type WorkerModelTemplateVM struct {
	Cmd     string `json:"cmd"`
	PreCmd  string `json:"pre_cmd"`
	PostCmd string `json:"post_cmd"`
}

func (wmt WorkerModelTemplate) Lint() error {
	if wmt.Name == "" {
		return WithStack(fmt.Errorf("missing worker model template name"))
	}
	if wmt.Docker != nil {
		if wmt.Docker.Cmd == "" {
			return WithStack(fmt.Errorf("missing docker cmd"))
		}
		if wmt.Docker.Shell == "" {
			return WithStack(fmt.Errorf("missing docker shell"))
		}
	}
	if wmt.VM != nil {
		if wmt.VM.Cmd == "" {
			return WithStack(fmt.Errorf("missing vm cmd"))
		}
		if wmt.VM.PostCmd == "" {
			return WithStack(fmt.Errorf("missing vm post_cmd to shutdown the VM"))
		}
	}
	return nil
}

func (wmt WorkerModelTemplate) GetName() string {
	return wmt.Name
}
