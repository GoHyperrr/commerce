package analytics

import (
	"context"
	"testing"
	"time"

	"github.com/GoHyperrr/mdk"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestAnalyticsModule(t *testing.T) {
	database, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	rt := mdk.NewTestRuntime(database)

	testProj := &mdk.TestProjector{}
	ctxMod := &mdk.TestContextModule{Proj: testProj}
	rt.SetModule("core.context", ctxMod)

	mod := NewModule()
	_ = mod.Init(context.Background(), rt)
	_ = database.AutoMigrate(mod.Models()...)

	t.Run("System Stats", func(t *testing.T) {
		now := time.Now()
		ended := now.Add(time.Second)
		testProj.Lineages = []mdk.LineageData{
			mdk.TestLineageData{
				ID:        "wf1",
				Name:      "test",
				State:     "COMPLETED",
				StartedAt: now,
				EndedAt:   &ended,
				Events: []mdk.Event{
					{
						Namespace: "workflow",
						Type:      "started",
						Payload:   map[string]any{"id": "wf1", "name": "test", "version": "v1"},
					},
				},
			},
		}
		
		stats := testProj.ListLineages()
		if len(stats) == 0 {
			t.Error("expected at least 1 lineage")
		}
	})

	t.Run("Module API", func(t *testing.T) {
		if mod.ID() != "commerce.analytics" {
			t.Error("invalid ID")
		}
		if mod.Repo() != nil {
			t.Error("repo should be nil")
		}
		if len(mod.Models()) != 0 {
			t.Error("models should be empty")
		}
	})
}
