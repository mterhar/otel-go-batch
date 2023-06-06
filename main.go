package batchscheduler

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func main() {

	// intialization code that starts up, does evaluations, connects to queue
	log.Println("starting scheduler")
	
	// create regular tracer for the scheduler
	if err := initTracer(); err != nil {
		log.Panic(err)
	}
	tracer := tp.Tracer("schedulerStarupTracer")
	ctx := context.Background()
	// defer func() { _ = tp.Shutdown(ctx) }()

	// make a little tree of spans
	var span trace.Span
	ctx, span = tracer.Start(ctx, "Evaluate the queue and environment")
	// defer span.End()

	span.SetAttributes(attribute.Int("job.run", 2001))

	var childSpan trace.Span
	ctx, childSpan = tracer.Start(ctx, "Child span for evaluating the queue")
	childSpan.SetAttributes(attribute.Int("queue.depth", 1000))

	var grandChildSpan trace.Span
	ctx, grandChildSpan = tracer.Start(ctx, "Grandchild span reporting no errors")
	grandChildSpan.SetAttributes(attribute.Bool("errors", false))

	grandChildSpan.End()
	childSpan.End()
	span.End()

	_ = tp.Shutdown(ctx)
	// we now have no active spans on that first trace.
	// the root span was sent so the job is started and a sampling decision can be made.
	// typically the last span comes out when the app ends, but in our case, we don't want that
	// Going forward, we want to treat the spans like metrics, so we'll create a new tracer for that.

	tracer = tp.Tracer("SchedulerRunnerTracer")
	ctx = context.Background() // reset context to get rid of the startup identifiers

	

	ctxWorker := context.Background()

	tpWorker, err := newWorkerTracer()
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = tpWorker.Shutdown(ctxWorker) }()

		
	var spanWorker trace.Span
	ctxWorker, spanWorker = tpWorker.Tracer("startNextJob").Start(ctxWorker, "make outer request")
	defer spanWorker.End()
	// This for loop is our fake job queue.
	for i := 0; i < 1000; i++ {
		// check for a number of iterations, make a new context and spanworker.
		if i%100 == 1 {
			ctxWorker, spanWorker = tpWorker.Tracer("startNextJob").Start(ctxWorker, "make outer request")
		}
	
		doSomeJobWork(ctxWorker, i)
	}

	// make an initial http request
	r, err := http.NewRequest("", "", nil)
	if err != nil {
		panic(err)
	}

	// This is roughly what an instrumented http client does.
	log.Println("The \"make outer request\" span should be recorded, because it is recorded with a Tracer from the SDK TracerProvider")
	// var span trace.Span
	ctx, span = tpWorker.Tracer("example/passthrough/outer").Start(ctx, "make outer request")
	defer span.End()
	r = r.WithContext(ctx)
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(r.Header))

	backendFunc := func(r *http.Request) {
		// This is roughly what an instrumented http server does.
		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		log.Println("The \"handle inner request\" span should be recorded, because it is recorded with a Tracer from the SDK TracerProvider")
		_, span := tp.Tracer("example/passthrough/inner").Start(ctx, "handle inner request")
		defer span.End()

		// Do "backend work"
		time.Sleep(time.Second)
	}
	// This handler will be a passthrough, since we didn't set a global TracerProvider
	passthroughHandler := handler.New(backendFunc)
	passthroughHandler.HandleHTTPReq(r)
}

// This section is for the scheduler's tracer.
var tp *sdktrace.TracerProvider

// initTracer creates and registers trace provider instance.
func initTracer() error {
	exp, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		return fmt.Errorf("failed to initialize stdouttrace exporter: %w", err)
	}
	bsp := sdktrace.NewBatchSpanProcessor(exp)
	tp = sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(bsp),
	)
	otel.SetTracerProvider(tp)
	return nil
}

// the next function is to create a new trace proider for the worker.
func newWorkerTracer() (*sdktrace.TracerProvider, error) {
	// replace with honeycomb exporter?
	exp, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize stdouttrace exporter: %w", err)
	}
	bsp := sdktrace.NewBatchSpanProcessor(exp)
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(bsp),
	)
	return tp, nil
}
