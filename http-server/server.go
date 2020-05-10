package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	RequestTypeCorrect      = 0
	RequestTypeInvalid      = 1
	RequestTypeEmptyFilter  = 2
	RequestTypeEmptyMessage = 3
)

const (
	LogstashInputPort      = 8182
	ServerLogPath          = "server.log"
	ReadmeFile             = "README.md"
	FilterFilePath         = "/usr/share/logstash/pipeline/filter.conf"
	OutputFilePath         = "/usr/share/logstash/output.json"
	PatternsDirectory      = "/usr/share/logstash/patterns"
	PatternsFileNameLength = 5
)

type logstashPipelineInput struct {
	Message string
	Filter  string
	Pattern string
	PatternsDirs string
}

func main() {
	http.HandleFunc("/", mainHandler)
	http.HandleFunc("/ping", pingHandler)
	http.HandleFunc("/upload", logstashPipelineHandler)

	logFile, _ := os.OpenFile(ServerLogPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	defer logFile.Close()

	log.SetOutput(logFile)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Print("Start server--------------")

	os.Remove(OutputFilePath)

	log.Fatal(http.ListenAndServe(":8181", nil))
}

// "/"
// Main page returns documentation about server.
func mainHandler(responseWriter http.ResponseWriter, request *http.Request) {
	documentation, err := ioutil.ReadFile(ReadmeFile)
	if err == nil {
		io.WriteString(responseWriter, string(documentation))
	} else {
		log.Printf("Cannot read Readme file: %s", err.Error())
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
	log.Println("Got new request")
	
	var pipelineOutput string

	responseWriter.Header().Set("Content-Type", "application/json")
	defer responseWriter.Write([]byte(pipelineOutput))

	pipelineInput, err := getPipelineInput(request)
	if err != nil {
		pipelineOutput = fmt.Sprintf("{\"Error\": \"%s\"}", err.Error())
		log.Println(err.Error())
		return
	}

	if pipelineInput.PatternsDirs != "" {
		changePatternsDirs(&pipelineInput, PatternsDirectory)
	}

	if pipelineInput.Pattern != "" {
		patternFilePath := filepath.Join(PatternsDirectory, "pattern")
		err := ioutil.WriteFile(patternFilePath, []byte(pipelineInput.Pattern), 0644)
		if err != nil {
			pipelineOutput = fmt.Sprintf("{\"Error\": \"Cannot write patterns file: %s\"}", err.Error())
			log.Println(err.Error())
			return
		}
		defer os.Remove(patternFilePath)
	}

	err = ioutil.WriteFile(FilterFilePath, []byte(pipelineInput.Filter), 0644)
	if err != nil {
		pipelineOutput = fmt.Sprintf("{\"Error\": \"Cannot write filter: %s\"}", err.Error())
		log.Println(err.Error())
		return
	}

	defer ioutil.WriteFile(FilterFilePath, []byte("filter{}\n"), 0644)

	// wait for restart pipeline (autoreload in 2 seconds)
	time.Sleep(5 * 1000 * time.Millisecond)

	pipelineOutput, err = processPipeline(pipelineInput, OutputFilePath, LogstashInputPort)	

	if err != nil {
		pipelineOutput = fmt.Sprintf("{\"Error\": \"%s\"}", err.Error())
		log.Println(err.Error())
	}
}

func getPipelineInput(request *http.Request) (logstashPipelineInput, error) {
	pipelineInput := logstashPipelineInput{}
	multiPartReader, err := request.MultipartReader()
	if err != nil {
		return pipelineInput, fmt.Errorf("Parse request error: %s", err.Error())
	}

	for {
		part, err := multiPartReader.NextPart()

		// This is OK, no more parts
		if err == io.EOF {
			break
		}

		// Any other error
		if err != nil {
			return pipelineInput, fmt.Errorf("Get new request part error: %s", err.Error())
		}

		var buffer bytes.Buffer
		io.Copy(&buffer, part)

		pipelineInput.Pattern = ""

		switch part.FormName() {
		case "filter":
			pipelineInput.Filter = buffer.String()
		case "message":
			pipelineInput.Message = buffer.String()
		case "patterns":
			pipelineInput.Pattern = fmt.Sprintf("%s\n%s", pipelineInput.Pattern, buffer.String())
		case "patterns_dir":
			pipelineInput.PatternsDirs = buffer.String()
		default:
			return pipelineInput, fmt.Errorf("Invalid multipart data in request")
		}
	}

	if pipelineInput.Filter == "" {
		return pipelineInput, fmt.Errorf("Empty filter")
	}

	if pipelineInput.Message == "" {
		return pipelineInput, fmt.Errorf("Empty message")
	}

	return pipelineInput, nil
}

func processPipeline(pipelineInput logstashPipelineInput, outputFilePath string, logstashInputPort int) (string, error) {
	err := processMessage(pipelineInput.Message, logstashInputPort)

	if err != nil {
		return "", fmt.Errorf("Cannot send message to Logstash: %s" + err.Error())
	}

	time.Sleep(5 * 1000 * time.Millisecond)

	defer os.Remove(outputFilePath)

	output, err := getLogstashOutput(outputFilePath)	
	if err != nil {
		return "", fmt.Errorf("Cannot read logstash output: %s", err.Error())
	}

	return output, nil
}

func processMessage(message string, logstashInputPort int) error {
	messages := strings.Split(message, "\n")

	connection, err := net.ListenUDP("udp", &net.UDPAddr{Port: 8180})
	if err != nil {
		return fmt.Errorf("Cannot connect to port 8180/udp: %s", err.Error())
	}
	defer connection.Close()

	for try := 0; try < 3; try++ {
		err = sendMessagesToLogstash(connection, messages, logstashInputPort)

		if err == nil {
			break
		}
		time.Sleep(1 * 1000 * time.Millisecond)
	}

	return err
}

func getLogstashOutput(fileName string) (string, error) {
	var output []byte
	var err error

	for try := 0; try < 10; try++ {
		output, err = ioutil.ReadFile(fileName)
		if err == nil {
			break
		}
		time.Sleep(3 * 1000 * time.Millisecond)
	}

	if err != nil {
		return "", fmt.Errorf("Cannot read output: %s", err.Error())
	}

	return string(output), nil
}

func sendMessagesToLogstash(connection *net.UDPConn, messages []string, port int) error {
	logstashAddress := net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: port}
	for _, message := range messages {
		_, err := connection.WriteToUDP([]byte(message), &logstashAddress)
		if err != nil {
			log.Printf("Cannot send message: %s", message)
			return err
		}
	}

	return nil
}

func changePatternsDirs(pipelineInput *logstashPipelineInput, actualPatternsDirectory string) {
	patternsDirectoriesList := strings.Split(pipelineInput.PatternsDirs, ",")
	for _, patternsDirectory := range patternsDirectoriesList {
		pipelineInput.Filter = strings.ReplaceAll(pipelineInput.Filter, patternsDirectory, actualPatternsDirectory)
	}
}
