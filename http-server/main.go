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

// "/"
// Main page returns documentation about server.
func mainHandler(responseWriter http.ResponseWriter, request *http.Request) {
    readmeContent, err := ioutil.ReadFile("README.md")
    if err == nil {
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
    response := logstashPipelineOutput{Output: "sample output"}

    responseJson, err := json.Marshal(response)
    if err != nil {
        http.Error(responseWriter, err.Error(), http.StatusInternalServerError)
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
