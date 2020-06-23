package logging

import (
	"fmt"
	"os"
	"runtime/debug"
	"time"

	"github.com/getsentry/sentry-go"
)

// Fatalf sends a message to sentry and then exits the program
func Fatal(err error, context string, params ...interface{}) {
	sentry.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetLevel(sentry.LevelFatal)
	})
	contextS := fmt.Sprintf(context, params...)
	message := fmt.Sprintf("error (%s): %+v", contextS, err)
	sentry.CaptureMessage(message)
	fmt.Fprintf(os.Stderr, "%s\n", message)

	// dump the stack
	debug.PrintStack()

	// flush remaining messages before quitting!
	sentry.Flush(2 * time.Second)
	os.Exit(1)
}

func Infof(format string, params ...interface{}) {
	message := fmt.Sprintf(format, params...)
	sentry.CaptureMessage(message)
}
