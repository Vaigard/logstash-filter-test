package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProcessPipeline(t *testing.T) {
	correctOutput := "output"
	filePath := "output"

	err := ioutil.WriteFile(filePath, []byte(correctOutput), 0644)
	if err != nil {
		assert.Fail(
			t,
			"Test run fail: cannot write output file in 'TestProcessPipeline'.",
		)
	}

	pipelineInput := logstashPipelineInput{
		Message:      "message",
		Filter:       "filter",
		Patterns:     "pattern",
		PatternsDirs: "dirs",
	}

	output, err := processPipeline(pipelineInput, filePath, 8181)
	if err != nil {
		assert.Fail(
			t,
			fmt.Sprintf("Function 'processPipeline' returns error: %s", err.Error()),
		)
	}

	assert.Equal(
		t,
		correctOutput,
		output,
		"Function 'processPipeline' returns incorrect output.",
	)
}

func TestProcessMessage(t *testing.T) {
	err := processMessage("test1\ntest2", 8181)
	if err != nil {
		assert.Fail(
			t,
			"Cannot send message to testServer with 'processMessage' function.",
		)
	}
}

func TestGetLogstashOutput(t *testing.T) {
	correctOutput := "test"
	filePath := "output"
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
	connection, err := net.ListenUDP("udp", &net.UDPAddr{Port: 8180})
	if err != nil {
		assert.Fail(
			t,
			"Test run fail: cannot connect to port 8180 in 'TestSendMessagesToLogstash'.",
		)
	}
	defer connection.Close()

	err = sendMessagesToLogstash(connection, []string{"test"}, 8181)
	if err != nil {
		assert.Fail(
			t,
			"Cannot send message to testServer with 'sendMessagesToLogstash' function.",
		)
	}
}

func TestChangePatternsDirs(t *testing.T) {
	pipelineInput := logstashPipelineInput{
		Message:      "message",
		Filter:       "qwe asd qwe zxc",
		Patterns:     "pattern",
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
