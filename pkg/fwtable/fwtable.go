package fwtable

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// Size of the file structure in bytes
const fwTableSize = 32

// FwTable represents the main table
type FwTable struct {
	Signature            string
	Checksum             uint32
	Version, Size, Extra int
	Parts                []*PartEntry
	Fws                  []*FwEntry
	// private
	content []byte
	table   struct {
		Signature   [8]byte
		Checksum    uint32
		Version     byte
		Reserved    [7]byte
		Paddings    uint32
		PartListLen uint32
		FwListLen   uint32
	}
}

func calculateChecksum(in []byte, start uint32) uint32 {
	for _, b := range in {
		start += uint32(b)
	}
	return start
}

func (fw *FwTable) checksum() uint32 {
	// Skip Signature and Checksum fields
	cs := calculateChecksum(fw.content[12:], 0)
	for _, p := range fw.Parts {
		cs = calculateChecksum(p.content, cs)
	}
	for _, f := range fw.Fws {
		cs = calculateChecksum(f.content, cs)
	}
	return cs
}

func (fw *FwTable) Validate() []error {
	ret := make([]error, 0)
	if fw.Signature != "VERONA__" {
		ret = append(ret, fmt.Errorf("Signature does not match, found '%s' expected 'VERONA__'", fw.Signature))
	}
	for _, v := range fw.table.Reserved {
		if v != 0 {
			ret = append(ret, fmt.Errorf("non-zero data on FwTableDesc reserved field"))
		}
	}
	cs := fw.checksum()
	if fw.table.Checksum != cs {
		ret = append(ret, fmt.Errorf("Checksum does not match, found 0x%08x expected 0x%08x", cs, fw.table.Checksum))
	}
	if len(ret) > 0 {
		return ret
	}
	return nil
}

func (fw *FwTable) ReadContent() error {
	err := binary.Read(bytes.NewReader(fw.content), binary.LittleEndian, &fw.table)
	if err != nil {
		return fmt.Errorf("could not read: %v", err)
	}
	fw.Signature = string(fw.table.Signature[:])
	fw.Checksum = fw.table.Checksum
	fw.Version = int(fw.table.Version)
	fw.Size = int(fw.table.Paddings)
	fw.Extra = fw.Size - 32 - int(fw.table.PartListLen) - int(fw.table.FwListLen)
	return nil
}

func (fw *FwTable) WriteContent() error {
	// Signature
	sigBytes := []byte(fw.Signature)[:8]
	for i := 0; i < 8; i++ {
		fw.table.Signature[i] = sigBytes[i]
	}
	// Version
	fw.table.Version = byte(fw.Version)
	// Reserved
	for i := 0; i < 7; i++ {
		fw.table.Reserved[i] = 0
	}
	// PartListLen
	fw.table.PartListLen = uint32(partEntrySize * len(fw.Parts))
	// FwListLen
	fw.table.FwListLen = uint32(fwEntrySize * len(fw.Fws))

	totalSize := fwTableSize + fw.table.PartListLen + fw.table.FwListLen
	// Make sure to line up on the sector size
	fw.table.Paddings = totalSize + 512 - (totalSize % 512)
	fw.Size = int(fw.table.Paddings)
	fw.Extra = int(fw.table.Paddings - totalSize)
	// Will update checksum after
	fw.table.Checksum = 0

	// Temporary Write to calculate checksum
	buffer := new(bytes.Buffer)
	err := binary.Write(buffer, binary.LittleEndian, &fw.table)
	if err != nil {
		return fmt.Errorf("writing header: %v", err)
	}
	copy(fw.content, buffer.Bytes())

	// Update parts content
	for i, p := range fw.Parts {
		err = p.WriteContent()
		if err != nil {
			return fmt.Errorf("part %d - %v", i+1, err)
		}
	}
	// Update FWs content
	for i, f := range fw.Fws {
		err = f.WriteContent()
		if err != nil {
			return fmt.Errorf("firmware %d - %v", i+1, err)
		}
	}
	// Recalculate Checksum
	fw.table.Checksum = fw.checksum()

	// Write content again with updated checksum
	buffer = new(bytes.Buffer)
	err = binary.Write(buffer, binary.LittleEndian, &fw.table)
	if err != nil {
		return fmt.Errorf("re-writing header: %v", err)
	}
	// Save content
	copy(fw.content, buffer.Bytes())

	return nil
}

func (fw *FwTable) Export(w io.Writer) error {
	n, err := w.Write(fw.content)
	if n != len(fw.content) {
		return fmt.Errorf("partial write in header (wrote %d out of %d bytes)", n, len(fw.content))
	}
	if err != nil {
		return fmt.Errorf("error writing header: %v", err)
	}
	written := len(fw.content)
	for i, p := range fw.Parts {
		n, err = w.Write(p.content)
		if n != len(p.content) {
			return fmt.Errorf("partial write in Part %d (wrote %d out of %d bytes)", i, n, len(p.content))
		}
		if err != nil {
			return fmt.Errorf("error writing Part %d: %v", i, err)
		}
		written += len(p.content)
	}
	// Update FWs content
	for i, f := range fw.Fws {
		n, err = w.Write(f.content)
		if n != len(f.content) {
			return fmt.Errorf("partial write in Firmware %d (wrote %d out of %d bytes)", i, n, len(f.content))
		}
		if err != nil {
			return fmt.Errorf("error writing Firmware %d: %v", i, err)
		}
		written += len(f.content)
	}
	// Append padding
	n, err = w.Write(make([]byte, (fw.Size - written)))
	if n != (fw.Size - written) {
		return fmt.Errorf("partial write on padding (wrote %d out of %d bytes)", n, fw.Size-written)
	}
	if err != nil {
		return fmt.Errorf("error writing paddings: %v", err)
	}
	return nil
}

func New(r io.Reader) (*FwTable, error) {
	buffer := make([]byte, fwTableSize)
	_, err := r.Read(buffer)
	if err != nil {
		return nil, fmt.Errorf("read: %v", err)
	}
	fw := &FwTable{content: buffer}
	err = fw.ReadContent()
	if err != nil {
		return nil, err
	}

	if int(fw.table.PartListLen)%partEntrySize != 0 {
		return nil, fmt.Errorf("partition entries size is not aligned: %d", fw.table.PartListLen)
	}
	fw.Parts = make([]*PartEntry, int(fw.table.PartListLen)/partEntrySize)

	if int(fw.table.FwListLen)%fwEntrySize != 0 {
		return nil, fmt.Errorf("partition entries size is not aligned: %d", fw.table.FwListLen)
	}
	fw.Fws = make([]*FwEntry, int(fw.table.FwListLen)/fwEntrySize)

	for i := range fw.Parts {
		buffer = make([]byte, partEntrySize)
		_, err := r.Read(buffer)
		if err != nil {
			return nil, fmt.Errorf("read part %d: %v", i, err)
		}
		part := &PartEntry{content: buffer}
		err = part.ReadContent()
		if err != nil {
			return nil, fmt.Errorf("part %d: %v", i, err)
		}
		fw.Parts[i] = part
	}

	for i := range fw.Fws {
		buffer = make([]byte, fwEntrySize)
		_, err := r.Read(buffer)
		if err != nil {
			return nil, fmt.Errorf("read part %d: %v", i, err)
		}
		f := &FwEntry{content: buffer}
		err = f.ReadContent()
		if err != nil {
			return nil, fmt.Errorf("firmware %d: %v", i, err)
		}
		fw.Fws[i] = f
	}

	return fw, nil
}
