//go:build e2e
// +build e2e

package gp

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/simulot/immich-go/internal/metadata"
	"github.com/telemachus/humane"
)

func TestReadBigTakeout(t *testing.T) {
	f, err := os.Create("bigread.log")
	if err != nil {
		panic(err)
	}

	l := slog.New(humane.NewHandler(f, &humane.Options{Level: slog.LevelInfo}))
	j := fileevent.NewRecorder(l)
	m, err := filepath.Glob("../../../test-data/full_takeout/*.zip")
	if err != nil {
		t.Error(err)
		return
	}
	cnt := 0
	fsyss, err := fshelper.ParsePath(m)
	flags := &ImportFlags{
		SupportedMedia: metadata.DefaultSupportedMedia,
	}

	to, err := NewTakeout(context.Background(), j, flags, fsyss...)
	if err != nil {
		t.Error(err)
		return
	}

	assets, err := to.Browse(context.Background())
	if err != nil {
		t.Error(err)
		return
	}

	for range assets {
		cnt++
	}
	l.Info(fmt.Sprintf("files seen %d", cnt))
	j.Report()
}