package ai

import (
	"context"
	"sync"
	"sync/atomic"
)

type StreamProcessor struct {
	chunkFn    func(StreamChunk) error
	completeFn func(*CompletionResult) error
	errorFn    func(error) error
	progressFn func(float64) error
	cancelled  atomic.Bool
	chunkCount int64
}

type StreamOption func(*StreamProcessor)

func WithChunkHandler(fn func(StreamChunk) error) StreamOption {
	return func(sp *StreamProcessor) { sp.chunkFn = fn }
}

func WithCompleteHandler(fn func(*CompletionResult) error) StreamOption {
	return func(sp *StreamProcessor) { sp.completeFn = fn }
}

func WithErrorHandler(fn func(error) error) StreamOption {
	return func(sp *StreamProcessor) { sp.errorFn = fn }
}

func WithProgressHandler(fn func(float64) error) StreamOption {
	return func(sp *StreamProcessor) { sp.progressFn = fn }
}

func NewStreamProcessor(opts ...StreamOption) *StreamProcessor {
	sp := &StreamProcessor{}
	for _, opt := range opts {
		opt(sp)
	}
	return sp
}

func (sp *StreamProcessor) HandleChunk(chunk StreamChunk) error {
	atomic.AddInt64(&sp.chunkCount, 1)
	if sp.chunkFn != nil {
		return sp.chunkFn(chunk)
	}
	return nil
}

func (sp *StreamProcessor) OnComplete(result *CompletionResult) error {
	if sp.completeFn != nil {
		return sp.completeFn(result)
	}
	return nil
}

func (sp *StreamProcessor) OnError(err error) error {
	if sp.errorFn != nil {
		return sp.errorFn(err)
	}
	return nil
}

func (sp *StreamProcessor) OnProgress(progress float64) error {
	if sp.progressFn != nil {
		return sp.progressFn(progress)
	}
	return nil
}

func (sp *StreamProcessor) Cancel() {
	sp.cancelled.Store(true)
}

func (sp *StreamProcessor) IsCancelled() bool {
	return sp.cancelled.Load()
}

func (sp *StreamProcessor) ChunkCount() int64 {
	return atomic.LoadInt64(&sp.chunkCount)
}

func ProcessStream(ctx context.Context, provider AIProvider, req CompletionRequest, opts ...StreamOption) (<-chan StreamChunk, error) {
	handler := NewStreamProcessor(opts...)
	ch, err := provider.GenerateStream(ctx, req)
	if err != nil {
		return nil, err
	}

	out := make(chan StreamChunk)
	go func() {
		defer close(out)
		for chunk := range ch {
			_ = handler.HandleChunk(chunk)
			out <- chunk
			if chunk.Done {
				_ = handler.OnComplete(&CompletionResult{
					Content:      "",
					ProviderName: provider.Name(),
					FinishReason: chunk.FinishReason,
				})
				return
			}
		}
	}()
	return out, nil
}

func CollectStream(ctx context.Context, provider AIProvider, req CompletionRequest) (string, error) {
	var fullContent string
	handler := NewStreamProcessor(
		WithChunkHandler(func(chunk StreamChunk) error {
			fullContent += chunk.Content
			return nil
		}),
	)

	ch, err := provider.GenerateStream(ctx, req)
	if err != nil {
		return "", err
	}

	for chunk := range ch {
		if chunk.Error != nil {
			return fullContent, chunk.Error
		}
		_ = handler.HandleChunk(chunk)
		if chunk.Done {
			break
		}
	}

	return fullContent, nil
}

type StreamAccumulator struct {
	Content string
	mu      sync.Mutex
}

func NewStreamAccumulator() *StreamAccumulator {
	return &StreamAccumulator{}
}

func (sa *StreamAccumulator) HandleChunk(chunk StreamChunk) error {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	sa.Content += chunk.Content
	return nil
}

func (sa *StreamAccumulator) OnComplete(result *CompletionResult) error {
	return nil
}

func (sa *StreamAccumulator) OnError(err error) error {
	return nil
}

func (sa *StreamAccumulator) OnProgress(progress float64) error {
	return nil
}

func (sa *StreamAccumulator) Text() string {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	return sa.Content
}
