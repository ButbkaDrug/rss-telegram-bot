package store

import (
	"os"
)

// Super simple file store.
type Simplestore struct{
    // Who knows why I put it there? But let is be :)
    os.File
    // Allows to define storage location
	Filepath string
}

// Creates a new instance of a simple store with default configuration
func NewSimplestore() Simplestore {
	return Simplestore{
        Filepath: ".users/data",
    }
}

// Creates new instance of a simple store with a specific path.
func NewSimplestoreWithFilepath(p string) Simplestore {
    s := Simplestore{}
    s.Filepath = p

    return s
}

// Read from the file
func (s Simplestore) Read() ([]byte, error) {
	return os.ReadFile(s.Filepath)

}

// Writes to a file
func (s Simplestore) Write(data []byte) error {
	return os.WriteFile(s.Filepath, data, 0644)
}
