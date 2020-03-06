package main

import (
    "io"
    "io/ioutil"
    "net/http"
    "log"
    "encoding/json"
)

type RequestTypes struct {
    Invalid                  int   -1
    EmptyFilter              int    0
    MessageFilter            int    1
    MessageFilterExpected    int    2
    Filter                   int    3
}

type logstashPipelineInput struct {
    Message     string      `json:"message"`
    Filter      string      `json:"filter"`
    Expected    string      `json:"expected"`
}

type logstashPipelineOutput struct {
    Output      string      `json:"output"`
    Diff        string      `json:"diff"`
    Lint        string      `json:"lint"`
    Status      string      `json:"status"`
}

func getRequestType(request *http.Request) (requestType int) {
    requestBody interface{}
    requestType = RequestTypes.Filter
    if err := json.NewDecoder(request.Body).Decode(requestBody); err != nil {
        requestType = RequestTypes.Invalid
    }

    if requestBody.Filter == nil {
        requestType = RequestTypes.EmptyFilter
    }

    if requestBody.Message != nil and requestBody.Expected != nil {
        requestType = RequestTypes.MessageFilterExpected
    }

    if requestBody.Message != nil and requestBody.Expected == nil {
        requestType = RequestTypes.MessageFilter
    }

    return requestType
}

// "/"
// Main page returns documentation about server.
func mainHandler(responseWriter http.ResponseWriter, request *http.Request) {
    readmeContent, error := ioutil.ReadFile("README.md")
    if error == nil {
        io.WriteString(responseWriter, string(readmeContent))
    } else {
        io.WriteString(responseWriter, "Logstash filters tester's server\n")
    }
}

// "/ping"
// Allow simply check if server works.
func pingHandler(responseWriter http.ResponseWriter, request *http.Request) {
    io.WriteString(responseWriter, "pong")
}

// "/upload"
// Gets the logstash filter and testing data.
func logstashPipelineHandler(responseWriter http.ResponseWriter, request *http.Request) {
    response := logstashPipelineOutput{}

    switch requestType := getRequestType(request); requestType {
    case RequestTypes.Invalid:
        response = logstashPipelineOutput{Status: "Invalid request body"}
    case RequestTypes.EmptyFilter:
        response = logstashPipelineOutput{Status: "Empty filter field"}
    case RequestTypes.Filter:
        response = logstashPipelineOutput{Status: "Filter"}
    case RequestTypes.MessageFilter:
        response = logstashPipelineOutput{Status: "MessageFilter"}
    case RequestTypes.MessageFilterExpected:
        response = logstashPipelineOutput{Status: "MessageFilterExpected"}
    default:
        response = logstashPipelineOutput{Status: "Internal server error"}
    }

    responseJson, marshalError := json.Marshal(response)
    if marshalError != nil {
        http.Error(responseWriter, marshalError.Error(), http.StatusInternalServerError)
        return
    }

    responseWriter.Header().Set("Content-Type", "application/json")
    responseWriter.Write(responseJson)
}

func main() {
    http.HandleFunc("/", mainHandler)
    http.HandleFunc("/ping", pingHandler)
    http.HandleFunc("/upload", logstashPipelineHandler)

    log.Fatal(http.ListenAndServe(":8081", nil))
}
