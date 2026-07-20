package config

import (
	"log/slog"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// ConfigWatcher watches the config directory and reloads registries on file changes.
//
// The tag registry is deliberately NOT hot-reloaded here (CYB-3263). Since
// open-vocabulary tag validation (CYB-3246) landed, asset_tags is the runtime
// source of truth for tag data; tag_registry.yaml provides only the
// enum/propagation/source contracts, loaded once at startup. Watching and
// live-reloading the YAML created a second, drifting source of truth that could
// silently change validation behavior without a deploy — so the tag branch was
// removed. algo/action-label registries remain hot-reloadable.
type ConfigWatcher struct {
	watcher             *fsnotify.Watcher
	algoRegistry        *AlgoRegistry
	actionLabelRegistry *ActionLabelRegistry
	done                chan struct{}
	wg                  sync.WaitGroup
}

// NewConfigWatcher creates a watcher that monitors the given directory
// and reloads the provided registries when config files change.
func NewConfigWatcher(configDir string, algoReg *AlgoRegistry, actionLabelReg *ActionLabelRegistry) (*ConfigWatcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	if err := w.Add(configDir); err != nil {
		w.Close()
		return nil, err
	}
	cw := &ConfigWatcher{
		watcher:             w,
		algoRegistry:        algoReg,
		actionLabelRegistry: actionLabelReg,
		done:                make(chan struct{}),
	}
	cw.wg.Add(1)
	go cw.loop()
	return cw, nil
}

func (cw *ConfigWatcher) loop() {
	defer cw.wg.Done()

	// Debounce timer — coalesce rapid saves into a single reload.
	const debounce = 100 * time.Millisecond
	timer := time.NewTimer(0)
	if !timer.Stop() {
		<-timer.C
	}
	pending := map[string]struct{}{}

	for {
		select {
		case <-cw.done:
			timer.Stop()
			return
		case event, ok := <-cw.watcher.Events:
			if !ok {
				return
			}
			if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
				continue
			}
			base := filepath.Base(event.Name)
			if base == "algo_registry.yaml" || base == "action_label_registry.yaml" {
				pending[base] = struct{}{}
				timer.Reset(debounce)
			}
		case err, ok := <-cw.watcher.Errors:
			if !ok {
				return
			}
			slog.Error("config watcher error", "err", err)
		case <-timer.C:
			for name := range pending {
				switch name {
				case "algo_registry.yaml":
					if err := cw.algoRegistry.Reload(); err != nil {
						slog.Error("algo_registry reload failed", "err", err)
					} else {
						slog.Info("algo_registry reloaded")
					}
				case "action_label_registry.yaml":
					if cw.actionLabelRegistry == nil {
						continue
					}
					if err := cw.actionLabelRegistry.Reload(); err != nil {
						slog.Error("action_label_registry reload failed", "err", err)
					} else {
						slog.Info("action_label_registry reloaded")
					}
				}
			}
			pending = map[string]struct{}{}
		}
	}
}

// Stop shuts down the watcher and waits for the goroutine to exit.
func (cw *ConfigWatcher) Stop() {
	close(cw.done)
	cw.watcher.Close()
	cw.wg.Wait()
}
