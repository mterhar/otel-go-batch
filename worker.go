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
	// If we need to make outbound requests from the job, we need to attach the right context for propagation

	var span trace.Span

	ctx, span = tracerWorker.Start(ctx, "Do http request thing")
	defer span.End()

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
