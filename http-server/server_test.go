package main

import (
	"net"
	"net/http"
	"os"
	"io/ioutil"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"path/filepath"

	"github.com/stretchr/testify/assert"
)

const TestingFileDirectory = "testing-files"

func TestGetPipelineInput(t *testing.T) {

}

func TestProcessPipeline(t *testing.T) {

}

func TestProcessMessage(t *testing.T) {
	
}

func TestGetLogstashOutput(t *testing.T) {
	correctOutput := "test"
	filePath := filepath.Join(TestingFileDirectory, "output")
	defer os.Remove(filePath)
	err := ioutil.WriteFile(filePath, []byte(correctOutput), 0644)
	if err != nil {
		assert.Fail(
			t,
			"Cannot write test file in 'TestGetLogstashOutput'.",
		)
	}

	output, err := getLogstashOutput(filePath)
	if err != nil {
		assert.Fail(
			t,
			"Cannot read file in 'getLogstashOutput'.",
		)
	}

	assert.Equal(
		t,
		correctOutput,
		output,
		"Function 'getLogstashOutput' returns incorrect output.",
	)
}

func TestSendMessagesToLogstash(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {}))

	testServerPort, err := strconv.Atoi(strings.Split(testServer.URL, ":")[2])
	if err != nil {
		assert.Fail(
			t,
			"Test run fail: cannot get testServer port in 'TestSendMessagesToLogstash'.",
		)
	}

	connection, err := net.ListenUDP("udp", &net.UDPAddr{Port: 8180})
	if err != nil {
		assert.Fail(
			t,
			"Test run fail: cannot connect to port 8180 in 'TestSendMessagesToLogstash'.",
		)
	}
	defer connection.Close()

	err = sendMessagesToLogstash(connection, []string{"test"}, testServerPort)
	if err != nil {
		assert.Fail(
			t,
			"Cannot send message to testServer with 'sendMessagesToLogstash' function.",
		)
	}
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
