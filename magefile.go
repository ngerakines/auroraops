//+build mage

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

var Default = Build

func BuildAuroraOps(ctx context.Context) error {
	fmt.Println("Building auroraops...")
	output := envOrDefault("AURORAOPS_BINARY", "auroraops")
	if runtime.GOOS == "windows" {
		output += ".exe"
	}
	return sh.RunV("go", "build", "-o", output, "cmd/auroraops/main.go")
}

func BuildStatusHost(ctx context.Context) error {
	fmt.Println("Building statushost...")
	output := "statushost"
	if runtime.GOOS == "windows" {
		output += ".exe"
	}
	return sh.RunV("go", "build", "-o", output, "cmd/statushost/main.go")
}

func Build(ctx context.Context) error {
	fmt.Println("Building...")
	mg.CtxDeps(ctx, BuildStatusHost, BuildAuroraOps)
	return nil
}

func Release() (err error) {
	if os.Getenv("TAG") == "" {
		return errors.New("TAG environment variable is required")
	}
	if err := sh.RunV("git", "tag", "-a", "$TAG"); err != nil {
		return err
	}
	if err := sh.RunV("git", "push", "origin", "$TAG"); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			sh.RunV("git", "tag", "--delete", "$TAG")
			sh.RunV("git", "push", "--delete", "origin", "$TAG")
		}
	}()
	return sh.RunV("goreleaser")
}

func envOrDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
