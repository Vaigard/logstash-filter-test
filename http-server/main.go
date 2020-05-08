package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
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
	LocalOutboundPort      = 8180
	ServerPort             = ":8181"
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

	log.Fatal(http.ListenAndServe(ServerPort, nil))
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
	multiPartReader, err := request.MultipartReader()
	if err != nil {
		log.Printf("Parse request: %s", err.Error())
		return pipelineInput, RequestTypeInvalid
	}

	for {
		part, err := multiPartReader.NextPart()

		// This is OK, no more parts
		if err == io.EOF {
			break
		}

		// Any other error
		if err != nil {
			log.Printf("Get new request part: %s", err.Error())
			return pipelineInput, RequestTypeInvalid
		}

		var buffer bytes.Buffer
		io.Copy(&buffer, part)

		switch part.FormName() {
		case "filter":
			pipelineInput.Filter = buffer.String()
		case "message":
			pipelineInput.Message = buffer.String()
		case "patterns":
			writePatternsFile(buffer.String())
		case "patterns_dir":
			defer changePatternsDirs(&pipelineInput, buffer.String())
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
	err := ioutil.WriteFile(FilterFilePath, []byte(filter), 0644)
	if err != nil {
		errorMessage := "Cannot write filter: " + err.Error()
		log.Print(errorMessage)
		return fmt.Sprintf("{\"Error\": \"%s\"}", errorMessage)
	}

	defer ioutil.WriteFile(FilterFilePath, []byte("filter{}\n"), 0644)

	// wait for restart pipeline (autoreload in 2 seconds)
	time.Sleep(5 * 1000 * time.Millisecond)

	err = processMessage(message)

	if err != nil {
		errorMessage := "Cannot send message to Logstash: " + err.Error()
		log.Print(errorMessage)
		return fmt.Sprintf("{\"Error\": \"%s\"}", errorMessage)
	}

	time.Sleep(5 * 1000 * time.Millisecond)

	defer os.Remove(OutputFilePath)

	output := getLogstashOutput()

	cleanPatternsDirectory(PatternsDirectory)

	return output
}

func processMessage(message string) error {
	messages := strings.Split(message, "\n")

	connection, err := net.ListenUDP("udp", &net.UDPAddr{Port: LocalOutboundPort})
	if err != nil {
		errorMessage := "Cannot connect to port 1234/udp: " + err.Error()
		log.Print(errorMessage)
		return err
	}
	defer connection.Close()

	for try := 0; try < 3; try++ {
		err = sendMessagesToLogstash(connection, messages, LogstashInputPort)

		if err == nil {
			break
		} else {
			log.Printf("Try to send message to Logstash: %s", err.Error())
		}
		time.Sleep(1 * 1000 * time.Millisecond)
	}

	return err
}

func getLogstashOutput() string {
	var output []byte
	var err error

	for try := 0; try < 5; try++ {
		output, err = ioutil.ReadFile(OutputFilePath)
		if err == nil {
			break
		} else {
			log.Print(err.Error())
		}
		time.Sleep(3 * 1000 * time.Millisecond)
	}

	if err != nil {
		errorMessage := "Cannot read output: " + err.Error()
		log.Print(errorMessage)
		return errorMessage
	}

	return string(output)
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

func changePatternsDirs(pipelineInput *logstashPipelineInput, patternsDirectories string) {
	patternsDirectoriesList := strings.Split(patternsDirectories, ",")
	for _, patternsDirectory := range patternsDirectoriesList {
		pipelineInput.Filter = strings.ReplaceAll(pipelineInput.Filter, patternsDirectory, PatternsDirectory)
	}
}

func writePatternsFile(patterns string) {
	patternsFileName := PatternsDirectory + "/" + randomString(PatternsFileNameLength)
	err := ioutil.WriteFile(patternsFileName, []byte(patterns), 0644)
	if err != nil {
		errorMessage := "Cannot write patterns file: " + err.Error()
		log.Print(errorMessage)
	}
}

func cleanPatternsDirectory(patternsDirectory string) {
	directory, err := os.Open(patternsDirectory)
	if err != nil {
		errorMessage := "Cannot open patterns directory: " + err.Error()
		log.Print(errorMessage)
		return
	}
	defer directory.Close()

	names, err := directory.Readdirnames(-1)
	if err != nil {
		errorMessage := "Cannot get pattern files names: " + err.Error()
		log.Print(errorMessage)
		return
	}

	for _, name := range names {
		err = os.RemoveAll(filepath.Join(patternsDirectory, name))
		if err != nil {
			errorMessage := "Cannot delete patterns file %s: " + err.Error()
			log.Printf(errorMessage, name)
		}
	}
}

func randomString(length int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	name := make([]byte, length)
	for letter := range name {
		name[letter] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(name)
}
