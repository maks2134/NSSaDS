package network

import (
	"NSSaDS/lab4/internal/domain"
	"context"
	"sync"
	"sync/atomic"
	"time"
)

type ThreadPool struct {
	minWorkers      int
	maxWorkers      int
	queueSize       int
	workerTimeout   time.Duration
	expandThreshold float64

	taskQueue   chan func()
	workerQueue chan struct{}
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup

	activeWorkers  int32
	currentWorkers int32
	completedTasks int64
	queuedTasks    int32
}

func NewThreadPool(config *domain.ThreadPoolConfig) domain.ThreadPool {
	ctx, cancel := context.WithCancel(context.Background())

	return &ThreadPool{
		minWorkers:      config.MinWorkers,
		maxWorkers:      config.MaxWorkers,
		queueSize:       config.QueueSize,
		workerTimeout:   config.WorkerTimeout,
		expandThreshold: config.ExpandThreshold,
		taskQueue:       make(chan func(), config.QueueSize),
		workerQueue:     make(chan struct{}, config.MaxWorkers),
		ctx:             ctx,
		cancel:          cancel,
	}
}

func (tp *ThreadPool) Start(ctx context.Context) error {
	for i := 0; i < tp.minWorkers; i++ {
		tp.addWorker()
	}

	go tp.manager()

	return nil
}

func (tp *ThreadPool) Stop() error {
	tp.cancel()
	close(tp.taskQueue)
	tp.wg.Wait()
	return nil
}

func (tp *ThreadPool) Submit(task func()) error {
	select {
	case tp.taskQueue <- task:
		atomic.AddInt32(&tp.queuedTasks, 1)
		tp.checkAndExpand()
		return nil
	case <-tp.ctx.Done():
		return tp.ctx.Err()
	default:
		return domain.ErrQueueFull
	}
}

func (tp *ThreadPool) Stats() *domain.PoolStats {
	return &domain.PoolStats{
		ActiveWorkers:  int(atomic.LoadInt32(&tp.activeWorkers)),
		QueuedTasks:    int(atomic.LoadInt32(&tp.queuedTasks)),
		CompletedTasks: atomic.LoadInt64(&tp.completedTasks),
		MinWorkers:     tp.minWorkers,
		MaxWorkers:     tp.maxWorkers,
		CurrentWorkers: int(atomic.LoadInt32(&tp.currentWorkers)),
	}
}

func (tp *ThreadPool) addWorker() {
	if atomic.LoadInt32(&tp.currentWorkers) >= int32(tp.maxWorkers) {
		return
	}

	atomic.AddInt32(&tp.currentWorkers, 1)
	tp.wg.Add(1)

	go func() {
		defer tp.wg.Done()
		defer atomic.AddInt32(&tp.currentWorkers, -1)

		for {
			select {
			case task, ok := <-tp.taskQueue:
				if !ok {
					return
				}

				atomic.AddInt32(&tp.activeWorkers, 1)
				task()
				atomic.AddInt32(&tp.activeWorkers, -1)
				atomic.AddInt64(&tp.completedTasks, 1)
				atomic.AddInt32(&tp.queuedTasks, -1)

			case <-tp.ctx.Done():
				return
			case <-time.After(tp.workerTimeout):
				if atomic.LoadInt32(&tp.currentWorkers) > int32(tp.minWorkers) {
					return
				}
			}
		}
	}()
}

func (tp *ThreadPool) checkAndExpand() {
	current := atomic.LoadInt32(&tp.currentWorkers)
	queued := atomic.LoadInt32(&tp.queuedTasks)

	if current >= int32(tp.maxWorkers) {
		return
	}

	if float64(queued)/float64(tp.queueSize) > tp.expandThreshold {
		tp.addWorker()
	}
}

func (tp *ThreadPool) manager() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-tp.ctx.Done():
			return
		case <-ticker.C:
			tp.optimize()
		}
	}
}

func (tp *ThreadPool) optimize() {
	current := atomic.LoadInt32(&tp.currentWorkers)
	active := atomic.LoadInt32(&tp.activeWorkers)
	queued := atomic.LoadInt32(&tp.queuedTasks)

	if current > int32(tp.minWorkers) &&
		active < current/2 &&
		queued == 0 {
		atomic.AddInt32(&tp.currentWorkers, -1)
		tp.workerQueue <- struct{}{}
	}
}
