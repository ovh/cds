package workflowv3

type Secrets map[string]Secret

func (s Secrets) ExistSecret(secretName string) bool {
	_, ok := s[secretName]
	return ok
}

type Secret string
