package concurrencyUtils

import (
	"context"
	"log"
)

func OrChannel(cancelSignal ...<-chan struct{}) <-chan struct{} {
	switch len(cancelSignal) {
	case 0:
		return nil
	case 1:
		return cancelSignal[0]
	}

	finalCancelSignal := make(chan struct{})
	go func() {
		defer close(finalCancelSignal)
		switch len(cancelSignal) {
		case 2:
			select {
			case <-cancelSignal[0]:
			case <-cancelSignal[1]:
			}
		default:
			select {
			case <-cancelSignal[0]:
			case <-cancelSignal[1]:
			case <-cancelSignal[2]:
			case <-OrChannel(append(cancelSignal[3:], finalCancelSignal)...):
			}
		}
	}()

	return finalCancelSignal
}

func OrDone[T any](done <-chan struct{}, channel <-chan T) <-chan T {
	valStream := make(chan T)
	go func() {
		defer close(valStream)
		for {
			select {
			case <-done:
				return
			case val, ok := <-channel:
				if !ok {
					return
				}
				select {
				case <-done:
					return
				case valStream <- val:
				}
			}
		}
	}()

	return valStream
}

func Bridge[T any](ctx context.Context, channels <-chan <-chan T) <-chan T {
	valStream := make(chan T)
	go func() {
		defer close(valStream)
		for {
			var stream <-chan T
			select {
			case <-ctx.Done():
				return
			case subChannels, ok := <-channels:
				if !ok {
					return
				}
				stream = subChannels
			}

			for val := range OrDone(ctx.Done(), stream) {
				select {
				case <-ctx.Done():
					return
				case valStream <- val:
				}
			}
		}
	}()

	return valStream
}

type ChanStruct[U any] struct {
	value U
	err   error
}

func (p ChanStruct[U]) GetError() error {
	return p.err
}

func (p ChanStruct[U]) GetResult() (U, error) {
	if p.err != nil {
		var nilReuslt U
		return nilReuslt, p.err
	}

	return p.value, nil
}

func Pipeline[T, U any](ctx context.Context, inStream <-chan T, fn func(ctx context.Context, input T) (U, error)) <-chan ChanStruct[U] {
	outStream := make(chan ChanStruct[U])
	go func() {
		defer close(outStream)

		for val := range inStream {
			select {
			case <-ctx.Done():
				return
			default:
			}

			result, err := fn(ctx, val)
			select {
			case <-ctx.Done():
				return
			case outStream <- ChanStruct[U]{value: result, err: err}:
			}
		}
	}()

	return outStream
}

func primeFinder(ctx context.Context, intStream <-chan int) <-chan int {
	primeStream := make(chan int)
	go func() {
		defer close(primeStream)
		for integer := range intStream {
			prime := true
			for i := 2; i < integer; i++ {
				if integer%i == 0 {
					prime = false
					break
				}
			}

			if prime {
				select {
				case <-ctx.Done():
					return
				case primeStream <- integer:
				}
			}
		}
	}()
	return primeStream
}

func toInt(ctx context.Context, valueStream <-chan any) <-chan int {
	intStream := make(chan int)

	go func() {
		defer close(intStream)
		for v := range valueStream {
			select {
			case <-ctx.Done():
				return
			case intStream <- v.(int):
			}
		}
	}()

	return intStream
}

func repeatFn(ctx context.Context, fn func() any) <-chan any {
	valueStream := make(chan any)

	go func() {
		defer close(valueStream)
		for {
			select {
			case <-ctx.Done():
				return
			case valueStream <- fn():
			}
		}
	}()

	return valueStream
}

func take(ctx context.Context, valueStream <-chan string, num int) <-chan string {
	takeStream := make(chan string)

	go func() {
		defer close(takeStream)

		for i := num; i > 0 || i == -1; {
			if i != -1 {
				i--
			}
			select {
			case <-ctx.Done():
				return
			case takeStream <- <-valueStream:
			}
		}
	}()

	return takeStream
}

type Worker struct {
	msgChan  <-chan []byte
	handler  func(ctx context.Context, payload []byte) error
	quitChan chan struct{}
}

func NewWorker(msgChan <-chan []byte, handler func(ctx context.Context, payload []byte) error) *Worker {
	return &Worker{
		msgChan:  msgChan,
		handler:  handler,
		quitChan: make(chan struct{}),
	}
}

func (w *Worker) Start(ctx context.Context) {
	go func() {
		combinedDone := OrChannel(ctx.Done(), w.quitChan)

		for payload := range OrDone(combinedDone, w.msgChan) {
			if err := w.handler(ctx, payload); err != nil {
				log.Printf("[Worker] Handler error: %v", err)
			}
		}

		log.Println("[Worker] Shutting down cleanly!")

	}()
}

func (w *Worker) Stop() {
	close(w.quitChan)
}