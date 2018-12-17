package sdk

type WorkflowData struct {
	Node  Node   `json:"node" db:"-" cli:"-"`
	Joins []Node `json:"joins" db:"-" cli:"-"`
}

func (w *WorkflowData) AncestorsNames(n Node) []string {
	res, ok := w.Node.ancestorNames(n.Name)
	if ok {
		return res
	}

	for _, j := range w.Joins {
		resAncestor, found := (&j).ancestorNames(n.Name)
		if found {
			return resAncestor
		}
	}
	return nil
}

// GetHooks returns the list of all hooks in the workflow tree
func (w *WorkflowData) GetHooks() map[string]NodeHook {
	if w == nil {
		return nil
	}
	res := map[string]NodeHook{}
	for _, n := range w.Array() {
		for _, h := range n.Hooks {
			res[h.UUID] = h
		}
	}
	return res
}

// GetHooksMapRef returns the list of all hooks in the workflow tree
func (w *WorkflowData) GetHooksMapRef() map[string]NodeHook {
	if w == nil {
		return nil
	}
	res := make(map[string]NodeHook)
	for _, n := range w.Array() {
		for _, h := range n.Hooks {
			res[h.Ref] = h
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
	for _, n := range w.Array() {
		if n.Name == s {
			return n
		}
	}
	return nil
}
