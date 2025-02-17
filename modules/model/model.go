/* VectorModel interface
32 bit and 8 bit model structs
LoadModel and GetEmbedding methods for both structs
LoadVectorModel function to load either 32 bit or 8 bit model based on file extension
*/

package model

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
)

// VectorModel interface defines the methods that all vector models must implement
type VectorModel interface {
	LoadModel(filename string) error
	GetEmbedding(token string) interface{}
}

// Word2VecModel represents a 32-bit floating point Word2Vec model
type Word2VecModel struct {
	Vectors map[string][]float32
	Size    int
}

// LoadModel loads a 32-bit floating point Word2Vec model from a file
func (m *Word2VecModel) LoadModel(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	var vocabSize, vectorSize int
	fmt.Fscanf(reader, "%d %d\n", &vocabSize, &vectorSize)

	m.Vectors = make(map[string][]float32, vocabSize)
	m.Size = vectorSize

	for i := 0; i < vocabSize; i++ {
		word, err := reader.ReadString(' ')
		if err != nil {
			return fmt.Errorf("failed to read word: %v", err)
		}
		word = strings.TrimSpace(word)

		vector := make([]float32, vectorSize)
		for j := 0; j < vectorSize; j++ {
			err := binary.Read(reader, binary.LittleEndian, &vector[j])
			if err != nil {
				return fmt.Errorf("failed to read vector: %v", err)
			}
		}
		m.Vectors[word] = vector
	}

	return nil
}

// GetEmbedding returns the vector embedding of a token for the 32-bit model
func (m *Word2VecModel) GetEmbedding(token string) interface{} {
	vec, ok := m.Vectors[token]
	if !ok {
		return make([]float32, m.Size)
	}
	return vec
}

// QuantizedWord2VecModel represents an 8-bit integer quantized Word2Vec model
type QuantizedWord2VecModel struct {
	Vectors map[string][]int8
	Min     float32
	Max     float32
	Size    int
}

// LoadModel loads an 8-bit integer quantized Word2Vec model from a file
func (m *QuantizedWord2VecModel) LoadModel(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	var vocabSize, vectorSize int32
	if err := binary.Read(file, binary.LittleEndian, &vocabSize); err != nil {
		return fmt.Errorf("failed to read vocab size: %v", err)
	}
	if err := binary.Read(file, binary.LittleEndian, &vectorSize); err != nil {
		return fmt.Errorf("failed to read vector size: %v", err)
	}
	m.Size = int(vectorSize)

	if err := binary.Read(file, binary.LittleEndian, &m.Min); err != nil {
		return fmt.Errorf("failed to read min value: %v", err)
	}
	if err := binary.Read(file, binary.LittleEndian, &m.Max); err != nil {
		return fmt.Errorf("failed to read max value: %v", err)
	}

	m.Vectors = make(map[string][]int8, vocabSize)

	for i := 0; i < int(vocabSize); i++ {
		word, err := readNullTerminatedString(file)
		if err != nil {
			return fmt.Errorf("failed to read word: %v", err)
		}

		vector := make([]int8, vectorSize)
		if err := binary.Read(file, binary.LittleEndian, &vector); err != nil {
			return fmt.Errorf("failed to read vector: %v", err)
		}

		m.Vectors[word] = vector
	}

	return nil
}

// GetEmbedding returns the vector embedding of a token for the 8-bit quantized model
func (m *QuantizedWord2VecModel) GetEmbedding(token string) interface{} {
	vec, ok := m.Vectors[token]
	if !ok {
		return make([]int8, m.Size)
	}
	return vec
}

// Helper function to read null-terminated strings
func readNullTerminatedString(reader io.Reader) (string, error) {
	var bytes []byte
	for {
		var b [1]byte
		_, err := reader.Read(b[:])
		if err != nil {
			return "", err
		}
		if b[0] == 0 {
			break
		}
		bytes = append(bytes, b[0])
	}
	return string(bytes), nil
}

// LoadVectorModel loads either a 32-bit or 8-bit model based on the file extension
func LoadVectorModel(filename string) (VectorModel, error) {
	var model VectorModel

	if strings.HasSuffix(filename, ".bin") {
		model = &Word2VecModel{}
	} else if strings.HasSuffix(filename, ".8int.bin") {
		model = &QuantizedWord2VecModel{}
	} else {
		return nil, fmt.Errorf("unsupported file format")
	}

	err := model.LoadModel(filename)
	if err != nil {
		return nil, err
	}

	return model, nil
}
