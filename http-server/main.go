package main

import (
    "io"
    "io/ioutil"
    "net/http"
    "log"
    "encoding/json"
)

type logstashPipelineInput struct {
    Message     string      `json:"message"`
    Filter      string      `json:"filter"`
}

type logstashPipelineOutput struct {
    Output      string      `json:"output"`
}

func checkRequest(request *http.Request) (error int) {
    error = 0
    return error
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
    checkRequestError := checkRequest(request)

    response := logstashPipelineOutput{}

    if checkRequestError == 0 {
        response = logstashPipelineOutput{Output: "correct request"}
    } else {
        response = logstashPipelineOutput{Output: "bad request"}
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
