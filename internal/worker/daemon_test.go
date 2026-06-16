package worker

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/andrejsstepanovs/prowiki/internal/domain"
)

// Mocks

type MockQueue struct {
	mu           sync.Mutex
	jobsToReturn []domain.Job
	FailCalls    []int64
	CompCalls    []int64
}

func (m *MockQueue) ClaimBatch(ctx context.Context, limit int) ([]domain.Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.jobsToReturn) == 0 {
		return nil, nil
	}
	jobs := m.jobsToReturn
	m.jobsToReturn = nil
	return jobs, nil
}
func (m *MockQueue) Complete(ctx context.Context, jobID int64, fn domain.TxFunc) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CompCalls = append(m.CompCalls, jobID)
	return nil
}
func (m *MockQueue) Fail(ctx context.Context, jobID int64, errPayload string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.FailCalls = append(m.FailCalls, jobID)
	return nil
}
func (m *MockQueue) Enqueue(ctx context.Context, jobs ...domain.Job) error { return nil }

func (m *MockQueue) GetCompCalls() []int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	res := make([]int64, len(m.CompCalls))
	copy(res, m.CompCalls)
	return res
}

func (m *MockQueue) GetFailCalls() []int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	res := make([]int64, len(m.FailCalls))
	copy(res, m.FailCalls)
	return res
}

type MockDispatcher struct{}

func (m *MockDispatcher) Dispatch(ctx context.Context, job domain.Job) (domain.TxFunc, error) {
	if job.Stage == "PANIC" {
		panic("boom")
	}
	if job.Stage == "ERR" {
		return nil, errors.New("err")
	}
	return func(tx *sql.Tx) error { return nil }, nil
}

func TestDaemon(t *testing.T) {
	q := &MockQueue{
		jobsToReturn: []domain.Job{
			{ID: 1, Stage: "OK"},
			{ID: 2, Stage: "PANIC"},
			{ID: 3, Stage: "ERR"},
		},
	}
	d := &MockDispatcher{}

	daemon := NewDaemon(q, d, 10*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	go daemon.Start(ctx)

	time.Sleep(50 * time.Millisecond)
	cancel()

	compCalls := q.GetCompCalls()
	failCalls := q.GetFailCalls()

	if len(compCalls) != 1 || compCalls[0] != 1 {
		t.Fatalf("expected job 1 to complete, got %v", compCalls)
	}

	if len(failCalls) != 2 {
		t.Fatalf("expected jobs 2 and 3 to fail, got %v", failCalls)
	}
}
