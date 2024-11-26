package repository

import (
	"context"
	"fmt"
	"sync"

	"E.E/internal/core/domain"
)

type MemoryRepository struct {
	jobs     map[string]*domain.EncryptionJob
	history  map[string][]domain.JobHistoryEntry
	mu       sync.RWMutex
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		jobs:    make(map[string]*domain.EncryptionJob),
		history: make(map[string][]domain.JobHistoryEntry),
	}
}

func (r *MemoryRepository) Create(ctx context.Context, job *domain.EncryptionJob) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.jobs[job.ID]; exists {
		return fmt.Errorf("job already exists with ID: %s", job.ID)
	}

	r.jobs[job.ID] = job
	return nil
}

func (r *MemoryRepository) Update(ctx context.Context, job *domain.EncryptionJob) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.jobs[job.ID]; !exists {
		return fmt.Errorf("job not found with ID: %s", job.ID)
	}

	r.jobs[job.ID] = job
	return nil
}

func (r *MemoryRepository) Get(ctx context.Context, jobID string) (*domain.EncryptionJob, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	job, exists := r.jobs[jobID]
	if !exists {
		return nil, nil
	}

	return job, nil
}

func (r *MemoryRepository) List(ctx context.Context) ([]*domain.EncryptionJob, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	jobs := make([]*domain.EncryptionJob, 0, len(r.jobs))
	for _, job := range r.jobs {
		jobs = append(jobs, job)
	}
	return jobs, nil
}

func (r *MemoryRepository) Delete(ctx context.Context, jobID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.jobs[jobID]; !exists {
		return fmt.Errorf("job not found with ID: %s", jobID)
	}

	delete(r.jobs, jobID)
	return nil
}

func (r *MemoryRepository) AddJobHistory(ctx context.Context, jobID string, entry domain.JobHistoryEntry) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.history[jobID] = append(r.history[jobID], entry)
	return nil
}

func (r *MemoryRepository) GetJobHistory(ctx context.Context, jobID string) ([]domain.JobHistoryEntry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.history[jobID], nil
}

func (r *MemoryRepository) HealthCheck(ctx context.Context) error {
	return nil // Memory repository is always healthy
}

func (r *MemoryRepository) Close() error {
	return nil // Nothing to close for memory repository
} 