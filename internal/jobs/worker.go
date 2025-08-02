package jobs

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"korus/internal/database"
)

type Worker struct {
	id       int
	queue    *Queue
	handlers map[string]JobHandler
	stopCh   chan struct{}
	wg       *sync.WaitGroup
}

type JobHandler interface {
	Handle(ctx context.Context, job *Job) error
}

type JobHandlerFunc func(ctx context.Context, job *Job) error

func (f JobHandlerFunc) Handle(ctx context.Context, job *Job) error {
	return f(ctx, job)
}

type WorkerPool struct {
	workers     []*Worker
	queue       *Queue
	handlers    map[string]JobHandler
	workerCount int
	stopCh      chan struct{}
	wg          sync.WaitGroup
}

func NewWorker(id int, queue *Queue) *Worker {
	return &Worker{
		id:       id,
		queue:    queue,
		handlers: make(map[string]JobHandler),
		stopCh:   make(chan struct{}),
		wg:       &sync.WaitGroup{},
	}
}

func NewWorkerPool(workerCount int, db *database.DB) *WorkerPool {
	queue := NewQueue(db)
	
	return &WorkerPool{
		workers:     make([]*Worker, workerCount),
		queue:       queue,
		handlers:    make(map[string]JobHandler),
		workerCount: workerCount,
		stopCh:      make(chan struct{}),
	}
}

func (w *Worker) RegisterHandler(jobType string, handler JobHandler) {
	w.handlers[jobType] = handler
}

func (w *Worker) Start(ctx context.Context) {
	w.wg.Add(1)
	go w.run(ctx)
}

func (w *Worker) Stop() {
	close(w.stopCh)
	w.wg.Wait()
}

func (w *Worker) run(ctx context.Context) {
	defer w.wg.Done()
	
	log.Printf("Worker %d started", w.id)
	defer log.Printf("Worker %d stopped", w.id)

	ticker := time.NewTicker(5 * time.Second) // Check for jobs every 5 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		case <-ticker.C:
			if err := w.processJob(ctx); err != nil {
				log.Printf("Worker %d error processing job: %v", w.id, err)
			}
		}
	}
}

func (w *Worker) processJob(ctx context.Context) error {
	// Get supported job types
	var jobTypes []string
	for jobType := range w.handlers {
		jobTypes = append(jobTypes, jobType)
	}

	if len(jobTypes) == 0 {
		return nil // No handlers registered
	}

	// Dequeue job
	job, err := w.queue.Dequeue(ctx, jobTypes)
	if err != nil {
		return fmt.Errorf("failed to dequeue job: %w", err)
	}

	if job == nil {
		return nil // No jobs available
	}

	log.Printf("Worker %d processing job %d (type: %s)", w.id, job.ID, job.JobType)

	// Get handler
	handler, exists := w.handlers[job.JobType]
	if !exists {
		return w.queue.Fail(ctx, job.ID, fmt.Sprintf("no handler for job type: %s", job.JobType))
	}

	// Process job with timeout
	jobCtx, cancel := context.WithTimeout(ctx, 30*time.Minute) // 30 minute timeout
	defer cancel()

	start := time.Now()
	err = handler.Handle(jobCtx, job)
	duration := time.Since(start)

	if err != nil {
		log.Printf("Worker %d job %d failed after %v: %v", w.id, job.ID, duration, err)
		
		// Retry job up to 3 times
		if job.Attempts < 3 {
			return w.queue.Retry(ctx, job.ID, 3)
		}
		
		return w.queue.Fail(ctx, job.ID, err.Error())
	}

	log.Printf("Worker %d job %d completed in %v", w.id, job.ID, duration)
	return w.queue.Complete(ctx, job.ID)
}

func (wp *WorkerPool) RegisterHandler(jobType string, handler JobHandler) {
	wp.handlers[jobType] = handler
}

func (wp *WorkerPool) Start(ctx context.Context) error {
	log.Printf("Starting worker pool with %d workers", wp.workerCount)

	// Create and start workers
	for i := 0; i < wp.workerCount; i++ {
		worker := NewWorker(i+1, wp.queue)
		
		// Register all handlers with each worker
		for jobType, handler := range wp.handlers {
			worker.RegisterHandler(jobType, handler)
		}
		
		wp.workers[i] = worker
		worker.Start(ctx)
	}

	// Start cleanup routine
	wp.wg.Add(1)
	go wp.cleanupRoutine(ctx)

	return nil
}

func (wp *WorkerPool) Stop() {
	log.Println("Stopping worker pool...")
	
	close(wp.stopCh)

	// Stop all workers
	for _, worker := range wp.workers {
		worker.Stop()
	}

	wp.wg.Wait()
	log.Println("Worker pool stopped")
}

func (wp *WorkerPool) EnqueueJob(ctx context.Context, jobType string, payload interface{}) (*Job, error) {
	return wp.queue.Enqueue(ctx, jobType, payload)
}

func (wp *WorkerPool) GetQueue() *Queue {
	return wp.queue
}

func (wp *WorkerPool) cleanupRoutine(ctx context.Context) {
	defer wp.wg.Done()
	
	ticker := time.NewTicker(1 * time.Hour) // Cleanup every hour
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-wp.stopCh:
			return
		case <-ticker.C:
			// Cleanup completed jobs older than 24 hours
			olderThan := time.Now().Add(-24 * time.Hour)
			count, err := wp.queue.CleanupCompletedJobs(ctx, olderThan)
			if err != nil {
				log.Printf("Failed to cleanup completed jobs: %v", err)
			} else if count > 0 {
				log.Printf("Cleaned up %d completed jobs", count)
			}
		}
	}
}

// Notification system using PostgreSQL LISTEN/NOTIFY
func (wp *WorkerPool) StartNotificationListener(ctx context.Context, db *database.DB) error {
	wp.wg.Add(1)
	go func() {
		defer wp.wg.Done()
		
		if err := wp.listenForJobNotifications(ctx, db); err != nil {
			log.Printf("Notification listener error: %v", err)
		}
	}()
	
	return nil
}

func (wp *WorkerPool) listenForJobNotifications(ctx context.Context, db *database.DB) error {
	conn, err := db.Pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection: %w", err)
	}
	defer conn.Release()

	// Listen for job notifications
	_, err = conn.Exec(ctx, "LISTEN new_job")
	if err != nil {
		return fmt.Errorf("failed to listen for notifications: %w", err)
	}

	log.Println("Listening for job notifications...")

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-wp.stopCh:
			return nil
		default:
			// Wait for notification with timeout
			notification, err := conn.Conn().WaitForNotification(ctx)
			if err != nil {
				// Check if context was cancelled
				if ctx.Err() != nil {
					return nil
				}
				log.Printf("Error waiting for notification: %v", err)
				continue
			}

			if notification.Channel == "new_job" {
				log.Printf("Received job notification: %s", notification.Payload)
				// Notification received, workers will pick up jobs on their next tick
			}
		}
	}
}

// Helper function to trigger job notifications
func (wp *WorkerPool) NotifyNewJob(ctx context.Context, db *database.DB, jobType string) error {
	_, err := db.ExecContext(ctx, "NOTIFY new_job, $1", jobType)
	return err
}