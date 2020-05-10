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

	defer cleanPatternsDirectory(PatternsDirectory)
	var pipelineOutput string

	pipelineInput, err := getPipelineInput(request)	

	if err == nil {
		pipelineOutput, err = processPipeline(pipelineInput, PatternsDirectory)
	}

	if err != nil {
		pipelineOutput = fmt.Sprintf("{\"Error\": \"%s\"}", err.Error())
	}
	
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.Write([]byte(pipelineOutput))
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

func processPipeline(pipelineInput logstashPipelineInput, patternsDirectory string) (string, error) {
	message := pipelineInput.Message
	filter := pipelineInput.Filter

	if pipelineInput.PatternsDirs != "" {
		changePatternsDirs(&pipelineInput, patternsDirectory)
	}

	if pipelineInput.Pattern != "" {
		patternFilePath := filepath.Join(patternsDirectory, "pattern")
		err := ioutil.WriteFile(patternFilePath, []byte(pipelineInput.Pattern), 0644)
		if err != nil {
			return "", fmt.Errorf("Cannot write patterns file: %s", err.Error())
		}
		defer os.Remove(patternFilePath)
	}

	log.Printf("Process new filter and message")
	err := ioutil.WriteFile(FilterFilePath, []byte(filter), 0644)
	if err != nil {
		return "", fmt.Errorf("Cannot write filter: %s", err.Error())
	}

	defer ioutil.WriteFile(FilterFilePath, []byte("filter{}\n"), 0644)

	// wait for restart pipeline (autoreload in 2 seconds)
	time.Sleep(5 * 1000 * time.Millisecond)

	err = processMessage(message)

	if err != nil {
		return "", fmt.Errorf("Cannot send message to Logstash: %s" + err.Error())
	}

	time.Sleep(5 * 1000 * time.Millisecond)

	defer os.Remove(OutputFilePath)

	output, err := getLogstashOutput(OutputFilePath)	
	if err != nil {
		return "", fmt.Errorf("Cannot read logstash output: %s", err.Error())
	}

	return output, nil
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

func writePatternsFile(patterns string, patternsDirectory string) error {
	patternsFileName := patternsDirectory + "/" + randomString(PatternsFileNameLength)
	err := ioutil.WriteFile(patternsFileName, []byte(patterns), 0644)
	if err != nil {
		return fmt.Errorf("Cannot write patterns file: %s", err.Error())
	}

	return nil
}

func cleanPatternsDirectory(patternsDirectory string) error {
	directory, err := os.Open(patternsDirectory)
	if err != nil {
		return fmt.Errorf("Cannot open patterns directory: %s", err.Error())
	}
	defer directory.Close()

	names, err := directory.Readdirnames(-1)
	if err != nil {
		return fmt.Errorf("Cannot get pattern files names: %s", err.Error())
	}

	for _, name := range names {
		err = os.RemoveAll(filepath.Join(patternsDirectory, name))
		if err != nil {
			return fmt.Errorf("Cannot delete patterns file %s: %s", name, err.Error())
		}
	}

	return nil
}

func randomString(length int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	name := make([]byte, length)
	for letter := range name {
		name[letter] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(name)
}
