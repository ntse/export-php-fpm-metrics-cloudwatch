package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
)

// MetadataResponse represents the JSON response from the ECS metadata endpoint.
// See https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-metadata-endpoint-v4.html for more information.
type MetadataResponse struct {
	Cluster     string `json:"Cluster"`
	ServiceName string `json:"ServiceName"`
	TaskARN     string `json:"TaskARN"`
}

// PHPFPMStatus represents the JSON response from PHP-FPM status endpoint.
// See https://www.php.net/manual/en/fpm.status.php for more information.
type PHPFPMStatus struct {
	ListenQueue     int64 `json:"listen queue"`
	ActiveProcesses int64 `json:"active processes"`
	SlowRequests    int64 `json:"slow requests"`
	Yo              int64 `json:"yo"`
}

// GetContainerServiceName retrieves the name of the container service.
// It makes an HTTP GET request to the ECS_CONTAINER_METADATA_URI_V4 environment variable
// and parses the response to extract the service name.
// If successful, it returns the service name and nil error.
// Otherwise, it returns an empty string and the encountered error.
func GetContainerServiceName() (string, error) {
	resp, err := http.Get(fmt.Sprintf("%s/task", os.Getenv("ECS_CONTAINER_METADATA_URI_V4")))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var metadataResponse MetadataResponse
	err = json.NewDecoder(resp.Body).Decode(&metadataResponse)
	if err != nil {
		return "", err
	}

	return metadataResponse.ServiceName, nil
}

func ExportToCloudwatch(svc cloudwatchiface.CloudWatchAPI, phpfpmstatus PHPFPMStatus, servicename string) (*cloudwatch.PutMetricDataOutput, error) {

	PutMetricDataOutput, err := svc.PutMetricData(&cloudwatch.PutMetricDataInput{
		Namespace: aws.String("Monitoring/PHP-FPM"),
		MetricData: []*cloudwatch.MetricDatum{
			&cloudwatch.MetricDatum{
				MetricName: aws.String("ListenQueue"),
				Unit:       aws.String("Count"),
				Value:      aws.Float64(float64(phpfpmstatus.ListenQueue)),
				Dimensions: []*cloudwatch.Dimension{
					&cloudwatch.Dimension{
						Name:  aws.String("ServiceName"),
						Value: aws.String(servicename),
					},
				},
			},
			&cloudwatch.MetricDatum{
				MetricName: aws.String("ActiveProcesses"),
				Unit:       aws.String("Count"),
				Value:      aws.Float64(float64(phpfpmstatus.ActiveProcesses)),
				Dimensions: []*cloudwatch.Dimension{
					&cloudwatch.Dimension{
						Name:  aws.String("ServiceName"),
						Value: aws.String(servicename),
					},
				},
			},
			&cloudwatch.MetricDatum{
				MetricName: aws.String("SlowRequests"),
				Unit:       aws.String("Count"),
				Value:      aws.Float64(float64(phpfpmstatus.SlowRequests)),
				Dimensions: []*cloudwatch.Dimension{
					&cloudwatch.Dimension{
						Name:  aws.String("ServiceName"),
						Value: aws.String(servicename),
					},
				},
			},
		},
	})

	if err != nil {
		fmt.Println("Error adding metrics:", err.Error())
		return PutMetricDataOutput, err
	}

	fmt.Printf("%+v\n", PutMetricDataOutput)

	return PutMetricDataOutput, nil
}

func main() {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	svc := cloudwatch.New(sess)

	svc_name, err := GetContainerServiceName()
	if err != nil {
		fmt.Println("Error getting service name:", err.Error())
		return
	}

	ExportToCloudwatch(svc, PHPFPMStatus{}, svc_name)
}
