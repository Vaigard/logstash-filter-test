package main

import (
    "io"
    "io/ioutil"
    "net/http"
    "log"
    "encoding/json"
    "bytes"
    "time"
    "os"
    "bufio"
    "strings"
)

const (
    RequestTypeInvalid                 = -1
    RequestTypeEmptyFilter             = 0
    RequestTypeMessageFilter           = 1
    RequestTypeMessageFilterExpected   = 2
    RequestTypeFilter                  = 3
)

ServerLogPath := "server.log"
var InputFilePath string
var FilterFilePath string
var OutputFilePath string

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

func checkConfig(args []string) (string, string, string) {
    // default logstash docker paths
    inputFilePath  := "/usr/share/logstash/input.txt"
    filterFilePath := "/usr/share/logstash/pipeline/filter.conf"
    outputFilePath := "/usr/share/logstash/output.json"

    if len(args) == 1 {
        log.Print("No user config, using default configuration\n")
        return inputFilePath, filterFilePath, outputFilePath
    }

    filename := args[1]

    if len(filename) == 0 {
        return inputFilePath, filterFilePath, outputFilePath
    }

    file, err := os.Open(filename)
    if err != nil {
        log.Printf("Cannot read config file %s, using default configuration\n", filename)
        return inputFilePath, filterFilePath, outputFilePath
    }
    defer file.Close()
    
    reader := bufio.NewReader(file)

    config := map[string]string{
        "input": inputFilePath,
        "filter": filterFilePath,
        "output": outputFilePath,
    }

    for {
        line, err := reader.ReadString('\n')
        
        if equal := strings.Index(line, "="); equal >= 0 {
            if key := strings.TrimSpace(line[:equal]); len(key) > 0 {
                value := ""
                if len(line) > equal {
                    value = strings.TrimSpace(line[equal+1:])
                }

                config[key] = value
            }
        }
        if err == io.EOF {
            break
        }
        if err != nil {
            return inputFilePath, filterFilePath, outputFilePath
        }
    }

    inputFilePath = config["input"]
    filterFilePath = config["filter"]
    outputFilePath = config["output"]

    log.Printf("Loaded config file %s\n", filename)
    return inputFilePath, filterFilePath, outputFilePath
 }

func lintFitler(filter string) string {
    return "lintFilter"
}

func processMessage(message string, filter string) string {
    err := ioutil.WriteFile(FilterFilePath, []byte(filter), 0644)
    if err != nil {
        return "Cannot write filter: " + err.Error()
    }

    time.Sleep(10 * 1000 * time.Millisecond)

    err = ioutil.WriteFile(InputFilePath, []byte(message), 0644)
    if err != nil {
        return "Cannot write message: " + err.Error()
    }

    defer ioutil.WriteFile(InputFilePath, []byte("~~~~~~~~~~~~~~~\n\n"), 0644)

    time.Sleep(10 * 1000 * time.Millisecond)

    output, err := ioutil.ReadFile(OutputFilePath)
    if err != nil {
        return "Cannot read output: " + err.Error()
    }
    
    ioutil.WriteFile(FilterFilePath, []byte("filter{}\n"), 0644)
    time.Sleep(5 * 1000 * time.Millisecond)
    os.Remove(OutputFilePath)

    return string(output)
}

func compareOutput(expected string, actual string) string {
    return "compareOutput"
}

func testPipeline(pipelineInput logstashPipelineInput, requestType int) logstashPipelineOutput {
    if requestType == RequestTypeFilter {
        return logstashPipelineOutput{Lint: lintFitler(pipelineInput.Filter)}
    }

    pipelineOutput := logstashPipelineOutput{Output: processMessage(pipelineInput.Message, pipelineInput.Filter)}

    if requestType == RequestTypeMessageFilterExpected {
        pipelineOutput.Diff = compareOutput(pipelineInput.Expected, pipelineOutput.Output)
    }

    return pipelineOutput
}

func getPipelineInput(request *http.Request) (logstashPipelineInput, int) {
    pipelineInput := logstashPipelineInput{}
    multiPartReader, error := request.MultipartReader()
    if error != nil {
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
            return pipelineInput, RequestTypeInvalid
        }

        var buffer bytes.Buffer
        io.Copy(&buffer, part)

        switch part.FormName() {
        case "filter":
            pipelineInput.Filter = buffer.String()
        case "message":
            pipelineInput.Message = buffer.String()
        case "expected":
            pipelineInput.Expected = buffer.String()
        default:
            return pipelineInput, RequestTypeInvalid
        }
    }

    if pipelineInput.Filter == "" {
        return pipelineInput, RequestTypeEmptyFilter
    }

    if pipelineInput.Message != "" && pipelineInput.Expected != "" {
        return pipelineInput, RequestTypeMessageFilterExpected
    }

    if pipelineInput.Message != "" && pipelineInput.Expected == "" {
        return pipelineInput, RequestTypeMessageFilter
    }

    return pipelineInput, RequestTypeFilter
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
    pipelineOutput := logstashPipelineOutput{}

    pipelineInput, requestType := getPipelineInput(request)

    switch requestType {
    case RequestTypeInvalid:
        pipelineOutput = logstashPipelineOutput{Status: "Invalid request"}
    case RequestTypeEmptyFilter:
        pipelineOutput = logstashPipelineOutput{Status: "Empty filter field"}
    default:
        pipelineOutput = testPipeline(pipelineInput, requestType)
        pipelineOutput.Status = "OK"
    }

    responseJson, marshalError := json.Marshal(pipelineOutput)
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

    logFile, _ := os.OpenFile(ServerLogPath, os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
    defer logFile.Close()

    log.SetOutput(logFile)

    InputFilePath, FilterFilePath, OutputFilePath = checkConfig(os.Args)    

    os.Remove(InputFilePath)
    os.Remove(OutputFilePath)

    log.Fatal(http.ListenAndServe(":8081", nil))
}
