package sdk

type WorkflowData struct {
	Node  Node   `json:"node" db:"-" cli:"-"`
	Joins []Node `json:"joins" db:"-" cli:"-"`
}

// GetHooks returns the list of all hooks in the workflow tree
func (w *WorkflowData) GetHooks() map[string]NodeHook {
	if w == nil {
		return nil
	}

	res := map[string]NodeHook{}

	a := w.Node.GetHooks()
	for k, v := range a {
		res[k] = v
	}

	for _, j := range w.Joins {
		b := j.GetHooks()
		for k, v := range b {
			res[k] = v
		}
	}
	return res
}

func (w *WorkflowData) Array() []*Node {
	nodes := make([]*Node, 0)
	nodes = w.Node.array(nodes)
	for i := range w.Joins {
		nodes = w.Joins[i].array(nodes)
	}
	return nodes
}

func (w *WorkflowData) Maps() map[int64]*Node {
	nodes := make(map[int64]*Node, 0)
	nodes = w.Node.maps(nodes)
	for i := range w.Joins {
		nodes = w.Joins[i].maps(nodes)
	}
	return nodes
}

func (w *WorkflowData) NodeByRef(ref string) *Node {
	n := (&w.Node).nodeByRef(ref)
	if n != nil {
		return n
	}
	for i := range w.Joins {
		n = (&w.Joins[i]).nodeByRef(ref)
		if n != nil {
			return n
		}
	}
	return nil
}

func (w *WorkflowData) NodeByID(ID int64) *Node {
	n := (&w.Node).nodeByID(ID)
	if n != nil {
		return n
	}
	for i := range w.Joins {
		n = (&w.Joins[i]).nodeByID(ID)
		if n != nil {
			return n
		}
	}
	return nil
}
func (w *WorkflowData) NodeByName(s string) *Node {
	n := w.Node.GetNodeByName(s)
	if n != nil {
		return n
	}
	for _, j := range w.Joins {
		n = j.GetNodeByName(s)
		if n != nil {
			return n
		}
	}
	return nil
}
