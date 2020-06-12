package fwtable

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

var FwTypes = []string{
	"Reserved",
	"Bootcode",
	"Kernel",
	"RescueDeviceTree",
	"KernelDeviceTree",
	"RescueRootFS",
	"KernelRootFS",
	"Audio",
	"AudioFile",
	"VideoFile",
	"Ext4",
	"Ubifs",
	"Squash",
	"Ext3",
	"Odd",
	"YAFFS2",
	"ISO",
	"Swap",
	"NTFS",
	"JFFS2",
	"ImageFile",
	"ImageFile1",
	"ImageFile2",
	"AudioFile1",
	"AudioFile2",
	"VideoFile1",
	"VideoFile2",
	"Video",
	"Video2",
	"eCPU",
	"Tee",
	"GoldKernel",
	"GoldRescueDeviceTree",
	"GoldRescueRootfs",
	"GoldAudio",
	"GoldTee",
	"Config",
	"uBoot",
	"BL31",
	"Hypervisor",
	"GoldBL31",
	"RSAKeyFW",
	"RSAKeyTee",
	"RescueKernel",
	"RescueAudio",
	"RescueConfig",
	"Unknown",
}

// Size of the file structure in bytes
const fwEntrySize = 32

// FwEntry represents one firmware in the table.
type FwEntry struct {
	Type                    string
	Lzma                    bool
	ReadOnly                bool
	Version, Length         int
	Paddings, DiskOffset    int
	TargetAddress, Checksum uint32

	content []byte
	table   struct {
		Type          uint8
		Options       byte
		Version       uint32
		TargetAddress uint32
		Offset        uint32
		Length        uint32
		Paddings      uint32
		Checksum      uint32
		Reserved      [6]byte
	}
}

func (fw *FwTable) FindFwByType(fwType string) *FwEntry {
	var fwe *FwEntry
	for _, f := range fw.Fws {
		if f.Type == fwType {
			fwe = f
			break
		}
	}
	return fwe
}

func (fw *FwEntry) ReadFile(r io.ByteReader) error {
	// Start checksum
	cs := uint32(0)
	size := 0
	var currentError error
	for currentError == nil {
		b, currentError := r.ReadByte()
		if currentError == io.EOF {
			break
		}
		// Update checksum
		cs += uint32(b)
		size++
	}
	if currentError != nil {
		return fmt.Errorf("error reading at %d mark: %v", size, currentError)
	}

	fw.Checksum = cs
	fw.Length = size
	// Pad to the nearest 512
	fw.Paddings = size + 512 - (size % 512)
	return nil
}

func (fw *FwEntry) WriteContent() error {
	match := false
	for i, v := range FwTypes {
		if fw.Type == v {
			fw.table.Type = uint8(i)
			match = true
			break
		}
	}
	if !match {
		return fmt.Errorf("invalid type '%s'", fw.Type)
	}
	options := byte(0)
	if fw.Lzma {
		options |= (1 << 6)
	}
	if fw.ReadOnly {
		options |= (1 << 7)
	}
	fw.table.Options = options
	fw.table.Version = uint32(fw.Version)
	fw.table.TargetAddress = fw.TargetAddress
	fw.table.Offset = uint32(fw.DiskOffset)
	fw.table.Length = uint32(fw.Length)
	fw.table.Paddings = uint32(fw.Paddings)
	fw.table.Checksum = fw.Checksum
	for i := 0; i < 6; i++ {
		fw.table.Reserved[i] = 0
	}
	buffer := new(bytes.Buffer)
	err := binary.Write(buffer, binary.LittleEndian, &fw.table)
	if err != nil {
		return fmt.Errorf("writing: %v", err)
	}
	fw.content = buffer.Bytes()
	return nil
}

func (fw *FwEntry) ReadContent() error {
	err := binary.Read(bytes.NewReader(fw.content), binary.LittleEndian, &fw.table)
	if err != nil {
		return err
	}
	if int(fw.table.Type) >= len(FwTypes) {
		fw.Type = fmt.Sprintf("Invalid (%d)", fw.table.Type)
	} else {
		fw.Type = FwTypes[fw.table.Type]
	}
	fw.Lzma = (fw.table.Options & (1 << 6)) == (1 << 6)
	fw.ReadOnly = (fw.table.Options & (1 << 7)) == (1 << 7)
	fw.Version = int(fw.table.Version)
	fw.Length = int(fw.table.Length)
	fw.Paddings = int(fw.table.Paddings)
	fw.DiskOffset = int(fw.table.Offset)
	fw.TargetAddress = fw.table.TargetAddress
	fw.Checksum = fw.table.Checksum
	return nil
}
