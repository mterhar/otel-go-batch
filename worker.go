package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func doSomeJobWork(ctx context.Context, jobNumber int64) error {
	if jobNumber%17 == 0 {
		return errors.New("couldn't start")
	}
	log.Printf("starting job %d \n", jobNumber)

	var tracerWorker = tpWorker.Tracer("example/otel-go-batch")
	var spanWorker trace.Span
	ctx, spanWorker = tracerWorker.Start(ctx, "Worker side: Start job")
	defer spanWorker.End()

	spanWorker.SetAttributes(attribute.Int64("job.number", jobNumber))
	spanWorker.SetAttributes(attribute.String("job.emitted_by", "worker"))
	// If we need to make outbound requests from the job, we need to attach the right context for propagation

	var span trace.Span

	ctx, span = tracerWorker.Start(ctx, "Do http request thing")
	defer span.End()
	span.SetAttributes(attribute.String("job.emitted_by", "worker"))

	httpTarget := "http://localhost"
	span.SetAttributes(attribute.String("job.web_target", httpTarget))
	log.Printf("http requesting %s \n", httpTarget)

	httpRequest, err := http.NewRequest("GET", httpTarget, nil)
	if err != nil {
		fmt.Println(err)
	}

	if jobNumber%36 == 0 {
		span.SetStatus(codes.Error, "error in the middle of the job")
		return errors.New("died in the middle")
	}
	time.Sleep((29 * time.Millisecond))

	// this is how to explicitly send the trace propagation headers.
	// it's probably not needed here since the only context passed in was for the job
	// if scheduler and job trace contexts are both present, this will help
	httpRequest = httpRequest.WithContext(ctx)
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(httpRequest.Header))

	// rather than sending it, we'll just take a look
	httpStr := formatRequest(httpRequest)
	if err != nil {
		fmt.Println(err)
	}

	spanWorker.AddEvent("WebRequest", trace.WithAttributes(attribute.String("request.as_string", httpStr)))
	sleepTime := jobNumber % 10 * 80
	time.Sleep(time.Duration(sleepTime) * time.Millisecond)

	return nil
}

func doSomeLengthyJobWork(ctx context.Context, jobNumber int64) error {
	if jobNumber%17 == 0 {
		return errors.New("couldn't start")
	}
	log.Printf("starting job %d \n", jobNumber)

	var tracerWorker = tpWorker.Tracer("example/otel-go-batch")
	var spanWorker trace.Span
	ctx, spanWorker = tracerWorker.Start(ctx, "Worker side: Start lengthy job")
	defer spanWorker.End()

	spanWorker.SetAttributes(attribute.String("job.emitted_by", "worker"))

	// randomly return error statuses
	if seededRand.Intn(100) < 12 {
		spanWorker.SetStatus(codes.Error, "error in the middle of the job")
		return errors.New("died in the middle")
	}

	loops := seededRand.Intn(24) + 2
	for i := 0; i < loops; i += 1 {
		_, span := tracerWorker.Start(ctx, "Doing some stuff")
		defer span.End()

		span.SetAttributes(attribute.String("job.emitted_by", "worker"))
		span.SetAttributes(attribute.String("worker.loop", fmt.Sprintf("Loop %v of %v", i+1, loops)))
		// This line is included in here because there can be issues.
		// 1. If span is reassigned on the next loop, it won't send the replaced span
		// 2. If the function closes while the goroutine is sleeping, it won't send the last span
		// Doing something like this helps ensure that you're emitting everything you expect

		time.Sleep(time.Duration(seededRand.Intn(300)+30) * time.Millisecond)

		// let these spans finish at some random period.
		go func(s trace.Span) {
			time.Sleep(time.Duration(seededRand.Intn(50)) * time.Millisecond)
			s.End()
		}(span)
	}

	return nil
}

// nowhere in here are we emitting spans.
// the return string array is collecting a list of errors.
func doSomeDetailedJobWork(jobNumber int64) []string {
	// the majority have no errors
	if seededRand.Intn(100) < 50 {
		return nil
	}

	if jobNumber%17 == 0 {
		return []string{"couldn't start"}
	}
	log.Printf("starting job %d \n", jobNumber)

	var errorRecord []string

	// randomly return error statuses
	if seededRand.Intn(100) < 12 {
		errorRecord = append(errorRecord, "problems at the beginning")
	}

	// loop through abunch of tasks
	loops := seededRand.Intn(1000) + 2
	for i := 0; i < loops; i += 1 {
		if seededRand.Intn(100) < 12 {
			errorRecord = append(errorRecord, fmt.Sprintf("error in job %v on task %v", jobNumber, i))
		}
	}

	return errorRecord
}

// formatRequest generates string representation of a request
// from https://medium.com/doing-things-right/pretty-printing-http-requests-in-golang-a918d5aaa000
func formatRequest(r *http.Request) string {
	// Create return string
	var request []string
	// Add the request string
	url := fmt.Sprintf("%v %v %v", r.Method, r.URL, r.Proto)
	request = append(request, url)
	// Add the host
	request = append(request, fmt.Sprintf("Host: %v", r.Host))
	// Loop through headers
	for name, headers := range r.Header {
		name = strings.ToLower(name)
		for _, h := range headers {
			request = append(request, fmt.Sprintf("%v: %v", name, h))
		}
	}

	// If this is a POST, add post data
	if r.Method == "POST" {
		r.ParseForm()
		request = append(request, "\n")
		request = append(request, r.Form.Encode())
	}
	// Return the request as a string
	return strings.Join(request, "\n")
}
