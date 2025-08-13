package wav

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"

	"path/filepath"
	"strings"
	"zham-app/utils"
)

// ReformatWAV converts a given WAV file to the specified number of channels,
// either mono (1 channel) or stereo (2 channels).
func ConvertToWAV(inputFilePath string, channels int, resampleRate int) (string, error) {
	if _, err := os.Stat(inputFilePath); err != nil {
		return "", fmt.Errorf("input file does not exist: %v", err)
	}

	if channels < 1 || channels > 2 {
		channels = 1
	}

	if resampleRate < 1 {
		resampleRate = 44100
	}

	fileExt := filepath.Ext(inputFilePath)
	outputFile := strings.TrimSuffix(inputFilePath, fileExt) + "rfm.wav"

	tmpFile := filepath.Join(filepath.Dir(outputFile), "tmp_"+filepath.Base(outputFile))
	defer os.Remove(tmpFile)

	cmd := exec.Command(
		"ffmpeg",
		"-y",
		"-i", inputFilePath,
		"-c", "pcm_s16le",
		"-ar", fmt.Sprint(resampleRate),
		"-ac", fmt.Sprint(channels),
		tmpFile,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to convert to wav: %v, outputFile %v, output %v", err, string(output), outputFile)
	}

	if err := utils.MoveFile(tmpFile, outputFile); err != nil {
		return "", fmt.Errorf("failed to rename temporary file to output file: %v", err)
	}

	return outputFile, nil
}

// WavBytesToFloat64 converts a slice of bytes from a .wav file to a slice of float64 samples
func WavBytesToSamples(input []byte) ([]float64, error) {
	if len(input)%2 != 0 {
		return nil, errors.New("invalid input length")
	}

	numSamples := len(input) / 2
	output := make([]float64, numSamples)

	for i := 0; i < len(input); i += 2 {
		// Interpret bytes as a 16-bit signed integer (little-endian)
		sample := int16(binary.LittleEndian.Uint16(input[i : i+2]))

		// Scale the sample to the range [-1, 1]
		output[i/2] = float64(sample) / 32768.0
	}

	return output, nil
}

func ConverterToWAV(r *http.Request, resampleRate int) ([]float64, error) {
	file, header, err := r.FormFile("audio")
	if err != nil {
		return nil, fmt.Errorf("failed to create file")
	}
	defer file.Close()

	fmt.Printf("Received file: %s\n", header.Filename)

	uploadedPath := "tmp/" + header.Filename
	outputFile, err := os.Create(uploadedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file in path %v", uploadedPath)
	}

	if _, err := io.Copy(outputFile, file); err != nil {
		return nil, fmt.Errorf("failed to copy into file")
	}
	// defer outputFile.Close()

	wavFile, err := ConvertToWAV(uploadedPath, 1, resampleRate)
	if err != nil {
		return nil, fmt.Errorf("error: %v, output %v", err, string(wavFile))
	}

	data, err := os.ReadFile(wavFile)
	if err != nil {
		return nil, err
	}

	if len(data) < 44 {
		return nil, errors.New("invalid WAV file size (too small)")
	}

	byteData := data[44:]
	sample, err := WavBytesToSamples(byteData)
	if err != nil {
		return nil, errors.New("invalid WAV file size (too small)")
	}

	utils.DeleteFile(wavFile)

	err = outputFile.Close()
	if err != nil {
		fmt.Println("outputFile.Close() error", err)
	}

	utils.DeleteFile(uploadedPath)

	return sample, nil
}
