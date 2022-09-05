package sdk

import "fmt"

type V2WorkerModel struct {
	Name        string                `json:"name"`
	From        string                `json:"from"`
	Description string                `json:"description,omitempty"`
	Docker      *WorkerModelDocker    `json:"docker,omitempty"`
	Openstack   *WorkerModelOpenstack `json:"openstack,omitempty"`
	VSphere     *WorkerModelVSphere   `json:"vsphere,omitempty"`
}

func (wm V2WorkerModel) GetName() string {
	return wm.Name
}

func (wm V2WorkerModel) Lint() error {
	if wm.Name == "" {
		return WithStack(fmt.Errorf("missing worker model template name"))
	}
	if wm.VSphere != nil && (wm.Openstack != nil || wm.Docker != nil) ||
		wm.Docker != nil && (wm.Openstack != nil || wm.VSphere != nil) ||
		wm.Openstack != nil && (wm.Docker != nil || wm.VSphere != nil) {
		return WithStack(fmt.Errorf("worker model cannot have multiple types"))
	}

	switch {
	case wm.Docker != nil:
		if wm.Docker.Image == "" {
			return WithStack(fmt.Errorf("missing image path"))
		}
		if wm.From != "" && (wm.Docker.Cmd != "" || wm.Docker.Shell != "" || len(wm.Docker.Envs) > 0) {
			return WithStack(fmt.Errorf("you can't override worker model template (cmd,shell,envs)"))
		}
	case wm.Openstack != nil:
		if wm.Openstack.Flavor == "" {
			return WithStack(fmt.Errorf("missing flavor"))
		}
		if wm.Openstack.Image == "" {
			return WithStack(fmt.Errorf("missing image"))
		}
		if wm.From != "" && (wm.Openstack.Cmd != "" || wm.Openstack.PreCmd != "" || wm.Openstack.PostCmd != "") {
			return WithStack(fmt.Errorf("you can't override worker model template (cmd,pre_cmd,post_cmd)"))
		}
	case wm.VSphere != nil:
		if wm.VSphere.Image == "" {
			return WithStack(fmt.Errorf("missing image"))
		}
		if wm.VSphere.Username == "" || wm.VSphere.Password == "" {
			return WithStack(fmt.Errorf("missing vm credentials"))
		}
		if wm.From != "" && (wm.VSphere.Cmd != "" || wm.VSphere.PreCmd != "" || wm.VSphere.PostCmd != "") {
			return WithStack(fmt.Errorf("you can't override worker model template (cmd,pre_cmd,post_cmd)"))
		}
	}
	return nil
}

type WorkerModelDocker struct {
	Image    string            `json:"image"`
	Registry string            `json:"registry,omitempty"`
	Username string            `json:"username,omitempty"`
	Password string            `json:"password,omitempty"`
	Cmd      string            `json:"cmd,omitempty"`
	Shell    string            `json:"shell"`
	Envs     map[string]string `json:"envs"`
}

type WorkerModelOpenstack struct {
	Image   string `json:"image"`
	Flavor  string `json:"flavor"`
	Cmd     string `json:"cmd,omitempty"`
	PreCmd  string `json:"pre_cmd"`
	PostCmd string `json:"post_cmd"`
}

type WorkerModelVSphere struct {
	Image    string `json:"image"`
	Username string `json:"username"`
	Password string `json:"password"`
	Cmd      string `json:"cmd,omitempty"`
	PreCmd   string `json:"pre_cmd"`
	PostCmd  string `json:"post_cmd"`
}
