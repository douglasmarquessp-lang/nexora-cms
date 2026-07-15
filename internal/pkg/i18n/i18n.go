package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"path"
	"sync"
)

type Bundle struct {
	mu       sync.RWMutex
	locales  map[string]map[string]string
	fallback string
}

func NewBundle(fallback string) *Bundle {
	return &Bundle{
		locales:  make(map[string]map[string]string),
		fallback: fallback,
	}
}

func (b *Bundle) LoadFromFS(fsys embed.FS, pattern string) error {
	entries, err := fsys.ReadDir(".")
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if pattern != "" {
			matched, err := path.Match(pattern, entry.Name())
			if err != nil || !matched {
				continue
			}
		}

		data, err := fsys.ReadFile(entry.Name())
		if err != nil {
			return err
		}

		locale := entry.Name()[:len(entry.Name())-5]
		var messages map[string]string
		if err := json.Unmarshal(data, &messages); err != nil {
			return fmt.Errorf("failed to parse %s: %w", entry.Name(), err)
		}

		b.mu.Lock()
		b.locales[locale] = messages
		b.mu.Unlock()
	}

	return nil
}

func (b *Bundle) T(locale, key string, args ...interface{}) string {
	b.mu.RLock()
	messages, ok := b.locales[locale]
	b.mu.RUnlock()

	if !ok {
		messages = b.locales[b.fallback]
	}

	msg, ok := messages[key]
	if !ok {
		msg = key
	}

	if len(args) > 0 {
		return fmt.Sprintf(msg, args...)
	}

	return msg
}
