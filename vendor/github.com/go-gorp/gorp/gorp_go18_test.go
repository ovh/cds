// Copyright 2012 James Cooper. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package gorp provides a simple way to marshal Go structs to and from
// SQL databases.  It uses the database/sql package, and should work with any
// compliant database/sql driver.
//
// Source code and project home:
// https://github.com/go-gorp/gorp
//

// +build go1.8

package gorp_test

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

// Drivers that don't support cancellation.
var unsupportedDrivers map[string]bool = map[string]bool{
	"mymysql": true,
}

type SleepDialect interface {
	// string to sleep for d duration
	SleepClause(d time.Duration) string
}

func TestWithNotCanceledContext(t *testing.T) {
	dbmap := initDbMap()
	defer dropAndClose(dbmap)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	withCtx := dbmap.WithContext(ctx)

	_, err := withCtx.Exec("SELECT 1")
	assert.Nil(t, err)
}

func TestWithCanceledContext(t *testing.T) {
	dialect, driver := dialectAndDriver()
	if unsupportedDrivers[driver] {
		t.Skip("Cancellation is not yet supported by all drivers. Not known to be supported in %s.", driver)
	}

	sleepDialect, ok := dialect.(SleepDialect)
	if !ok {
		t.Skip("Sleep is not supported in all dialects. Not known to be supported in %s.", driver)
	}

	dbmap := initDbMap()
	defer dropAndClose(dbmap)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	withCtx := dbmap.WithContext(ctx)

	startTime := time.Now()

	_, err := withCtx.Exec("SELECT " + sleepDialect.SleepClause(1*time.Second))

	if d := time.Since(startTime); d > 500*time.Millisecond {
		t.Errorf("too long execution time: %s", d)
	}

	switch driver {
	case "postgres":
		// pq doesn't return standard deadline exceeded error
		if err.Error() != "pq: canceling statement due to user request" {
			t.Errorf("expected context.DeadlineExceeded, got %v", err)
		}
	default:
		if err != context.DeadlineExceeded {
			t.Errorf("expected context.DeadlineExceeded, got %v", err)
		}
	}
}
