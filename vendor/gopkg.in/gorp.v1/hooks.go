// Copyright 2012 James Cooper. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package gorp provides a simple way to marshal Go structs to and from
// SQL databases.  It uses the database/sql package, and should work with any
// compliant database/sql driver.
//
// Source code and project home:
// https://github.com/go-gorp/gorp

package gorp

//++ TODO v2-phase3: HasPostGet => PostGetter, HasPostDelete => PostDeleter, etc.

// HasPostGet provides PostGet() which will be executed after the GET statement.
type HasPostGet interface {
	PostGet(SqlExecutor) error
}

// HasPostDelete provides PostDelete() which will be executed after the DELETE statement
type HasPostDelete interface {
	PostDelete(SqlExecutor) error
}

// HasPostUpdate provides PostUpdate() which will be executed after the UPDATE statement
type HasPostUpdate interface {
	PostUpdate(SqlExecutor) error
}

// HasPostInsert provides PostInsert() which will be executed after the INSERT statement
type HasPostInsert interface {
	PostInsert(SqlExecutor) error
}

// HasPreDelete provides PreDelete() which will be executed before the DELETE statement.
type HasPreDelete interface {
	PreDelete(SqlExecutor) error
}

// HasPreUpdate provides PreUpdate() which will be executed before UPDATE statement.
type HasPreUpdate interface {
	PreUpdate(SqlExecutor) error
}

// HasPreInsert provides PreInsert() which will be executed before INSERT statement.
type HasPreInsert interface {
	PreInsert(SqlExecutor) error
}
