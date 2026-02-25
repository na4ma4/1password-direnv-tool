//go:build mage

package main

import (
	"context"

	"github.com/magefile/mage/mg"

	//mage:import
	"github.com/dosquad/mage"
)

func init() {
	// dyndep.Add(dyndep.Golang,
	// 	func(ctx context.Context) error { mg.F(mage.Golang.Generate); return nil },
	// )
}

// TestLocal update, protoc, format, tidy, lint & test.
func TestLocal(ctx context.Context) {
	mg.CtxDeps(ctx, mage.Test)
}

var Default = TestLocal
