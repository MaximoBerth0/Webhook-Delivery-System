package worker

import (
	"context"
	"log/slog"
	"sync"
	"time"
	"webhook-delivery-system/internal/delivery"
	"webhook-delivery-system/internal/infrastructure/config"
)

type Dispatcher struct {
	deliverySvc    deliveryService
	deliveryWorker *DeliveryWorker
	logger         *slog.Logger
	config         config.WorkerConfig
	cancel         context.CancelFunc
	wg             sync.WaitGroup
}

func NewDispatcher(
	svc deliveryService,
	worker *DeliveryWorker,
	logger *slog.Logger,
	cfg config.WorkerConfig,
) *Dispatcher {
	return &Dispatcher{
		deliverySvc:    svc,
		deliveryWorker: worker,
		logger:         logger,
		config:         cfg,
	}
}

func (d *Dispatcher) Run(ctx context.Context) {
	ctx, d.cancel = context.WithCancel(ctx)

	// job channel (buffered for performance)
	jobChan := make(chan delivery.Delivery, d.config.BatchSize*2)

	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		defer close(jobChan)
		d.fetchLoop(ctx, jobChan)
	}()

	for i := 0; i < d.config.Concurrency; i++ {
		d.wg.Add(1)
		go func(workerID int) {
			defer d.wg.Done()
			d.workerLoop(ctx, workerID, jobChan)
		}(i)
	}
}

func (d *Dispatcher) Stop() {
	if d.cancel != nil {
		d.cancel()
	}
	d.wg.Wait()
}

func (d *Dispatcher) fetchLoop(ctx context.Context, jobChan chan<- delivery.Delivery) {
	for {
		select {
		case <-ctx.Done():
			d.logger.Info("fetcher shutting down")
			return
		default:
			// handle scheduled attempts
			if err := d.deliveryWorker.ProcessScheduledAttempts(ctx); err != nil {
				d.logger.Error("error processing scheduled attempts",
					slog.String("error", err.Error()))
			}

			// Fetch pending deliveries
			deliveries, err := d.deliverySvc.GetPending(ctx, d.config.BatchSize)
			if err != nil {
				d.logger.Error("error fetching pending deliveries",
					slog.String("error", err.Error()))
				d.sleep(ctx)
				continue
			}

			if len(deliveries) == 0 {
				d.sleep(ctx)
				continue
			}

			d.logger.Debug("fetched deliveries batch",
				slog.Int("count", len(deliveries)))

			// push each delivery to the channel
			for _, del := range deliveries {
				select {
				case jobChan <- del:
				case <-ctx.Done():
					return
				}
			}
		}
	}
}

// consumes from channel, processes jobs
func (d *Dispatcher) workerLoop(ctx context.Context, workerID int, jobChan <-chan delivery.Delivery) {
	d.logger.Info("worker started", slog.Int("worker_id", workerID))

	for {
		select {
		case <-ctx.Done():
			d.logger.Info("worker shutting down", slog.Int("worker_id", workerID))
			return

		case del, ok := <-jobChan:
			if !ok {
				// Channel closed, no more jobs
				d.logger.Info("worker finished (channel closed)", slog.Int("worker_id", workerID))
				return
			}

			d.logger.Debug("processing delivery",
				slog.Int("worker_id", workerID),
				slog.String("delivery_id", del.ID))

			if err := d.deliveryWorker.ProcessFirstAttempt(ctx, &del); err != nil {
				d.logger.Error("delivery failed",
					slog.Int("worker_id", workerID),
					slog.String("delivery_id", del.ID),
					slog.String("error", err.Error()))
			}
		}
	}
}

func (d *Dispatcher) sleep(ctx context.Context) {
	select {
	case <-ctx.Done():
	case <-time.After(d.config.PollInterval):
	}
}
