package sdk

type WorkflowData struct {
	Node  Node   `json:"node" db:"-" cli:"-"`
	Joins []Node `json:"joins" db:"-" cli:"-"`
}

func (w *WorkflowData) Maps() map[int64]*Node {
	nodes := make(map[int64]*Node, 0)
	w.Node.maps(&nodes)
	for i := range w.Joins {
		w.Joins[i].maps(&nodes)
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
