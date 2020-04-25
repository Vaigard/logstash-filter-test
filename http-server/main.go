package main

import (
    "io"
    "io/ioutil"
    "net/http"
    "net"
    "log"
    "bytes"
    "time"
    "os"
    "strings"
    "fmt"
    "encoding/json"
)

const (
    RequestTypeCorrect          = 0
    RequestTypeInvalid          = 1
    RequestTypeEmptyFilter      = 2
    RequestTypeEmptyMessage     = 3
)

const (
    LogstashPlainInputPort      = 8082
    LogstashJsonInputPort       = 8083
    ServerPort                  = ":8081"
    ServerLogPath               = "server.log"
    ReadmeFile                  = "README.md"
    FilterFilePath              = "/usr/share/logstash/pipeline/filter.conf"
    OutputFilePath              = "/usr/share/logstash/output.json"
    PatternsDirectory           = "/usr/share/logstash/patterns"
)

type logstashPipelineInput struct {
    Message     string
    Filter      string
}

func main() {
    http.HandleFunc("/", mainHandler)
    http.HandleFunc("/ping", pingHandler)
    http.HandleFunc("/upload", logstashPipelineHandler)

    logFile, _ := os.OpenFile(ServerLogPath, os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
    defer logFile.Close()

    log.SetOutput(logFile)
    log.SetFlags(log.LstdFlags | log.Lshortfile)
    log.Print("Start server--------------")

    os.Remove(OutputFilePath)

    log.Fatal(http.ListenAndServe(ServerPort, nil))
}

// "/"
// Main page returns documentation about server.
func mainHandler(responseWriter http.ResponseWriter, request *http.Request) {
    documentation, error := ioutil.ReadFile(ReadmeFile)
    if error == nil {
        io.WriteString(responseWriter, string(documentation))
    } else {
        log.Printf("Cannot read Readme file: %s", error.Error())
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
    log.Print("Got new request")

    pipelineInput, requestType := getPipelineInput(request)

    var pipelineOutput string

    switch requestType {
    case RequestTypeInvalid:
        pipelineOutput = "{\"Error\": \"Invalid request\"}"
    case RequestTypeEmptyFilter:
        pipelineOutput = "{\"Error\": \"Empty filter\"}"
    case RequestTypeEmptyMessage:
        pipelineOutput = "{\"Error\": \"Empty message\"}"
    default:
        pipelineOutput = processPipeline(pipelineInput)
    }

    responseWriter.Header().Set("Content-Type", "application/json")
    responseWriter.Write([]byte(pipelineOutput))
}

func getPipelineInput(request *http.Request) (logstashPipelineInput, int) {
    pipelineInput := logstashPipelineInput{}
    multiPartReader, error := request.MultipartReader()
    if error != nil {
        log.Printf("Parse request: %s", error.Error())
        return pipelineInput, RequestTypeInvalid
    }

    for {
        part, error := multiPartReader.NextPart()

        // This is OK, no more parts
        if error == io.EOF {
            break
        }

        // Any other error
        if error != nil {
            log.Printf("Get new request part: %s", error.Error())
            return pipelineInput, RequestTypeInvalid
        }

        var buffer bytes.Buffer
        io.Copy(&buffer, part)

        switch part.FormName() {
        case "filter":
            pipelineInput.Filter = buffer.String()
        case "message":
            pipelineInput.Message = buffer.String()
        default:
            return pipelineInput, RequestTypeInvalid
        }
    }

    if pipelineInput.Filter == "" {
        return pipelineInput, RequestTypeEmptyFilter
    }

    if pipelineInput.Message == "" {
        return pipelineInput, RequestTypeEmptyMessage
    }

    return pipelineInput, RequestTypeCorrect
}

func processPipeline(pipelineInput logstashPipelineInput) string {
    message := pipelineInput.Message
    filter := pipelineInput.Filter
    log.Printf("Process new filter and message")
    error := ioutil.WriteFile(FilterFilePath, []byte(filter), 0644)
    if error != nil {
        errorMessage := "Cannot write filter: " + error.Error()
        log.Print(errorMessage)
        return fmt.Sprintf("{\"Error\": \"%s\"}", errorMessage)
    }

    defer ioutil.WriteFile(FilterFilePath, []byte("filter{}\n"), 0644)

    // wait for restart pipeline (autoreload in 2 seconds)
    time.Sleep(5 * 1000 * time.Millisecond)

    error = processMessage(message)    

    if error != nil {
        errorMessage := "Cannot send message to Logstash: " + error.Error()
        log.Print(errorMessage)
        return fmt.Sprintf("{\"Error\": \"%s\"}", errorMessage)
    }

    time.Sleep(5 * 1000 * time.Millisecond)

    defer os.Remove(OutputFilePath)

    output := getLogstashOutput()  

    return output
}

func processMessage(message string) error {
    messages := strings.Split(message, "\n")

    port := LogstashPlainInputPort

    if json.Valid([]byte(messages[0])) {
        log.Print("Here is JSON messages")
        port = LogstashJsonInputPort
    }

    connection, error := net.ListenUDP("udp", &net.UDPAddr{Port: 1234})
    if error != nil {
        errorMessage := "Cannot connect to port 1234/udp: " + error.Error()
        log.Print(errorMessage)
        return error
    }
    defer connection.Close()

    for try := 0; try < 3; try++ {
        error = sendMessagesToLogstash(connection, messages, port)

        if error == nil {
            break
        } else {
            log.Printf("Try to send message to Logstash: %s", error.Error())
        }
        time.Sleep(1 * 1000 * time.Millisecond)
    }

    return error
} 

func getLogstashOutput() string {
    var output []byte
    var error error

    for try := 0; try < 5; try++ {
        output, error = ioutil.ReadFile(OutputFilePath)
        if error == nil {
            break
        } else {
            log.Print(error.Error())
        }
        time.Sleep(3 * 1000 * time.Millisecond)
    }

    if error != nil {
        errorMessage := "Cannot read output: " + error.Error()
        log.Print(errorMessage)
        return errorMessage
    }

    return string(output)
}

func sendMessagesToLogstash(connection* net.UDPConn, messages []string, port int) error {
    logstashAddress := net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: port}
    for _, message := range messages {
        log.Printf("Message: %s", message)
        _, error := connection.WriteToUDP([]byte(message), &logstashAddress)
        if error != nil {
            return error
        }
    }

    return nil
}
