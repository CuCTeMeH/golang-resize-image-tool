package handlers

import (
	"bytes"
	"fmt"
	"github.com/CuCTeMeH/golang-resize-image-tool/model"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/ducmeit1/imaging"
	"github.com/gorilla/mux"
	"net/http"
	"os"
)

type ResizeCropHandler struct {
	initialized bool
	s3Handler   S3Bucket
}

var regional string

func (s *ResizeCropHandler) init() error {
	if !s.initialized {
		s.getConfig()
		s.s3Handler = new(S3Handler)
		s.initialized = true
	}
	return nil
}

func (s *ResizeCropHandler) getConfig() {
	regional = os.Getenv("regional")
}

func (s *ResizeCropHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := s.init(); err != nil {
		fmt.Printf("Error here: %v\n", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	originalBucket := r.URL.Query().Get("original_bucket")
	originalFolder := r.URL.Query().Get("original_folder")

	resizeBucket := r.URL.Query().Get("resize_bucket")
	resizedFolder := r.URL.Query().Get("resize_folder")

	vars := mux.Vars(r)
	//Bind params to model
	img := &model.Image{
		Optional: vars["optional"],
	}
	//Validate model state
	if !img.IsMatchFormat() {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	fmt.Printf("Img: %v | %v | %v | %v | %v | %v\n", img.FileName, img.Crop, img.Extension, img.Dimension, img.Height, img.Width)
	ctx := r.Context()
	sess := session.Must(session.NewSession())
	sess.Config.Region = aws.String(regional)

	originalKey := img.GetS3Key(originalFolder, img.FileName)
	exist, data, err := s.s3Handler.DownloadImage(ctx, sess, originalBucket, originalKey)
	if !exist {
		fmt.Printf("Not found image with key: %v\n", originalKey)
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	if err != nil {
		fmt.Printf("Download image error : %v\n", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	//Decode from downloaded data
	originalImage, err := imaging.Decode(bytes.NewReader(data))
	if err != nil {
		fmt.Printf("Decode image error: %v\n", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	//Resize image
	resizedImage := img.ResizeOrCrop(originalImage)
	if resizedImage == nil {
		fmt.Printf("Resized image error: %v\n", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	//Encode resize image and upload to s3
	var bufferEncode = new(bytes.Buffer)
	errEncode := imaging.Encode(bufferEncode, resizedImage, model.ParseExtension(model.ParseContentType(img.Extension)))
	if errEncode != nil {
		fmt.Printf("Encode image error: %v\n", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	//Upload to S3
	targetKey := img.GetS3Key(resizedFolder, img.GetOutputFileName())
	output, err := s.s3Handler.UploadImage(ctx, sess, resizeBucket, targetKey, bufferEncode.Bytes())

	if err != nil {
		fmt.Printf("Upload image error: %v\n", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	//Serve file just uploaded
	fmt.Printf("New image has uploaded at: %v\n", output.Location)
	http.Redirect(w, r, output.Location, http.StatusTemporaryRedirect)
}
