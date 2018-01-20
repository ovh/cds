/*
Copyright (c) 2017 VMware, Inc. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package task

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/vmware/govmomi/govc/cli"
	"github.com/vmware/govmomi/govc/flags"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/types"
)

type recent struct {
	*flags.DatacenterFlag

	max    int
	follow bool
}

func init() {
	cli.Register("tasks", &recent{})
}

func (cmd *recent) Register(ctx context.Context, f *flag.FlagSet) {
	cmd.DatacenterFlag, ctx = flags.NewDatacenterFlag(ctx)
	cmd.DatacenterFlag.Register(ctx, f)

	f.IntVar(&cmd.max, "n", 25, "Output the last N tasks")
	f.BoolVar(&cmd.follow, "f", false, "Follow recent task updates")
}

func (cmd *recent) Description() string {
	return `Display info for recent tasks.

When a task has completed, the result column includes the task duration on success or
error message on failure.  If a task is still in progress, the result column displays
the completion percentage and the task ID.  The task ID can be used as an argument to
the 'task.cancel' command.

By default, all recent tasks are included (via TaskManager), but can be limited by PATH
to a specific inventory object.

Examples:
  govc tasks
  govc tasks -f
  govc tasks -f /dc1/host/cluster1`
}

func (cmd *recent) Usage() string {
	return "[PATH]"
}

func (cmd *recent) Process(ctx context.Context) error {
	if err := cmd.DatacenterFlag.Process(ctx); err != nil {
		return err
	}
	return nil
}

func chop(s string) string {
	if len(s) < 30 {
		return s
	}

	return s[:29] + "*"
}

func (cmd *recent) Run(ctx context.Context, f *flag.FlagSet) error {
	c, err := cmd.Client()
	if err != nil {
		return err
	}

	m := c.ServiceContent.TaskManager

	watch := *m

	if f.NArg() == 1 {
		refs, merr := cmd.ManagedObjects(ctx, f.Args())
		if merr != nil {
			return merr
		}
		watch = refs[0]
	}

	v, err := view.NewManager(c).CreateTaskView(ctx, &watch)
	if err != nil {
		return nil
	}

	defer v.Destroy(context.Background())

	v.Follow = cmd.follow

	stamp := "15:04:05"
	tmpl := "%-30s %-30s %13s %9s %9s %9s %s\n"
	fmt.Fprintf(cmd.Out, tmpl, "Task", "Target", "Initiator", "Queued", "Started", "Completed", "Result")

	var last string
	updated := false

	return v.Collect(ctx, func(tasks []types.TaskInfo) {
		if !updated && len(tasks) > cmd.max {
			tasks = tasks[len(tasks)-cmd.max:]
		}
		updated = true

		for _, info := range tasks {
			var user string

			switch x := info.Reason.(type) {
			case *types.TaskReasonUser:
				user = x.UserName
			}

			if info.EntityName == "" || user == "" {
				continue
			}

			ruser := strings.SplitN(user, "\\", 2)
			if len(ruser) == 2 {
				user = ruser[1] // discard domain
			}

			queued := info.QueueTime.Format(stamp)
			start := "-"
			end := start

			if info.StartTime != nil {
				start = info.StartTime.Format(stamp)
			}

			result := fmt.Sprintf("%2d%% %s", info.Progress, info.Task)

			if info.CompleteTime != nil {
				if info.State == types.TaskInfoStateError {
					result = info.Error.LocalizedMessage
				} else {
					result = fmt.Sprintf("%s (%s)", info.State, info.CompleteTime.Sub(*info.StartTime).String())
				}

				end = info.CompleteTime.Format(stamp)
			}

			name := strings.TrimSuffix(info.Name, "_Task")
			switch name {
			case "Destroy", "Rename":
				name = info.Entity.Type + "." + name
			}

			item := fmt.Sprintf(tmpl, name, chop(info.EntityName), user, queued, start, end, result)

			if item == last {
				continue // task info was updated, but the fields we display were not
			}
			last = item

			fmt.Fprint(cmd.Out, item)
		}
	})
}
