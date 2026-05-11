package worker

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

type Dispatcher struct {
	deliverySvc    deliveryService
	deliveryWorker *DeliveryWorker
	logger         *slog.Logger
	concurrency    int
	pollInterval   time.Duration
	batchSize      int
	cancel         context.CancelFunc
	wg             sync.WaitGroup
}

func NewDispatcher(
	svc deliveryService,
	worker *DeliveryWorker,
	logger *slog.Logger,
	concurrency int,
	pollInterval time.Duration,
	batchSize int,
) *Dispatcher {
	return &Dispatcher{
		deliverySvc:    svc,
		deliveryWorker: worker,
		logger:         logger,
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
	d.wg.Wait()
}

func (d *Dispatcher) loop(ctx context.Context, workerID int) {
	for {
		select {
		case <-ctx.Done():
			d.logger.Info("worker shutting down", slog.Int("worker_id", workerID))
			return
		default:
			if err := d.deliveryWorker.ProcessScheduledAttempts(ctx); err != nil {
				d.logger.Error("error processing scheduled attempts",
					slog.Int("worker_id", workerID),
					slog.String("error", err.Error()),
				)
			}

			deliveries, err := d.deliverySvc.GetPending(ctx, d.batchSize)
			if err != nil {
				d.logger.Error("error fetching pending deliveries",
					slog.Int("worker_id", workerID),
					slog.String("error", err.Error()),
				)
				d.sleep(ctx)
				continue
			}

			if len(deliveries) == 0 {
				d.sleep(ctx)
				continue
			}

			d.logger.Debug("picked up deliveries",
				slog.Int("worker_id", workerID),
				slog.Int("count", len(deliveries)),
			)

			for _, del := range deliveries {
				del := del
				if err := d.deliveryWorker.ProcessFirstAttempt(ctx, &del); err != nil {
					d.logger.Error("delivery failed",
						slog.Int("worker_id", workerID),
						slog.String("delivery_id", del.ID),
						slog.String("error", err.Error()),
					)
				}
			}
		}
	}
}

func (d *Dispatcher) sleep(ctx context.Context) {
	select {
	case <-ctx.Done():
	case <-time.After(d.pollInterval):
	}
}
