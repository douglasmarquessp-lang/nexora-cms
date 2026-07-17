package ai

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
)

func TestNewStreamProcessor(t *testing.T) {
	sp := NewStreamProcessor()
	if sp == nil {
		t.Fatal("expected non-nil processor")
	}
}

func TestStreamProcessor_HandleChunk(t *testing.T) {
	var gotContent string
	sp := NewStreamProcessor(
		WithChunkHandler(func(chunk StreamChunk) error {
			gotContent += chunk.Content
			return nil
		}),
	)

	sp.HandleChunk(StreamChunk{Content: "Hello ", Index: 0, Done: false})
	sp.HandleChunk(StreamChunk{Content: "World", Index: 1, Done: true})

	if gotContent != "Hello World" {
		t.Errorf("expected 'Hello World', got '%s'", gotContent)
	}
}

func TestStreamProcessor_OnComplete(t *testing.T) {
	var completed bool
	sp := NewStreamProcessor(
		WithCompleteHandler(func(result *CompletionResult) error {
			completed = true
			return nil
		}),
	)

	sp.OnComplete(&CompletionResult{Content: "Done"})
	if !completed {
		t.Error("expected complete handler called")
	}
}

func TestStreamProcessor_OnError(t *testing.T) {
	var gotErr error
	sp := NewStreamProcessor(
		WithErrorHandler(func(err error) error {
			gotErr = err
			return err
		}),
	)

	testErr := errors.New("test error")
	sp.OnError(testErr)
	if gotErr != testErr {
		t.Errorf("expected test error, got %v", gotErr)
	}
}

func TestStreamProcessor_OnProgress(t *testing.T) {
	var progress float64
	sp := NewStreamProcessor(
		WithProgressHandler(func(p float64) error {
			progress = p
			return nil
		}),
	)

	sp.OnProgress(0.5)
	if progress != 0.5 {
		t.Errorf("expected 0.5, got %f", progress)
	}
}

func TestStreamProcessor_Cancel(t *testing.T) {
	sp := NewStreamProcessor()
	if sp.IsCancelled() {
		t.Error("expected not cancelled initially")
	}
	sp.Cancel()
	if !sp.IsCancelled() {
		t.Error("expected cancelled")
	}
}

func TestStreamProcessor_ChunkCount(t *testing.T) {
	sp := NewStreamProcessor()
	sp.HandleChunk(StreamChunk{Content: "a", Index: 0, Done: false})
	sp.HandleChunk(StreamChunk{Content: "b", Index: 1, Done: false})

	if count := sp.ChunkCount(); count != 2 {
		t.Errorf("expected 2 chunks, got %d", count)
	}
}

func TestStreamProcessor_DefaultHandlers(t *testing.T) {
	sp := NewStreamProcessor()

	err := sp.HandleChunk(StreamChunk{Content: "test"})
	if err != nil {
		t.Errorf("expected no error from default handler, got %v", err)
	}

	err = sp.OnComplete(&CompletionResult{})
	if err != nil {
		t.Errorf("expected no error from default complete, got %v", err)
	}

	err = sp.OnError(errors.New("err"))
	if err != nil {
		t.Errorf("expected no error from default error handler, got %v", err)
	}

	err = sp.OnProgress(0.5)
	if err != nil {
		t.Errorf("expected no error from default progress, got %v", err)
	}
}

func TestCollectStream(t *testing.T) {
	p := NewMockProvider("test", "test-model", nil)
	ctx := context.Background()

	content, err := CollectStream(ctx, p, CompletionRequest{Prompt: "Hello"})
	if err != nil {
		t.Fatalf("CollectStream failed: %v", err)
	}
	if content == "" {
		t.Error("expected non-empty content")
	}
}

func TestNewStreamAccumulator(t *testing.T) {
	sa := NewStreamAccumulator()
	if sa == nil {
		t.Fatal("expected non-nil accumulator")
	}

	sa.HandleChunk(StreamChunk{Content: "Hello ", Index: 0})
	sa.HandleChunk(StreamChunk{Content: "World", Index: 1})
	sa.OnComplete(&CompletionResult{})
	sa.OnError(nil)
	sa.OnProgress(1.0)

	if sa.Text() != "Hello World" {
		t.Errorf("expected 'Hello World', got '%s'", sa.Text())
	}
}

func TestStreamProcessor_ChunkHandlerError(t *testing.T) {
	expectedErr := errors.New("chunk error")
	sp := NewStreamProcessor(
		WithChunkHandler(func(chunk StreamChunk) error {
			return expectedErr
		}),
	)

	err := sp.HandleChunk(StreamChunk{Content: "test"})
	if err != expectedErr {
		t.Errorf("expected chunk error, got %v", err)
	}
}

func TestProcessStream(t *testing.T) {
	p := NewMockProvider("test", "test-model", nil)
	ctx := context.Background()

	var chunkCount atomic.Int64
	ch, err := ProcessStream(ctx, p, CompletionRequest{Prompt: "Hello"},
		WithChunkHandler(func(chunk StreamChunk) error {
			chunkCount.Add(1)
			return nil
		}),
	)
	if err != nil {
		t.Fatalf("ProcessStream failed: %v", err)
	}

	for range ch {
		// consume all chunks
	}
	if chunkCount.Load() == 0 {
		t.Error("expected chunks to be processed")
	}
}
