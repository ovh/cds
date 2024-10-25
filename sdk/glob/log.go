package glob

import "fmt"

type LogFunc func(a ...any) (n int, err error)

var (
	DebugEnabled bool    = false
	DebugFunc    LogFunc = fmt.Println
)

func Debug(format string, args ...interface{}) {
	if DebugEnabled {
		if len(args) > 0 {
			DebugFunc("[GLOB]" + fmt.Sprintf(format, args...))
		} else {
			DebugFunc("[GLOB]" + format)
		}
	}
}
