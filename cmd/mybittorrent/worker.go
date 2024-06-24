package main

// TODO: implement worker pool to improve download speed

// type WorkerPool struct {

// 	// The maximum number of goroutines (i.e., workers) maintained by the Worker Pool.
// 	maxWorkers int

// 	// The task queue where tasks are submitted and stored
// 	taskQueue chan func()

// 	// The worker queue from which workers retrieve tasks to execute.
// 	workerQueue chan func()
// 	stoppedChan chan struct{}
// 	stopSignal  chan struct{}

// 	// The waiting queue, used to cache tasks when there are no available workers
// 	// and the current number of workers has reached the maximum limit.
// 	// waitingQueue deque.Deque[func()]
// 	stopLock sync.Mutex
// 	stopOnce sync.Once
// 	stopped  bool
// 	waiting  int32
// 	wait     bool
// }

// func New(maxWorkers int) *WorkerPool {
// 	// There must be at least one worker.
// 	if maxWorkers < 1 {
// 		maxWorkers = 1
// 	}

// 	pool := &WorkerPool{
// 		maxWorkers:  maxWorkers,
// 		taskQueue:   make(chan func()),
// 		workerQueue: make(chan func()),
// 		stopSignal:  make(chan struct{}),
// 		stoppedChan: make(chan struct{}),
// 	}

// 	// Start the task dispatcher.
// 	go pool.dispatch()

// 	return pool
// }

// func (p *WorkerPool) Submit(task func()) {
// 	if task != nil {
// 		p.taskQueue <- task
// 	}
// }

// // blocks and waits for the submitted task to complete execution
// func (p *WorkerPool) SubmitWait(task func()) {
// 	if task == nil {
// 		return
// 	}
// 	doneChan := make(chan struct{})
// 	p.taskQueue <- func() {
// 		task()
// 		close(doneChan)
// 	}
// 	<-doneChan
// }
