package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"os"
	"reflect"
	"strings"
)

func main() {

	services := []string{"s3", "ec2"}

	for _, service := range services {
		collectService(service)
	}
}

// service 별 수집
func collectService(service string) {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-east-1"),
	)
	if err != nil {
		panic(err)
	}
	opsMap := make(map[string]AWSOperation)

	//zap.S().Debugf("collect : %s", service)
	//start := time.Now()
	switch service {
	case "s3":
		client := s3.NewFromConfig(cfg)
		opsMap = getAWSClient(reflect.TypeOf(client))
	case "ec2":
		client := ec2.NewFromConfig(cfg)
		opsMap = getAWSClient(reflect.TypeOf(client))
	}

	b, err := json.MarshalIndent(opsMap, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling operation map: %v\n", err)
		return
	} else {
		_ = os.WriteFile(fmt.Sprintf("aws_%s_operations.json", service), b, 0644)
	}
	return
}

func getAWSClient(ops reflect.Type) map[string]AWSOperation {
	// Initialize a map to store the input and output JSON structures.
	opMaps := make(map[string]AWSOperation)

	for i := 0; i < ops.NumMethod(); i++ {
		// Get the operation name and type.
		method := ops.Method(i)
		methodName := method.Name
		methodType := method.Func.Type()

		var inputMaps map[string]interface{}
		var outputMaps map[string]interface{}

		// Get the input and output types for the operation.
		for j := 0; j < methodType.NumIn(); j++ {
			inputType := methodType.In(j)
			inputTypeName := inputType.String()
			switch {
			case strings.Contains(inputTypeName, "Client"):
				continue
			case strings.Contains(inputTypeName, "context"):
				continue
			case strings.Contains(inputTypeName, "Options"):
				continue
			}
			//fmt.Printf("  in #%d : %s\n", j, inputTypeName)
			inputMap := getStructFields(inputType.Elem())
			if len(inputMap) > 0 {
				inputMaps = inputMap
			}
		}

		for j := 0; j < methodType.NumOut(); j++ {
			outType := methodType.Out(j)
			outTypeName := outType.String()
			switch {
			case strings.Contains(outTypeName, "error"):
				continue
			}
			//fmt.Printf("  out #%d : %s\n", j, outTypeName)
			outMap := getStructFields(outType.Elem())
			if len(outMap) > 0 {
				outputMaps = outMap
			}
		}

		// Add the input and output maps to the operation map.
		opMaps[methodName] = AWSOperation{
			//MethodName:     methodName,
			MethodRequest:  &inputMaps,
			MethodResponse: &outputMaps,
		}
	}

	// Marshal the operation map to JSON.
	/*	opJSON, err := json.MarshalIndent(opMaps, "", "  ")
		if err != nil {
			fmt.Printf("Error marshaling operation map: %v\n", err)
		} else {
			fmt.Printf("Operation map:\n%s\n", opJSON)
		}*/
	return opMaps
}

func getStructFields(structType reflect.Type) map[string]interface{} {
	fields := make(map[string]interface{})
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		switch field.Name {
		case "noSmithyDocumentSerde", "ResultMetadata":
			continue
		}
		jsonTag := field.Tag.Get("json")
		if jsonTag != "" {
			fields[jsonTag] = field.Type.String()
		} else {
			fields[field.Name] = field.Type.String()
		}
	}
	return fields
}

type AWSOperation struct {
	//MethodName     string                  `json:"method_name" `
	MethodRequest  *map[string]interface{} `json:"requestParameters" `
	MethodResponse *map[string]interface{} `json:"responseElements" `
}
