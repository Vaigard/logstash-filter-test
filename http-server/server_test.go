package main

import (
	// "encoding/json"
	// "fmt"
	// "net/http"
	// "net/http/httptest"
	// "net/url"
	// "strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetPipelineInput(t *testing.T) {

}

func TestProcessPipeline(t *testing.T) {

}

func TestProcessMessage(t *testing.T) {
	
}

func TestGetLogstashOutput(t *testing.T) {

}

func TestSendMessagesToLogstash(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
	}))

	assert.Fail(
		t,
		testServer.URL,
	)
}

func TestChangePatternsDirs(t *testing.T) {
	pipelineInput := logstashPipelineInput{
		Message: "message",
		Filter: "qwe asd qwe zxc",
		Pattern: "pattern",
		PatternsDirs: "asd,zxc",
	}

	actualFilter := "qwe qwe qwe qwe"
	patternsDirectory := "qwe"

	changePatternsDirs(&pipelineInput, patternsDirectory)

	assert.Equal(
		t,
		actualFilter,
		pipelineInput.Filter,
		"Function 'changePatternsDirs' make incorrect Filter value.",
	)
}
