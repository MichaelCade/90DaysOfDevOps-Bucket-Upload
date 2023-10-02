package main

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gorilla/mux"
)

func main() {
	// Retrieve AWS credentials and configuration from environment variables
	awsAccessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	awsSecretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	awsRegion := os.Getenv("AWS_REGION")
	awsBucket := os.Getenv("AWS_BUCKET")

	// Initialize AWS session with retrieved credentials and region
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(awsRegion),
		Credentials: credentials.NewStaticCredentials(awsAccessKey, awsSecretKey, ""),
	})
	if err != nil {
		// Handle error
		panic(err)
	}

	// Create an S3 client
	svc := s3.New(sess)

	// Create a Gorilla Mux router
	router := mux.NewRouter()

	// Serve the HTML page for uploading files at the root path
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Read the HTML template file
		tmpl, err := template.ParseFiles("templates/upload.html")
		if err != nil {
			http.Error(w, "Failed to load the HTML template", http.StatusInternalServerError)
			return
		}

		// Render the HTML template
		err = tmpl.Execute(w, nil)
		if err != nil {
			http.Error(w, "Failed to render the HTML template", http.StatusInternalServerError)
			return
		}
	}).Methods("GET")

	// Define your upload route
	router.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		// Parse the uploaded file
		file, header, err := r.FormFile("file")
		if err != nil {
			// Handle error
			http.Error(w, "Failed to read the uploaded file", http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Use the original file name as the object key
		objectKey := header.Filename

		fmt.Printf("Uploading file: %s to S3 bucket: %s\n", objectKey, awsBucket)

		// Initialize a new multipart upload on S3
		initResp, err := svc.CreateMultipartUpload(&s3.CreateMultipartUploadInput{
			Bucket: aws.String(awsBucket),
			Key:    aws.String(objectKey),
		})
		if err != nil {
			// Handle error
			fmt.Printf("Failed to create multipart upload on S3: %v\n", err)
			http.Error(w, "Failed to create multipart upload on S3", http.StatusInternalServerError)
			return
		}
		uploadID := *initResp.UploadId

		// Configure the part size (e.g., 5 MB)
		partSize := int64(5 * 1024 * 1024)

		// Calculate the number of parts
		fileSize := header.Size
		numParts := (fileSize + partSize - 1) / partSize

		// Upload file parts
		var completedParts []*s3.CompletedPart
		for partNumber := int64(1); partNumber <= numParts; partNumber++ {
			partData := make([]byte, partSize)
			bytesRead, err := file.Read(partData)
			if err != nil {
				// Handle error
				fmt.Printf("Failed to read file part: %v\n", err)
				http.Error(w, "Failed to read file part", http.StatusInternalServerError)
				return
			}

			// Upload the part to S3
			uploadResp, err := svc.UploadPart(&s3.UploadPartInput{
				Bucket:     aws.String(awsBucket),
				Key:        aws.String(objectKey),
				UploadId:   aws.String(uploadID),
				PartNumber: aws.Int64(partNumber),
				Body:       strings.NewReader(string(partData[:bytesRead])),
			})
			if err != nil {
				// Handle error
				fmt.Printf("Failed to upload file part to S3: %v\n", err)
				http.Error(w, "Failed to upload file part to S3", http.StatusInternalServerError)
				return
			}

			completedParts = append(completedParts, &s3.CompletedPart{
				ETag:       uploadResp.ETag,
				PartNumber: aws.Int64(partNumber),
			})
		}

		// Complete the multipart upload
		_, err = svc.CompleteMultipartUpload(&s3.CompleteMultipartUploadInput{
			Bucket:          aws.String(awsBucket),
			Key:             aws.String(objectKey),
			UploadId:        aws.String(uploadID),
			MultipartUpload: &s3.CompletedMultipartUpload{Parts: completedParts},
		})
		if err != nil {
			// Handle error
			fmt.Printf("Failed to complete multipart upload on S3: %v\n", err)
			http.Error(w, "Failed to complete multipart upload on S3", http.StatusInternalServerError)
			return
		}

		// File uploaded successfully
		// You can redirect the user to a confirmation page
		http.Redirect(w, r, "/confirmation", http.StatusSeeOther)
	}).Methods("POST")

	// Serve the confirmation page
	router.HandleFunc("/confirmation", func(w http.ResponseWriter, r *http.Request) {
		// Read the HTML template file
		tmpl, err := template.ParseFiles("templates/confirmation.html")
		if err != nil {
			http.Error(w, "Failed to load the HTML template", http.StatusInternalServerError)
			return
		}

		// Render the HTML template
		err = tmpl.Execute(w, nil)
		if err != nil {
			http.Error(w, "Failed to render the HTML template", http.StatusInternalServerError)
			return
		}
	}).Methods("GET")

	// Print a message indicating that the server is starting
	fmt.Println("Server is starting and listening on port 8080...")

	// Start the HTTP server
	http.ListenAndServe(":8080", router)
}
