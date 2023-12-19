package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"

	fcgiclient "github.com/tomasen/fcgi_client"
)

// PHPFPMStatus is a struct that represents the JSON response from PHP-FPM status endpoint.
// We only need a few fields, but the rest are here for reference. See https://www.php.net/manual/en/fpm.status.php for more information.
type PHPFPMStatus struct {
	Pool            string          `json:"pool"`
	ProcessManager  string          `json:"process manager"`
	StartTime       int64           `json:"start time"`
	StartSince      int64           `json:"start since"`
	AcceptedConn    int64           `json:"accepted conn"`
	ListenQueue     int64           `json:"listen queue"`
	MaxListenQueue  int64           `json:"max listen queue"`
	ListenQueueLen  int64           `json:"listen queue len"`
	IdleProcesses   int64           `json:"idle processes"`
	ActiveProcesses int64           `json:"active processes"`
	TotalProcesses  int64           `json:"total processes"`
	MaxActiveProcs  int64           `json:"max active processes"`
	MaxChildren     int64           `json:"max children reached"`
	SlowRequests    int64           `json:"slow requests"`
	Processes       []PHPFPMProcess `json:"processes"`
}

// PHPFPMProcess is a struct that represents a process in the JSON response from PHP-FPM status endpoint.
// We don't use any of these fields but they may be useful in the future.
type PHPFPMProcess struct {
	PID             int64   `json:"pid"`
	State           string  `json:"state"`
	StartTime       int64   `json:"start time"`
	StartSince      int64   `json:"start since"`
	Requests        int64   `json:"requests"`
	RequestDuration int64   `json:"request duration"`
	RequestMethod   string  `json:"request method"`
	RequestURI      string  `json:"request uri"`
	ContentLength   int64   `json:"content length"`
	User            string  `json:"user"`
	Script          string  `json:"script"`
	LastRequestCPU  float64 `json:"last request cpu"`
	LastRequestMem  int64   `json:"last request memory"`
}

// Export metrics to CloudWatch. If we can't export them, we'll log an error.
func exportToCloudWatch(response PHPFPMStatus, ServiceName string) {

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	svc := cloudwatch.New(sess)

	log.Println("Exporting metrics...")
	log.Println("ListenQueue:", response.ListenQueue)
	log.Println("ActiveProcesses:", response.ActiveProcesses)
	log.Println("MaxListenQueue:", response.MaxListenQueue)

	_, err := svc.PutMetricData(&cloudwatch.PutMetricDataInput{
		Namespace: aws.String("Monitoring/PHP-FPM"),
		MetricData: []*cloudwatch.MetricDatum{
			&cloudwatch.MetricDatum{
				MetricName: aws.String("ListenQueue"),
				Unit:       aws.String("Count"),
				Value:      aws.Float64(float64(response.ListenQueue)),
				Dimensions: []*cloudwatch.Dimension{
					&cloudwatch.Dimension{
						Name:  aws.String("ServiceName"),
						Value: aws.String(ServiceName),
					},
				},
			},
			&cloudwatch.MetricDatum{
				MetricName: aws.String("ActiveProcesses"),
				Unit:       aws.String("Count"),
				Value:      aws.Float64(float64(response.ActiveProcesses)),
				Dimensions: []*cloudwatch.Dimension{
					&cloudwatch.Dimension{
						Name:  aws.String("ServiceName"),
						Value: aws.String(ServiceName),
					},
				},
			},
			&cloudwatch.MetricDatum{
				MetricName: aws.String("MaxListenQueue"),
				Unit:       aws.String("Count"),
				Value:      aws.Float64(float64(response.MaxListenQueue)),
				Dimensions: []*cloudwatch.Dimension{
					&cloudwatch.Dimension{
						Name:  aws.String("ServiceName"),
						Value: aws.String(ServiceName),
					},
				},
			},
		},
	})
	if err != nil {
		fmt.Println("Error adding metrics:", err.Error())
		return
	}

}

// Retrieve stats from PHP-FPM using FastCGI protocol and return them as a PHPFPMStatus struct.
func retrievePhpFpmStats() PHPFPMStatus {
	fcgi, err := fcgiclient.Dial("tcp", "127.0.0.1:9000")
	if err != nil {
		log.Printf("Error connecting to PHP-FPM: %v", err)
	}
	defer fcgi.Close()

	// Send a request to PHP-FPM
	env := map[string]string{
		"SCRIPT_FILENAME": "/status",
		"SCRIPT_NAME":     "/status",
		"SERVER_SOFTWARE": "go / fcgiclient",
		"REMOTE_ADDR":     "127.0.0.1",
		"QUERY_STRING":    "json&full",
	}

	resp, err := fcgi.Get(env)
	if err != nil {
		log.Printf("Error sending request: %v", err)
	}
	defer resp.Body.Close()

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response: %v", err)
	}

	var response PHPFPMStatus
	json.Unmarshal(content, &response)

	return response
}

// Retrieve service name of the ECS task from metadata endpoint. If we can't retrieve it, we'll return "NOT_SET" and log an error message.
func retrieveServiceName() string {
	serviceName := "NOT_SET"
	metadataUrl := os.Getenv("ECS_CONTAINER_METADATA_URI_V4")
	infoEndpoint := metadataUrl + "/latest/meta-data/instance-id"

	request, err := http.NewRequest("GET", infoEndpoint, nil)
	if err != nil {
		log.Printf("Error creating request: %v. Defaulting to %v", err, serviceName)
		return serviceName
	}

	request.Header.Set("X-Aws-Ec2-Metadata-Token-Ttl-Seconds", "21600")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		log.Printf("Error retrieving instance ID: %v. Defaulting to %v", err, serviceName)
		return serviceName
	}

	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Printf("Error reading response body: %v. Defaulting to %v", err, serviceName)
		return serviceName
	}

	if response.StatusCode != 200 {
		log.Printf("Received non-200 response from meta-data URL: %v. Defaulting to %v", response.StatusCode, serviceName)
		return serviceName
	}

	serviceName = string(body)
	return serviceName
}

func main() {

	ServiceName := retrieveServiceName()

	for {
		// Retrieve stats from PHP-FPM and export them to CloudWatch
		response := retrievePhpFpmStats()
		exportToCloudWatch(response, ServiceName)

		time.Sleep(30 * time.Second)
	}

}
