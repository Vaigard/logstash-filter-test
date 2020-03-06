package main

import (
    "io"
    "io/ioutil"
    "net/http"
    "log"
    "encoding/json"
)

const (
    RequestTypeInvalid                 = -1
    RequestTypeEmptyFilter             = 0
    RequestTypeMessageFilter           = 1
    RequestTypeMessageFilterExpected   = 2
    RequestTypeFilter                  = 3
)

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

func getRequestType(request *http.Request) int {
    requestBody, err := ioutil.ReadAll(request.Body)
    defer request.Body.Close()
    if err != nil {
        return RequestTypeInvalid
    }

    log.Print("UNMARSHAL\n")

    requestBodyJson := logstashPipelineInput{}
    err = json.Unmarshal(requestBody, &requestBodyJson)
    if err != nil {
        log.Printf(err.Error())
        return RequestTypeInvalid
    }

    log.Print("REQUEST\n")
    log.Print(requestBodyJson)

    if requestBodyJson.Filter == "" {
        return RequestTypeEmptyFilter
    }

    if requestBodyJson.Message != "" && requestBodyJson.Expected != "" {
        return RequestTypeMessageFilterExpected
    }

    if requestBodyJson.Message != "" && requestBodyJson.Expected == "" {
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
