package switchfs

import (
	"encoding/binary"
	"errors"
	"io"
)

const (
	PfsfileEntryTableSize = 0x18
	HfsfileEntryTableSize = 0x40
	pfs0Magic             = "PFS0"
	hfs0Magic             = "HFS0"
)

type fileEntry struct {
	StartOffset uint64
	Size        uint64
	Name        string
}

// PFS0 struct to represent PFS0 filesystem of NSP
type PFS0 struct {
	Filepath  string
	Size      uint64
	HeaderLen uint16
	Files     []fileEntry
}

// https://wiki.oatmealdome.me/PFS0_(File_Format)
func ReadPfs0File(filePath string) (*PFS0, error) {

	file, err := OpenFile(filePath)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	p, err := readPfs0(file, 0x0)
	if err != nil {
		return nil, err
	}
	p.Filepath = filePath
	return p, nil
}

func readPfs0(reader io.ReaderAt, offset int64) (*PFS0, error) {

	header := make([]byte, 0xC)
	n, err := reader.ReadAt(header, offset)
	if err != nil {
		return nil, err
	}
	if n != 0xC {
		return nil, errors.New("failed to read file")
	}
	var fileEntryTableSize uint16
	if string(header[:0x4]) == pfs0Magic {
		fileEntryTableSize = PfsfileEntryTableSize
	} else if string(header[:0x4]) == hfs0Magic {
		fileEntryTableSize = HfsfileEntryTableSize
	} else {
		return nil, errors.New("Invalid NSP headerBytes. Expected 'PFS0'/'HFS0', got '" + string(header[:0x4]) + "'")
	}
	p := &PFS0{}

	fileCount := binary.LittleEndian.Uint16(header[0x4:0x8])
	stringsLen := binary.LittleEndian.Uint16(header[0x8:0xC])

	// Compute the entry-table offset in a wide int so a large fileCount cannot
	// silently overflow (the original uint16 math could wrap and produce a tiny
	// offset, leading to out-of-range reads/panics later).
	fileEntryTableOffset := int64(0x10) + int64(fileEntryTableSize)*int64(fileCount)

	if fileEntryTableOffset+int64(stringsLen) > 0x7fffffff {
		return nil, errors.New("invalid PFS0/HFS0 header: implausible size")
	}
	p.HeaderLen = uint16(fileEntryTableOffset) + stringsLen
	fileNamesBuffer := make([]byte, stringsLen)
	_, err = reader.ReadAt(fileNamesBuffer, offset+fileEntryTableOffset)
	if err != nil {
		return nil, err
	}

	p.Files = make([]fileEntry, fileCount)
	// go over the fileEntries
	for i := uint16(0); i < fileCount; i++ {
		fileEntryTable := make([]byte, fileEntryTableSize)
		_, err = reader.ReadAt(fileEntryTable, offset+int64(0x10)+int64(fileEntryTableSize)*int64(i))
		if err != nil {
			return nil, err
		}

		fileOffset := binary.LittleEndian.Uint64(fileEntryTable[0:8])
		fileSize := binary.LittleEndian.Uint64(fileEntryTable[8:16])

		// The name offset is read straight from the file; bound it against the
		// actual string table before slicing to avoid an out-of-range panic.
		nameOffset := binary.LittleEndian.Uint32(fileEntryTable[16:20])
		if nameOffset > uint32(len(fileNamesBuffer)) {
			return nil, errors.New("invalid PFS0/HFS0 file entry: name offset out of range")
		}
		var nameBytes []byte
		for _, b := range fileNamesBuffer[nameOffset:] {
			if b == 0x0 {
				break
			} else {
				nameBytes = append(nameBytes, b)
			}
		}

		p.Files[i] = fileEntry{fileOffset + uint64(p.HeaderLen), fileSize, string(nameBytes)}
	}
	p.HeaderLen += stringsLen

	return p, nil
}
