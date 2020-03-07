package main

import (
    "io"
    "io/ioutil"
    "net/http"
    "log"
    "encoding/json"
    "bytes"
)

const (
    RequestTypeInvalid                 = -1
    RequestTypeEmptyFilter             = 0
    RequestTypeMessageFilter           = 1
    RequestTypeMessageFilterExpected   = 2
    RequestTypeFilter                  = 3
)

type logstashPipelineInput struct {
    Message     string
    Filter      string
    Expected    string
}

type logstashPipelineOutput struct {
    Output      string      `json:"output"`
    Diff        string      `json:"diff"`
    Lint        string      `json:"lint"`
    Status      string      `json:"status"`
}

func getRequestType(request *http.Request) int {
    multiPartReader, error := request.MultipartReader()
    if error != nil {
        return RequestTypeInvalid
    }

    pipelineData := logstashPipelineInput{}

    for {
        part, error := multiPartReader.NextPart()

        // This is OK, no more parts
        if error == io.EOF {
            break
        }

        // Any other error
        if error != nil {
            return RequestTypeInvalid
        }

        var buffer bytes.Buffer
        io.Copy(&buffer, part)

        switch part.FormName() {
        case "filter":
            pipelineData.Filter = buffer.String()
        case "message":
            pipelineData.Message = buffer.String()
        case "expected":
            pipelineData.Expected = buffer.String()
        default:
            return RequestTypeInvalid
        }
    }

    log.Println(pipelineData.Filter, pipelineData.Message, pipelineData.Expected)

    if pipelineData.Filter == "" {
        return RequestTypeEmptyFilter
    }

    if pipelineData.Message != "" && pipelineData.Expected != "" {
        return RequestTypeMessageFilterExpected
    }

    if pipelineData.Message != "" && pipelineData.Expected == "" {
        return RequestTypeMessageFilter
    }

    return RequestTypeFilter
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
    case RequestTypeInvalid:
        response = logstashPipelineOutput{Status: "Invalid request body"}
    case RequestTypeEmptyFilter:
        response = logstashPipelineOutput{Status: "Empty filter field"}
    case RequestTypeFilter:
        response = logstashPipelineOutput{Status: "Filter"}
    case RequestTypeMessageFilter:
        response = logstashPipelineOutput{Status: "MessageFilter"}
    case RequestTypeMessageFilterExpected:
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
