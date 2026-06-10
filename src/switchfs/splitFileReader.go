package switchfs

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/avast/retry-go/v5"
)

type ReadAtCloser interface {
	io.ReaderAt
	io.Closer
}

type splitFile struct {
	info      []os.FileInfo
	files     []ReadAtCloser
	path      string
	chunkSize int64
}

type fileWrapper struct {
	file ReadAtCloser
	path string
}

func NewFileWrapper(filePath string) (*fileWrapper, error) {
	result := fileWrapper{}
	result.path = filePath
	file, err := _openFile(filePath)
	if err != nil {
		return nil, err
	}
	result.file = file
	return &result, nil
}

func (sp *fileWrapper) ReadAt(p []byte, off int64) (n int, err error) {
	if sp.file != nil {
		return sp.file.ReadAt(p, off)
	}
	return 0, errors.New("file is not opened")
}

func (sp *fileWrapper) Close() error {

	if sp.file != nil {
		return sp.file.Close()
	}

	return nil
}

func isSplitPart(fileName string, prefix string) bool {
	if !strings.HasPrefix(fileName, prefix) {
		return false
	}
	suffix := fileName[len(prefix):]
	if len(suffix) == 0 {
		return false
	}
	for i := 0; i < len(suffix); i++ {
		if suffix[i] < '0' || suffix[i] > '9' {
			return false
		}
	}
	return true
}

func NewSplitFileReader(filePath string) (*splitFile, error) {
	result := splitFile{}
	dir := filepath.Dir(filePath)
	baseName := filepath.Base(filePath)

	i := len(baseName) - 1
	for i >= 0 && baseName[i] >= '0' && baseName[i] <= '9' {
		i--
	}
	prefix := baseName[:i+1]

	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	type partInfo struct {
		info os.FileInfo
		num  int
	}
	var matchedParts []partInfo
	maxPart := -1

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := file.Name()
		if isSplitPart(name, prefix) {
			suffix := name[len(prefix):]
			partNum, err := strconv.Atoi(suffix)
			if err == nil {
				info, err := file.Info()
				if err == nil {
					matchedParts = append(matchedParts, partInfo{info: info, num: partNum})
					if partNum > maxPart {
						maxPart = partNum
					}
				}
			}
		}
	}

	if len(matchedParts) == 0 {
		return nil, errors.New("no split files found")
	}

	result.path = dir
	result.info = make([]os.FileInfo, maxPart+1)
	result.files = make([]ReadAtCloser, maxPart+1)

	for _, p := range matchedParts {
		result.info[p.num] = p.info
	}

	// Determine chunk size from part 0
	if result.info[0] != nil {
		result.chunkSize = result.info[0].Size()
	} else {
		// fallback to the size of the first available part
		for _, info := range result.info {
			if info != nil {
				result.chunkSize = info.Size()
				break
			}
		}
	}

	if result.chunkSize == 0 {
		return nil, errors.New("chunk size is 0 or no parts found")
	}

	return &result, nil
}

func (sp *splitFile) ReadAt(p []byte, off int64) (n int, err error) {
	//calculate the part containing the offset
	part := int(off / sp.chunkSize)

	if part < 0 || part >= len(sp.info) || sp.info[part] == nil {
		return 0, errors.New("missing part " + strconv.Itoa(part))
	}

	if len(sp.files) == 0 || sp.files[part] == nil {
		file, err := _openFile(filepath.Join(sp.path, sp.info[part].Name()))
		if err != nil {
			return 0, err
		}
		sp.files[part] = file
	}
	off = off - sp.chunkSize*int64(part)

	if off < 0 || off > sp.info[part].Size() {
		return 0, errors.New("offset is out of bounds")
	}
	return sp.files[part].ReadAt(p, off)
}

func _openFile(path string) (*os.File, error) {
	var file *os.File
	var err error
	err = retry.New(retry.Attempts(5)).Do(
		func() error {
			file, err = os.Open(path)
			return err
		},
	)
	return file, err
}

func (sp *splitFile) Close() error {
	for _, file := range sp.files {
		if file != nil {
			file.Close()
		}
	}
	return nil
}

func OpenFile(filePath string) (ReadAtCloser, error) {
	if len(filePath) == 0 {
		return nil, errors.New("empty file path")
	}
	//check if it's a split file
	if _, err := strconv.Atoi(filePath[len(filePath)-1:]); err == nil {
		return NewSplitFileReader(filePath)
	} else {
		return NewFileWrapper(filePath)
	}
}
