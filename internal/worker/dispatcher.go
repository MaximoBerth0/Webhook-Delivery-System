package worker

import (
	"context"
	"log"
	"sync"
	"time"
)

type Dispatcher struct {
	deliverySvc    deliveryService
	deliveryWorker *DeliveryWorker
	concurrency    int
	pollInterval   time.Duration
	batchSize      int
	cancel         context.CancelFunc
	wg             sync.WaitGroup
}

func NewDispatcher(
	svc deliveryService,
	worker *DeliveryWorker,
	concurrency int,
	pollInterval time.Duration,
	batchSize int,
) *Dispatcher {
	return &Dispatcher{
		deliverySvc:    svc,
		deliveryWorker: worker,
		concurrency:    concurrency,
		pollInterval:   pollInterval,
		batchSize:      batchSize,
	}
}

func (d *Dispatcher) Run(ctx context.Context) {
	ctx, d.cancel = context.WithCancel(ctx)

	for i := 0; i < d.concurrency; i++ {
		d.wg.Add(1)
		go func(workerID int) {
			defer d.wg.Done()
			d.loop(ctx, workerID)
		}(i)
	}
}

func (d *Dispatcher) Stop() {
	if d.cancel != nil {
		d.cancel()
	}
	d.wg.Wait() // blocks until all goroutines finish cleanly
}

func (d *Dispatcher) loop(ctx context.Context, workerID int) {
	for {
		select {
		case <-ctx.Done():
			log.Printf("worker %d shutting down", workerID)
			return
		default:
			deliveries, err := d.deliverySvc.GetPending(ctx, d.batchSize)
			if err != nil {
				log.Printf("worker %d: error fetching pending: %v", workerID, err)
				d.sleep(ctx)
				continue
			}

			if len(deliveries) == 0 {
				d.sleep(ctx)
				continue
			}

			log.Printf("worker %d: picked up %d deliveries", workerID, len(deliveries))

			for _, del := range deliveries {
				del := del
				d.deliveryWorker.processDelivery(ctx, &del)
			}
		}
	}
}

// sleep respects context cancellation so shutdown is immediate
func (d *Dispatcher) sleep(ctx context.Context) {
	select {
	case <-ctx.Done():
	case <-time.After(d.pollInterval):
	}
}
