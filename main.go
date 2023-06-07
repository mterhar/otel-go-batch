package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func main() {
	jobIdentifier := String(12)
	// intialization code that starts up, does evaluations, connects to queue
	log.Println("starting scheduler: " + jobIdentifier)

	// create regular tracer for the scheduler
	ctx, err := initTracer()
	if err != nil {
		log.Panic(err)
	}
	tracer := tp.Tracer("example/otel-go-batch")
	defer func() { _ = tp.Shutdown(ctx) }()

	// make a little tree of spans
	var span trace.Span
	ctx, span = tracer.Start(ctx, "Evaluate the queue and environment")
	// Since we are going to manually kill this span, I don't know if we want to do the normal defer
	// If the app is killed, it won't send.
	defer span.End()

	span.SetAttributes(attribute.String("job.run", jobIdentifier))
	span.SetAttributes(attribute.String("job.emitted_by", "scheduler"))

	var childSpan trace.Span
	ctx2, childSpan := tracer.Start(ctx, "Child span for evaluating the queue")
	childSpan.SetAttributes(attribute.Int("queue.depth", 1000))
	childSpan.SetAttributes(attribute.String("job.emitted_by", "scheduler"))

	var grandChildSpan trace.Span
	_, grandChildSpan = tracer.Start(ctx2, "Grandchild span reporting no errors")
	grandChildSpan.SetAttributes(attribute.Bool("errors", false))
	grandChildSpan.SetAttributes(attribute.String("job.emitted_by", "scheduler"))

	grandChildSpan.End()
	childSpan.End()
	// startupTraceSpanLink := trace.LinkFromContext(ctx, ))
	startupTraceSpanLink := trace.Link{
		SpanContext: span.SpanContext(),
		Attributes: []attribute.KeyValue{
			attribute.String("name", "Link to job start"),
			attribute.String("job.run", jobIdentifier),
			attribute.String("job.emitted_by", "scheduler"),
		},
	}
	fmt.Printf("Startup trace span link: %#v \n", startupTraceSpanLink)

	var spanWorker trace.Span
	// This for loop is our fake job queue.
	var i = 1
	for ; i <= 100; i++ {
		ctx, spanWorker = tp.Tracer("example/otel-go-batch").Start(ctx, "Job started") //, trace.WithLinks(startupTraceSpanLink))
		defer spanWorker.End()
		spanWorker.SetAttributes(attribute.Int("job.number", i))
		spanWorker.SetAttributes(attribute.String("job.emitted_by", "scheduler"))

		err := doSomeJobWork(ctx, int64(i))
		if err != nil {
			spanWorker.SetStatus(codes.Error, "Job failed for some reason")
		}
	}

	// do some job cleanup stuff?
	log.Println("Done with the batch jobs")
}

// This section is for the scheduler's tracer.
var tp *sdktrace.TracerProvider

// initTracer creates and registers trace provider instance.
func initTracer() (context.Context, error) {
	ctx := context.Background()
	client := otlptracegrpc.NewClient()
	exp, err := otlptrace.New(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize grpc exporter: %w", err)
	}
	bsp := sdktrace.NewBatchSpanProcessor(exp)
	tp = sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(bsp),
	)
	otel.SetTracerProvider(tp)
	return ctx, nil
}

// helper functions unrelated to otel

// Make a random string for job identifier
// from https://www.calhoun.io/creating-random-strings-in-go/
const charset = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

func StringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func String(length int) string {
	return StringWithCharset(length, charset)
}
