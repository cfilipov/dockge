package main

import (
	"flag"
	"os"
	"testing"

	"github.com/cfilipov/dockge/internal/testutil"
)

var (
	flagMockDaemon = flag.String("mock-daemon", "", "path to mock daemon binary")
	flagMockDocker = flag.String("mock-docker", "", "path to mock docker CLI binary")
)

func TestMain(m *testing.M) {
	flag.Parse()
	if *flagMockDaemon != "" {
		testutil.StartDaemon(*flagMockDaemon, *flagMockDocker)
	}
	code := m.Run()
	testutil.StopDaemon()
	os.Exit(code)
}
