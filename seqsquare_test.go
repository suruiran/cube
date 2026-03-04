package cube

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSeqSquare_Concurrency(t *testing.T) {
	sq := NewSeqSquare[string](&SeqSquareOptions{
		MaxKeys:    100,
		MaxWaiters: 1000,
	})

	const (
		key         = "0.0"
		numRoutines = 100
		iterations  = 50
	)

	var counter int64
	var wg sync.WaitGroup
	wg.Add(numRoutines)

	start := time.Now()

	for i := range numRoutines {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				unlock, err := sq.Acquire(context.Background(), key)
				if err != nil {
					t.Errorf("Routine %d failed to acquire: %v", id, err)
					return
				}

				current := atomic.AddInt64(&counter, 1)
				if current%10 == 0 {
					time.Sleep(time.Microsecond * 10)
				}
				unlock()
			}
		}(i)
	}

	wg.Wait()
	expected := int64(numRoutines * iterations)
	if counter != expected {
		t.Errorf("Counter mismatch: got %d, want %d", counter, expected)
	}
	t.Logf("Concurrency test passed. Time taken: %v", time.Since(start))
}

func TestSeqSquare_ContextTimeout(t *testing.T) {
	sq := NewSeqSquare[string](&SeqSquareOptions{
		MaxWaiters: 10,
	})

	key := "timeout_key"

	unlock1, _ := sq.Acquire(context.Background(), key)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
	defer cancel()

	_, err := sq.Acquire(ctx, key)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected timeout error, got: %v", err)
	}

	unlock1()

	unlock2, err := sq.Acquire(context.Background(), key)
	if err != nil {
		t.Fatalf("Failed to acquire lock after timeout: %v", err)
	}
	unlock2()
}

func TestSeqSquare_CleanupAndRecount(t *testing.T) {
	opts := &SeqSquareOptions{
		MaxKeys:          10,
		CleanInterval:    time.Second * 1,
		RecountKeysSteps: 2,
	}
	sq := NewSeqSquare[int](opts)

	for i := 0; i < 10; i++ {
		unlock, _ := sq.Acquire(context.Background(), i)
		unlock()
	}

	_, err := sq.Acquire(context.Background(), 100)
	if !errors.Is(err, ErrSeqSquareKeysFull) {
		t.Errorf("Expected KeysFull error, got %v", err)
	}

	t.Log("Waiting for tryclean to sweep items...")
	time.Sleep(time.Second * 4)

	unlock, err := sq.Acquire(context.Background(), 999)
	if err != nil {
		t.Errorf("Should be able to acquire after cleanup, but got: %v", err)
	} else {
		unlock()
	}
}

func BenchmarkSeqSquare_AcquireRelease(b *testing.B) {
	sq := NewSeqSquare[int](nil)
	ctx := b.Context()

	for i := 0; b.Loop(); i++ {
		unlock, _ := sq.Acquire(ctx, i%128)
		unlock()
	}
}

func BenchmarkSeqSquare_HotKey(b *testing.B) {
	sq := NewSeqSquare[string](nil)
	ctx := context.Background()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			unlock, _ := sq.Acquire(ctx, "same_key")
			unlock()
		}
	})
}
