package event

import (
	"fmt"
	"os"

	"github.com/docker/docker/pkg/namesgenerator"
)

var hostname, cdsname string

// Routine initializes and run event routine dequeue
func Routine() {
	var err error
	hostname, err = os.Hostname()
	if err != nil {
		hostname = fmt.Sprintf("Error while getting Hostname: %s", err.Error())
	}
	cdsname = namesgenerator.GetRandomName(0)

	kafkaRoutine()
}
